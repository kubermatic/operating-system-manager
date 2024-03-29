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

package bootstrap

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CloudInitNamespace    = "cloud-init-settings"
	cloudInitGetterSecret = "cloud-init-getter-token"
)

func ExtractAPIServerToken(ctx context.Context, client ctrlruntimeclient.Client) (string, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: cloudInitGetterSecret, Namespace: CloudInitNamespace}, secret); err != nil {
		return "", fmt.Errorf("failed to get %s secrets in namespace %s: %w", cloudInitGetterSecret, CloudInitNamespace, err)
	}

	token := secret.Data["token"]
	if token != nil {
		return string(token), nil
	}

	return "", errors.New("failed to fetch api server token")
}
