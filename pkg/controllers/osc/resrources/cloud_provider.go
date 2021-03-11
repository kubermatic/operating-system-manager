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

package resrources

import (
	"encoding/json"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
)

func GetCloudProviderFromMachineDeployment(md *clusterv1alpha1.MachineDeployment) (*v1alpha1.CloudProviderSpec, error) {
	cloudProvider := &struct {
		CloudProvider     string                `json:"cloudProvider"`
		CloudProviderSpec *runtime.RawExtension `json:"cloudProviderSpec"`
	}{}

	if err := json.Unmarshal(md.Spec.Template.Spec.ProviderSpec.Value.Raw, cloudProvider); err != nil {
		return nil, err
	}

	return &v1alpha1.CloudProviderSpec{
		Name: cloudProvider.CloudProvider,
		Spec: *cloudProvider.CloudProviderSpec,
	}, nil
}
