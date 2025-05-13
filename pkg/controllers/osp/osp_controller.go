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

package osp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"k8c.io/operating-system-manager/deploy/osps"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/resources/reconciling"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "OperatingSystemProfileController"

	ospsDefaultDirName = "default"
)

type ospMap map[string]*v1alpha1.OperatingSystemProfile

type Reconciler struct {
	client.Client
	log         *zap.SugaredLogger
	defaultOSPs ospMap

	namespace string
}

func Add(mgr manager.Manager, log *zap.SugaredLogger, namespace string, workerCount int, disableDefaultOSPs bool) error {
	// if the default osps creation is disabled then there is no need to load the default osps and only custom osps
	// should be used.
	if disableDefaultOSPs {
		return nil
	}

	defaultOSPs, err := loadDefaultOSPs()
	if err != nil {
		return fmt.Errorf("failed to load default OSPs: %w", err)
	}

	reconciler := &Reconciler{
		Client:      mgr.GetClient(),
		log:         log,
		defaultOSPs: defaultOSPs,
		namespace:   namespace,
	}

	// trigger controller once upon startup to bootstrap the default OSPs
	bootstrapping := make(chan event.GenericEvent, len(defaultOSPs))
	for name := range defaultOSPs {
		bootstrapping <- event.GenericEvent{
			Object: &v1alpha1.OperatingSystemProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			},
		}
	}

	_, err = builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workerCount,
		}).
		For(&v1alpha1.OperatingSystemProfile{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return object.GetNamespace() == namespace
		}))).
		WatchesRawSource(source.Channel(bootstrapping, &handler.EnqueueRequestForObject{})).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrlruntime.Request) (reconcile.Result, error) {
	if err := r.reconcileOSP(ctx, req.Name); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcileOSP(ctx context.Context, name string) error {
	osp, ok := r.defaultOSPs[name]
	if !ok {
		return nil
	}

	r.log.Debugw("Reconciling OSP resource...", "osp", name)

	ospReconcilers := []reconciling.NamedOperatingSystemProfileReconcilerFactory{
		ospReconciler(name, osp),
	}

	if err := reconciling.ReconcileOperatingSystemProfiles(ctx, ospReconcilers, r.namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile OSP: %w", err)
	}

	return nil
}

func ospReconciler(name string, source *v1alpha1.OperatingSystemProfile) reconciling.NamedOperatingSystemProfileReconcilerFactory {
	return func() (string, reconciling.OperatingSystemProfileReconciler) {
		return name, func(osp *v1alpha1.OperatingSystemProfile) (*v1alpha1.OperatingSystemProfile, error) {
			// only attempt an update if our OSP is newer
			if osp.Spec.Version != source.Spec.Version {
				osp.Spec = source.Spec
			}

			return osp, nil
		}
	}
}

func loadDefaultOSPs() (ospMap, error) {
	ospDefaultDir, err := osps.FS.ReadDir(ospsDefaultDirName)
	if err != nil {
		return nil, fmt.Errorf("failed to read OSPs default directory: %w", err)
	}

	var defaultOSPs = make(ospMap, len(ospDefaultDir))
	for _, ospFile := range ospDefaultDir {
		filename := ospFile.Name()

		osp, err := parseOSPFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read OSP %s: %w", filename, err)
		}

		// Remove file extension .yaml to get OSP name
		ospName := strings.ReplaceAll(filename, ".yaml", "")

		defaultOSPs[ospName] = osp
	}

	return defaultOSPs, nil
}

func parseOSPFile(filename string) (*v1alpha1.OperatingSystemProfile, error) {
	content, err := osps.FS.ReadFile(filepath.Join(ospsDefaultDirName, filename))
	if err != nil {
		return nil, err
	}

	osp := &v1alpha1.OperatingSystemProfile{}
	if err := yamlutil.Unmarshal(content, osp); err != nil {
		return nil, err
	}

	return osp, nil
}
