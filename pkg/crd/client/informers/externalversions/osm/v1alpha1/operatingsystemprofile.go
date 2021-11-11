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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	versioned "k8c.io/operating-system-manager/pkg/crd/client/clientset/versioned"
	internalinterfaces "k8c.io/operating-system-manager/pkg/crd/client/informers/externalversions/internalinterfaces"
	v1alpha1 "k8c.io/operating-system-manager/pkg/crd/client/listers/osm/v1alpha1"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// OperatingSystemProfileInformer provides access to a shared informer and lister for
// OperatingSystemProfiles.
type OperatingSystemProfileInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.OperatingSystemProfileLister
}

type operatingSystemProfileInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewOperatingSystemProfileInformer constructs a new informer for OperatingSystemProfile type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewOperatingSystemProfileInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredOperatingSystemProfileInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredOperatingSystemProfileInformer constructs a new informer for OperatingSystemProfile type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredOperatingSystemProfileInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OperatingsystemmanagerV1alpha1().OperatingSystemProfiles(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OperatingsystemmanagerV1alpha1().OperatingSystemProfiles(namespace).Watch(context.TODO(), options)
			},
		},
		&osmv1alpha1.OperatingSystemProfile{},
		resyncPeriod,
		indexers,
	)
}

func (f *operatingSystemProfileInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredOperatingSystemProfileInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *operatingSystemProfileInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&osmv1alpha1.OperatingSystemProfile{}, f.defaultInformer)
}

func (f *operatingSystemProfileInformer) Lister() v1alpha1.OperatingSystemProfileLister {
	return v1alpha1.NewOperatingSystemProfileLister(f.Informer().GetIndexer())
}
