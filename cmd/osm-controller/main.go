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
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/containerruntime"
	"k8c.io/operating-system-manager/pkg/bootstrap"
	"k8c.io/operating-system-manager/pkg/clusterinfo"
	"k8c.io/operating-system-manager/pkg/controllers/osc"
	"k8c.io/operating-system-manager/pkg/controllers/osp"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"
	providerconfig "k8c.io/operating-system-manager/pkg/providerconfig/config"
	"k8c.io/operating-system-manager/pkg/resources/reconciling"
	"k8c.io/operating-system-manager/pkg/util/certificate"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type options struct {
	workerCount             int
	namespace               string
	containerRuntime        string
	externalCloudProvider   bool
	pauseImage              string
	initialTaints           string
	enableLeaderElection    bool
	clusterDNSIPs           string
	workerClusterKubeconfig string
	kubeconfig              string
	kubeletFeatureGates     string

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

	overrideBootstrapKubeletAPIServer string
	bootstrapTokenServiceAccountName  string
	caBundleFile                      string
}

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(osmv1alpha1.AddToScheme(scheme))
	utilruntime.Must(clusterv1alpha1.AddToScheme(scheme))
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

	flag.StringVar(&opt.kubeletFeatureGates, "node-kubelet-feature-gates", "RotateKubeletServerCertificate=true", "Feature gates to set on the kubelet")

	flag.StringVar(&opt.nodeHTTPProxy, "node-http-proxy", "", "If set, it configures the 'HTTP_PROXY' & 'HTTPS_PROXY' environment variable on the nodes.")
	flag.StringVar(&opt.nodeNoProxy, "node-no-proxy", ".svc,.cluster.local,localhost,127.0.0.1", "If set, it configures the 'NO_PROXY' environment variable on the nodes.")
	flag.StringVar(&opt.nodeInsecureRegistries, "node-insecure-registries", "", "Comma separated list of registries which should be configured as insecure on the container runtime")
	flag.StringVar(&opt.nodeRegistryMirrors, "node-registry-mirrors", "", "Comma separated list of Docker image mirrors")

	if opt.nodeContainerdRegistryMirrors == nil {
		opt.nodeContainerdRegistryMirrors = containerruntime.RegistryMirrorsFlags{}
	}
	flag.Var(&opt.nodeContainerdRegistryMirrors, "node-containerd-registry-mirrors", "Configure registry mirrors endpoints. Can be used multiple times to specify multiple mirrors")
	flag.StringVar(&opt.nodeRegistryCredentialsSecret, "node-registry-credentials-secret", "", "A Secret object reference, that contains auth info for image registry in namespace/secret-name form, example: kube-system/registry-credentials. See doc at https://github.com/kubermaric/machine-controller/blob/main/docs/registry-authentication.md")

	flag.StringVar(&opt.healthProbeAddress, "health-probe-address", "127.0.0.1:8085", "The address on which the liveness check on /healthz and readiness check on /readyz will be available")
	flag.StringVar(&opt.metricsAddress, "metrics-address", "127.0.0.1:8080", "The address on which Prometheus metrics will be available under /metrics")

	flag.StringVar(&opt.workerHealthProbeAddress, "worker-health-probe-address", "127.0.0.1:8086", "For worker manager, the address on which the liveness check on /healthz and readiness check on /readyz will be available")
	flag.StringVar(&opt.workerMetricsAddress, "worker-metrics-address", "127.0.0.1:8081", "For worker manager, the address on which Prometheus metrics will be available under /metrics")
	flag.BoolVar(&opt.enableLeaderElection, "leader-elect", true, "Enable leader election for controller manager.")
	flag.StringVar(&opt.overrideBootstrapKubeletAPIServer, "override-bootstrap-kubelet-apiserver", "", "Override for the API server address used in worker nodes bootstrap-kubelet.conf")
	flag.StringVar(&opt.bootstrapTokenServiceAccountName, "bootstrap-token-service-account-name", "", "When set use the service account token from this SA as bootstrap token instead of creating a temporary one. Passed in namespace/name format")
	flag.StringVar(&opt.caBundleFile, "ca-bundle", "", "Path to a file containing all PEM-encoded CA certificates. Will be used for Kubernetes CA certificates.")

	flag.Parse()

	if len(opt.namespace) == 0 {
		klog.Fatal("-namespace is required")
	}

	if opt.containerRuntime != "containerd" {
		klog.Fatalf("%s not supported; containerd is the only supported container runtimes", opt.containerRuntime)
	}

	var (
		err          error
		customCACert string
	)

	if len(opt.caBundleFile) > 0 {
		customCACert, err = retrieveCustomCACertificate(opt.caBundleFile)
		if err != nil {
			klog.Fatalf("-ca-bundle is invalid: %s", err.Error())
		}
	}

	opt.kubeconfig = flag.Lookup("kubeconfig").Value.(flag.Getter).Get().(string)

	// Parse flags
	parsedClusterDNSIPs, err := parseClusterDNSIPs(opt.clusterDNSIPs)
	if err != nil {
		klog.Fatalf("invalid cluster dns specified: %v", err)
	}

	parsedKubeletFeatureGates, err := parseKubeletFeatureGates(opt.kubeletFeatureGates)
	if err != nil {
		klog.Fatalf("invalid kubelet feature gates specified: %v", err)
	}

	var bootstrapTokenServiceAccountName *types.NamespacedName
	if opt.bootstrapTokenServiceAccountName != "" {
		flagParts := strings.Split(opt.bootstrapTokenServiceAccountName, "/")
		if flagPartsLen := len(flagParts); flagPartsLen != 2 {
			klog.Fatalf("splitting the bootstrap-token-service-account-name flag value in '/' returned %d parts, expected exactly two", flagPartsLen)
		}
		bootstrapTokenServiceAccountName = &types.NamespacedName{Namespace: flagParts[0], Name: flagParts[1]}
	}

	// Build container-runtime configuration
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

	// Configure logger
	logger, err := zap.NewProduction()
	if err != nil {
		klog.Fatal(err)
	}
	log := logger.Sugar()

	reconciling.Configure(log)

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
			Scheme: scheme,
		})
		if err != nil {
			klog.Fatalf("failed to build worker client: %v", err)
		}

		workerMgr, err = manager.New(workerClusterConfig, manager.Options{
			Scheme:           scheme,
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

	caCert := customCACert
	if opt.caBundleFile == "" {
		caCert, err = certificate.GetCACert(opt.kubeconfig, mgr.GetConfig())
		if err != nil {
			klog.Fatal("failed to load CA certificate", zap.Error(err))
		}
	}

	kubeconfigProvider := clusterinfo.New(workerClient, caCert)
	bootstrappingManager := bootstrap.New(workerClient, kubeconfigProvider, bootstrapTokenServiceAccountName, opt.overrideBootstrapKubeletAPIServer)

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
		bootstrappingManager,
		caCert,
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
		containerRuntimeConfig,
		opt.nodeRegistryCredentialsSecret,
		parsedKubeletFeatureGates,
	); err != nil {
		klog.Fatal(err)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatalf("Failed to start OSC controller: %v", zap.Error(err))
	}
}

