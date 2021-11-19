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

package vsphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"k8c.io/operating-system-manager/pkg/cloudprovider/vsphere/types"
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
		opts types.GlobalOpts
		err  error
	)

	vsphereURL, err := getURL(rawConfig)
	if err != nil {
		return nil, err
	}

	opts.VCenterPort = vsphereURL.Port()
	opts.User, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.Username, "VSPHERE_USERNAME")
	if err != nil {
		return nil, err
	}

	opts.Password, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.Password, "VSPHERE_PASSWORD")
	if err != nil {
		return nil, err
	}

	opts.InsecureFlag, err = config.GetConfigVarResolver().GetConfigVarBoolValueOrEnv(rawConfig.AllowInsecure, "VSPHERE_ALLOW_INSECURE")
	if err != nil {
		return nil, err
	}

	opts.ClusterID, err = config.GetConfigVarResolver().GetConfigVarStringValue(rawConfig.Cluster)
	if err != nil {
		return nil, err
	}

	workingDir := rawConfig.Folder
	// Default to basedir
	if workingDir == "" {
		workingDir = fmt.Sprintf("/%s/vm", rawConfig.Datacenter)
	}

	cloudConfig := &types.CloudConfig{
		Global: opts,
		Disk: types.DiskOpts{
			SCSIControllerType: "pvscsi",
		},
		Workspace: types.WorkspaceOpts{
			Datacenter:       rawConfig.Datacenter,
			VCenterIP:        vsphereURL.Hostname(),
			DefaultDatastore: rawConfig.Datastore,
			Folder:           workingDir,
		},
		VirtualCenter: map[string]*types.VirtualCenterConfig{
			vsphereURL.Hostname(): {
				VCenterPort: vsphereURL.Port(),
				Datacenters: rawConfig.Datacenter,
				User:        opts.User,
				Password:    opts.Password,
			},
		},
	}

	return cloudConfig, nil
}

func getURL(rawConfig types.RawConfig) (*url.URL, error) {
	vsphereURL, err := config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.VSphereURL, "VSPHERE_ADDRESS")
	if err != nil {
		return nil, err
	}

	// Required because url.Parse returns an empty string for the hostname if there was no schema
	if !strings.HasPrefix(vsphereURL, "https://") {
		vsphereURL = "https://" + vsphereURL
	}

	u, err := url.Parse(vsphereURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse '%s' as url: %v", vsphereURL, err)
	}
	return u, nil
}
