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

package generator

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	clusterv1alpha1 "k8c.io/machine-controller/pkg/apis/cluster/v1alpha1"
	mcbootstrap "k8c.io/machine-controller/pkg/bootstrap"
	"k8c.io/machine-controller/pkg/jsonutil"
	providerconfigtypes "k8c.io/machine-controller/pkg/providerconfig/types"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/providerconfig"
)

const (
	defaultUnitsPath = "/etc/systemd/system/"
	base64Encoding   = "b64"
)

// CloudConfigGenerator generates the machine bootstrapping and provisioning configurations for the corresponding operating system config
type CloudConfigGenerator interface {
	Generate(config *osmv1alpha1.OSCConfig, provisioningUtility osmv1alpha1.ProvisioningUtility, operatingSystem osmv1alpha1.OperatingSystem, cloudProvider osmv1alpha1.CloudProvider, md clusterv1alpha1.MachineDeployment, secretType mcbootstrap.CloudConfigSecret) ([]byte, error)
}

// DefaultCloudConfigGenerator represents the default generator of the machine provisioning configurations
type DefaultCloudConfigGenerator struct {
	unitsPath string
}

// NewDefaultCloudConfigGenerator creates a new CloudConfigGenerator.
func NewDefaultCloudConfigGenerator(unitsPath string) CloudConfigGenerator {
	if unitsPath == "" {
		unitsPath = defaultUnitsPath
	}

	return &DefaultCloudConfigGenerator{
		unitsPath: unitsPath,
	}
}

func (d *DefaultCloudConfigGenerator) Generate(config *osmv1alpha1.OSCConfig, provisioningUtility osmv1alpha1.ProvisioningUtility, operatingSystem osmv1alpha1.OperatingSystem, cloudProvider osmv1alpha1.CloudProvider, md clusterv1alpha1.MachineDeployment, secretType mcbootstrap.CloudConfigSecret) ([]byte, error) {
	provisioner, err := GetProvisioningUtility(operatingSystem, md)
	if err != nil {
		return nil, fmt.Errorf("failed to determine provisioning utility: %w", err)
	}

	if provisioningUtility != "" && provisioner != provisioningUtility {
		return nil, fmt.Errorf("specified provisioning utility %q is not supported by the OperatingSystemConfig", provisioningUtility)
	}

	var files []*fileSpec
	for _, file := range config.Files {
		content := file.Content.Inline.Data
		// Ignition doesn't support base64 encoding
		if file.Content.Inline.Encoding == base64Encoding && provisioner == osmv1alpha1.ProvisioningUtilityCloudInit {
			content = base64.StdEncoding.EncodeToString([]byte(file.Content.Inline.Data))
		}

		fSpec := &fileSpec{
			Path:     file.Path,
			Content:  content,
			Encoding: file.Content.Inline.Encoding,
		}
		permissions := fmt.Sprintf("%v", file.Permissions)
		// Convert to an octal value for file permissions.
		if len(permissions) == 3 {
			permissions = "0" + permissions
		}
		fSpec.Permissions = &permissions
		files = append(files, fSpec)
	}

	var units []*unitSpec
	for _, unit := range config.Units {
		uSpec := &unitSpec{
			Name: unit.Name,
		}

		if unit.Enable != nil {
			uSpec.Enable = *unit.Enable
		}

		if unit.Mask != nil {
			uSpec.Mask = *unit.Mask
		}

		if unit.Content != nil {
			uSpec.Content = *unit.Content
		}

		for _, dropIn := range unit.DropIns {
			dSpec := &dropInSpec{
				Name:    dropIn.Name,
				Content: dropIn.Content,
			}
			uSpec.DropIns = append(uSpec.DropIns, *dSpec)
		}
		units = append(units, uSpec)
	}

	// Retrieve Operating System Config.
	providerConfig := providerconfigtypes.Config{}
	if err := jsonutil.StrictUnmarshal(md.Spec.Template.Spec.ProviderSpec.Value.Raw, &providerConfig); err != nil {
		return nil, fmt.Errorf("failed to decode provider configs: %w", err)
	}

	osConfig, err := providerconfig.LoadConfig((providerConfig.OperatingSystemSpec))
	if err != nil {
		return nil, err
	}

	// Fetch user data template based on the provisioning utility
	userDataTemplate := getUserDataTemplate(provisioner, md.Name, string(cloudProvider))
	tmpl, err := template.New("user-data").Funcs(TxtFuncMap()).Parse(userDataTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user-data template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, &struct {
		Files             []*fileSpec
		Units             []*unitSpec
		UserSSHKeys       []string
		CloudInitModules  *osmv1alpha1.CloudInitModule
		CloudProviderName string
		OperatingSystem   string
		ConfigurationType string
		OSConfig          providerconfig.Config
	}{
		Files:             files,
		Units:             units,
		UserSSHKeys:       config.UserSSHKeys,
		CloudInitModules:  config.CloudInitModules,
		CloudProviderName: string(cloudProvider),
		OperatingSystem:   string(operatingSystem),
		ConfigurationType: string(secretType),
		OSConfig:          *osConfig,
	}); err != nil {
		return nil, err
	}

	if provisioner == osmv1alpha1.ProvisioningUtilityCloudInit {
		return buf.Bytes(), nil
	}

	return toIgnition(buf.String())
}

func getUserDataTemplate(p osmv1alpha1.ProvisioningUtility, mdName, cloudProvider string) string {
	if p == osmv1alpha1.ProvisioningUtilityIgnition {
		return ignitionTemplate
	}

	if cloudProvider == "edge" {
		return strings.ReplaceAll(cloudInitTemplate, "<MACHINE_NAME>", mdName)
	}

	return cloudInitTemplate
}

