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
	"encoding/json"
	"fmt"
	"net"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
	log                   *zap.SugaredLogger
	namespace             string
	clusterAddress        string
	containerRuntime      string
	externalCloudProvider bool
	pauseImage            string
	initialTaints         string
	generator             generator.CloudConfigGenerator
	clusterDNSIPs         []net.IP
	kubeconfig            string
	nodeHTTPProxy         string
	nodeNoProxy           string
	podCIDR               string
	nodePortRange         string
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	namespace string,
	clusterName string,
	workerCount int,
	clusterDNSIPs []net.IP,
	kubeconfig string,
	generator generator.CloudConfigGenerator,
	containerRuntime string,
	externalCloudProvider bool,
	pauseImage string,
	initialTaints string,
	nodeHTTPProxy string,
	nodeNoProxy string,
	podCIDR string,
	nodePortRange string) error {
	reconciler := &Reconciler{
		Client:                mgr.GetClient(),
		log:                   log,
		namespace:             namespace,
		clusterAddress:        clusterName,
		generator:             generator,
		kubeconfig:            kubeconfig,
		clusterDNSIPs:         clusterDNSIPs,
		containerRuntime:      containerRuntime,
		pauseImage:            pauseImage,
		initialTaints:         initialTaints,
		externalCloudProvider: externalCloudProvider,
		nodeHTTPProxy:         nodeHTTPProxy,
		nodeNoProxy:           nodeNoProxy,
		podCIDR:               podCIDR,
		nodePortRange:         nodePortRange,
	}
	log.Info("Reconciling OSC resource..")
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: workerCount})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &clusterv1alpha1.MachineDeployment{}}, &handler.EnqueueRequestForObject{}, filterMachineDeploymentPredicate()); err != nil {
		return fmt.Errorf("failed to watch MachineDeployments: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrlruntime.Request) (reconcile.Result, error) {
	log := r.log.With("request", req)
	log.Info("Reconciling OSC resource..")

	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := r.Get(ctx, req.NamespacedName, machineDeployment); err != nil {
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

	supportedMD, err := r.isSupportedMachineDeployment(ctx, machineDeployment)
	if err != nil || !supportedMD {
		return reconcile.Result{}, err
	}

	// Add finalizer if it doesn't exist
	if !kuberneteshelper.HasFinalizer(machineDeployment, MachineDeploymentCleanupFinalizer) {
		kuberneteshelper.AddFinalizer(machineDeployment, MachineDeploymentCleanupFinalizer)
		if err := r.Client.Update(ctx, machineDeployment); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	err = r.reconcile(ctx, machineDeployment)
	if err != nil {
		r.log.Errorw("Reconciling failed", zap.Error(err))
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
	if err := r.Get(ctx, types.NamespacedName{Name: ospName, Namespace: r.namespace}, osp); err != nil {
		return fmt.Errorf("failed to get OperatingSystemProfile: %v", err)
	}

	if err := reconciling.ReconcileOperatingSystemConfigs(ctx, []reconciling.NamedOperatingSystemConfigCreatorGetter{
		resources.OperatingSystemConfigCreator(
			md,
			osp,
			r.kubeconfig,
			r.clusterDNSIPs,
			r.containerRuntime,
			r.externalCloudProvider,
			r.pauseImage,
			r.initialTaints,
			r.nodeHTTPProxy,
			r.nodeNoProxy,
			r.nodePortRange,
			r.podCIDR,
		),
	}, r.namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile provisioning operating system config: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileSecrets(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	oscName := fmt.Sprintf(resources.MachineDeploymentSubresourceNamePattern, md.Name, resources.ProvisioningCloudConfig)
	osc := &osmv1alpha1.OperatingSystemConfig{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: CloudInitSettingsNamespace, Name: oscName}, osc); err != nil {
		return fmt.Errorf("failed to list OperatingSystemConfigs: %v", err)
	}

	provisionData, err := r.generator.Generate(osc)
	if err != nil {
		return fmt.Errorf("failed to generate provisioning data")
	}

	if err := reconciling.ReconcileSecrets(ctx, []reconciling.NamedSecretCreatorGetter{
		resources.CloudConfigSecretCreator(md.Name, resources.ProvisioningCloudConfig, provisionData),
	}, r.namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile provisioning secrets: %v", err)
	}
	r.log.Infof("successfully generated provisioning secret: %v", fmt.Sprintf(resources.MachineDeploymentSubresourceNamePattern, md.Name, resources.ProvisioningCloudConfig))

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
	if err := r.Get(ctx, types.NamespacedName{Name: oscName, Namespace: r.namespace}, osc); err != nil {
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
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: r.namespace}, secret); err != nil {
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

// isSupportedMachineDeployment ensures that MachineDeployment is complaint with the OperatingSystemProfile
func (r *Reconciler) isSupportedMachineDeployment(ctx context.Context, md *clusterv1alpha1.MachineDeployment) (bool, error) {
	ospName := md.Annotations[resources.MachineDeploymentOSPAnnotation]
	osp := &osmv1alpha1.OperatingSystemProfile{}
	if err := r.Get(ctx, types.NamespacedName{Name: ospName, Namespace: r.namespace}, osp); err != nil {
		return false, fmt.Errorf("Failed to get OperatingSystemProfile: %v", err)
	}

	// Get providerConfig from machineDeployment
	providerConfig := providerconfigtypes.Config{}
	err := json.Unmarshal(md.Spec.Template.Spec.ProviderSpec.Value.Raw, &providerConfig)
	if err != nil {
		return false, fmt.Errorf("Failed to decode provider configs: %v", err)
	}

	// Ensure that OSP supports the operating system
	if osp.Spec.OSName != osmv1alpha1.OperatingSystem(providerConfig.OperatingSystem) {
		// Just log an error instead of returning it to avoid re-queues
		r.log.Errorw("OperatingSystemProfile does not support the OperatingSystem specified in MachineDeployment")
		return false, nil
	}

	// Ensure that OSP supports the cloud provider
	supportedCloudProvider := false
	for _, cloudProvider := range osp.Spec.SupportedCloudProviders {
		if providerconfigtypes.CloudProvider(cloudProvider.Name) == providerConfig.CloudProvider {
			supportedCloudProvider = true
			break
		}
	}

	// Ensure that OSP supports the operating system
	if !supportedCloudProvider {
		// Just log an error instead of returning it to avoid re-queues
		r.log.Errorw("OperatingSystemProfile does not support the CloudProvider specified in MachineDeployment")
		return false, nil
	}
	return true, nil
}

// filterMachineDeploymentPredicate will filter machine deployments based on the presence of OSP annotation
func filterMachineDeploymentPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			md := e.Object.(*clusterv1alpha1.MachineDeployment)
			return md.Annotations[resources.MachineDeploymentOSPAnnotation] != ""
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			md := e.Object.(*clusterv1alpha1.MachineDeployment)
			return md.Annotations[resources.MachineDeploymentOSPAnnotation] != ""
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			md := e.ObjectNew.(*clusterv1alpha1.MachineDeployment)
			return md.Annotations[resources.MachineDeploymentOSPAnnotation] != ""
		},
		GenericFunc: func(e event.GenericEvent) bool {
			md := e.Object.(*clusterv1alpha1.MachineDeployment)
			return md.Annotations[resources.MachineDeploymentOSPAnnotation] != ""
		},
	}
}
