/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

	"go.uber.org/zap"

	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "OperatingSystemProfileController"
)

type Reconciler struct {
	client.Client

	log *zap.SugaredLogger
}

func Add(mgr manager.Manager, log *zap.SugaredLogger, namespace string, workerCount int) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),
		log:    log,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: workerCount})
	if err != nil {
		return err
	}
	return c.Watch(&source.Kind{Type: &v1alpha1.OperatingSystemProfile{}}, &handler.EnqueueRequestForObject{},
		predicate.NewPredicateFuncs(func(o metav1.Object, _ runtime.Object) bool { return o.GetNamespace() == namespace }))
}

func (r *Reconciler) Reconcile(req ctrlruntime.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := r.log.With("request", req)
	log.Debug("Reconciling OSP resource..")

	profile := &v1alpha1.OperatingSystemProfile{}
	if err := r.Get(ctx, req.NamespacedName, profile); err != nil && kerrors.IsNotFound(err) {
		return reconcile.Result{}, fmt.Errorf("failed to get OSP resource: %v", err)
	}

	if profile.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}
