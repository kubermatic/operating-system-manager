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
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/kubermatic/machine-controller/pkg/ini"
)

const (
	cloudConfigTpl = `[Global]
user              = {{ .Global.User | iniEscape }}
password          = {{ .Global.Password | iniEscape }}
port              = {{ .Global.VCenterPort | iniEscape }}
insecure-flag     = {{ .Global.InsecureFlag }}

[Disk]
scsicontrollertype = {{ .Disk.SCSIControllerType | iniEscape }}

[Workspace]
server            = {{ .Workspace.VCenterIP | iniEscape }}
datacenter        = {{ .Workspace.Datacenter | iniEscape }}
folder            = {{ .Workspace.Folder | iniEscape }}
default-datastore = {{ .Workspace.DefaultDatastore | iniEscape }}

{{ range $name, $vc := .VirtualCenter }}
[VirtualCenter {{ $name | iniEscape }}]
user = {{ $vc.User | iniEscape }}
password = {{ $vc.Password | iniEscape }}
port = {{ $vc.VCenterPort }}
datacenters = {{ $vc.Datacenters | iniEscape }}
{{ end }}
`
)

type WorkspaceOpts struct {
	VCenterIP        string `gcfg:"server"`
	Datacenter       string `gcfg:"datacenter"`
	Folder           string `gcfg:"folder"`
	DefaultDatastore string `gcfg:"default-datastore"`
}

type DiskOpts struct {
	SCSIControllerType string `dcfg:"scsicontrollertype"`
}

type GlobalOpts struct {
	User         string `gcfg:"user"`
	Password     string `gcfg:"password"`
	InsecureFlag bool   `gcfg:"insecure-flag"`
	VCenterPort  string `gcfg:"port"`
	ClusterID    string `gcfg:"cluster-id"`
}

type VirtualCenterConfig struct {
	User        string `gcfg:"user"`
	Password    string `gcfg:"password"`
	VCenterPort string `gcfg:"port"`
	Datacenters string `gcfg:"datacenters"`
}

// CloudConfig is used to read and store information from the cloud configuration file
type CloudConfig struct {
	Global    GlobalOpts
	Disk      DiskOpts
	Workspace WorkspaceOpts

	VirtualCenter map[string]*VirtualCenterConfig
}

// ToString renders the cloud configuration as string.
func (cc *CloudConfig) ToString() (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["iniEscape"] = ini.Escape

	tpl, err := template.New("cloud-config").Funcs(funcMap).Parse(cloudConfigTpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse the cloud config template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, cc); err != nil {
		return "", fmt.Errorf("failed to execute cloud config template: %w", err)
	}

	return buf.String(), nil
}
