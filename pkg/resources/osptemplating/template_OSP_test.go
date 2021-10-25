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

package osptemplating

import (
	"testing"

	"github.com/go-test/deep"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8s.io/utils/pointer"
)

func TestContainerRuntimeSetup(t *testing.T) {
	tests := []struct {
		inputOSP         *osmv1alpha1.OperatingSystemProfile
		expectedOSPFiles []osmv1alpha1.File
		containerRuntime string
	}{
		{
			containerRuntime: "containerd",
			inputOSP: &osmv1alpha1.OperatingSystemProfile{
				Spec: osmv1alpha1.OperatingSystemProfileSpec{
					SupportedContainerRuntimes: []osmv1alpha1.ContainerRuntimeSpec{
						{
							Name: "containerd",
							ScriptFile: `
{{- define "containerRuntimeScript" }}
multiline
container runtime setup script
{{- end }}`,
							ConfigFile: `
{{- define "containerRuntimeConfig" }}
multiline
container runtime config file
{{- end }}`,
						},
					},
					Files: []osmv1alpha1.File{
						{
							Path:        "/opt/bin/setup",
							Permissions: pointer.Int32Ptr(755),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Encoding: "b64",
									Data: `
first part of the script
{{ template "containerRuntimeScript" }}

last part of the script`,
								},
							},
						},
						{
							Path:        "/etc/containerd/config.toml",
							Permissions: pointer.Int32Ptr(755),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Encoding: "b64",
									Data: `
{{- template "containerRuntimeConfig" }}`,
								},
							},
						},
					},
				},
			},
			expectedOSPFiles: []osmv1alpha1.File{
				{
					Path:        "/opt/bin/setup",
					Permissions: pointer.Int32Ptr(755),
					Content: osmv1alpha1.FileContent{
						Inline: &osmv1alpha1.FileContentInline{
							Encoding: "b64",
							Data: `
first part of the script

multiline
container runtime setup script

last part of the script`,
						},
					},
				},
				{
					Path:        "/etc/containerd/config.toml",
					Permissions: pointer.Int32Ptr(755),
					Content: osmv1alpha1.FileContent{
						Inline: &osmv1alpha1.FileContentInline{
							Encoding: "b64",
							Data: `
multiline
container runtime config file`,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		osp := tc.inputOSP.DeepCopy()
		if err := SetupContainerRuntime(tc.containerRuntime, osp); err != nil {
			t.Fatalf(err.Error())
		}
		if diff := deep.Equal(osp.Spec.Files, tc.expectedOSPFiles); diff != nil {
			t.Errorf("expected OSP is different from received one: %v", diff)
		}
	}
}
