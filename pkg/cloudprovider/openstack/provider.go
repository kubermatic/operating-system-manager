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

package openstack

import (
	"encoding/json"
	"errors"
	"fmt"

	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"k8c.io/operating-system-manager/pkg/cloudprovider/openstack/types"
	"k8c.io/operating-system-manager/pkg/providerconfig/config"

	"k8s.io/klog"
)

func GetCloudConfig(pconfig providerconfigtypes.Config, kubeletVersion string) (string, error) {
	c, err := getConfig(pconfig, kubeletVersion)
	if err != nil {
		return "", fmt.Errorf("failed to parse config: %v", err)
	}

	s, err := c.ToString()
	if err != nil {
		return "", fmt.Errorf("failed to convert cloud-config to string: %v", err)
	}

	return s, nil
}
func getConfig(pconfig providerconfigtypes.Config, kubeletVersion string) (*types.CloudConfig, error) {
	if pconfig.CloudProviderSpec.Raw == nil {
		return nil, errors.New("CloudProviderSpec in the MachineDeployment cannot be empty")
	}

	rawConfig := types.RawConfig{}
	if err := json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CloudProviderSpec: %v", err)
	}

	var (
		opts types.GlobalOpts
		err  error
	)

	// Ignore Region not found as Region might not be found and we can default it later
	opts.Region, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.Region, "OS_REGION_NAME")
	if err != nil {
		klog.V(6).Infof("Region from configuration or environment variable not found")
	}

	// We ignore errors here because the OS domain is only required when using Identity API V3
	opts.DomainName, _ = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.DomainName, "OS_DOMAIN_NAME")

	opts.AuthURL, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.IdentityEndpoint, "OS_AUTH_URL")
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"identityEndpoint\" field, error = %v", err)
	}

	trustDevicePath, _, err := config.GetConfigVarResolver().GetConfigVarBoolValue(rawConfig.TrustDevicePath)
	if err != nil {
		return nil, err
	}

	// Retrieve authentication config, username/password or application credentials
	err = getConfigAuth(&opts, &rawConfig)
	if err != nil {
		return nil, err
	}

	cloudConfig := &types.CloudConfig{
		Global: opts,
		BlockStorage: types.BlockStorageOpts{
			BSVersion:       "auto",
			TrustDevicePath: trustDevicePath,
			IgnoreVolumeAZ:  true,
		},
		LoadBalancer: types.LoadBalancerOpts{
			ManageSecurityGroups: true,
		},
		Version: kubeletVersion,
	}

	if rawConfig.NodeVolumeAttachLimit != nil {
		cloudConfig.BlockStorage.NodeVolumeAttachLimit = *rawConfig.NodeVolumeAttachLimit
	}

	return cloudConfig, nil
}

func getConfigAuth(c *types.GlobalOpts, rawConfig *types.RawConfig) error {
	var err error
	c.ApplicationCredentialID, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.ApplicationCredentialID, "OS_APPLICATION_CREDENTIAL_ID")
	if err != nil {
		return fmt.Errorf("failed to get the value of \"applicationCredentialID\" field, error = %v", err)
	}
	if c.ApplicationCredentialID != "" {
		klog.V(6).Infof("applicationCredentialID from configuration or environment was found.")
		c.ApplicationCredentialSecret, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.ApplicationCredentialSecret, "OS_APPLICATION_CREDENTIAL_SECRET")
		if err != nil {
			return fmt.Errorf("failed to get the value of \"applicationCredentialSecret\" field, error = %v", err)
		}
		return nil
	}
	c.Username, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.Username, "OS_USER_NAME")
	if err != nil {
		return fmt.Errorf("failed to get the value of \"username\" field, error = %v", err)
	}
	c.Password, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.Password, "OS_PASSWORD")
	if err != nil {
		return fmt.Errorf("failed to get the value of \"password\" field, error = %v", err)
	}
	c.ProjectName, err = getProjectNameOrTenantName(rawConfig)
	if err != nil {
		return fmt.Errorf("failed to get the value of \"projectName\" field or fallback to \"tenantName\" field, error = %v", err)
	}
	c.ProjectID, err = getProjectIDOrTenantID(rawConfig)
	if err != nil {
		return fmt.Errorf("failed to get the value of \"projectID\" or fallback to\"tenantID\" field, error = %v", err)
	}
	return nil
}

// Get the Project name from config or env var. If not defined fallback to tenant name
func getProjectNameOrTenantName(rawConfig *types.RawConfig) (string, error) {
	projectName, err := config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.ProjectName, "OS_PROJECT_NAME")
	if err == nil && len(projectName) > 0 {
		return projectName, nil
	}

	//fallback to tenantName
	return config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.TenantName, "OS_TENANT_NAME")
}

// Get the Project id from config or env var. If not defined fallback to tenant id
func getProjectIDOrTenantID(rawConfig *types.RawConfig) (string, error) {
	projectID, err := config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.ProjectID, "OS_PROJECT_ID")
	if err == nil && len(projectID) > 0 {
		return projectID, nil
	}

	//fallback to tenantName
	return config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.TenantID, "OS_TENANT_ID")
}
