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

package config

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"k8c.io/operating-system-manager/pkg/providerconfig/config/types"

	corev1 "k8s.io/api/core/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigVarResolver struct {
	ctx       context.Context
	client    ctrlruntimeclient.Client
	namespace string
}

var (
	// CABundle is set globally once by main() function
	// This is shared globally since the enclosing values won't change during the controller lifecycle
	ConfigVarResolverInstance ConfigVarResolver
)

func (cvr *ConfigVarResolver) GetConfigVarStringValue(configVar types.ConfigVarString) (string, error) {
	// We need all three of these to fetch and use a secret
	if configVar.SecretKeyRef.Name != "" && configVar.SecretKeyRef.Namespace != "" && configVar.SecretKeyRef.Key != "" {
		secret := &corev1.Secret{}
		name := ktypes.NamespacedName{Namespace: configVar.SecretKeyRef.Namespace, Name: configVar.SecretKeyRef.Name}
		if err := cvr.client.Get(cvr.ctx, name, secret); err != nil {
			return "", fmt.Errorf("error retrieving secret '%s' from namespace '%s': '%v'", configVar.SecretKeyRef.Name, configVar.SecretKeyRef.Namespace, err)
		}
		if val, ok := secret.Data[configVar.SecretKeyRef.Key]; ok {
			return string(val), nil
		}
		return "", fmt.Errorf("secret '%s' in namespace '%s' has no key '%s'", configVar.SecretKeyRef.Name, configVar.SecretKeyRef.Namespace, configVar.SecretKeyRef.Key)
	}

	// We need all three of these to fetch and use a configmap
	if configVar.ConfigMapKeyRef.Name != "" && configVar.ConfigMapKeyRef.Namespace != "" && configVar.ConfigMapKeyRef.Key != "" {
		configMap := &corev1.ConfigMap{}
		name := ktypes.NamespacedName{Namespace: configVar.ConfigMapKeyRef.Namespace, Name: configVar.ConfigMapKeyRef.Name}
		if err := cvr.client.Get(cvr.ctx, name, configMap); err != nil {
			return "", fmt.Errorf("error retrieving configmap '%s' from namespace '%s': '%v'", configVar.ConfigMapKeyRef.Name, configVar.ConfigMapKeyRef.Namespace, err)
		}
		if val, ok := configMap.Data[configVar.ConfigMapKeyRef.Key]; ok {
			return val, nil
		}
		return "", fmt.Errorf("configmap '%s' in namespace '%s' has no key '%s'", configVar.ConfigMapKeyRef.Name, configVar.ConfigMapKeyRef.Namespace, configVar.ConfigMapKeyRef.Key)
	}

	return configVar.Value, nil
}

// GetConfigVarStringValueOrEnv tries to get the value from ConfigVarString, when it fails, it falls back to
// getting the value from an environment variable specified by envVarName parameter
func (cvr *ConfigVarResolver) GetConfigVarStringValueOrEnv(configVar types.ConfigVarString, envVarName string) (string, error) {
	cfgVar, err := cvr.GetConfigVarStringValue(configVar)
	if err == nil && len(cfgVar) > 0 {
		return cfgVar, err
	}

	envVal, _ := os.LookupEnv(envVarName)
	return envVal, nil
}

func (cvr *ConfigVarResolver) GetConfigVarBoolValueOrEnv(configVar types.ConfigVarBool, envVarName string) (bool, error) {
	cvs := types.ConfigVarString{Value: strconv.FormatBool(configVar.Value), SecretKeyRef: configVar.SecretKeyRef}
	stringVal, err := cvr.GetConfigVarStringValue(cvs)
	if err != nil {
		return false, err
	}
	if stringVal == "" {
		envVal, envValFound := os.LookupEnv(envVarName)
		if !envValFound {
			return false, fmt.Errorf("all mechanisms(value, secret, configMap) of getting the value failed, including reading from environment variable = %s which was not set", envVarName)
		}
		stringVal = envVal
	}
	boolVal, err := strconv.ParseBool(stringVal)
	if err != nil {
		return false, err
	}
	return boolVal, nil
}

// SetConfigVarResolver will instantiate the global ConfigVarResolver Instance
func SetConfigVarResolver(ctx context.Context, client ctrlruntimeclient.Client, namespace string) {
	ConfigVarResolverInstance = ConfigVarResolver{
		ctx:       ctx,
		client:    client,
		namespace: namespace,
	}
}

func GetConfigVarResolver() *ConfigVarResolver {
	return &ConfigVarResolverInstance
}
