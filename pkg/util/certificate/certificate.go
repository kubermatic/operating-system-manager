/*
Copyright 2022 The Operating System Manager contributors.

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

package certificate

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func getCACertFromKubeconfigPath(kubeconfigPath string) (string, error) {
	kubeconfig, err := fileToClientConfig(kubeconfigPath)
	if err != nil {
		return "", err
	}
	if len(kubeconfig.Clusters) != 1 {
		return "", fmt.Errorf("kubeconfig does not contain exactly one cluster, can not extract server address")
	}
	// Clusters is a map so we have to use range here
	for _, clusterConfig := range kubeconfig.Clusters {
		return string(clusterConfig.CertificateAuthorityData), nil
	}

	return "", fmt.Errorf("no CACert found")
}

func fileToClientConfig(kubeconfigPath string) (*clientcmdapi.Config, error) {
	return clientcmd.LoadFromFile(kubeconfigPath)
}

func GetCACert(kubeconfigPath string, config *rest.Config) (string, error) {
	if kubeconfigPath != "" {
		return getCACertFromKubeconfigPath(kubeconfigPath)
	}
	if config != nil && config.CAData != nil {
		return string(config.CAData), nil
	}
	return "", fmt.Errorf("no CA certificate found")
}
