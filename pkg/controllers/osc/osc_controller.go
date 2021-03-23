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

package osc

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	"k8c.io/operating-system-manager/pkg/controllers/osc/resrources"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"
	"k8c.io/operating-system-manager/pkg/resources/reconciling"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "operating-system-config-controller"
)

type Reconciler struct {
	client.Client
	log            *zap.SugaredLogger
	namespace      string
	clusterAddress string
	generator      generator.CloudInitGenerator

	clusterDNSIPs []net.IP
	kubeconfig    string
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	namespace string,
	clusterName string,
	workerCount int,
	clusterDNSIPs []net.IP,
	kubeconfig string,
	generator generator.CloudInitGenerator) error {
	reconciler := &Reconciler{
		Client:         mgr.GetClient(),
		log:            log,
		namespace:      namespace,
		clusterAddress: clusterName,
		generator:      generator,
		kubeconfig:     kubeconfig,
		clusterDNSIPs:  clusterDNSIPs,
	}
	log.Info("Reconciling OSC resource..")
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: workerCount})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &clusterv1alpha1.MachineDeployment{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch MachineDeployments: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(req ctrlruntime.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := r.log.With("request", req)
	log.Info("Reconciling OSC resource..")

	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := r.Get(ctx, req.NamespacedName, machineDeployment); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	if machineDeployment.DeletionTimestamp != nil {
		// TODO(mq): delete oscs and secrets when md is deleted.
		return reconcile.Result{}, nil
	}

	err := r.reconcile(ctx, machineDeployment)
	if err != nil {
		r.log.Errorw("Reconciling failed", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	if md.Annotations[resrources.MachineDeploymentOSPAnnotation] == "" {
		r.log.Warnw("Ignoring OSM request: no OperatingSystemProfile found. This could influence the provisioning of the machine")
		return nil
	}

	if err := r.reconcileOperatingSystemConfigs(ctx, md); err != nil {
		return fmt.Errorf("failed to reconcile operating system config: %v", err)
	}

	if err := r.reconcileSecrets(ctx, md); err != nil {
		return fmt.Errorf("failed to reconcile secrets: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileOperatingSystemConfigs(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	ospName := md.Annotations[resrources.MachineDeploymentOSPAnnotation]
	osp := &v1alpha1.OperatingSystemProfile{}
	if err := r.Get(ctx, types.NamespacedName{Name: ospName, Namespace: r.namespace}, osp); err != nil {
		return fmt.Errorf("failed to get OperatingSystemProfile: %v", err)
	}

	// bootstrapOsp, err := resources.BootstrapOSP("", md)
	// if err != nil {
	// 	return err
	// }

	// if err := reconciling.ReconcileOperatingSystemConfigs(ctx, []reconciling.NamedOperatingSystemConfigCreatorGetter{
	// 	// TODO(mq): add api server address
	// 	resrources.OperatingSystemConfigCreator(false, md, bootstrapOsp),
	// }, r.namespace, r.Client); err != nil {
	// 	return fmt.Errorf("failed to reconcile cloud-init bootstrap operating system config: %v", err)
	// }

	if err := reconciling.ReconcileOperatingSystemConfigs(ctx, []reconciling.NamedOperatingSystemConfigCreatorGetter{
		resrources.OperatingSystemConfigCreator(
			true,
			md,
			osp,
			r.kubeconfig,
			r.clusterDNSIPs,
		),
	}, r.namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile cloud-init provision operating system config: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileSecrets(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	oscList := &v1alpha1.OperatingSystemConfigList{}
	if err := r.List(ctx, oscList, &client.ListOptions{Namespace: r.namespace}); err != nil {
		return fmt.Errorf("failed to list OperatingSystemConfigs: %v", err)
	}

	oscs := oscList.Items
	for i := range oscs {
		switch oscs[i].Name {
		case fmt.Sprintf("%s-osc-%s", md.Name, resrources.BootstrapCloudInit):
			// bootstrapData, err := r.generator.Generate(&oscs[i], md)
			// if err != nil {
			// 	return fmt.Errorf("failed to generate bootstrap cloud-init data")
			// }

			// if err := reconciling.ReconcileSecrets(ctx, []reconciling.NamedSecretCreatorGetter{
			// 	resrources.CloudInitSecretCreator(md.Name, resrources.BootstrapCloudInit, bootstrapData),
			// }, r.namespace, r.Client); err != nil {
			// 	return fmt.Errorf("failed to reconcile cloud-init bootstrap secrets: %v", err)
			// }
		case fmt.Sprintf("%s-osc-%s", md.Name, resrources.ProvisioningCloudInit):
			provisionData, err := r.generator.Generate(&oscs[i], md)
			if err != nil {
				return fmt.Errorf("failed to generate provisioning cloud-init data")
			}

			if err := reconciling.ReconcileSecrets(ctx, []reconciling.NamedSecretCreatorGetter{
				resrources.CloudInitSecretCreator(md.Name, resrources.ProvisioningCloudInit, provisionData),
			}, r.namespace, r.Client); err != nil {
				return fmt.Errorf("failed to reconcile cloud-init provisioning secrets: %v", err)
			}
		default:
			r.log.Debugw("skipping osc %s secret reconciliation for machine deployment %s", oscs[i].Name, md.Name)
		}
	}
	return nil
}
