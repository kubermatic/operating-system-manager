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

	"github.com/go-logr/logr"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"

	admissionv1 "k8s.io/api/admission/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const ospNamePattern = "osp-%s"

// AdmissionHandler for mutating MachineDeployment CRD.
type AdmissionHandler struct {
	log     logr.Logger
	decoder *admission.Decoder
}

// NewAdmissionHandler returns a new mutation AdmissionHandler for MachineDeployments..
func NewAdmissionHandler() *AdmissionHandler {
	return &AdmissionHandler{}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register("/mutate-v1alpha1-machinedeployment", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("machine-deployment-mutation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
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

		switch providerConfig.OperatingSystem {
		case providerconfigtypes.OperatingSystemUbuntu,
			providerconfigtypes.OperatingSystemCentOS,
			providerconfigtypes.OperatingSystemFlatcar,
			providerconfigtypes.OperatingSystemAmazonLinux2,
			providerconfigtypes.OperatingSystemRockyLinux,
			providerconfigtypes.OperatingSystemSLES,
			providerconfigtypes.OperatingSystemRHEL:

			md.Annotations[resources.MachineDeploymentOSPAnnotation] = fmt.Sprintf(ospNamePattern, providerConfig.OperatingSystem)
		default:
			return fmt.Errorf("failed to populate OSP annotation for machinedeployment with unsupported Operating System %s", providerConfig.OperatingSystem)
		}
	}

	return nil
}
