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

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/util"
	"k8c.io/operating-system-manager/pkg/admission"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type options struct {
	kubeconfig            string
	userClusterKubeconfig string
	namespace             string

	admissionListenAddress string
	admissionTLSCertPath   string
	admissionTLSKeyPath    string
	caBundleFile           string
}

func main() {
	klog.InitFlags(nil)

	opt := &options{}

	if flag.Lookup("kubeconfig") == nil {
		flag.StringVar(&opt.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	}

	flag.StringVar(&opt.userClusterKubeconfig, "user-cluster-kubeconfig", "", "Path to a user cluster's kubeconfig")
	flag.StringVar(&opt.admissionListenAddress, "listen-address", ":9876", "The address on which the MutatingWebhook will listen on")
	flag.StringVar(&opt.admissionTLSCertPath, "tls-cert-path", "/tmp/cert/cert.pem", "The path of the TLS cert for the MutatingWebhook")
	flag.StringVar(&opt.admissionTLSKeyPath, "tls-key-path", "/tmp/cert/key.pem", "The path of the TLS key for the MutatingWebhook")
	flag.StringVar(&opt.caBundleFile, "ca-bundle", "", "path to a file containing all PEM-encoded CA certificates (will be used instead of the host's certificates if set)")
	flag.StringVar(&opt.namespace, "namespace", "", "The namespace where the OSC controller will run.")
	flag.Parse()

	// User Cluster kubeconfig is required
	if len(opt.userClusterKubeconfig) == 0 {
		klog.Fatal("-user-cluster-kubeconfig is required")
	}

	// Namespace is required
	if len(opt.namespace) == 0 {
		klog.Fatal("-namespace is required")
	}

	opt.kubeconfig = flag.Lookup("kubeconfig").Value.(flag.Getter).Get().(string)

	if opt.caBundleFile != "" {
		if err := util.SetCABundleFile(opt.caBundleFile); err != nil {
			klog.Fatalf("-ca-bundle is invalid: %v", err)
		}
	}

	var (
		err                    error
		seedCfg, userCfg       *rest.Config
		seedClient, userClient ctrlruntimeclient.Client
	)

	// build config for seed cluster
	seedCfg, err = clientcmd.BuildConfigFromFlags("", opt.kubeconfig)
	if err != nil {
		klog.Fatalf("error building kubeconfig: %v", err)
	}

	// build client for seed cluster
	seedClient, err = ctrlruntimeclient.New(seedCfg, ctrlruntimeclient.Options{})
	if err != nil {
		klog.Fatalf("failed to build seed client: %v", err)
	}

	// build config for user cluster
	userCfg, err = clientcmd.BuildConfigFromFlags("", opt.userClusterKubeconfig)
	if err != nil {
		klog.Fatalf("error building user cluster kubeconfig: %v", err)
	}

	// build client for user cluster
	userClient, err = ctrlruntimeclient.New(userCfg, ctrlruntimeclient.Options{})
	if err != nil {
		klog.Fatalf("failed to build user client: %v", err)
	}

	srv, err := admission.New(opt.admissionListenAddress, opt.namespace, userClient, seedClient)
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
