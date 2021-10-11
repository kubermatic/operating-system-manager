package resources

import (
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func GetCACert(kubeconfigPath string) (string, error) {
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

// StringifyKubeconfig marshals a kubeconfig to its text form
func StringifyKubeconfig(kubeconfigPath string) (string, error) {
	kubeconfig, err := fileToClientConfig(kubeconfigPath)
	if err != nil {
		return "", err
	}
	kubeconfigBytes, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error writing kubeconfig: %v", err)
	}

	return string(kubeconfigBytes), nil
}
