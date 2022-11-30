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

package reconciling

import (
	"context"
	"fmt"

	"k8c.io/reconciler/pkg/reconciling"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
)

// OperatingSystemProfileReconciler defines an interface to create/update OperatingSystemProfiles.
type OperatingSystemProfileReconciler = func(existing *osmv1alpha1.OperatingSystemProfile) (*osmv1alpha1.OperatingSystemProfile, error)

// NamedOperatingSystemProfileReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedOperatingSystemProfileReconcilerFactory = func() (name string, reconciler OperatingSystemProfileReconciler)

// OperatingSystemProfileObjectWrapper adds a wrapper so the OperatingSystemProfileReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func OperatingSystemProfileObjectWrapper(reconciler OperatingSystemProfileReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*osmv1alpha1.OperatingSystemProfile))
		}
		return reconciler(&osmv1alpha1.OperatingSystemProfile{})
	}
}

// ReconcileOperatingSystemProfiles will create and update the OperatingSystemProfiles coming from the passed OperatingSystemProfileReconciler slice.
func ReconcileOperatingSystemProfiles(ctx context.Context, namedFactories []NamedOperatingSystemProfileReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := OperatingSystemProfileObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &osmv1alpha1.OperatingSystemProfile{}, false); err != nil {
			return fmt.Errorf("failed to ensure OperatingSystemProfile %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// OperatingSystemConfigReconciler defines an interface to create/update OperatingSystemConfigs.
type OperatingSystemConfigReconciler = func(existing *osmv1alpha1.OperatingSystemConfig) (*osmv1alpha1.OperatingSystemConfig, error)

// NamedOperatingSystemConfigReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedOperatingSystemConfigReconcilerFactory = func() (name string, reconciler OperatingSystemConfigReconciler)

// OperatingSystemConfigObjectWrapper adds a wrapper so the OperatingSystemConfigReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func OperatingSystemConfigObjectWrapper(reconciler OperatingSystemConfigReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*osmv1alpha1.OperatingSystemConfig))
		}
		return reconciler(&osmv1alpha1.OperatingSystemConfig{})
	}
}

// ReconcileOperatingSystemConfigs will create and update the OperatingSystemConfigs coming from the passed OperatingSystemConfigReconciler slice.
func ReconcileOperatingSystemConfigs(ctx context.Context, namedFactories []NamedOperatingSystemConfigReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := OperatingSystemConfigObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &osmv1alpha1.OperatingSystemConfig{}, false); err != nil {
			return fmt.Errorf("failed to ensure OperatingSystemConfig %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}
