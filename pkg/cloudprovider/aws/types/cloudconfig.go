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
	cloudConfigTpl = `[global]
Zone={{ .Global.Zone | iniEscape }}
VPC={{ .Global.VPC | iniEscape }}
SubnetID={{ .Global.SubnetID | iniEscape }}
`
)

type CloudConfig struct {
	Global GlobalOpts
}

type GlobalOpts struct {
	Zone     string
	VPC      string
	SubnetID string
}

// ToString renders the cloud configuration as string.
func (cc *CloudConfig) ToString() (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["iniEscape"] = ini.Escape

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
