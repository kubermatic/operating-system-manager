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

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/containerruntime"
	"k8c.io/operating-system-manager/pkg/controllers/osc"
	"k8c.io/operating-system-manager/pkg/controllers/osp"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"
	providerconfig "k8c.io/operating-system-manager/pkg/providerconfig/config"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type options struct {
	workerCount             int
	namespace               string
	clusterName             string
	containerRuntime        string
	externalCloudProvider   bool
	pauseImage              string
	initialTaints           string
	nodePortRange           string
	podCidr                 string
	enableLeaderElection    bool
	clusterDNSIPs           string
	workerClusterKubeconfig string
	kubeconfig              string

	healthProbeAddress       string
	metricsAddress           string
	workerHealthProbeAddress string
	workerMetricsAddress     string

	// Flags for configuring CRI
	nodeInsecureRegistries        string
	nodeRegistryMirrors           string
	nodeRegistryCredentialsSecret string
	nodeContainerdRegistryMirrors containerruntime.RegistryMirrorsFlags

	// Flags for proxy
	nodeHTTPProxy string
	nodeNoProxy   string
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme.Scheme))
	utilruntime.Must(osmv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clusterv1alpha1.AddToScheme(scheme.Scheme))
}

func main() {
	klog.InitFlags(nil)

	opt := &options{}

	if flag.Lookup("kubeconfig") == nil {
		flag.StringVar(&opt.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	}
	flag.StringVar(&opt.workerClusterKubeconfig, "worker-cluster-kubeconfig", "", "Path to kubeconfig of cluster where provisioning secrets are created")
	flag.IntVar(&opt.workerCount, "worker-count", 10, "Number of workers which process reconciliation in parallel.")
	flag.StringVar(&opt.namespace, "namespace", "", "The namespace where the OSC controller will run.")
	flag.StringVar(&opt.containerRuntime, "container-runtime", "containerd", "container runtime to deploy.")
	flag.BoolVar(&opt.externalCloudProvider, "external-cloud-provider", false, "cloud-provider Kubelet flag set to external.")
	flag.StringVar(&opt.clusterDNSIPs, "cluster-dns", "10.10.10.10", "Comma-separated list of DNS server IP address.")
	flag.StringVar(&opt.pauseImage, "pause-image", "", "pause image to use in Kubelet.")
	flag.StringVar(&opt.initialTaints, "initial-taints", "", "taints to use when creating the node.")

	flag.StringVar(&opt.podCidr, "pod-cidr", "172.25.0.0/16", "The network ranges from which POD networks are allocated")
	flag.StringVar(&opt.nodePortRange, "node-port-range", "30000-32767", "A port range to reserve for services with NodePort visibility")

	flag.StringVar(&opt.nodeHTTPProxy, "node-http-proxy", "", "If set, it configures the 'HTTP_PROXY' & 'HTTPS_PROXY' environment variable on the nodes.")
	flag.StringVar(&opt.nodeNoProxy, "node-no-proxy", ".svc,.cluster.local,localhost,127.0.0.1", "If set, it configures the 'NO_PROXY' environment variable on the nodes.")
	flag.StringVar(&opt.nodeInsecureRegistries, "node-insecure-registries", "", "Comma separated list of registries which should be configured as insecure on the container runtime")
	flag.StringVar(&opt.nodeRegistryMirrors, "node-registry-mirrors", "", "Comma separated list of Docker image mirrors")

	flag.StringVar(&opt.healthProbeAddress, "health-probe-address", "127.0.0.1:8085", "The address on which the liveness check on /healthz and readiness check on /readyz will be available")
	flag.StringVar(&opt.metricsAddress, "metrics-address", "127.0.0.1:8080", "The address on which Prometheus metrics will be available under /metrics")

	flag.StringVar(&opt.workerHealthProbeAddress, "worker-health-probe-address", "127.0.0.1:8086", "For worker manager, the address on which the liveness check on /healthz and readiness check on /readyz will be available")
	flag.StringVar(&opt.workerMetricsAddress, "worker-metrics-address", "127.0.0.1:8081", "For worker manager, the address on which Prometheus metrics will be available under /metrics")
	flag.BoolVar(&opt.enableLeaderElection, "leader-elect", true, "Enable leader election for controller manager.")

	flag.Parse()

	if len(opt.namespace) == 0 {
		klog.Fatal("-namespace is required")
	}

	if !(opt.containerRuntime == "docker" || opt.containerRuntime == "containerd") {
		klog.Fatalf("%s not supported; containerd, docker are the supported container runtimes", opt.containerRuntime)
	}

	opt.kubeconfig = flag.Lookup("kubeconfig").Value.(flag.Getter).Get().(string)

	// out-of-cluster config was not provided using the flag, try to use the in-cluster config.
	if opt.kubeconfig == "" {
		opt.kubeconfig = getKubeConfigPath()
	}

	// Parse flags
	parsedClusterDNSIPs, err := parseClusterDNSIPs(opt.clusterDNSIPs)
	if err != nil {
		klog.Fatalf("invalid cluster dns specified: %v", err)
	}

	containerRuntimeOpts := containerruntime.Opts{
		ContainerRuntime:          opt.containerRuntime,
		ContainerdRegistryMirrors: opt.nodeContainerdRegistryMirrors,
		InsecureRegistries:        opt.nodeInsecureRegistries,
		PauseImage:                opt.pauseImage,
		RegistryMirrors:           opt.nodeRegistryMirrors,
		RegistryCredentialsSecret: opt.nodeRegistryCredentialsSecret,
	}
	containerRuntimeConfig, err := containerruntime.BuildConfig(containerRuntimeOpts)
	if err != nil {
		klog.Fatalf("failed to generate container runtime config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		klog.Fatal(err)
	}
	log := logger.Sugar()

	// Create manager with client against in-cluster config
	mgr, err := createManager(opt)
	if err != nil {
		klog.Fatalf("failed to create runtime manager: %v", err)
	}

	// Start with assuming that current cluster will be used as worker cluster
	workerMgr := mgr
	workerClient := mgr.GetClient()

	// Handling for worker cluster
	if opt.workerClusterKubeconfig != "" {
		workerClusterConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: opt.workerClusterKubeconfig},
			&clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			klog.Fatal(err)
		}

		// Build dedicated client for worker cluster, some read actions fail on the split client created by manager due to informers not syncing in-time
		workerClient, err = ctrlruntimeclient.New(workerClusterConfig, ctrlruntimeclient.Options{
			Scheme: scheme.Scheme,
		})
		if err != nil {
			klog.Fatalf("failed to build worker client: %v", err)
		}

		workerMgr, err = manager.New(workerClusterConfig, manager.Options{
			LeaderElection:   opt.enableLeaderElection,
			LeaderElectionID: "operating-system-manager-worker-manager",
			// We use hard-coded namespace kube-system here since manager uses worker cluster config
			LeaderElectionNamespace: "kube-system",
			HealthProbeBindAddress:  opt.workerHealthProbeAddress,
			MetricsBindAddress:      opt.workerMetricsAddress,
			Port:                    9444,
		})
		if err != nil {
			klog.Fatal(err)
		}

		// "-worker-cluster-kubeconfig" was not empty and a valid kubeconfig was provided,
		// point workerClient to the external cluster
		// Use workerClusterKubeconfig since the machines will exist on that cluster
		opt.kubeconfig = opt.workerClusterKubeconfig

		if err := mgr.Add(workerMgr); err != nil {
			klog.Fatal("failed to add workers cluster mgr to main mgr", zap.Error(err))
		}
	}

	// Instantiate ConfigVarResolver
	providerconfig.SetConfigVarResolver(context.Background(), workerMgr.GetClient(), opt.namespace)

	// Setup OSP controller
	if err := osp.Add(mgr, log, opt.namespace, opt.workerCount); err != nil {
		klog.Fatal(err)
	}

	// Setup OSC controller
	if err := osc.Add(
		workerMgr,
		log,
		workerClient,
		mgr.GetClient(),
		opt.kubeconfig,
		opt.namespace,
		opt.workerCount,
		parsedClusterDNSIPs,
		generator.NewDefaultCloudConfigGenerator(""),
		opt.containerRuntime,
		opt.externalCloudProvider,
		opt.pauseImage,
		opt.initialTaints,
		opt.nodeHTTPProxy,
		opt.nodeNoProxy,
		opt.nodePortRange,
		opt.podCidr,
		containerRuntimeConfig,
		opt.nodeRegistryCredentialsSecret,
	); err != nil {
		klog.Fatal(err)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatalf("Failed to start OSC controller: %v", zap.Error(err))
	}
}
