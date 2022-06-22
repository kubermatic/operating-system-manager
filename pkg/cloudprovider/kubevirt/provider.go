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

package kubevirt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/operating-system-manager/pkg/cloudprovider/kubevirt/types"
	"k8c.io/operating-system-manager/pkg/providerconfig/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	envKubevirtKubeconfig = "KUBEVIRT_KUBECONFIG"
)

func GetCloudConfig(pconfig providerconfigtypes.Config) (string, error) {
	c, err := getConfig(pconfig)
	if err != nil {
		return "", fmt.Errorf("failed to parse config: %w", err)
	}

	s, err := c.ToString()
	if err != nil {
		return "", fmt.Errorf("failed to convert cloud-config to string: %w", err)
	}

	return s, nil
}
func getConfig(pconfig providerconfigtypes.Config) (*types.CloudConfig, error) {
	if pconfig.CloudProviderSpec.Raw == nil {
		return nil, errors.New("CloudProviderSpec in the MachineDeployment cannot be empty")
	}

	rawConfig := types.RawConfig{}
	if err := json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CloudProviderSpec: %w", err)
	}

	var (
		opts types.GlobalOpts
		err  error
	)

	// Kubeconfig was specified directly in the Machine/MachineDeployment CR. In this case we need to ensure that the value is base64 encoded.
	if rawConfig.Auth.Kubeconfig.Value != "" {
		val, err := base64.StdEncoding.DecodeString(rawConfig.Auth.Kubeconfig.Value)
		if err != nil {
			// An error here means that this is not a valid base64 string
			// We can be more explicit here with the error for visibility. Webhook will return this error if we hit this scenario.
			return nil, fmt.Errorf("failed to decode base64 encoded kubeconfig. Expected value is a base64 encoded Kubeconfig in JSON or YAML format: %w", err)
		}
		opts.Kubeconfig = string(val)
	} else {
		// Environment variable or secret reference was used for providing the value of kubeconfig
		// We have to be lenient in this case and allow unencoded values as well.
		opts.Kubeconfig, err = config.GetConfigVarResolver().GetConfigVarStringValueOrEnv(rawConfig.Auth.Kubeconfig, envKubevirtKubeconfig)
		if err != nil {
			return nil, fmt.Errorf(`failed to get value of "kubeconfig" field: %w`, err)
		}
		val, err := base64.StdEncoding.DecodeString(opts.Kubeconfig)
		// We intentionally ignore errors here with an assumption that an unencoded YAML or JSON must have been passed on
		// in this case.
		if err == nil {
			opts.Kubeconfig = string(val)
		}
	}

	opts.Namespace = getNamespace()

	cloudConfig := &types.CloudConfig{
		Global: opts,
	}

	return cloudConfig, nil
}

// getNamespace returns the namespace where the VM is created.
// VM is created in a dedicated namespace <cluster-id>
// which is the namespace where the machine-controller pod is running.
// Defaults to `kube-system`.
func getNamespace() string {
	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		// Useful especially for ci tests.
		ns = metav1.NamespaceSystem
	}
	return ns
}
