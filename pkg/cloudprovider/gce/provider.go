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

package gce

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/operating-system-manager/pkg/cloudprovider/gce/types"
	"k8c.io/operating-system-manager/pkg/providerconfig/config"
)

func GetCloudConfig(pconfig providerconfig.Config) (string, error) {
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
func getConfig(pconfig providerconfig.Config) (*types.CloudConfig, error) {
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

	opts.NodeTags = rawConfig.Tags
	opts.ProjectID, err = getProjectID(rawConfig)
	if err != nil {
		return nil, err
	}
	opts.LocalZone, err = config.GetConfigVarResolver().GetStringValue(rawConfig.Zone)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve zone: %w", err)
	}

	opts.MultiZone, _, err = config.GetConfigVarResolver().GetBoolValue(rawConfig.MultiZone)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve multizone: %w", err)
	}
	opts.Regional, _, err = config.GetConfigVarResolver().GetBoolValue(rawConfig.Regional)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve regional: %w", err)
	}

	opts.NetworkName, err = config.GetConfigVarResolver().GetStringValue(rawConfig.Network)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve network: %w", err)
	}

	opts.SubnetworkName, err = config.GetConfigVarResolver().GetStringValue(rawConfig.Subnetwork)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve subnetwork: %w", err)
	}

	opts.ProjectID, err = getProjectID(rawConfig)
	if err != nil {
		return nil, err
	}

	cloudConfig := &types.CloudConfig{
		Global: opts,
	}

	return cloudConfig, nil
}

func getProjectID(rawConfig types.RawConfig) (string, error) {
	serviceAccount, err := config.GetConfigVarResolver().GetStringValueOrEnv(rawConfig.ServiceAccount, "GOOGLE_SERVICE_ACCOUNT")
	if err != nil {
		return "", fmt.Errorf("cannot retrieve service account: %w", err)
	}

	sa, err := base64.StdEncoding.DecodeString(serviceAccount)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 service account: %w", err)
	}
	sam := map[string]string{}
	err = json.Unmarshal(sa, &sam)
	if err != nil {
		return "", fmt.Errorf("failed unmarshalling service account: %w", err)
	}
	return sam["project_id"], nil
}
