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

package containerruntime

import (
	"fmt"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
)

func SetupContainerRuntime(containerRuntime string, osp *osmv1alpha1.OperatingSystemProfile) {
	var crToSet *osmv1alpha1.ContainerRuntimeSpec
	for _, cr := range osp.Spec.SupportedContainerRuntimes {
		if cr.Name == containerRuntime {
			if crToSet == nil {
				crToSet = &cr
			}
		}
	}

	for _, file := range osp.Spec.Files {
		if file.Path == crToSet.ConfigFileName {
			file.Content.Inline.Data = fmt.Sprintf(file.Content.Inline.Data, crToSet.ScriptFile)
		}
	}
}
