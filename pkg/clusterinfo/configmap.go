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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	configMapName           = "cluster-info"
	kubernetesEndpointsName = "kubernetes"
	securePortName          = "https"
)

func New(client controllerruntimeclient.Client, caCert string) *KubeconfigProvider {
	return &KubeconfigProvider{
		client: client,
		caCert: caCert,
	}
}

type KubeconfigProvider struct {
	caCert string
	client controllerruntimeclient.Client
}

func (p *KubeconfigProvider) GetKubeconfig(ctx context.Context) (*clientcmdapi.Config, error) {
	cm, err := p.getKubeconfigFromConfigMap(ctx)
	if err != nil {
		klog.V(6).Infof("could not get cluster-info kubeconfig from configmap: %v", err)
		klog.V(6).Info("falling back to retrieval via endpoint")
		return p.buildKubeconfigFromEndpoint(ctx)
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

func (p *KubeconfigProvider) buildKubeconfigFromEndpoint(ctx context.Context) (*clientcmdapi.Config, error) {
	endpoint := &corev1.Endpoints{}
	if err := p.client.Get(ctx, types.NamespacedName{Name: kubernetesEndpointsName, Namespace: metav1.NamespaceDefault}, endpoint); err != nil {
		return nil, err
	}

	if len(endpoint.Subsets) == 0 {
		return nil, errors.New("no subsets in the kubernetes endpoints resource")
	}
	subset := endpoint.Subsets[0]

	if len(subset.Addresses) == 0 {
		return nil, errors.New("no addresses in the first subset of the kubernetes endpoints resource")
	}
	address := subset.Addresses[0]

	ip := net.ParseIP(address.IP)
	if ip == nil {
		return nil, errors.New("could not parse ip from ")
	}

	getSecurePort := func(endpointSubset corev1.EndpointSubset) *corev1.EndpointPort {
		for _, p := range subset.Ports {
			if p.Name == securePortName {
				return &p
			}
		}
		return nil
	}

	port := getSecurePort(subset)
	if port == nil {
		return nil, errors.New("no secure port in the subset")
	}
	url := fmt.Sprintf("https://%s", net.JoinHostPort(ip.String(), strconv.Itoa(int(port.Port))))

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