type fileSpec struct {
	Path        string
	Content     string
	Encoding    string
	Permissions *string
	Name        string
}

type unitSpec struct {
	Name    string
	Enable  bool
	Mask    bool
	Content string
	DropIns []dropInSpec
}

type dropInSpec struct {
	Name    string
	Content string
}

var cloudInitTemplate = `#cloud-config
{{- /* Hostname is configured only for the bootstrap configuration */}}
{{- if eq .ConfigurationType "bootstrap" -}}
{{- if ne .CloudProviderName "aws" -}}
{{- /* Never set the hostname on AWS nodes. Kubernetes(kube-proxy) requires the hostname to be the private dns name */}}
{{- /* machine-controller will replace "<MACHINE_NAME>" placeholder with the name of the machine */}}
hostname: <MACHINE_NAME>
{{- end -}}
{{ end }}

{{ if .OSConfig.DistUpgradeOnBoot -}}
package_upgrade: true
package_reboot_if_required: true
{{ end -}}

ssh_pwauth: false

ssh_authorized_keys:
{{ range $_, $key := .UserSSHKeys -}}
- '{{ $key }}'
{{ end -}}

write_files:
{{- range $_, $file := .Files }}
- path: '{{ $file.Path }}'
  permissions: '{{or $file.Permissions 0644}}'
{{- if $file.Encoding }}
  encoding: '{{ $file.Encoding }}'
{{- end }}
  content: |-
{{ $file.Content | indent 4 }}
{{ end }}

{{- /* Hostname is configured only for the bootstrap configuration */}}
{{- if eq .ConfigurationType "bootstrap" -}}
{{ if ne .CloudProviderName "aws" }}
{{- /* Never set the hostname on AWS nodes. Kubernetes(kube-proxy) requires the hostname to be the private dns name */}}
{{- /* machine-controller will replace "<MACHINE_NAME>" placeholder with the name of the machine */}}
- path: /etc/machine-name
  permissions: '0600'
  content: |-
        <MACHINE_NAME>
{{ end }}
{{- end -}}

{{- if .Units -}}
coreos:
  units:
{{- range $_, $unit := .Units }}
  - name: "{{ $unit.Name }}"
    enable: {{or $unit.Enable false}}
    {{ if $unit.Enable -}}
	command: start
	{{- end }}
    mask: {{or $unit.Mask false}}
{{ if $unit.Content }}
    content: |
{{ $unit.Content | indent 6 }}
{{- end }}
{{ if $unit.Content }}
    drop-ins:
{{- range $_, $dropIn := $unit.DropIns }}
      - name: "{{ $dropIn.Name }}"
        content: |
{{ $dropIn.Content | indent 10 }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{- if .CloudInitModules -}}
{{ if .CloudInitModules.BootCMD }}
bootcmd:
{{- range $_, $cmd := .CloudInitModules.BootCMD }}
- {{ $cmd }}
{{- end }}
{{ end }}

{{- if .CloudInitModules.RunCMD }}
runcmd:
{{- range $_, $val := .CloudInitModules.RunCMD }}
- {{ $val -}}
{{ end }}
{{ end }}

{{- if .CloudInitModules.RHSubscription }}
rh_subscription:
{{- range $key, $val := .CloudInitModules.RHSubscription }}
    {{ $key }}: {{ $val -}}
{{ end }}
{{ end }}

{{- if .CloudInitModules.YumRepos }}
yum_repos:
{{- range $key, $val := .CloudInitModules.YumRepos }}
    {{ $key }}:
{{- range $prop, $propVal := $val }}
       {{ $prop }}: {{ $propVal }}
{{- end }}
{{- end }}
{{ end }}

{{- if .CloudInitModules.YumRepoDir }}
yum_repo_dir: {{ .CloudInitModules.YumRepoDir }}
{{- end }}
{{- end }}`

var ignitionTemplate = `passwd:
{{- if ne (len .UserSSHKeys) 0 }}
  users:
    - name: core
      ssh_authorized_keys:
        {{range .UserSSHKeys}}- {{.}}
        {{end}}
{{- end }}
storage:
  files:
{{- /* Hostname is configured only for the bootstrap configuration */}}
{{- if eq .ConfigurationType "bootstrap" -}}
{{- if ne .CloudProviderName "aws" -}}
{{- /* Never set the hostname on AWS nodes. Kubernetes(kube-proxy) requires the hostname to be the private dns name */}}
{{- /* machine-controller will replace "<MACHINE_NAME>" placeholder with the name of the machine */}}
  - path: /etc/machine-name
    mode: 0600
    filesystem: root
    contents:
        inline: '<MACHINE_NAME>'
{{ end }}
{{ end }}
{{- range $_, $file := .Files }}
  - path: '{{ $file.Path }}'
    mode: {{or $file.Permissions 0644}}
    filesystem: root
    contents:
        inline: |
{{ $file.Content | indent 10 }}
{{- end }}
systemd:
  units:
{{- range $_, $unit := .Units }}
  - name: {{ $unit.Name }}
    enabled: {{or $unit.Enable false}}
    mask: {{or $unit.Mask false}}
{{ if $unit.Content }}
    contents: |
{{ $unit.Content | indent 6 }}
{{- end }}
{{ if $unit.Content }}
    dropins:
{{- range $_, $dropIn := $unit.DropIns }}
      - name: {{ $dropIn.Name }}
        contents: |
{{ $dropIn.Content | indent 10 }}
{{- end }}
{{- end }}
{{- end }}
`
