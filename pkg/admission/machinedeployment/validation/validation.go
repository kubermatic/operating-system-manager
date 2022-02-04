/*
Copyright 2022 The Operating System Manager contributors.

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

package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	admissionv1 "k8s.io/api/admission/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for validating MachineDeployment CRD.
type AdmissionHandler struct {
	log     logr.Logger
	decoder *admission.Decoder

	client    ctrlruntimeclient.Client
	namespace string
}

// NewAdmissionHandler returns a new validation AdmissionHandler.
func NewAdmissionHandler(client ctrlruntimeclient.Client, namespace string) *AdmissionHandler {
	return &AdmissionHandler{}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register("/machinedeployment", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("machine-deployment-validation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	allErrs := field.ErrorList{}
	md := &clusterv1alpha1.MachineDeployment{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, md); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		allErrs = append(allErrs, h.validateMachineDeployments(ctx, *md)...)

	case admissionv1.Update:
		if err := h.decoder.Decode(req, md); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding machinedeployment: %w", err))
		}
		allErrs = append(allErrs, h.validateMachineDeployments(ctx, *md)...)

	case admissionv1.Delete:
		// NOP we don't need validations for delete operations

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on machinedeployment resources", req.Operation))
	}

	if len(allErrs) > 0 {
		return webhook.Denied(fmt.Sprintf("machinedeployment validation request %s denied: %v", req.UID, allErrs))
	}

	return webhook.Allowed(fmt.Sprintf("machinedeployment validation request %s allowed", req.UID))
}

func (h *AdmissionHandler) validateMachineDeployments(ctx context.Context, md clusterv1alpha1.MachineDeployment) field.ErrorList {
	return ValidateMachineDeployment(ctx, md, h.client, h.namespace)
}

func ValidateMachineDeployment(ctx context.Context, md clusterv1alpha1.MachineDeployment, client ctrlruntimeclient.Client, namespace string) field.ErrorList {
	allErrs := field.ErrorList{}

	ospName := md.Annotations[resources.MachineDeploymentOSPAnnotation]
	// Ignoring request since no OperatingSystemProfile found
	if len(ospName) == 0 {
		// Returning an empty list here as MD without this annotation shouldn't be validated
		return allErrs
	}

	osp := &osmv1alpha1.OperatingSystemProfile{}
	err := client.Get(ctx, types.NamespacedName{Name: ospName, Namespace: namespace}, osp)
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
