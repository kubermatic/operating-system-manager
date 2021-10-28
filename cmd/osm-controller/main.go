/*
Copyright 2021 The Operating System Manager contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/operating-system-manager/pkg/controllers/osc"
	"k8c.io/operating-system-manager/pkg/controllers/osp"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"

	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

type options struct {
	workerCount           int
	namespace             string
	clusterName           string
	containerRuntime      string
	externalCloudProvider bool
	pauseImage            string
	initialTaints         string

	clusterDNSIPs string
	kubeconfig    string
}

func main() {
	klog.InitFlags(nil)

	opt := &options{}

	if flag.Lookup("kubeconfig") == nil {
		flag.StringVar(&opt.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	}

	flag.IntVar(&opt.workerCount, "worker-count", 10, "Number of workers which process reconciliation in parallel.")
	flag.StringVar(&opt.clusterName, "cluster-name", "", "The cluster where the OSC will run.")
	flag.StringVar(&opt.namespace, "namespace", "", "The namespace where the OSC controller will run.")
	flag.StringVar(&opt.containerRuntime, "container-runtime", "containerd", "container runtime to deploy.")
	flag.BoolVar(&opt.externalCloudProvider, "external-cloud-provider", false, "cloud-provider Kubelet flag set to external.")
	flag.StringVar(&opt.clusterDNSIPs, "cluster-dns", "10.10.10.10", "Comma-separated list of DNS server IP address.")
	flag.StringVar(&opt.pauseImage, "pause'image", "", "pause image to use in Kubelet.")
	flag.StringVar(&opt.initialTaints, "initial-taints", "", "taints to use when creating the node.")

	flag.Parse()

	if len(opt.namespace) == 0 {
		klog.Fatal("-namespace is required")
	}
	opt.kubeconfig = flag.Lookup("kubeconfig").Value.(flag.Getter).Get().(string)

	if err := validateClusterDNSIPs(opt.clusterDNSIPs); err != nil {
		klog.Fatalf("invalid cluster dns specified: %v", err)
	}

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		klog.Error(err, "could not create manager")
		os.Exit(1)
	}
	if err = v1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Fatal(err)
	}
	// because we watch MachineDeployments
	if err = clusterv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Fatal(err)
	}
	logger, err := zap.NewProduction()
	if err != nil {
		klog.Fatal(err)
	}

	log := logger.Sugar()

	if err := osp.Add(mgr, log, opt.namespace, opt.workerCount); err != nil {
		klog.Fatal(err)
	}

	if err := osc.Add(
		mgr,
		log,
		opt.namespace,
		opt.clusterName,
		opt.workerCount,
		opt.clusterDNSIPs,
		opt.kubeconfig,
		generator.NewDefaultCloudInitGenerator(""),
		opt.containerRuntime,
		opt.externalCloudProvider,
		opt.pauseImage,
		opt.initialTaints,
	); err != nil {
		klog.Fatal(err)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		klog.Fatalf("Failed to start OSC controller: %v", zap.Error(err))
	}
}

func validateClusterDNSIPs(s string) error {
	sips := strings.Split(s, ",")
	for _, sip := range sips {
		ip := net.ParseIP(strings.TrimSpace(sip))
		if ip == nil {
			return fmt.Errorf("unable to parse ip %s", sip)
		}
	}
	return nil
}
