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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMachineDeploymentValidation(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = osmv1alpha1.AddToScheme(scheme)
	_ = clusterv1alpha1.AddToScheme(scheme)

	osp := getOperatingSystemProfile()
	fakeClient := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(scheme).
		WithObjects(osp).
		Build()

	ah := AdmissionHandler{
		client:    fakeClient,
		namespace: "default",
	}

	tests := []struct {
		name              string
		machineDeployment clusterv1alpha1.MachineDeployment
		expectedError     string
	}{
		{
			name:              "MachineDeployment with no OSP annotation",
			machineDeployment: generateMachineDeployment(t, "", "ubuntu", "aws"),
			expectedError:     "[]",
		},
		{
			name:              "MachineDeployment with non-existing OSP specified",
			machineDeployment: generateMachineDeployment(t, "invalid", "ubuntu", "aws"),
			expectedError:     `[spec.template.spec.providerSpec.OperatingSystem: Invalid value: "ubuntu": OperatingSystemProfile does not support the OperatingSystem specified in MachineDeployment spec.template.spec.providerSpec.CloudProvider: Invalid value: "aws": OperatingSystemProfile does not support the CloudProvider specified in MachineDeployment]`,
		},
		{
			name:              "MachineDeployment with in-compatible OS",
			machineDeployment: generateMachineDeployment(t, "ubuntu", "sles", "aws"),
			expectedError:     `[spec.template.spec.providerSpec.OperatingSystem: Invalid value: "sles": OperatingSystemProfile does not support the OperatingSystem specified in MachineDeployment]`,
		},
		{
			name:              "MachineDeployment with in-compatible cloud provider",
			machineDeployment: generateMachineDeployment(t, "ubuntu", "ubuntu", "azure"),
			expectedError:     `[spec.template.spec.providerSpec.CloudProvider: Invalid value: "azure": OperatingSystemProfile does not support the CloudProvider specified in MachineDeployment]`,
		},
		{
			name:              "MachineDeployment with compatible OS and cloud provider",
			machineDeployment: generateMachineDeployment(t, "ubuntu", "ubuntu", "aws"),
			expectedError:     "[]",
		},
	}
	for _, tc := range tests {
		tc := tc // scopelint fix
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateMachineDeployment(context.TODO(), tc.machineDeployment, ah.client, ah.namespace)
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
	}
}

func generateMachineDeployment(t *testing.T, osp string, os providerconfigtypes.OperatingSystem, cloudprovider string) v1alpha1.MachineDeployment {
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
