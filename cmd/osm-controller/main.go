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
	"strings"
	"time"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/operating-system-manager/pkg/controllers/osc"
	"k8c.io/operating-system-manager/pkg/controllers/osp"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"
	providerconfig "k8c.io/operating-system-manager/pkg/providerconfig/config"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
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
	nodeHTTPProxy         string
	nodeNoProxy           string
	nodePortRange         string
	podCidr               string

	clusterDNSIPs         string
	userClusterKubeconfig string

	healthProbeAddress string
	metricsAddress     string
}

const (
	defaultLeaderElectionNamespace = "kube-system"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(clusterv1alpha1.AddToScheme(scheme))
}

func main() {
	klog.InitFlags(nil)

	opt := &options{}

	flag.StringVar(&opt.userClusterKubeconfig, "user-cluster-kubeconfig", "", "Path to a user cluster kubeconfig where provisioning secrets are created")
	flag.IntVar(&opt.workerCount, "worker-count", 10, "Number of workers which process reconciliation in parallel.")
	flag.StringVar(&opt.clusterName, "cluster-name", "", "The cluster where the OSC will run.")
	flag.StringVar(&opt.namespace, "namespace", "", "The namespace where the OSC controller will run.")
	flag.StringVar(&opt.containerRuntime, "container-runtime", "containerd", "container runtime to deploy.")
	flag.BoolVar(&opt.externalCloudProvider, "external-cloud-provider", false, "cloud-provider Kubelet flag set to external.")
	flag.StringVar(&opt.clusterDNSIPs, "cluster-dns", "10.10.10.10", "Comma-separated list of DNS server IP address.")
	flag.StringVar(&opt.pauseImage, "pause-image", "", "pause image to use in Kubelet.")
	flag.StringVar(&opt.initialTaints, "initial-taints", "", "taints to use when creating the node.")
	flag.StringVar(&opt.nodeHTTPProxy, "node-http-proxy", "", "If set, it configures the 'HTTP_PROXY' & 'HTTPS_PROXY' environment variable on the nodes.")
	flag.StringVar(&opt.nodeNoProxy, "node-no-proxy", ".svc,.cluster.local,localhost,127.0.0.1", "If set, it configures the 'NO_PROXY' environment variable on the nodes.")
	flag.StringVar(&opt.podCidr, "pod-cidr", "172.25.0.0/16", "The network ranges from which POD networks are allocated")
	flag.StringVar(&opt.nodePortRange, "node-port-range", "30000-32767", "A port range to reserve for services with NodePort visibility")

	flag.StringVar(&opt.healthProbeAddress, "health-probe-address", "127.0.0.1:8085", "The address on which the liveness check on /healthz and readiness check on /readyz will be available")
	flag.StringVar(&opt.metricsAddress, "metrics-address", "127.0.0.1:8080", "The address on which Prometheus metrics will be available under /metrics")

	flag.Parse()

	if len(opt.namespace) == 0 {
		klog.Fatal("-namespace is required")
	}

	if len(opt.userClusterKubeconfig) == 0 {
		klog.Fatal("-user-cluster-kubeconfig is required")
	}

	if !(opt.containerRuntime == "docker" || opt.containerRuntime == "containerd") {
		klog.Fatalf("%s not supported; containerd, docker are the supported container runtimes", opt.containerRuntime)
	}

	opt.userClusterKubeconfig = flag.Lookup("user-cluster-kubeconfig").Value.(flag.Getter).Get().(string)

	parsedClusterDNSIPs, err := parseClusterDNSIPs(opt.clusterDNSIPs)
	if err != nil {
		klog.Fatalf("invalid cluster dns specified: %v", err)
	}

	mgr, err := createManager(opt)
	if err != nil {
		klog.Fatalf("failed to create runtime manager: %v", err)
	}

	// Instantiate ConfigVarResolver
	providerconfig.SetConfigVarResolver(context.Background(), mgr.GetClient(), opt.namespace)

	logger, err := zap.NewProduction()
	if err != nil {
		klog.Fatal(err)
	}
	log := logger.Sugar()

	// Setup OSP controller
	if err = (&osp.Reconciler{
		Client:      mgr.GetClient(),
		Log:         log,
		WorkerCount: opt.workerCount,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatalf("unable to create osp controller with err: %v", err)
	}

	// Build config for user cluster
	userCfg, err := clientcmd.BuildConfigFromFlags("", opt.userClusterKubeconfig)
	if err != nil {
		klog.Fatalf("error building user cluster kubeconfig: %v", err)
	}

	// Build client for user cluster
	userClient, err := ctrlruntimeclient.New(userCfg, ctrlruntimeclient.Options{})
	if err != nil {
		klog.Fatalf("failed to build user client: %v", err)
	}

	// Build dedicated clientset for informers
	userClientset := kubernetes.NewForConfigOrDie(userCfg)

	// Build informers for OSC controller
	userInformerFactory := informers.NewSharedInformerFactory(userClientset, time.Minute*30)
	err = mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		userInformerFactory.Start(ctx.Done())
		userInformerFactory.WaitForCacheSync(ctx.Done())
		return nil
	}))
	if err != nil {
		klog.Fatalf("error adding InformerFactory to the manager: %v", err)
	}

	// Setup OSC controller
	if err = (&osc.Reconciler{
		Client:                mgr.GetClient(),
		Log:                   log,
		UserClient:            userClient,
		WorkerCount:           opt.workerCount,
		Namespace:             osc.CloudInitSettingsNamespace,
		ClusterAddress:        opt.clusterName,
		ContainerRuntime:      opt.containerRuntime,
		ExternalCloudProvider: opt.externalCloudProvider,
		PauseImage:            opt.pauseImage,
		InitialTaints:         opt.initialTaints,
		Generator:             generator.NewDefaultCloudConfigGenerator(""),
		ClusterDNSIPs:         parsedClusterDNSIPs,
		UserClusterKubeconfig: opt.userClusterKubeconfig,
		NodeHTTPProxy:         opt.nodeHTTPProxy,
		NodeNoProxy:           opt.nodeNoProxy,
		PodCIDR:               opt.podCidr,
		NodePortRange:         opt.nodePortRange,
	}).SetupWithManager(mgr); err != nil {
		klog.Fatalf("unable to create osc controller with err: %v", err)
	}

	log.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		klog.Fatalf("Failed to start OSC controller: %v", zap.Error(err))
	}
}

func createManager(opt *options) (manager.Manager, error) {
	// Manager options
	options := manager.Options{
		Scheme: scheme,
		// TODO(waleed) make it work
		LeaderElection:          false,
		LeaderElectionID:        "operating-system-manager",
		LeaderElectionNamespace: defaultLeaderElectionNamespace,
		HealthProbeBindAddress:  opt.healthProbeAddress,
		MetricsBindAddress:      opt.metricsAddress,
		Port:                    9443,
		Namespace:               opt.namespace, // namespaced-scope when the value is not an empty string
	}

	mgr, err := manager.New(config.GetConfigOrDie(), options)
	if err != nil {
		return nil, fmt.Errorf("error building ctrlruntime manager: %v", err)
	}

	// Add health endpoints
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("failed to add health check: %v", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("failed to add readiness check: %v", err)
	}
	return mgr, nil
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
