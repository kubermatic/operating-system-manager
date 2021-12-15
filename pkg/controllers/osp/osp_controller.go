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

	"go.uber.org/zap"

	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "OperatingSystemProfileController"
)

type Reconciler struct {
	client.Client
	Log         *zap.SugaredLogger
	WorkerCount int
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrlruntime.Request) (reconcile.Result, error) {
	log := r.Log.With("request", req)
	log.Info("Reconciling OSP resource..")

	profile := &v1alpha1.OperatingSystemProfile{}
	if err := r.Get(ctx, req.NamespacedName, profile); err != nil && kerrors.IsNotFound(err) {
		return reconcile.Result{}, fmt.Errorf("failed to get OSP resource: %v", err)
	}

	if profile.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr manager.Manager) error {
	return ctrlruntime.NewControllerManagedBy(mgr).
		For(&v1alpha1.OperatingSystemProfile{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.WorkerCount}).
		Complete(r)
}