func createManager(opt *options) (manager.Manager, error) {
	// Manager options
	options := manager.Options{
		Scheme:                  scheme,
		LeaderElection:          opt.enableLeaderElection,
		LeaderElectionID:        "operating-system-manager",
		LeaderElectionNamespace: opt.namespace,
		HealthProbeBindAddress:  opt.healthProbeAddress,
		MetricsBindAddress:      opt.metricsAddress,
		Port:                    9443,
	}

	if opt.workerClusterKubeconfig != "" {
		options.Namespace = opt.namespace
	}

	mgr, err := manager.New(config.GetConfigOrDie(), options)
	if err != nil {
		return nil, fmt.Errorf("error building ctrlruntime manager: %w", err)
	}

	// Add health endpoints
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("failed to add health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("failed to add readiness check: %w", err)
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

func parseKubeletFeatureGates(s string) (map[string]bool, error) {
	featureGates := map[string]bool{}
	sFeatureGates := strings.Split(s, ",")

	for _, featureGate := range sFeatureGates {
		sFeatureGate := strings.Split(featureGate, "=")
		if len(sFeatureGate) != 2 {
			return nil, fmt.Errorf("invalid kubelet feature gate: %q", featureGate)
		}

		featureGateEnabled, err := strconv.ParseBool(sFeatureGate[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse kubelet feature gate: %q", featureGate)
		}

		featureGates[sFeatureGate[0]] = featureGateEnabled
	}
	// Feature gate RotateKubeletServerCertificate is always enforced as a default
	if _, ok := featureGates["RotateKubeletServerCertificate"]; !ok {
		featureGates["RotateKubeletServerCertificate"] = true
	}
	return featureGates, nil
}

func retrieveCustomCACertificate(filepath string) (string, error) {
	cert, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %q: %w", filepath, err)
	}

	caBundle := x509.NewCertPool()
	if !caBundle.AppendCertsFromPEM(cert) {
		return "", errors.New("file does not contain valid PEM-encoded certificates")
	}

	return string(cert), err
}
