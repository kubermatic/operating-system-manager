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

package resources

import (
	"fmt"

	"k8c.io/operating-system-manager/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

// CloudConfigSecretCreator returns a function to create a secret that contains the cloud-init or ignition configurations.
func CloudConfigSecretCreator(mdName string, oscType CloudConfigSecret, data []byte) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		secretName := fmt.Sprintf(MachineDeploymentSubresourceNamePattern, mdName, oscType)
		return secretName, func(sec *corev1.Secret) (*corev1.Secret, error) {
			if sec.Data == nil {
				sec.Data = map[string][]byte{}
			}
			sec.Data["cloud-config"] = data

			// Cloud config secret is immutable
			sec.Immutable = pointer.Bool(true)
			return sec, nil
		}
	}
}
