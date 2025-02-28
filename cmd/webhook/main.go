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
	"log"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	mdmutation "k8c.io/operating-system-manager/pkg/admission/machinedeployment/mutation"
	oscvalidation "k8c.io/operating-system-manager/pkg/admission/operatingsystemconfig/validation"
	ospvalidation "k8c.io/operating-system-manager/pkg/admission/operatingsystemprofile/validation"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	osmlog "k8c.io/operating-system-manager/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
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
	logFlags := osmlog.NewDefaultOptions()
	logFlags.AddFlags(flag.CommandLine)

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

	if err := logFlags.Validate(); err != nil {
		log.Fatalf("Invalid options: %v", err)
	}

	rawLog := osmlog.New(logFlags.Debug, logFlags.Format)
	log := rawLog.Sugar()
	// set the logger used by controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	if len(opt.namespace) == 0 {
		log.Fatal("-namespace is required")
	}
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		WebhookServer: webhook.NewServer(webhook.Options{
			CertDir: opt.certDir,
			Port:    9443,
		}),
		HealthProbeBindAddress:  opt.probeAddr,
		LeaderElection:          opt.enableLeaderElection,
		LeaderElectionNamespace: opt.namespace,
		LeaderElectionID:        "operating-system-manager-leader-lock",
		Metrics:                 metricsserver.Options{BindAddress: opt.metricsAddr},
		Scheme:                  scheme,
	})
	if err != nil {
		log.Fatal("failed to create the manager", zap.Error(err))
	}

	// Register webhooks
	oscvalidation.NewAdmissionHandler(log, scheme).SetupWebhookWithManager(mgr)
	ospvalidation.NewAdmissionHandler(log, scheme).SetupWebhookWithManager(mgr)
	mdmutation.NewAdmissionHandler(log, scheme).SetupWebhookWithManager(mgr)

	// Add health endpoints
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Fatalf("failed to add health check: %v", zap.Error(err))
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Fatalf("failed to add readiness check: %v", zap.Error(err))
	}

	klog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatalf("failed to start OSC controller: %v", zap.Error(err))
	}
}
