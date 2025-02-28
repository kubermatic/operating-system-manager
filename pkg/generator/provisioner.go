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

package generator

import (
	"fmt"

	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/jsonutil"
	"k8c.io/machine-controller/sdk/providerconfig"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/providerconfig/flatcar"
)

// GetProvisioningUtility returns the provisioning utility for the given machine
func GetProvisioningUtility(osName osmv1alpha1.OperatingSystem, md clusterv1alpha1.MachineDeployment) (osmv1alpha1.ProvisioningUtility, error) {
	// We need to check if `ProvisioningUtility` was explicitly specified in the machine deployment. If not then we
	// will always default to `ignition`.
	if osName == osmv1alpha1.OperatingSystemFlatcar {
		providerConfig := providerconfig.Config{}
		if err := jsonutil.StrictUnmarshal(md.Spec.Template.Spec.ProviderSpec.Value.Raw, &providerConfig); err != nil {
			return "", fmt.Errorf("failed to decode provider configs: %w", err)
		}

		config, err := flatcar.LoadConfig(providerConfig.OperatingSystemSpec)
		if err != nil {
			return "", err
		}

		if config.ProvisioningUtility == "" {
			return osmv1alpha1.ProvisioningUtilityIgnition, err
		}
		return osmv1alpha1.ProvisioningUtility(config.ProvisioningUtility), err
	}
	// Only flatcar supports ignition.
	return osmv1alpha1.ProvisioningUtilityCloudInit, nil
}
