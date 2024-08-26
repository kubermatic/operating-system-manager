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
	"encoding/json"
	"errors"
	"fmt"

	providerconfigtypes "k8c.io/machine-controller/pkg/providerconfig/types"
	"k8c.io/operating-system-manager/pkg/cloudprovider/aws/types"
	"k8c.io/operating-system-manager/pkg/providerconfig/config"
)

func GetCloudConfig(pconfig providerconfigtypes.Config) (string, error) {
	c, err := getConfig(pconfig)
	if err != nil {
		return "", fmt.Errorf("failed to parse config: %w", err)
	}

	s, err := c.ToString()
	if err != nil {
		return "", fmt.Errorf("failed to convert cloud-config to string: %w", err)
	}

	return s, nil
}
func getConfig(pconfig providerconfigtypes.Config) (*types.CloudConfig, error) {
	if pconfig.CloudProviderSpec.Raw == nil {
		return nil, errors.New("CloudProviderSpec in the MachineDeployment cannot be empty")
	}

	rawConfig := types.RawConfig{}
	if err := json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CloudProviderSpec: %w", err)
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
