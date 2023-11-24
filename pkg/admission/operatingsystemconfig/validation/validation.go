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
	"fmt"
	"net/http"

	"go.uber.org/zap"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	admissionv1 "k8s.io/api/admission/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for validating OperatingSystemConfig CRD.
type AdmissionHandler struct {
	log     *zap.SugaredLogger
	decoder *admission.Decoder
}

// NewAdmissionHandler returns a new validation AdmissionHandler.
func NewAdmissionHandler(log *zap.SugaredLogger, scheme *runtime.Scheme) *AdmissionHandler {
	return &AdmissionHandler{
		log:     log,
		decoder: admission.NewDecoder(scheme),
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/operatingsystemconfig", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) Handle(_ context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	osc := &osmv1alpha1.OperatingSystemConfig{}
	oldOSC := &osmv1alpha1.OperatingSystemConfig{}

	switch req.Operation {
	case admissionv1.Update:
		if err := h.decoder.Decode(req, osc); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding osc: %w", err))
		}
		if err := h.decoder.DecodeRaw(req.OldObject, oldOSC); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("error occurred while decoding old osc: %w", err))
		}
		err := h.validateUpdate(osc, oldOSC)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("operatingSystemConfig validation request %s denied: %v", req.UID, err))
		}

	case admissionv1.Create, admissionv1.Delete:
		// NOP we always allow create, delete operarions at the moment

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on osc resources", req.Operation))
	}

	return webhook.Allowed(fmt.Sprintf("operatingSystemConfig validation request %s allowed", req.UID))
}

func (h *AdmissionHandler) validateUpdate(osc, oldOSC *osmv1alpha1.OperatingSystemConfig) error {
	// Updates for OperatingSystemConfig Spec are not allowed
	if equal := apiequality.Semantic.DeepEqual(oldOSC.Spec, osc.Spec); !equal {
		return fmt.Errorf("OperatingSystemConfig is immutable and updates are not allowed")
	}
	return nil
}
