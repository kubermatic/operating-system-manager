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

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	mdmutation "k8c.io/operating-system-manager/pkg/admission/machinedeployment/mutation"
	oscvalidation "k8c.io/operating-system-manager/pkg/admission/operatingsystemconfig/validation"
	ospvalidation "k8c.io/operating-system-manager/pkg/admission/operatingsystemprofile/validation"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type options struct {
	namespace string

	metricsAddr          string
	enableLeaderElection bool
	probeAddr            string
	certDir              string
}

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(clusterv1alpha1.AddToScheme(scheme))
}

func main() {
	klog.InitFlags(nil)

	opt := &options{}

	flag.StringVar(&opt.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&opt.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&opt.enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&opt.namespace, "namespace", "", "The namespace where the OSC webhook will run.")
	flag.StringVar(&opt.certDir, "cert-dir", "/tmp/k8s-webhook-server/serving-certs",
		"Directory that contains the server key(tls.key) and certificate(tls.crt).")
	flag.Parse()

	if len(opt.namespace) == 0 {
		klog.Fatal("-namespace is required")
	}

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		CertDir:                 opt.certDir,
		HealthProbeBindAddress:  opt.probeAddr,
		LeaderElection:          opt.enableLeaderElection,
		LeaderElectionNamespace: opt.namespace,
		LeaderElectionID:        "operating-system-manager-leader-lock",
		MetricsBindAddress:      opt.metricsAddr,
		Port:                    9443,
		Scheme:                  scheme,
	})
	if err != nil {
		klog.Fatal("failed to create the manager", zap.Error(err))
	}

	// Register webhooks
	oscvalidation.NewAdmissionHandler().SetupWebhookWithManager(mgr)
	ospvalidation.NewAdmissionHandler().SetupWebhookWithManager(mgr)
	mdmutation.NewAdmissionHandler().SetupWebhookWithManager(mgr)

	// Add health endpoints
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		klog.Fatalf("failed to add health check: %v", zap.Error(err))
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		klog.Fatalf("failed to add readiness check: %v", zap.Error(err))
	}

	klog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatalf("failed to start OSC controller: %v", zap.Error(err))
	}
}
