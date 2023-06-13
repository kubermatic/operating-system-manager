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
	"io/fs"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"k8c.io/operating-system-manager/deploy/osps"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/resources/reconciling"
	predicateutil "k8c.io/operating-system-manager/pkg/util/predicate"

	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	ctrlruntime "sigs.k8s.io/controller-runtime"
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

type Reconciler struct {
	client.Client
	log             *zap.SugaredLogger
	defaultOSPFiles map[string][]byte

	namespace string
}

func Add(mgr manager.Manager, log *zap.SugaredLogger, namespace string, workerCount int) error {
	ospDefaultDir, err := osps.FS.ReadDir(ospsDefaultDirName)
	if err != nil {
		return fmt.Errorf("failed to read osps default directory: %w", err)
	}

	var defaultOSPFiles = make(map[string][]byte, len(ospDefaultDir))
	for _, ospFile := range ospDefaultDir {
		defaultOSPFile, err := fs.ReadFile(osps.FS, filepath.Join(ospsDefaultDirName, ospFile.Name()))
		if err != nil {
			return fmt.Errorf("failed to read osp file %s: %w", ospFile.Name(), err)
		}

		defaultOSPFiles[ospFile.Name()] = defaultOSPFile
	}

	reconciler := &Reconciler{
		Client:          mgr.GetClient(),
		log:             log,
		defaultOSPFiles: defaultOSPFiles,
		namespace:       namespace,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: workerCount})
	if err != nil {
		return err
	}

	// Since the osp controller cares about only creating the default OSP resources, we need to watch for the creation
	// of any random resource in the underlying namespace where osm is deployed. We picked deployments for this and added additional
	// event filtering to avoid redundant reconciliation/requeues.
	if err := c.Watch(
		&source.Kind{Type: &appsv1.Deployment{}},
		&handler.EnqueueRequestForObject{},
		filterDeploymentPredicate(),
		predicateutil.ByNamespace(namespace),
	); err != nil {
		return fmt.Errorf("failed to create watch for deployments: %w", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrlruntime.Request) (reconcile.Result, error) {
	r.log.Info("Reconciling default OSP resource..")

	if err := r.reconcile(ctx); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcile(ctx context.Context) error {
	var ospReconcilers []reconciling.NamedOperatingSystemProfileReconcilerFactory
	for name, ospFile := range r.defaultOSPFiles {
		osp, err := parseYAMLToObject(ospFile)
		if err != nil {
			return fmt.Errorf("failed to parse osp %s: %w", name, err)
		}

		// Remove file extension .yaml from the OSP name
		name = strings.ReplaceAll(name, ".yaml", "")

		// Check if OSP already exists
		existingOSP := &v1alpha1.OperatingSystemProfile{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: name, Namespace: r.namespace}, existingOSP); err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to retrieve existing OperatingSystemProfile: %w", err)
		}

		// Since OSPs are immutable, we only want to reconcile resources when the version is different.
		if osp.Spec.Version != existingOSP.Spec.Version {
			// OSP already exists
			osp.SetResourceVersion(existingOSP.GetResourceVersion())
			osp.SetGeneration(existingOSP.GetGeneration())

			ospReconcilers = append(ospReconcilers, ospReconciler(name, osp))
		}
	}

	if err := reconciling.ReconcileOperatingSystemProfiles(ctx,
		ospReconcilers,
		r.namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile osps: %w", err)
	}

	return nil
}

func ospReconciler(name string, osp *v1alpha1.OperatingSystemProfile) reconciling.NamedOperatingSystemProfileReconcilerFactory {
	return func() (string, reconciling.OperatingSystemProfileReconciler) {
		return name, func(*v1alpha1.OperatingSystemProfile) (*v1alpha1.OperatingSystemProfile, error) {
			return osp, nil
		}
	}
}

func parseYAMLToObject(ospByte []byte) (*v1alpha1.OperatingSystemProfile, error) {
	osp := &v1alpha1.OperatingSystemProfile{}
	if err := yamlutil.Unmarshal(ospByte, osp); err != nil {
		return nil, err
	}

	return osp, nil
}

// filterDeploymentPredicate filters out all deployment events except the creation one.
func filterDeploymentPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}
