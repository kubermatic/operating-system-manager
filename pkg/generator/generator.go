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

// CloudInitGenerator generates the cloud-init configurations for the corresponding operating system config
type CloudInitGenerator interface {
	Generate(osc *osmv1alpha1.OperatingSystemConfig) ([]byte, error)
}

// DefaultCloudInitGenerator represents the default generator of the cloud-init configuration
type DefaultCloudInitGenerator struct {
	unitsPath string
}

// NewDefaultCloudInitGenerator creates a new cloud-init generator.
func NewDefaultCloudInitGenerator(unitsPath string) CloudInitGenerator {
	if unitsPath == "" {
		unitsPath = defaultUnitsPath
	}

	return &DefaultCloudInitGenerator{
		unitsPath: unitsPath,
	}
}

func (d *DefaultCloudInitGenerator) Generate(osc *osmv1alpha1.OperatingSystemConfig) ([]byte, error) {

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

	tmpl, err := template.New("user-data").Funcs(TxtFuncMap()).Parse(cloudInitTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cloud-init template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, &struct {
		Files       []*fileSpec
		UserSSHKeys []string
	}{
		Files:       files,
		UserSSHKeys: osc.Spec.UserSSHKeys,
	}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type fileSpec struct {
	Path        string
	Content     string
	Permissions *string
	Name        string
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
