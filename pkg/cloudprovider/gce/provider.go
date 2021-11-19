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

	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"k8c.io/operating-system-manager/pkg/cloudprovider/gce/types"
	"k8c.io/operating-system-manager/pkg/providerconfig/config"
)

func GetCloudConfig(pconfig providerconfigtypes.Config) (string, error) {
	c, err := getConfig(pconfig)
	if err != nil {
		return "", fmt.Errorf("failed to parse config: %v", err)
	}

	s, err := c.ToString()
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
		err error
	)

	opts := types.GlobalOpts{
		NodeTags:       rawConfig.Tags,
		LocalZone:      rawConfig.Zone,
		MultiZone:      rawConfig.MultiZone,
		Regional:       rawConfig.Regional,
		NetworkName:    rawConfig.Network,
		SubnetworkName: rawConfig.Subnetwork,
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
	serviceAccount, err := config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.ServiceAccount, "GOOGLE_SERVICE_ACCOUNT")
	if err != nil {
		return "", fmt.Errorf("cannot retrieve service account: %v", err)
	}

	sa, err := base64.StdEncoding.DecodeString(serviceAccount)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 service account: %v", err)
	}
	sam := map[string]string{}
	err = json.Unmarshal(sa, &sam)
	if err != nil {
		return "", fmt.Errorf("failed unmarshalling service account: %v", err)
	}
	return sam["project_id"], nil
}
