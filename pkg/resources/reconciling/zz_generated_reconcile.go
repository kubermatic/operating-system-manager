// This file is generated. DO NOT EDIT.
package reconciling

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// SecretCreator defines an interface to create/update Secrets
type SecretCreator = func(existing *corev1.Secret) (*corev1.Secret, error)

// NamedSecretCreatorGetter returns the name of the resource and the corresponding creator function
type NamedSecretCreatorGetter = func() (name string, create SecretCreator)

// SecretObjectWrapper adds a wrapper so the SecretCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func SecretObjectWrapper(create SecretCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*corev1.Secret))
		}
		return create(&corev1.Secret{})
	}
}

// ReconcileSecrets will create and update the Secrets coming from the passed SecretCreator slice
func ReconcileSecrets(ctx context.Context, namedGetters []NamedSecretCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := SecretObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &corev1.Secret{}, false); err != nil {
			return fmt.Errorf("failed to ensure Secret %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}

// OperatingSystemConfigCreator defines an interface to create/update OperatingSystemConfigs
type OperatingSystemConfigCreator = func(existing *osmv1alpha1.OperatingSystemConfig) (*osmv1alpha1.OperatingSystemConfig, error)

// NamedOperatingSystemConfigCreatorGetter returns the name of the resource and the corresponding creator function
type NamedOperatingSystemConfigCreatorGetter = func() (name string, create OperatingSystemConfigCreator)

// OperatingSystemConfigObjectWrapper adds a wrapper so the OperatingSystemConfigCreator matches ObjectCreator.
// This is needed as Go does not support function interface matching.
func OperatingSystemConfigObjectWrapper(create OperatingSystemConfigCreator) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		if existing != nil {
			return create(existing.(*osmv1alpha1.OperatingSystemConfig))
		}
		return create(&osmv1alpha1.OperatingSystemConfig{})
	}
}

// ReconcileOperatingSystemConfigs will create and update the OperatingSystemConfigs coming from the passed OperatingSystemConfigCreator slice
func ReconcileOperatingSystemConfigs(ctx context.Context, namedGetters []NamedOperatingSystemConfigCreatorGetter, namespace string, client ctrlruntimeclient.Client, objectModifiers ...ObjectModifier) error {
	for _, get := range namedGetters {
		name, create := get()
		createObject := OperatingSystemConfigObjectWrapper(create)
		createObject = createWithNamespace(createObject, namespace)
		createObject = createWithName(createObject, name)

		for _, objectModifier := range objectModifiers {
			createObject = objectModifier(createObject)
		}

		if err := EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, createObject, client, &osmv1alpha1.OperatingSystemConfig{}, false); err != nil {
			return fmt.Errorf("failed to ensure OperatingSystemConfig %s/%s: %v", namespace, name, err)
		}
	}

	return nil
}
