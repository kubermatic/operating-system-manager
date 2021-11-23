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
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path"
	"strings"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/operating-system-manager/pkg/controllers/osc"
	"k8c.io/operating-system-manager/pkg/controllers/osp"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"
	providerconfig "k8c.io/operating-system-manager/pkg/providerconfig/config"

	"k8s.io/client-go/util/homedir"
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
	cniVersion            string
	containerdVersion     string
	nodeHTTPProxy         string
	nodeNoProxy           string

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
	flag.StringVar(&opt.pauseImage, "pause-image", "", "pause image to use in Kubelet.")
	flag.StringVar(&opt.initialTaints, "initial-taints", "", "taints to use when creating the node.")
	flag.StringVar(&opt.cniVersion, "cni-version", "", "CNI version to use in the cluster.")
	flag.StringVar(&opt.containerdVersion, "containerd-version", "", "Containerd version to use in the cluster.")
	flag.StringVar(&opt.nodeHTTPProxy, "node-http-proxy", "", "If set, it configures the 'HTTP_PROXY' & 'HTTPS_PROXY' environment variable on the nodes.")
	flag.StringVar(&opt.nodeNoProxy, "node-no-proxy", ".svc,.cluster.local,localhost,127.0.0.1", "If set, it configures the 'NO_PROXY' environment variable on the nodes.")

	flag.Parse()

	if len(opt.namespace) == 0 {
		klog.Fatal("-namespace is required")
	}

	if !(opt.containerRuntime == "docker" || opt.containerRuntime == "containerd") {
		klog.Fatalf("%s not supported; containerd, docker are the supported container runtimes", opt.containerRuntime)
	}
	if len(opt.cniVersion) == 0 {
		klog.Fatal("-cni-version is required")
	}
	if len(opt.containerdVersion) == 0 {
		klog.Fatal("-containerd-version is required")
	}

	opt.kubeconfig = flag.Lookup("kubeconfig").Value.(flag.Getter).Get().(string)

	// out-of-cluster config was not provided using the flag, try to use the in-cluster config.
	if opt.kubeconfig == "" {
		opt.kubeconfig = getKubeConfigPath()
	}

	parsedClusterDNSIPs, err := parseClusterDNSIPs(opt.clusterDNSIPs)
	if err != nil {
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

	// Instantiate ConfigVarResolver
	providerconfig.SetConfigVarResolver(context.Background(), mgr.GetClient(), opt.namespace)

	if err := osc.Add(
		mgr,
		log,
		osc.CloudInitSettingsNamespace,
		opt.clusterName,
		opt.workerCount,
		parsedClusterDNSIPs,
		opt.kubeconfig,
		generator.NewDefaultCloudConfigGenerator(""),
		opt.containerRuntime,
		opt.externalCloudProvider,
		opt.pauseImage,
		opt.initialTaints,
		opt.cniVersion,
		opt.containerdVersion,
		opt.nodeHTTPProxy,
		opt.nodeNoProxy,
	); err != nil {
		klog.Fatal(err)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		klog.Fatalf("Failed to start OSC controller: %v", zap.Error(err))
	}
}

func parseClusterDNSIPs(s string) ([]net.IP, error) {
	var ips []net.IP
	sips := strings.Split(s, ",")
	for _, sip := range sips {
		ip := net.ParseIP(strings.TrimSpace(sip))
		if ip == nil {
			return nil, fmt.Errorf("unable to parse ip %s", sip)
		}
		ips = append(ips, ip)
	}
	return ips, nil
}

// getKubeConfigPath returns the path to the kubeconfig file.
func getKubeConfigPath() string {
	if os.Getenv("KUBECONFIG") != "" {
		return os.Getenv("KUBECONFIG")
	}
	return path.Join(homedir.HomeDir(), ".kube/config")
}
