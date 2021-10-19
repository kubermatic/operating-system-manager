/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

	"k8c.io/operating-system-manager/pkg/admission"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type options struct {
	masterURL              string
	kubeconfig             string
	seedMasterURL          string
	seedKubeconfig         string
	provider               string
	clusterNamespace       string
	admissionListenAddress string
	admissionTLSCertPath   string
	admissionTLSKeyPath    string
	caBundleFile           string
}

func main() {
	opt := &options{}

	klog.InitFlags(nil)
	if flag.Lookup("kubeconfig") == nil {
		flag.StringVar(&opt.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	}
	if flag.Lookup("master") == nil {
		flag.StringVar(&opt.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	}
	flag.StringVar(&opt.seedKubeconfig, "seed-kubeconfig", "", "Path to the seed kubeconfig. Only required if used in KKP.")
	flag.StringVar(&opt.seedMasterURL, "seed-master", "", "The address of the Kubernetes API server for seed cluster. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&opt.provider, "provider", "", "The provider hosting the Kubernetes cluster")
	flag.StringVar(&opt.clusterNamespace, "cluster-namespace", "kube-system", "The namespace in which the cluster control plane is deployed")
	flag.StringVar(&opt.admissionListenAddress, "listen-address", ":9876", "The address on which the MutatingWebhook will listen on")
	flag.StringVar(&opt.admissionTLSCertPath, "tls-cert-path", "/tmp/cert/cert.pem", "The path of the TLS cert for the MutatingWebhook")
	flag.StringVar(&opt.admissionTLSKeyPath, "tls-key-path", "/tmp/cert/key.pem", "The path of the TLS key for the MutatingWebhook")
	flag.StringVar(&opt.caBundleFile, "ca-bundle", "", "path to a file containing all PEM-encoded CA certificates (will be used instead of the host's certificates if set)")
	flag.Parse()

	opt.kubeconfig = flag.Lookup("kubeconfig").Value.String()
	opt.masterURL = flag.Lookup("master").Value.String()

	if opt.provider == "" {
		klog.Fatalf("--provider flag is mandatory")
	}

	var client, seedClient ctrlruntimeclient.Client
	var seedCfg, userCfg *rest.Config
	var seedScheme = runtime.NewScheme()

	// setup user config
	cfg, err := clientcmd.BuildConfigFromFlags(opt.masterURL, opt.kubeconfig)
	if err != nil {
		klog.Fatalf("error building kubeconfig: %v", err)
	}

	// setup seed config
	if opt.seedKubeconfig != "" {
		seedCfg, err = clientcmd.BuildConfigFromFlags(opt.seedMasterURL, opt.seedKubeconfig)
		if err != nil {
			klog.Fatalf("error building seed kubeconfig: %v", err)
		}
	} else {
		seedCfg = userCfg
	}

	// add OSM resources to seed scheme
	if err := v1alpha1.AddToScheme(seedScheme); err != nil {
		klog.Fatal(err)
	}
	// create seed client
	if seedClient, err = ctrlruntimeclient.New(seedCfg, ctrlruntimeclient.Options{
		Scheme: seedScheme,
		Mapper: nil,
	}); err != nil {
		klog.Fatalf("failed to build seed client: %v", err)
	}

	// create user client
	if client, err = ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{}); err != nil {
		klog.Fatalf("failed to build client: %v", err)
	}

	srv, err := admission.New(opt.admissionListenAddress, opt.provider, opt.clusterNamespace, client, seedClient)
	if err != nil {
		klog.Fatalf("failed to create admission hook: %v", err)
	}

	if err := srv.ListenAndServeTLS(opt.admissionTLSCertPath, opt.admissionTLSKeyPath); err != nil {
		klog.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if err := srv.Close(); err != nil {
			klog.Fatalf("Failed to shutdown server: %v", err)
		}
	}()
	klog.Infof("Listening on %s", opt.admissionListenAddress)
	select {}
}
