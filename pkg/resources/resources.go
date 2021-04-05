package resources

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func GetServerAddressFromKubeconfig(kubeconfigPath string) (string, error) {
	kubeconfig, err := fileToClientConfig(kubeconfigPath)
	if err != nil {
		return "", err
	}
	if len(kubeconfig.Clusters) != 1 {
		return "", fmt.Errorf("kubeconfig does not contain exactly one cluster, can not extract server address")
	}
	// Clusters is a map so we have to use range here
	for _, clusterConfig := range kubeconfig.Clusters {
		return strings.Replace(clusterConfig.Server, "https://", "", -1), nil
	}

	return "", fmt.Errorf("no server address found")

}

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

func GetOSMBootstrapUserdata(kubeconfigPath string, machineName string, secretName string) (string, error) {
	kubeconfig, err := fileToClientConfig(kubeconfigPath)
	if err != nil {
		return "", err
	}
	var clusterName string
	for key := range kubeconfig.Clusters {
		clusterName = key
	}
	data := struct {
		Token      string
		SecretName string
		ServerURL  string
	}{
		Token:      kubeconfig.AuthInfos[kubeconfig.CurrentContext].Token,
		SecretName: secretName,
		ServerURL:  kubeconfig.Clusters[clusterName].Server,
	}
	bsScript, err := template.New("bootstrap-cloud-init").Parse(bootstrapBinContentTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse bootstrapBinContentTemplate template: %v", err)
	}
	script := &bytes.Buffer{}
	err = bsScript.Execute(script, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute bootstrapBinContentTemplate template: %v", err)
	}
	bsCloudInit, err := template.New("bootstrap-cloud-init").Parse(cloudInitTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse download-binaries template: %v", err)
	}

	cloudInit := &bytes.Buffer{}
	err = bsCloudInit.Execute(cloudInit, struct {
		Script  string
		Service string
	}{
		Script:  base64.StdEncoding.EncodeToString(script.Bytes()),
		Service: base64.StdEncoding.EncodeToString([]byte(bootstrapServiceContentTemplate)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute cloudInitTemplate template: %v", err)
	}
	return cloudInit.String(), nil
}

const (
	bootstrapBinContentTemplate = `#!/bin/bash
	set -xeuo pipefail
	apt get update && apt install -y curl jq
	curl   -k -v --header 'Authorization: Bearer {{ .Token }}' \
	{{ .ServerURL }}/api/v1/namespaces/kube-system/secrets/{{ .SecretName }} \\
	| jq '.data["cloud-init"]' -r| base64 -d > /etc/cloud/cloud.cfg.d/{{ .SecretName }}.cfg
	cloud-init clean
	cloud-init --file /etc/cloud/cloud.cfg.d/{{ .SecretName }}.cfg init
	systemctl start provision.service`

	bootstrapServiceContentTemplate = `[Install]
	WantedBy=multi-user.target
	
	[Unit]
	Requires=network-online.target
	After=network-online.target
	[Service]
	Type=oneshot
	RemainAfterExit=true
	ExecStart=/opt/bin/bootstrap`

	cloudInitTemplate = `#cloud-config
{{ if ne .CloudProviderName "aws" }}
hostname: {{ .MachineName }}
{{- /* Never set the hostname on AWS nodes. Kubernetes(kube-proxy) requires the hostname to be the private dns name */}}
{{ end }}
ssh_pwauth: no
write_files:
- path: /opt/bin/bootstrap
  permissions: '0755'
  encoding: b64
  content: |
    {{ .Script }}
- path: /etc/systemd/system/bootstrap.service
  permissions: '0644'
  encoding: b64
  content: |
    {{ .Service }}
runcmd:
- systemctl restart bootstrap.service
- systemctl daemon-reload
`
)
