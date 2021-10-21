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
	"fmt"
	"testing"

	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	ospcontroller "k8c.io/operating-system-manager/pkg/controllers/osp"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	"github.com/go-test/deep"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	mdTestName      = "mdtestName"
	mdTestNamespace = "mdTestNamespace"
)

var (
	seedScheme = runtime.NewScheme()
)

func TestMachineDeploymentMutation(t *testing.T) {
	tests := []struct {
		name                      string
		admissionData             admissionData
		machineDeployment         clusterv1alpha1.MachineDeployment
		operatingSystemProfiles   osmv1alpha1.OperatingSystemProfileList
		expectedMachineDeployment *clusterv1alpha1.MachineDeployment
		err                       error
	}{
		{
			name: "No OSP specified, existing default OSP",
			machineDeployment: clusterv1alpha1.MachineDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mdTestName,
					Namespace: mdTestNamespace,
				},
			},
			admissionData: admissionData{
				client:           fakectrlruntimeclient.NewClientBuilder().Build(),
				provider:         "aws",
				clusterNamespace: "kube-system",
			},
			operatingSystemProfiles: osmv1alpha1.OperatingSystemProfileList{
				Items: []osmv1alpha1.OperatingSystemProfile{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "default_osp_test",
							Namespace: "kube-system",
							Annotations: map[string]string{
								ospcontroller.DefaultOSPAnnotation: "aws",
							},
						},
						Spec: osmv1alpha1.OperatingSystemProfileSpec{
							SupportedCloudProviders: []osmv1alpha1.CloudProviderSpec{
								{
									Name: "aws",
								},
							},
						},
					},
				},
			},
			expectedMachineDeployment: &clusterv1alpha1.MachineDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mdTestName,
					Namespace: mdTestNamespace,
					Annotations: map[string]string{
						resources.MachineDeploymentOSPAnnotation: "default_osp_test",
					},
				},
			},
		},
		{
			name: "Existing OSP specified",
			machineDeployment: clusterv1alpha1.MachineDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mdTestName,
					Namespace: mdTestNamespace,
					Annotations: map[string]string{
						resources.MachineDeploymentOSPAnnotation: "osp_test",
					},
				},
			},
			admissionData: admissionData{
				client:           fakectrlruntimeclient.NewClientBuilder().Build(),
				provider:         "aws",
				clusterNamespace: "kube-system",
			},
			operatingSystemProfiles: osmv1alpha1.OperatingSystemProfileList{
				Items: []osmv1alpha1.OperatingSystemProfile{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "osp_test",
							Namespace: "kube-system",
						},
						Spec: osmv1alpha1.OperatingSystemProfileSpec{
							SupportedCloudProviders: []osmv1alpha1.CloudProviderSpec{
								{
									Name: "aws",
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "default_osp_test",
							Namespace: "kube-system",
							Annotations: map[string]string{
								ospcontroller.DefaultOSPAnnotation: "aws",
							},
						},
						Spec: osmv1alpha1.OperatingSystemProfileSpec{
							OSName: "alibaba",
							SupportedCloudProviders: []osmv1alpha1.CloudProviderSpec{
								{
									Name: "aws",
								},
							},
						},
					},
				},
			},
			expectedMachineDeployment: &clusterv1alpha1.MachineDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mdTestName,
					Namespace: mdTestNamespace,
					Annotations: map[string]string{
						resources.MachineDeploymentOSPAnnotation: "osp_test",
					},
				},
			},
		},
		{
			name: "Not existing OSP specified, default OSP existing",
			machineDeployment: clusterv1alpha1.MachineDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mdTestName,
					Namespace: mdTestNamespace,
					Annotations: map[string]string{
						resources.MachineDeploymentOSPAnnotation: "osp_test",
					},
				},
			},
			admissionData: admissionData{
				client:           fakectrlruntimeclient.NewClientBuilder().Build(),
				provider:         "aws",
				clusterNamespace: "kube-system",
			},
			operatingSystemProfiles: osmv1alpha1.OperatingSystemProfileList{
				Items: []osmv1alpha1.OperatingSystemProfile{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "default_osp_test",
							Namespace: "kube-system",
							Annotations: map[string]string{
								ospcontroller.DefaultOSPAnnotation: "aws",
							},
						},
						Spec: osmv1alpha1.OperatingSystemProfileSpec{
							SupportedCloudProviders: []osmv1alpha1.CloudProviderSpec{
								{
									Name: "aws",
								},
							},
						},
					},
				},
			},
			expectedMachineDeployment: &clusterv1alpha1.MachineDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mdTestName,
					Namespace: mdTestNamespace,
					Annotations: map[string]string{
						resources.MachineDeploymentOSPAnnotation: "default_osp_test",
					},
				},
			},
		},
		{
			name: "No OSP specified, no default OSP",
			machineDeployment: clusterv1alpha1.MachineDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mdTestName,
					Namespace: mdTestNamespace,
				},
			},
			admissionData: admissionData{
				client:           fakectrlruntimeclient.NewClientBuilder().Build(),
				provider:         "aws",
				clusterNamespace: "kube-system",
			},
			err: fmt.Errorf("cannot get default Operating System Profile for machineDeployment mdTestNamespace/mdtestName"),
		},
	}

	_ = osmv1alpha1.AddToScheme(seedScheme)

	for _, tc := range tests {
		tc.admissionData.seedClient = fakectrlruntimeclient.NewClientBuilder().WithScheme(seedScheme).WithLists(&tc.operatingSystemProfiles).Build()
		tc := tc // scopelint fix
		t.Run(tc.name, func(t *testing.T) {
			mutatedMachineDeployment, err := tc.admissionData.validateMachineDeployment(tc.machineDeployment)
			if tc.err != nil && err == nil {
				t.Errorf("expected err %v not triggered", tc.err)
				return
			}
			if err != nil && tc.err == nil {
				t.Errorf("unexpected err %v", err)
				return
			}
			if err != nil && tc.err != nil && err.Error() != tc.err.Error() {
				t.Errorf("received error %v different from the expected one %v", err, tc.err)
				return
			}
			if diff := deep.Equal(mutatedMachineDeployment, tc.expectedMachineDeployment); diff != nil {
				if diff != nil {
					t.Errorf("received machineDeployment different from the expected one, diff: %v", diff)
				}
			}
		})
	}
}
