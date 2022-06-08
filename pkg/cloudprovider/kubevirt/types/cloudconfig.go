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

package types

import "gopkg.in/yaml.v3"

// CloudConfig contains only the section global.
type CloudConfig struct {
	Global GlobalOpts
}

// GlobalOpts contains the values of the global section of the cloud configuration.
type GlobalOpts struct {
	// Kubeconfig used to connect to the cluster that runs KubeVirt
	Kubeconfig string `yaml:"kubeconfig"`

	// Namespace used in KubeVirt cloud-controller-manager as infra cluster namespace.
	Namespace string `yaml:"namespace"`
}

// ToString renders the cloud configuration as string.
func (cc *CloudConfig) ToString() (string, error) {
	out, err := yaml.Marshal(cc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
