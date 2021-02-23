package generator

import (
	"bytes"
	"encoding/base64"
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
func NewDefaultCloudInitGenerator(unitsPath string) (CloudInitGenerator, error) {
	if unitsPath == "" {
		unitsPath = defaultUnitsPath
	}

	return &DefaultCloudInitGenerator{
		unitsPath: unitsPath,
	}, nil
}

func (d *DefaultCloudInitGenerator) Generate(osc *osmv1alpha1.OperatingSystemConfig) ([]byte, error) {
	var files []*fileSpec
	for _, file := range osc.Spec.Files {
		fSpec := &fileSpec{
			Path:    file.Path,
			Content: base64.StdEncoding.EncodeToString([]byte(file.Content.Inline.Data)),
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

var cloudInitTemplate = `
#cloud-config

ssh_pwauth: no
ssh_authorized_keys:
{{ range $_, $key := .UserSSHKeys -}}
- '{{ $key }}'
{{ end -}}

write_files:
{{ range $_, $file := .Files -}}
- path: '{{ $file.Path }}'
{{- if $file.Permissions }}
  permissions: '{{ $file.Permissions }}'
{{- end }}
  encoding: b64
  content: |
    {{ $file.Content }}
{{ end -}}

runcmd:
{{ range $_, $cmd := runCMDs .Files -}}
- systemctl restart {{ $cmd }}
{{ end -}}
- systemctl daemon-reload
`
