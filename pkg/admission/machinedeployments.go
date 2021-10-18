/*
Copyright 2019 The Machine Controller Authors.

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

	"k8c.io/operating-system-manager/pkg/controllers/osc/resrources"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
)

func (ad *admissionData) mutateMachineDeployments(ar admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	machineDeploymentOriginal := clusterv1alpha1.MachineDeployment{}
	if err := json.Unmarshal(ar.Object.Raw, &machineDeploymentOriginal); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %v", err)
	}

	machineDeployment := validateMachineDeployment(machineDeploymentOriginal)

	return createAdmissionResponse(&machineDeploymentOriginal, machineDeployment)
}

func validateMachineDeployment(machineDeploymentOriginal clusterv1alpha1.MachineDeployment) *clusterv1alpha1.MachineDeployment {
	machineDeployment := machineDeploymentOriginal.DeepCopy()
	OSP, ok := machineDeploymentOriginal.Annotations[resrources.MachineDeploymentOSPAnnotation]
	if !ok {
		return machineDeployment
	}
	// if OSP does not match any existing OSP, then patch it
	return machineDeployment
}
