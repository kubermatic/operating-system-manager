/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	ospcontroller "k8c.io/operating-system-manager/pkg/controllers/osp"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (ad *admissionData) mutateMachineDeployments(ar admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	machineDeploymentOriginal := clusterv1alpha1.MachineDeployment{}
	if err := json.Unmarshal(ar.Object.Raw, &machineDeploymentOriginal); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %v", err)
	}

	machineDeployment, err := ad.validateMachineDeployment(machineDeploymentOriginal)
	if err != nil {
		return nil, err
	}

	return createAdmissionResponse(&machineDeploymentOriginal, machineDeployment)
}

func (ad *admissionData) validateMachineDeployment(machineDeploymentOriginal clusterv1alpha1.MachineDeployment) (*clusterv1alpha1.MachineDeployment, error) {
	machineDeployment := machineDeploymentOriginal.DeepCopy()
	osp := &osmv1alpha1.OperatingSystemProfile{}

	// check if the machineDeployment is annotated with an existing operatingSystemProfile
	var ospSet bool
	ospName := machineDeployment.Annotations[resources.MachineDeploymentOSPAnnotation]
	if ospName != "" {
		err := ad.seedClient.Get(context.TODO(), client.ObjectKey{Name: ospName, Namespace: ad.clusterNamespace}, osp)
		if err != nil && !kerrors.IsNotFound(err) {
			return nil, err
		}

		if err == nil {
			for _, provider := range osp.Spec.SupportedCloudProviders {
				if provider.Name == ad.provider {
					ospSet = true
				}
			}
		}
	}

	if ospSet {
		return machineDeployment, nil
	}

	// if the MachineDeployment needs to be patched, then retrieve the default OperatingSystemProfile
	// and patch the machineDeployment with the annotation specifying it
	ospList := &osmv1alpha1.OperatingSystemProfileList{}
	if err := ad.seedClient.List(context.TODO(), ospList); err != nil {
		return nil, err
	}
	for _, o := range ospList.Items {
		if provider := o.Annotations[ospcontroller.DefaultOSPAnnotation]; ad.provider == provider {
			if machineDeployment.Annotations == nil {
				machineDeployment.Annotations = make(map[string]string)
			}
			machineDeployment.Annotations[resources.MachineDeploymentOSPAnnotation] = o.Name
			return machineDeployment, nil
		}
	}

	return nil, fmt.Errorf("cannot get default Operating System Profile for machineDeployment %s/%s", machineDeployment.Namespace, machineDeployment.Name)
}
