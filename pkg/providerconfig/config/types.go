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

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GlobalObjectKeySelector is needed as we can not use v1.SecretKeySelector
// because it is not cross namespace
type GlobalObjectKeySelector struct {
	corev1.ObjectReference `json:",inline"`
	Key                    string `json:"key,omitempty"`
}

type GlobalSecretKeySelector GlobalObjectKeySelector
type GlobalConfigMapKeySelector GlobalObjectKeySelector

type ConfigVarString struct {
	Value           string                     `json:"value,omitempty"`
	SecretKeyRef    GlobalSecretKeySelector    `json:"secretKeyRef,omitempty"`
	ConfigMapKeyRef GlobalConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
}

type ConfigVarBool struct {
	Value           bool                       `json:"value,omitempty"`
	SecretKeyRef    GlobalSecretKeySelector    `json:"secretKeyRef,omitempty"`
	ConfigMapKeyRef GlobalConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
}

type ConfigVarResolver struct {
	ctx       context.Context
	client    ctrlruntimeclient.Client
	namespace string
}
