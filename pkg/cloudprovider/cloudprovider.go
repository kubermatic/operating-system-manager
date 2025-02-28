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
	"fmt"

	"github.com/Masterminds/semver/v3"

	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/operating-system-manager/pkg/cloudprovider/aws"
	"k8c.io/operating-system-manager/pkg/cloudprovider/azure"
	"k8c.io/operating-system-manager/pkg/cloudprovider/gce"
	"k8c.io/operating-system-manager/pkg/cloudprovider/kubevirt"
	"k8c.io/operating-system-manager/pkg/cloudprovider/openstack"
	"k8c.io/operating-system-manager/pkg/cloudprovider/vsphere"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
)

// GetCloudConfig will return the cloud-config for machine
func GetCloudConfig(external bool, pconfig providerconfig.Config, kubeletVersion string) (string, error) {
	// cloud-config is not required by the kubelet for external cloud providers.
	if external {
		return "", nil
	}

	// In-tree cloud providers have been disabled starting from k8s 1.29 hence we don't need to generate cloud-config for them.
	gteKube129Condition, _ := semver.NewConstraint(">= 1.29")
	kubeSemVer, err := semver.NewVersion(kubeletVersion)
	if err != nil {
		return "", fmt.Errorf("failed to parse kubelet version: %w", err)
	}

	if gteKube129Condition.Check(kubeSemVer) {
		return "", nil
	}

	cloudProvider := osmv1alpha1.CloudProvider(pconfig.CloudProvider)

	switch cloudProvider {
	case osmv1alpha1.CloudProviderAWS:
		return aws.GetCloudConfig(pconfig)
	case osmv1alpha1.CloudProviderAzure:
		return azure.GetCloudConfig(pconfig)
	case osmv1alpha1.CloudProviderGoogle:
		return gce.GetCloudConfig(pconfig)
	case osmv1alpha1.CloudProviderKubeVirt:
		return kubevirt.GetCloudConfig(pconfig)
	case osmv1alpha1.CloudProviderOpenstack:
		return openstack.GetCloudConfig(pconfig, kubeletVersion)
	case osmv1alpha1.CloudProviderVsphere:
		return vsphere.GetCloudConfig(pconfig)

	// cloud-config is not required for these cloud providers
	case osmv1alpha1.CloudProviderAlibaba, osmv1alpha1.CloudProviderAnexia, osmv1alpha1.CloudProviderDigitalocean,
		osmv1alpha1.CloudProviderHetzner, osmv1alpha1.CloudProviderLinode, osmv1alpha1.CloudProviderEquinixMetal,
		osmv1alpha1.CloudProviderScaleway, osmv1alpha1.CloudProviderNutanix, osmv1alpha1.CloudProviderVMwareCloudDirector,
		osmv1alpha1.CloudProviderOpenNebula, osmv1alpha1.CloudProviderEdge, osmv1alpha1.CloudProviderBaremetal:
		return "", nil
	}

	return "", errors.New("unknown cloud provider")
}

func KubeletCloudProviderConfig(cloudProvider providerconfig.CloudProvider, external bool) (inTreeCCM bool, outOfTree bool, err error) {
	switch osmv1alpha1.CloudProvider(cloudProvider) {
	case osmv1alpha1.CloudProviderAWS,
		osmv1alpha1.CloudProviderAzure,
		osmv1alpha1.CloudProviderGoogle,
		osmv1alpha1.CloudProviderVsphere:
		return !external, external, nil

	default:
		return false, external, nil
	}
}
