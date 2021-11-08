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

package generator

import (
	"bytes"
	"fmt"
	"text/template"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
)

const defaultUnitsPath = "/etc/systemd/system/"

// CloudConfigGenerator generates the machine provisioning configurations for the corresponding operating system config
type CloudConfigGenerator interface {
	Generate(osc *osmv1alpha1.OperatingSystemConfig) ([]byte, error)
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

func (d *DefaultCloudConfigGenerator) Generate(osc *osmv1alpha1.OperatingSystemConfig) ([]byte, error) {
	var files []*fileSpec
	for _, file := range osc.Spec.Files {
		fSpec := &fileSpec{
			Path:    file.Path,
			Content: file.Content.Inline.Data,
		}
		if file.Permissions != nil {
			permissions := fmt.Sprintf("%04o", *file.Permissions)
			fSpec.Permissions = &permissions
		}

		files = append(files, fSpec)
	}

	var units []*unitSpec
	for _, unit := range osc.Spec.Units {
		uSpec := &unitSpec{
			Name:    unit.Name,
			Enable: *unit.Enable,
			Mask:   *unit.Mask,
			Content: *unit.Content,
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
	userDataTemplate, err := getUserDataTemplate(osc.Spec.OSName)
	if err != nil {
		return nil, fmt.Errorf("failed to get an appropriate user-data template: %v", err)
	}

	tmpl, err := template.New("user-data").Funcs(TxtFuncMap()).Parse(userDataTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user-data template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, &struct {
		Files       []*fileSpec
		Units 	 []*unitSpec
		UserSSHKeys []string
	}{
		Files:       files,
		Units:       units,
		UserSSHKeys: osc.Spec.UserSSHKeys,
	}); err != nil {
		return nil, err
	}

	if GetProvisioningUtility(osc.Spec.OSName) == CloudInit {
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
	Permissions *string
	Name        string
}

type unitSpec struct {
	Name    string
	Enable bool
	Mask bool
	Content string
	DropIns []dropInSpec

}

type dropInSpec struct {
	Name    string
	Content string
}

var cloudInitTemplate = `#cloud-config
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
  content: |-
{{ $file.Content | indent 4 }}
{{ end }}
runcmd:
{{ range $_, $cmd := runCMDs .Files -}}
- systemctl restart {{ $cmd }}
{{ end -}}
- systemctl daemon-reload
`

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
{{- range $_, $file := .Files }}
  - path: '{{ $file.Path }}'
    mode: {{or $file.Permissions 0644}}
    filesystem: root
    contents:
        inline: |
{{ $file.Content | indent 10 }}
{{- end }}
{{- range $_, $unit := .Units }}
  - name: {{ $unit.Name }}
    enable: {{ $unit.Enable }}
    mask: {{ $unit.Mask }}
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
