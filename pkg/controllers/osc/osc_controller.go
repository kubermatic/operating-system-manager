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

package osc

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"
	"k8c.io/operating-system-manager/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "operating-system-config-controller"
	// MachineDeploymentCleanupFinalizer indicates that sub-resources created by OSC controller against a MachineDeployment should be deleted
	MachineDeploymentCleanupFinalizer = "kubermatic.io/cleanup-operating-system-configs"
	// CloudInitSettingsNamespace is the namespace in which OSCs and secrets are created by OSC controller
	CloudInitSettingsNamespace = "cloud-init-settings"
)

type Reconciler struct {
	client.Client
	ExternalClient client.Client
	Log            *zap.SugaredLogger
	WorkerCount    int

	Namespace                 string
	OSPNamespace              string
	ClusterAddress            string
	ContainerRuntime          string
	ExternalCloudProvider     bool
	PauseImage                string
	InitialTaints             string
	Generator                 generator.CloudConfigGenerator
	ClusterDNSIPs             []net.IP
	ExternalClusterKubeconfig string
	NodeHTTPProxy             string
	NodeNoProxy               string
	PodCIDR                   string
	NodePortRange             string
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrlruntime.Request) (reconcile.Result, error) {
	log := r.Log.With("request", req)
	log.Info("Reconciling OSC resource..")

	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := r.ExternalClient.Get(ctx, req.NamespacedName, machineDeployment); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Resource is marked for deletion
	if machineDeployment.DeletionTimestamp != nil {
		log.Debug("Cleaning up resources against machine deployment")
		if kuberneteshelper.HasFinalizer(machineDeployment, MachineDeploymentCleanupFinalizer) {
			return r.handleMachineDeploymentCleanup(ctx, machineDeployment)
		}
		// Finalizer doesn't exist so clean up is already done
		return reconcile.Result{}, nil
	}

	// Add finalizer if it doesn't exist
	if !kuberneteshelper.HasFinalizer(machineDeployment, MachineDeploymentCleanupFinalizer) {
		kuberneteshelper.AddFinalizer(machineDeployment, MachineDeploymentCleanupFinalizer)
		if err := r.ExternalClient.Update(ctx, machineDeployment); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	err := r.reconcile(ctx, machineDeployment)
	if err != nil {
		r.Log.Errorw("Reconciling failed", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	if err := r.reconcileOperatingSystemConfigs(ctx, md); err != nil {
		return fmt.Errorf("failed to reconcile operating system config: %v", err)
	}

	if err := r.reconcileSecrets(ctx, md); err != nil {
		return fmt.Errorf("failed to reconcile secrets: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileOperatingSystemConfigs(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	ospName := md.Annotations[resources.MachineDeploymentOSPAnnotation]
	osp := &osmv1alpha1.OperatingSystemProfile{}
	if err := r.Get(ctx, types.NamespacedName{Name: ospName, Namespace: r.OSPNamespace}, osp); err != nil {
		return fmt.Errorf("failed to get OperatingSystemProfile: %v", err)
	}

	if err := reconciling.ReconcileOperatingSystemConfigs(ctx, []reconciling.NamedOperatingSystemConfigCreatorGetter{
		resources.OperatingSystemConfigCreator(
			md,
			osp,
			r.ExternalClusterKubeconfig,
			r.ClusterDNSIPs,
			r.ContainerRuntime,
			r.ExternalCloudProvider,
			r.PauseImage,
			r.InitialTaints,
			r.NodeHTTPProxy,
			r.NodeNoProxy,
			r.NodePortRange,
			r.PodCIDR,
		),
	}, r.Namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile provisioning operating system config: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileSecrets(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	oscName := fmt.Sprintf(resources.MachineDeploymentSubresourceNamePattern, md.Name, resources.ProvisioningCloudConfig)
	osc := &osmv1alpha1.OperatingSystemConfig{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: r.Namespace, Name: oscName}, osc); err != nil {
		return fmt.Errorf("failed to list OperatingSystemConfigs: %v", err)
	}

	provisionData, err := r.Generator.Generate(osc)
	if err != nil {
		return fmt.Errorf("failed to generate provisioning data")
	}

	if err := reconciling.ReconcileSecrets(ctx, []reconciling.NamedSecretCreatorGetter{
		resources.CloudConfigSecretCreator(md.Name, resources.ProvisioningCloudConfig, provisionData),
	}, CloudInitSettingsNamespace, r.ExternalClient); err != nil {
		return fmt.Errorf("failed to reconcile provisioning secrets: %v", err)
	}
	r.Log.Infof("successfully generated provisioning secret: %v", fmt.Sprintf(resources.MachineDeploymentSubresourceNamePattern, md.Name, resources.ProvisioningCloudConfig))

	return nil
}

// handleMachineDeploymentCleanup handles the cleanup of resources created against a MachineDeployment
func (r *Reconciler) handleMachineDeploymentCleanup(ctx context.Context, md *clusterv1alpha1.MachineDeployment) (reconcile.Result, error) {
	// Delete OperatingSystemConfig
	if err := r.deleteOperatingSystemConfig(ctx, md); err != nil {
		return reconcile.Result{}, err
	}

	// Delete generated secrets
	if err := r.deleteGeneratedSecrets(ctx, md); err != nil {
		return reconcile.Result{}, err
	}

	// Remove finalizer
	kuberneteshelper.RemoveFinalizer(md, MachineDeploymentCleanupFinalizer)

	// Update instance
	err := r.Client.Update(ctx, md)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
	}

	return reconcile.Result{}, nil
}

// deleteOperatingSystemConfig deletes the OperatingSystemConfig created against a MachineDeployment
func (r *Reconciler) deleteOperatingSystemConfig(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	oscName := fmt.Sprintf(resources.MachineDeploymentSubresourceNamePattern, md.Name, resources.ProvisioningCloudConfig)
	osc := &osmv1alpha1.OperatingSystemConfig{}
	if err := r.Get(ctx, types.NamespacedName{Name: oscName, Namespace: r.Namespace}, osc); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to retrieve OperatingSystemConfig %s against MachineDeployment %s: %v", oscName, md.Name, err)
	}
	if err := r.Delete(ctx, osc); err != nil {
		return fmt.Errorf("failed to delete OperatingSystemConfig %s: %v against MachineDeployment %s", oscName, md.Name, err)
	}
	return nil
}

// deleteGeneratedSecrets deletes the secrets created against a MachineDeployment
func (r *Reconciler) deleteGeneratedSecrets(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	secretName := fmt.Sprintf(resources.MachineDeploymentSubresourceNamePattern, md.Name, resources.ProvisioningCloudConfig)
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: r.Namespace}, secret); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to retrieve secret %s against MachineDeployment %s: %v", secret, md.Name, err)
	}

	if err := r.Delete(ctx, secret); err != nil {
		return fmt.Errorf("failed to delete secret %s against MachineDeployment %s: %v", secret, md.Name, err)
	}
	return nil
}

func (r *Reconciler) SetupWithManager(mgr manager.Manager) error {
	return ctrlruntime.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.MachineDeployment{}).
		WithEventFilter(filterMachineDeploymentPredicate()).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.WorkerCount}).
		Complete(r)
}

// filterMachineDeploymentPredicate will filter machine deployments based on the presence of OSP annotation
func filterMachineDeploymentPredicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetAnnotations()[resources.MachineDeploymentOSPAnnotation] != ""
	})
}
