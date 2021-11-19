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

package azure

import (
	"encoding/json"
	"errors"
	"fmt"

	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/operating-system-manager/pkg/cloudprovider/azure/types"
	"k8c.io/operating-system-manager/pkg/providerconfig/config"
)

const (
	envClientID       = "AZURE_CLIENT_ID"
	envClientSecret   = "AZURE_CLIENT_SECRET"
	envTenantID       = "AZURE_TENANT_ID"
	envSubscriptionID = "AZURE_SUBSCRIPTION_ID"
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

	var err error

	cloudConfig := types.CloudConfig{
		Cloud:               types.AzureCloudProvider,
		UseInstanceMetadata: true,
		ResourceGroup:       rawConfig.ResourceGroup,
		Location:            rawConfig.Location,
		VNetName:            rawConfig.VNetName,
		SubnetName:          rawConfig.SubnetName,
		RouteTableName:      rawConfig.RouteTableName,
		SecurityGroupName:   rawConfig.SecurityGroupName,
		VnetResourceGroup:   rawConfig.VNetResourceGroup,
		LoadBalancerSku:     rawConfig.LoadBalancerSku,
	}

	cloudConfig.TenantID, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.TenantID, envTenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"tenantID\" field, error = %v", err)
	}
	cloudConfig.SubscriptionID, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.SubscriptionID, envSubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"subscriptionID\" field, error = %v", err)
	}
	cloudConfig.AADClientID, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.ClientID, envClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"clientID\" field, error = %v", err)
	}

	cloudConfig.AADClientSecret, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.ClientSecret, envClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"clientSecret\" field, error = %v", err)
	}

	if rawConfig.AssignAvailabilitySet == nil && rawConfig.AvailabilitySet != "" ||
		rawConfig.AssignAvailabilitySet != nil && *rawConfig.AssignAvailabilitySet && rawConfig.AvailabilitySet != "" {
		cloudConfig.PrimaryAvailabilitySetName = rawConfig.AvailabilitySet
	}

	return &cloudConfig, nil
}
