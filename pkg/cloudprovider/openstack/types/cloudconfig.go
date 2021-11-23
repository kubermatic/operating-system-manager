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

package types

import (
	"bytes"
	"fmt"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/kubermatic/machine-controller/pkg/ini"
)

//  use-octavia is enabled by default in CCM since v1.17.0, and disabled by
//  default with the in-tree cloud provider.
//  https://v1-18.docs.kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#load-balancer
const (
	cloudConfigTpl = `[Global]
auth-url    = {{ .Global.AuthURL | iniEscape }}
{{- if .Global.ApplicationCredentialID }}
application-credential-id     = {{ .Global.ApplicationCredentialID | iniEscape }}
application-credential-secret = {{ .Global.ApplicationCredentialSecret | iniEscape }}
{{- else }}
username    = {{ .Global.Username | iniEscape }}
password    = {{ .Global.Password | iniEscape }}
tenant-name = {{ .Global.ProjectName | iniEscape }}
tenant-id   = {{ .Global.ProjectID | iniEscape }}
{{- end }}
domain-name = {{ .Global.DomainName | iniEscape }}
region      = {{ .Global.Region | iniEscape }}

[LoadBalancer]
{{- if semverCompare "~1.9.10 || ~1.10.6 || ~1.11.1 || >=1.12.*" .Version }}
manage-security-groups = {{ .LoadBalancer.ManageSecurityGroups }}
{{- end }}

[BlockStorage]
{{- if semverCompare ">=1.9" .Version }}
ignore-volume-az  = {{ .BlockStorage.IgnoreVolumeAZ }}
{{- end }}
trust-device-path = {{ .BlockStorage.TrustDevicePath }}
bs-version        = {{ default "auto" .BlockStorage.BSVersion | iniEscape }}
{{- if .BlockStorage.NodeVolumeAttachLimit }}
node-volume-attach-limit = {{ .BlockStorage.NodeVolumeAttachLimit }}
{{- end }}
`
)

type LoadBalancerOpts struct {
	ManageSecurityGroups bool `gcfg:"manage-security-groups"`
}

type BlockStorageOpts struct {
	BSVersion             string `gcfg:"bs-version"`
	TrustDevicePath       bool   `gcfg:"trust-device-path"`
	IgnoreVolumeAZ        bool   `gcfg:"ignore-volume-az"`
	NodeVolumeAttachLimit uint   `gcfg:"node-volume-attach-limit"`
}

type GlobalOpts struct {
	AuthURL                     string `gcfg:"auth-url"`
	Username                    string
	Password                    string
	ApplicationCredentialID     string `gcfg:"application-credential-id"`
	ApplicationCredentialSecret string `gcfg:"application-credential-secret"`

	// project name formerly known as tenant name.
	// it serialized as tenant-name because openstack CCM reads only tenant-name. In CCM, internally project and tenant
	// are stored into tenant-name.
	ProjectName string `gcfg:"tenant-name"`

	// project id formerly known as tenant id.
	// serialized as tenant-id for same reason as ProjectName
	ProjectID  string `gcfg:"tenant-id"`
	DomainName string `gcfg:"domain-name"`
	Region     string
}

// CloudConfig is used to read and store information from the cloud configuration file
type CloudConfig struct {
	Global       GlobalOpts
	LoadBalancer LoadBalancerOpts
	BlockStorage BlockStorageOpts
	Version      string
}

// ToString renders the cloud configuration as string.
func (cc *CloudConfig) ToString() (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["iniEscape"] = ini.Escape
	funcMap["boolPtr"] = func(b *bool) string { return strconv.FormatBool(*b) }

	tpl, err := template.New("cloud-config").Funcs(funcMap).Parse(cloudConfigTpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse the cloud config template: %v", err)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, cc); err != nil {
		return "", fmt.Errorf("failed to execute cloud config template: %v", err)
	}

	return buf.String(), nil
}
