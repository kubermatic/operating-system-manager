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

package mutation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	clusterv1alpha1 "k8c.io/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfigtypes "k8c.io/machine-controller/pkg/providerconfig/types"
	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const ospNamePattern = "osp-%s"

// AdmissionHandler for mutating MachineDeployment CRD.
type AdmissionHandler struct {
	log     *zap.SugaredLogger
	decoder admission.Decoder
}

// NewAdmissionHandler returns a new validation AdmissionHandler.
func NewAdmissionHandler(log *zap.SugaredLogger, scheme *runtime.Scheme) *AdmissionHandler {
	return &AdmissionHandler{
		log:     log,
		decoder: admission.NewDecoder(scheme),
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register("/mutate-v1alpha1-machinedeployment", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) Handle(_ context.Context, req admission.Request) admission.Response {
	md := &clusterv1alpha1.MachineDeployment{}

	switch req.Operation {
	case admissionv1.Create, admissionv1.Update:
		if err := h.decoder.Decode(req, md); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding reqquest: %w", err))
		}

	case admissionv1.Delete:
		// NOP we don't need mutations for delete operations
		return admission.Allowed(fmt.Sprintf("machinedeployment mutation request %s allowed; mutation is not required for delete request", req.UID))

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on machinedeployment resources", req.Operation))
	}

	if err := MutateMachineDeployment(md); err != nil {
		return admission.Errored(http.StatusInternalServerError, fmt.Errorf("error occurred while mutating machinedeployment: %w", err))
	}

	marshaledMd, err := json.Marshal(md)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, fmt.Errorf("error occurred while marshalling mutated machinedeployment: %w", err))
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledMd)
}

func MutateMachineDeployment(md *clusterv1alpha1.MachineDeployment) error {
	providerConfig, err := providerconfigtypes.GetConfig(md.Spec.Template.Spec.ProviderSpec)
	if err != nil {
		return fmt.Errorf("failed to read MachineDeployment.Spec.Template.Spec.ProviderSpec: %w", err)
	}

	// Check for existing annotation if it doesn't exist or if the value is empty
	// inject the appropriate annotation.
	if val, ok := md.Annotations[resources.MachineDeploymentOSPAnnotation]; !ok || val == "" {
		if md.Annotations == nil {
			md.Annotations = make(map[string]string)
		}

		if providerConfig.CloudProvider == providerconfigtypes.CloudProviderAnexia && providerConfig.OperatingSystem == providerconfigtypes.OperatingSystemFlatcar {
			prov, err := generator.GetProvisioningUtility(v1alpha1.OperatingSystem(providerConfig.OperatingSystem), *md)
			// We are intentionally ignoring any errors here since in that case the standard workflow should be used to determine the OSP.
			if err == nil && prov == v1alpha1.ProvisioningUtilityCloudInit {
				md.Annotations[resources.MachineDeploymentOSPAnnotation] = "osp-flatcar-cloud-init"
				return nil
			}
		}

		switch providerConfig.OperatingSystem {
		case providerconfigtypes.OperatingSystemUbuntu,
			providerconfigtypes.OperatingSystemFlatcar,
			providerconfigtypes.OperatingSystemAmazonLinux2,
			providerconfigtypes.OperatingSystemRockyLinux,
			providerconfigtypes.OperatingSystemRHEL:

			md.Annotations[resources.MachineDeploymentOSPAnnotation] = fmt.Sprintf(ospNamePattern, providerConfig.OperatingSystem)
		default:
			return fmt.Errorf("failed to populate OSP annotation for machinedeployment with unsupported Operating System %s", providerConfig.OperatingSystem)
		}
	}

	return nil
}
