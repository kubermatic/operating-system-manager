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

package aws

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/kubermatic/machine-controller/pkg/ini"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"k8c.io/operating-system-manager/pkg/cloudprovider/aws/types"
	"k8c.io/operating-system-manager/pkg/providerconfig/config"
)

// TODO @Waleed Move this to a configmap
const (
	cloudConfigTpl = `[global]
Zone={{ .Global.Zone | iniEscape }}
VPC={{ .Global.VPC | iniEscape }}
SubnetID={{ .Global.SubnetID | iniEscape }}
RouteTableID={{ .Global.RouteTableID | iniEscape }}
RoleARN={{ .Global.RoleARN | iniEscape }}
KubernetesClusterID={{ .Global.KubernetesClusterID | iniEscape }}
DisableSecurityGroupIngress={{ .Global.DisableSecurityGroupIngress }}
ElbSecurityGroup={{ .Global.ElbSecurityGroup | iniEscape }}
DisableStrictZoneCheck={{ .Global.DisableStrictZoneCheck }}
`
)

func GetCloudConfig(pconfig providerconfigtypes.Config) (string, error) {
	c, err := getConfig(pconfig)
	if err != nil {
		return "", fmt.Errorf("failed to parse config: %v", err)
	}

	s, err := cloudConfigToString(c)
	if err != nil {
		return "", fmt.Errorf("failed to convert cloud-config to string: %v", err)
	}

	return s, nil
}
func getConfig(pconfig providerconfigtypes.Config) (*types.CloudConfig, error) {
	if pconfig.OperatingSystemSpec.Raw == nil {
		return nil, errors.New("operatingSystemSpec in the MachineDeployment cannot be empty")
	}

	rawConfig := types.RawConfig{}
	if err := json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CloudProviderSpec: %v", err)
	}

	var (
		opts types.GlobalOpts
		err  error
	)

	opts.Zone, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.AvailabilityZone)
	if err != nil {
		return nil, err
	}
	opts.VPC, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.VpcID)
	if err != nil {
		return nil, err
	}
	opts.SubnetID, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.SubnetID)
	if err != nil {
		return nil, err
	}
	cloudConfig := &types.CloudConfig{
		Global: opts,
	}

	return cloudConfig, nil
}
func cloudConfigToString(c *types.CloudConfig) (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["iniEscape"] = ini.Escape

	tpl, err := template.New("cloud-config").Funcs(funcMap).Parse(cloudConfigTpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse the cloud config template: %v", err)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, c); err != nil {
		return "", fmt.Errorf("failed to execute cloud config template: %v", err)
	}

	return buf.String(), nil
}
