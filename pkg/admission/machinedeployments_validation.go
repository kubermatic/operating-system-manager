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

package admission

import (
	"context"
	"encoding/json"
	"fmt"

	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateMachineDeployment(md clusterv1alpha1.MachineDeployment, client ctrlruntimeclient.Client, ospNamespace string) field.ErrorList {
	allErrs := field.ErrorList{}

	ospName := md.Annotations[resources.MachineDeploymentOSPAnnotation]
	// Ignoring request since no OperatingSystemProfile found
	if len(ospName) == 0 {
		// Returning an empty list here as MD without this annotation shouldn't be validated
		return allErrs
	}

	osp := &osmv1alpha1.OperatingSystemProfile{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: ospName, Namespace: ospNamespace}, osp)
	if err != nil && !kerrors.IsNotFound(err) {
		if kerrors.IsNotFound(err) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("metadata", "annotations", resources.MachineDeploymentOSPAnnotation), ospName, "OperatingSystemProfile  not found"))
		} else {
			allErrs = append(allErrs, field.Invalid(field.NewPath("metadata", "annotations", resources.MachineDeploymentOSPAnnotation), ospName, err.Error()))
		}
		// we can't validate further since OSP doesn't exist
		return allErrs
	}

	// Get providerConfig from machineDeployment
	providerConfig := providerconfigtypes.Config{}
	err = json.Unmarshal(md.Spec.Template.Spec.ProviderSpec.Value.Raw, &providerConfig)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "template", "spec", "providerSpec"), md.Spec.Template.Spec.ProviderSpec.Value.Raw, fmt.Sprintf("Failed to decode provider configs: %v", err)))
		return allErrs
	}

	// Ensure that OSP supports the operating system
	if osp.Spec.OSName != osmv1alpha1.OperatingSystem(providerConfig.OperatingSystem) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "template", "spec", "providerSpec", "OperatingSystem"), providerConfig.OperatingSystem, "OperatingSystemProfile does not support the OperatingSystem specified in MachineDeployment"))
	}

	// Ensure that OSP supports the cloud provider
	supportedCloudProvider := false
	for _, cloudProvider := range osp.Spec.SupportedCloudProviders {
		if providerconfigtypes.CloudProvider(cloudProvider.Name) == providerConfig.CloudProvider {
			supportedCloudProvider = true
			break
		}
	}

	// Ensure that OSP supports the operating system
	if !supportedCloudProvider {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "template", "spec", "providerSpec", "CloudProvider"), providerConfig.CloudProvider, "OperatingSystemProfile does not support the CloudProvider specified in MachineDeployment"))
	}
	return allErrs
}
