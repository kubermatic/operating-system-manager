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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
)

func (ad *admissionData) validateMachineDeployments(ar admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	machineDeployment := clusterv1alpha1.MachineDeployment{}
	if err := json.Unmarshal(ar.Object.Raw, &machineDeployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %v", err)
	}

	if errs := ValidateMachineDeployment(machineDeployment, ad.client, ad.ospNamespace); len(errs) > 0 {
		return nil, fmt.Errorf("validation failed: %v", errs)
	}

	return createAdmissionResponse(true), nil
}
