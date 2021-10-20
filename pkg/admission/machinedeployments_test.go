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
	"fmt"
	"testing"

	"k8c.io/operating-system-manager/pkg/controllers/osc/resrources"
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
	osmv1alpha1.AddToScheme(seedScheme)
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
				seedClient:       fakectrlruntimeclient.NewClientBuilder().WithScheme(seedScheme).Build(),
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
						resrources.MachineDeploymentOSPAnnotation: "default_osp_test",
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
						resrources.MachineDeploymentOSPAnnotation: "osp_test",
					},
				},
			},
			admissionData: admissionData{
				client:           fakectrlruntimeclient.NewClientBuilder().Build(),
				seedClient:       fakectrlruntimeclient.NewClientBuilder().WithScheme(seedScheme).Build(),
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
						resrources.MachineDeploymentOSPAnnotation: "osp_test",
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
						resrources.MachineDeploymentOSPAnnotation: "osp_test",
					},
				},
			},
			admissionData: admissionData{
				client:           fakectrlruntimeclient.NewClientBuilder().Build(),
				seedClient:       fakectrlruntimeclient.NewClientBuilder().WithScheme(seedScheme).Build(),
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
						resrources.MachineDeploymentOSPAnnotation: "default_osp_test",
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
				seedClient:       fakectrlruntimeclient.NewClientBuilder().WithScheme(seedScheme).Build(),
				provider:         "aws",
				clusterNamespace: "kube-system",
			},
			err: fmt.Errorf("cannot get default Operating System Profile for machineDeployment mdTestNamespace/mdtestName"),
		},
	}

	for _, test := range tests {
		test.admissionData.seedClient = fakectrlruntimeclient.NewClientBuilder().WithScheme(seedScheme).WithLists(&test.operatingSystemProfiles).Build()

		t.Run(test.name, func(t *testing.T) {
			mutatedMachineDeployment, err := test.admissionData.validateMachineDeployment(test.machineDeployment)
			if test.err != nil && err == nil {
				t.Errorf("expected err %v not triggered", test.err)
				return
			}
			if err != nil && test.err == nil {
				t.Errorf("unexpected err %v", err)
				return
			}
			if err != nil && test.err != nil && err.Error() != test.err.Error() {
				t.Errorf("received error %v different from the expected one %v", err, test.err)
				return
			}
			if diff := deep.Equal(mutatedMachineDeployment, test.expectedMachineDeployment); diff != nil {
				if diff != nil {
					t.Errorf("received machineDeployment different from the expected one, diff: %v", diff)
				}
			}
		})
	}
}
