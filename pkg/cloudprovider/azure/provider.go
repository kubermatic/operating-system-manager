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
	"errors"
	"fmt"

	"github.com/kubermatic/machine-controller/pkg/jsonutil"
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
	if err := jsonutil.StrictUnmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CloudProviderSpec: %w", err)
	}

	var (
		cloudConfig types.CloudConfig
		err         error
	)

	cloudConfig.Cloud = types.AzureCloudProvider
	cloudConfig.UseInstanceMetadata = true

	cloudConfig.TenantID, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.TenantID, envTenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"tenantID\" field, error = %w", err)
	}
	cloudConfig.SubscriptionID, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.SubscriptionID, envSubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"subscriptionID\" field, error = %w", err)
	}
	cloudConfig.AADClientID, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.ClientID, envClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"clientID\" field, error = %w", err)
	}

	cloudConfig.AADClientSecret, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.ClientSecret, envClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"clientSecret\" field, error = %w", err)
	}

	cloudConfig.ResourceGroup, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.ResourceGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"resourceGroup\" field, error = %w", err)
	}

	cloudConfig.Location, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.Location)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"location\" field, error = %w", err)
	}

	cloudConfig.VNetName, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.VNetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"vnetName\" field, error = %w", err)
	}

	cloudConfig.SubnetName, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.SubnetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"subnetName\" field, error = %w", err)
	}
	cloudConfig.RouteTableName, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.RouteTableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"routeTableName\" field, error = %w", err)
	}
	cloudConfig.SecurityGroupName, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.SecurityGroupName)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"securityGroupName\" field, error = %w", err)
	}

	cloudConfig.VnetResourceGroup, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.VNetResourceGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"vnetResourceGroup\" field, error = %w", err)
	}

	cloudConfig.LoadBalancerSku, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.LoadBalancerSku)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"loadBalancerSku\" field, error = %w", err)
	}

	availabilitySet, err := config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.AvailabilitySet)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"availabilitySet\" field, error = %w", err)
	}

	if rawConfig.AssignAvailabilitySet == nil && availabilitySet != "" ||
		rawConfig.AssignAvailabilitySet != nil && *rawConfig.AssignAvailabilitySet && availabilitySet != "" {
		cloudConfig.PrimaryAvailabilitySetName = availabilitySet
	}

	return &cloudConfig, nil
}
