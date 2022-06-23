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
	"text/template"

	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
)

const (
	defaultUnitsPath = "/etc/systemd/system/"
	base64Encoding   = "b64"
)

// CloudConfigGenerator generates the machine bootstrapping and provisioning configurations for the corresponding operating system config
type CloudConfigGenerator interface {
	Generate(config *osmv1alpha1.OSCConfig, operatingSystem v1alpha1.OperatingSystem, cloudProvider v1alpha1.CloudProvider) ([]byte, error)
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

func (d *DefaultCloudConfigGenerator) Generate(config *osmv1alpha1.OSCConfig, operatingSystem v1alpha1.OperatingSystem, cloudProvider v1alpha1.CloudProvider) ([]byte, error) {
	var files []*fileSpec
	for _, file := range config.Files {
		content := file.Content.Inline.Data
		if file.Content.Inline.Encoding == base64Encoding {
			content = base64.StdEncoding.EncodeToString([]byte(file.Content.Inline.Data))
		}

		fSpec := &fileSpec{
			Path:     file.Path,
			Content:  content,
			Encoding: file.Content.Inline.Encoding,
		}
		if file.Permissions != nil {
			permissions := fmt.Sprintf("%04o", *file.Permissions)
			fSpec.Permissions = &permissions
		}

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

	// Fetch user data template based on the provisioning utility
	userDataTemplate, err := getUserDataTemplate(operatingSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to get an appropriate user-data template: %w", err)
	}

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
	}{
		Files:             files,
		Units:             units,
		UserSSHKeys:       config.UserSSHKeys,
		CloudInitModules:  config.CloudInitModules,
		CloudProviderName: string(cloudProvider),
		OperatingSystem:   string(operatingSystem),
	}); err != nil {
		return nil, err
	}

	if GetProvisioningUtility(operatingSystem) == CloudInit {
		return buf.Bytes(), nil
	}

	return toIgnition(buf.String())
}

func getUserDataTemplate(osName osmv1alpha1.OperatingSystem) (string, error) {
	pUtil := GetProvisioningUtility(osName)
	switch pUtil {
	case CloudInit:
		return cloudInitTemplate, nil
	case Ignition:
		return ignitionTemplate, nil
	default:
		return "", fmt.Errorf("invalid provisioning utility %s, allowed values are %s or %s",
			pUtil, Ignition, CloudInit)
	}
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
{{- if ne .CloudProviderName "aws" -}}
{{- /* Never set the hostname on AWS nodes. Kubernetes(kube-proxy) requires the hostname to be the private dns name */}}
{{- /* machine-controller will replace "<MACHINE_NAME>" placeholder with the name of the machine */}}
hostname: <MACHINE_NAME>
{{ end }}
ssh_pwauth: no
ssh_authorized_keys:
{{ range $_, $key := .UserSSHKeys -}}
- '{{ $key }}'
{{ end -}}

write_files:
{{- range $_, $file := .Files }}
- path: '{{ $file.Path }}'
{{- if $file.Permissions }}
  permissions: '{{ $file.Permissions }}'
{{- end }}
{{- if $file.Encoding }}
  encoding: '{{ $file.Encoding }}'
{{- end }}
  content: |-
{{ $file.Content | indent 4 }}
{{ end }}
{{- if and (eq .CloudProviderName "openstack") (or (eq .OperatingSystem "centos") (eq .OperatingSystem "rhel")) -}}
{{- /*  The normal way of setting it via cloud-init is broken, see */}}
{{- /*  https://bugs.launchpad.net/cloud-init/+bug/1662542 */}}
{{- /* machine-controller will replace "<MACHINE_NAME>" placeholder with the name of the machine */}}
- path: /etc/hostname
  permissions: '0600'
  content: |
	<MACHINE_NAME>
{{ end }}
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
{{- if ne .CloudProviderName "aws" -}}
{{- /* Never set the hostname on AWS nodes. Kubernetes(kube-proxy) requires the hostname to be the private dns name */}}
{{- /* machine-controller will replace "<MACHINE_NAME>" placeholder with the name of the machine */}}
  - path: /etc/hostname
    mode: 0600
    filesystem: root
    contents:
        inline: '<MACHINE_NAME>'
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
