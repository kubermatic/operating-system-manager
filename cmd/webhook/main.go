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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/operating-system-manager/pkg/admission"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type options struct {
	ospNamespace string

	admissionListenAddress string
	admissionTLSCertPath   string
	admissionTLSKeyPath    string
}

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

	flag.StringVar(&opt.admissionListenAddress, "listen-address", ":9876", "The address on which the MutatingWebhook will listen on")
	flag.StringVar(&opt.admissionTLSCertPath, "tls-cert-path", "/home/waleed/src/local/osm/cert/cert.pem", "The path of the TLS cert for the MutatingWebhook")
	flag.StringVar(&opt.admissionTLSKeyPath, "tls-key-path", "/home/waleed/src/local/osm/cert/key.pem", "The path of the TLS key for the MutatingWebhook")
	flag.StringVar(&opt.ospNamespace, "osp-namespace", "kubermatic", "The namespace where the OSPs will exist")
	flag.Parse()

	// Build config for in-cluster cluster
	cfg, err := config.GetConfig()
	if err != nil {
		klog.Fatalf("error building kubeconfig: %v", err)
	}

	// Build client against in-cluster config
	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		klog.Fatalf("failed to build seed client: %v", err)
	}

	srv, err := admission.New(opt.admissionListenAddress, opt.ospNamespace, client)
	if err != nil {
		klog.Fatalf("failed to create admission hook: %v", err)
	}

	klog.Infof("starting webhook server on %s", opt.admissionListenAddress)

	if err := srv.ListenAndServeTLS(opt.admissionTLSCertPath, opt.admissionTLSKeyPath); err != nil {
		klog.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if err := srv.Close(); err != nil {
			klog.Fatalf("failed to shutdown server: %v", err)
		}
	}()
	klog.Infof("Listening on %s", opt.admissionListenAddress)
	select {}
}
