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
	"fmt"
	"testing"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestOpeartingSystemProfileValidation(t *testing.T) {
	tests := []struct {
		name          string
		osp           osmv1alpha1.OperatingSystemProfile
		ospUpdated    osmv1alpha1.OperatingSystemProfile
		expectedError string
	}{
		{
			name: "Update OSP spec",
			osp: osmv1alpha1.OperatingSystemProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ubuntu",
					Namespace: "default",
				},
				Spec: osmv1alpha1.OperatingSystemProfileSpec{
					OSName:  "ubuntu",
					Version: "1.0.0",
					Files: []osmv1alpha1.File{
						{
							Path:        "/opt/bin/test.service",
							Permissions: pointer.Int32Ptr(0700),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Data: "    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
								},
							},
						},
					},
				},
			},
			ospUpdated: osmv1alpha1.OperatingSystemProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ubuntu",
					Namespace: "default",
				},
				Spec: osmv1alpha1.OperatingSystemProfileSpec{
					OSName:  "ubuntu",
					Version: "1.0.0",
					Files: []osmv1alpha1.File{
						{
							Path:        "/opt/bin/test.service-updated",
							Permissions: pointer.Int32Ptr(0700),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Data: "    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
								},
							},
						},
					},
				},
			},
			expectedError: "OperatingSystemProfile is immutable. For updates .spec.version needs to be updated",
		},
		{
			name: "Should update OSP spec with version bump",
			osp: osmv1alpha1.OperatingSystemProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ubuntu",
					Namespace: "default",
				},
				Spec: osmv1alpha1.OperatingSystemProfileSpec{
					OSName:  "ubuntu",
					Version: "1.0.0",
					Files: []osmv1alpha1.File{
						{
							Path:        "/opt/bin/test.service",
							Permissions: pointer.Int32Ptr(0700),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Data: "    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
								},
							},
						},
					},
				},
			},
			ospUpdated: osmv1alpha1.OperatingSystemProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ubuntu",
					Namespace: "default",
				},
				Spec: osmv1alpha1.OperatingSystemProfileSpec{
					OSName:  "ubuntu",
					Version: "1.0.1",
					Files: []osmv1alpha1.File{
						{
							Path:        "/opt/bin/test.service-updated",
							Permissions: pointer.Int32Ptr(0700),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Data: "    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
								},
							},
						},
					},
				},
			},
			expectedError: "",
		},
		{
			name: "Update osp labels",
			osp: osmv1alpha1.OperatingSystemProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ubuntu",
					Namespace: "default",
				},
				Spec: osmv1alpha1.OperatingSystemProfileSpec{
					OSName: "ubuntu",
					Files: []osmv1alpha1.File{
						{
							Path:        "/opt/bin/test.service",
							Permissions: pointer.Int32Ptr(0700),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Data: "    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
								},
							},
						},
					},
				},
			},
			ospUpdated: osmv1alpha1.OperatingSystemProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ubuntu",
					Namespace: "default",
					Labels: map[string]string{
						"key": "value",
					},
				},
				Spec: osmv1alpha1.OperatingSystemProfileSpec{
					OSName: "ubuntu",
					Files: []osmv1alpha1.File{
						{
							Path:        "/opt/bin/test.service",
							Permissions: pointer.Int32Ptr(0700),
							Content: osmv1alpha1.FileContent{
								Inline: &osmv1alpha1.FileContentInline{
									Data: "    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
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
			errs := ValidateOperatingSystemProfileUpdate(tc.osp, tc.ospUpdated)
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
