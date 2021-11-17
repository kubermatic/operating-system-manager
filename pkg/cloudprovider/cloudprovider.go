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

package cloudprovider

import (
	"errors"

	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
)

// GetCloudConfig will return the cloud-config for machine
func GetCloudConfig(config providerconfigtypes.Config) (string, error) {
	cloudProvider := osmv1alpha1.CloudProvider(config.CloudProvider)
	switch cloudProvider {

	// cloud-config is not required for these cloud providers
	case osmv1alpha1.CloudProviderAlibaba:
	case osmv1alpha1.CloudProviderAnexia:
	case osmv1alpha1.CloudProviderDigitalocean:
	case osmv1alpha1.CloudProviderHetzner:
	case osmv1alpha1.CloudProviderLinode:
	case osmv1alpha1.CloudProviderPacket:
	case osmv1alpha1.CloudProviderScaleway:
		return "", nil
	}

	return "", errors.New("unknown cloud provider")
}