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
	"encoding/json"
	"fmt"
	"testing"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/go-test/deep"
)

func TestMutateMachineDeployment(t *testing.T) {
	tests := []struct {
		name                      string
		machineDeployment         *clusterv1alpha1.MachineDeployment
		expectedMachineDeployment *clusterv1alpha1.MachineDeployment
		expectedError             string
	}{
		{
			name: "MachineDeployment with no OSP annotation",
			machineDeployment: &clusterv1alpha1.MachineDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "default",
					Namespace:   "kube-system",
					Annotations: nil,
				},
				Spec: clusterv1alpha1.MachineDeploymentSpec{
					Template: clusterv1alpha1.MachineTemplateSpec{
						Spec: clusterv1alpha1.MachineSpec{
							Versions: clusterv1alpha1.MachineVersionInfo{
								Kubelet: "1.22.1",
							},
							ProviderSpec: clusterv1alpha1.ProviderSpec{
								Value: &runtime.RawExtension{
									Raw: generateRawConfig(t, "ubuntu", "aws"),
								},
							},
						},
					},
				},
			},
			expectedMachineDeployment: &clusterv1alpha1.MachineDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "kube-system",
					Annotations: map[string]string{
						"k8c.io/operating-system-profile": "osp-ubuntu",
					},
				},
				Spec: clusterv1alpha1.MachineDeploymentSpec{
					Template: clusterv1alpha1.MachineTemplateSpec{
						Spec: clusterv1alpha1.MachineSpec{
							Versions: clusterv1alpha1.MachineVersionInfo{
								Kubelet: "1.22.1",
							},
							ProviderSpec: clusterv1alpha1.ProviderSpec{
								Value: &runtime.RawExtension{
									Raw: generateRawConfig(t, "ubuntu", "aws"),
								},
							},
						},
					},
				},
			},
			expectedError: "",
		},
	}
	for _, tc := range tests {
		tc := tc // scopelint fix
		t.Run(tc.name, func(t *testing.T) {
			md := tc.machineDeployment.DeepCopy()
			errs := MutateMachineDeployment(md)
			if errs != nil && len(tc.expectedError) == 0 {
				t.Errorf("didn't expect err but got %v", errs)
				return
			}
			if errs == nil && len(tc.expectedError) > 0 {
				t.Errorf("expected err %v but got valid response", tc.expectedError)
				return
			}
			if errs != nil && tc.expectedError != fmt.Sprintf("%v", errs) {
				t.Errorf("actual error %v didn't match expected error %v", errs, tc.expectedError)
				return
			}

			if diff := deep.Equal(md, tc.expectedMachineDeployment); len(diff) > 0 {
				t.Errorf("result of mutation did not match expected MachineDeployment, diff: %+v", diff)
			}
		})
	}
}

func getOperatingSystemProfile() *osmv1alpha1.OperatingSystemProfile {
	return &osmv1alpha1.OperatingSystemProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ubuntu",
			Namespace: "default",
		},
		Spec: osmv1alpha1.OperatingSystemProfileSpec{
			OSName:    "ubuntu",
			OSVersion: "2.0",
			Version:   "1.0.0",
			SupportedCloudProviders: []osmv1alpha1.CloudProviderSpec{
				{
					Name: "aws",
				},
			},
			ProvisioningConfig: osmv1alpha1.OSPConfig{
				SupportedContainerRuntimes: []osmv1alpha1.ContainerRuntimeSpec{
					{
						Name: "containerd",
					},
				},
				Files: []osmv1alpha1.File{
					{
						Path: "/etc/systemd/journald.conf.d/max_disk_use.conf",
						Content: osmv1alpha1.FileContent{
							Inline: &osmv1alpha1.FileContentInline{
								Encoding: "b64",
								Data:     "test",
							},
						},
					},
				},
			},
		},
	}
}

func generateRawConfig(t *testing.T, os providerconfigtypes.OperatingSystem, cloudprovider string) []byte {
	pconfig := providerconfigtypes.Config{
		SSHPublicKeys:     []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c"},
		OperatingSystem:   os,
		CloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"cloudProvider":"aws", "cloudProviderSpec":"test-provider-spec"}`)},
		CloudProvider:     providerconfigtypes.CloudProvider(cloudprovider),
	}
	mdConfig, err := json.Marshal(pconfig)
	if err != nil {
		t.Fatalf("failed to generate machine deployment: %v", err)
	}

	return mdConfig
}

func generateMachineDeployment(t *testing.T, osp string, os providerconfigtypes.OperatingSystem, cloudprovider string) clusterv1alpha1.MachineDeployment {
	pconfig := providerconfigtypes.Config{
		SSHPublicKeys:     []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3c"},
		OperatingSystem:   os,
		CloudProviderSpec: runtime.RawExtension{Raw: []byte(`{"cloudProvider":"aws", "cloudProviderSpec":"test-provider-spec"}`)},
		CloudProvider:     providerconfigtypes.CloudProvider(cloudprovider),
	}
	mdConfig, err := json.Marshal(pconfig)
	if err != nil {
		t.Fatalf("failed to generate machine deployment: %v", err)
	}

	annotations := make(map[string]string)
	if osp != "" {
		annotations = map[string]string{
			resources.MachineDeploymentOSPAnnotation: osp,
		}
	}

	md := clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "default",
			Namespace:   "kube-system",
			Annotations: annotations,
		},
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Template: clusterv1alpha1.MachineTemplateSpec{
				Spec: clusterv1alpha1.MachineSpec{
					Versions: clusterv1alpha1.MachineVersionInfo{
						Kubelet: "1.22.1",
					},
					ProviderSpec: clusterv1alpha1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: mdConfig,
						},
					},
				},
			},
		},
	}
	return md
}
