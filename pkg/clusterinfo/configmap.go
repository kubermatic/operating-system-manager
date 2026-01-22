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

package clusterinfo

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	configMapName     = "cluster-info"
	kubernetesService = "kubernetes"
	securePortName    = "https"
)

func New(client ctrlruntimeclient.Client, caCert string) *KubeconfigProvider {
	return &KubeconfigProvider{
		client: client,
		caCert: caCert,
	}
}

type KubeconfigProvider struct {
	caCert string
	client ctrlruntimeclient.Client
}

func (p *KubeconfigProvider) GetKubeconfig(ctx context.Context) (*clientcmdapi.Config, error) {
	cm, err := p.getKubeconfigFromConfigMap(ctx)
	if err != nil {
		klog.V(6).Infof("could not get cluster-info kubeconfig from configmap: %v", err)
		klog.V(6).Info("falling back to retrieval via endpointslice")
		return p.buildKubeconfigFromEndpointSlice(ctx)
	}
	return cm, nil
}

func (p *KubeconfigProvider) getKubeconfigFromConfigMap(ctx context.Context) (*clientcmdapi.Config, error) {
	cm := &corev1.ConfigMap{}
	if err := p.client.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: metav1.NamespacePublic}, cm); err != nil {
		return nil, err
	}

	data, found := cm.Data["kubeconfig"]
	if !found {
		return nil, errors.New("no kubeconfig found in cluster-info configmap")
	}
	return clientcmd.Load([]byte(data))
}

func (p *KubeconfigProvider) buildKubeconfigFromEndpointSlice(ctx context.Context) (*clientcmdapi.Config, error) {
	slices := &discoveryv1.EndpointSliceList{}
	if err := p.client.List(ctx, slices,
		ctrlruntimeclient.InNamespace(metav1.NamespaceDefault),
		ctrlruntimeclient.MatchingLabels{discoveryv1.LabelServiceName: kubernetesService}); err != nil {
		return nil, fmt.Errorf("failed to list endpointslices: %w", err)
	}

	if len(slices.Items) == 0 {
		return nil, errors.New("no endpointslices found for kubernetes service")
	}

	for _, slice := range slices.Items {
		port := getSecurePortFromSlice(slice.Ports)
		if port == nil {
			continue
		}

		for _, endpoint := range slice.Endpoints {
			if endpoint.Conditions.Ready == nil || !*endpoint.Conditions.Ready {
				continue
			}

			if len(endpoint.Addresses) == 0 {
				continue
			}

			ip := net.ParseIP(endpoint.Addresses[0])
			if ip == nil {
				continue
			}

			url := fmt.Sprintf("https://%s", net.JoinHostPort(ip.String(), strconv.Itoa(int(*port.Port))))

			return &clientcmdapi.Config{
				Kind:       "Config",
				APIVersion: "v1",
				Clusters: map[string]*clientcmdapi.Cluster{
					"": {
						Server:                   url,
						CertificateAuthorityData: []byte(p.caCert),
					},
				},
			}, nil
		}
	}

	return nil, errors.New("no ready endpoint found in kubernetes endpointslices")
}

func getSecurePortFromSlice(ports []discoveryv1.EndpointPort) *discoveryv1.EndpointPort {
	for _, p := range ports {
		if p.Name != nil && *p.Name == securePortName && p.Port != nil {
			return &p
		}
	}
	return nil
}
