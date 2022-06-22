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
	"strings"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CloudInitNamespace = "cloud-init-settings"
	jwtTokenNamePrefix = "cloud-init-getter-token"
)

func ExtractAPIServerToken(ctx context.Context, client ctrlruntimeclient.Client) (string, error) {
	secretList := corev1.SecretList{}
	if err := client.List(ctx, &secretList, &ctrlruntimeclient.ListOptions{Namespace: CloudInitNamespace}); err != nil {
		return "", fmt.Errorf("failed to list secrets in namespace %s: %w", CloudInitNamespace, err)
	}

	for _, secret := range secretList.Items {
		if strings.HasPrefix(secret.Name, jwtTokenNamePrefix) {
			if secret.Data != nil {
				jwtToken := secret.Data["token"]
				if jwtToken != nil {
					token := string(jwtToken)
					return token, nil
				}
			}
		}
	}

	return "", errors.New("failed to fetch api server token")
}
