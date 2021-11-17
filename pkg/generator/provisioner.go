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
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
)

// ProvisioningUtility specifies the type of utility used for machine provisioning
type ProvisioningUtility string

const (
	Ignition  ProvisioningUtility = "ignition"
	CloudInit ProvisioningUtility = "cloud-init"
)

// GetProvisioningUtility returns the provisioning utility for the given machine
func GetProvisioningUtility(osName osmv1alpha1.OperatingSystem) ProvisioningUtility {
	switch osName {
	case osmv1alpha1.OperatingSystemFlatcar:
		return Ignition
	default:
		return CloudInit
	}
}
