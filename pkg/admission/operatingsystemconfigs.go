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
	"encoding/json"
	"fmt"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	admissionv1 "k8s.io/api/admission/v1"
)

func (ad *admissionData) validateOperatingSystemConfigs(ar admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	osc := osmv1alpha1.OperatingSystemConfig{}
	if err := json.Unmarshal(ar.Object.Raw, &osc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %v", err)
	}

	// Do not validate the spec if it hasn't changed
	if ar.Operation == admissionv1.Update {
		var oscOld osmv1alpha1.OperatingSystemConfig
		if err := json.Unmarshal(ar.OldObject.Raw, &oscOld); err != nil {
			return nil, fmt.Errorf("failed to unmarshal OldObject: %v", err)
		}
		if errs := ad.validateOperatingSystemConfigUpdate(oscOld, osc); len(errs) > 0 {
			return nil, fmt.Errorf("validation failed for update: %v", errs)
		}
	}
	return createAdmissionResponse(true), nil
}
