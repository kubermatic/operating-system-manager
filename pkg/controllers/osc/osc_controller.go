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
	"github.com/kubermatic/machine-controller/pkg/containerruntime"
	"k8c.io/operating-system-manager/pkg/bootstrap"
	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"
	kuberneteshelper "k8c.io/operating-system-manager/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	ControllerName = "operating-system-config-controller"
	// MachineDeploymentCleanupFinalizer indicates that sub-resources created by OSC controller against a MachineDeployment should be deleted
	MachineDeploymentCleanupFinalizer = "kubermatic.io/cleanup-operating-system-configs"
	// CloudInitSettingsNamespace is the namespace in which OSCs and secrets are created by OSC controller
	CloudInitSettingsNamespace = "cloud-init-settings"
)

type Reconciler struct {
	client.Client
	workerClient client.Client

	log *zap.SugaredLogger

	bootstrap bootstrap.Bootstrap

	namespace                     string
	containerRuntime              string
	externalCloudProvider         bool
	pauseImage                    string
	initialTaints                 string
	generator                     generator.CloudConfigGenerator
	clusterDNSIPs                 []net.IP
	caCert                        string
	nodeHTTPProxy                 string
	nodeNoProxy                   string
	nodeRegistryCredentialsSecret string
	containerRuntimeConfig        containerruntime.Config
	kubeletFeatureGates           map[string]bool
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerClient client.Client,
	client client.Client,
	bootstrap bootstrap.Bootstrap,
	caCert string,
	namespace string,
	workerCount int,
	clusterDNSIPs []net.IP,
	generator generator.CloudConfigGenerator,
	containerRuntime string,
	externalCloudProvider bool,
	pauseImage string,
	initialTaints string,
	nodeHTTPProxy string,
	nodeNoProxy string,
	containerRuntimeConfig containerruntime.Config,
	nodeRegistryCredentialsSecret string,
	kubeletFeatureGates map[string]bool) error {
	reconciler := &Reconciler{
		log:                           log,
		workerClient:                  workerClient,
		Client:                        client,
		bootstrap:                     bootstrap,
		caCert:                        caCert,
		namespace:                     namespace,
		generator:                     generator,
		clusterDNSIPs:                 clusterDNSIPs,
		containerRuntime:              containerRuntime,
		pauseImage:                    pauseImage,
		initialTaints:                 initialTaints,
		externalCloudProvider:         externalCloudProvider,
		nodeHTTPProxy:                 nodeHTTPProxy,
		nodeNoProxy:                   nodeNoProxy,
		containerRuntimeConfig:        containerRuntimeConfig,
		nodeRegistryCredentialsSecret: nodeRegistryCredentialsSecret,
		kubeletFeatureGates:           kubeletFeatureGates,
	}
	log.Info("Reconciling OSC resource..")
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: workerCount})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &clusterv1alpha1.MachineDeployment{}}, &handler.EnqueueRequestForObject{}, filterMachineDeploymentPredicate()); err != nil {
		return fmt.Errorf("failed to watch MachineDeployments: %w", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrlruntime.Request) (reconcile.Result, error) {
	log := r.log.With("request", req)
	log.Info("Reconciling OSC resource..")

	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := r.workerClient.Get(ctx, req.NamespacedName, machineDeployment); err != nil {
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
		if err := r.workerClient.Update(ctx, machineDeployment); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	err := r.reconcile(ctx, machineDeployment)
	if err != nil {
		r.log.Errorw("Reconciling failed", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	if err := r.reconcileOperatingSystemConfigs(ctx, md); err != nil {
		return fmt.Errorf("failed to reconcile operating system config: %w", err)
	}

	if err := r.reconcileSecrets(ctx, md); err != nil {
		return fmt.Errorf("failed to reconcile secrets: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileOperatingSystemConfigs(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	// Check if OSC already exists, in that case we don't need to do anything since OSC are immutable
	oscName := fmt.Sprintf(resources.OperatingSystemConfigNamePattern, md.Name, md.Namespace)
	osc := &osmv1alpha1.OperatingSystemConfig{}
	if err := r.Get(ctx, types.NamespacedName{Name: oscName, Namespace: r.namespace}, osc); err == nil {
		// Early return since the object already exists
		return nil
	}

	ospName := md.Annotations[resources.MachineDeploymentOSPAnnotation]
	osp := &osmv1alpha1.OperatingSystemProfile{}

	if err := r.Get(ctx, types.NamespacedName{Name: ospName, Namespace: r.namespace}, osp); err != nil {
		return fmt.Errorf("failed to get OperatingSystemProfile: %w", err)
	}

	if r.nodeRegistryCredentialsSecret != "" {
		registryCredentials, err := containerruntime.GetContainerdAuthConfig(ctx, r.Client, r.nodeRegistryCredentialsSecret)
		if err != nil {
			return fmt.Errorf("failed to get containerd auth config: %w", err)
		}
		r.containerRuntimeConfig.RegistryCredentials = registryCredentials
	}

	bootstrapKubeconfig, err := r.bootstrap.CreateBootstrapKubeconfig(ctx, fmt.Sprintf("%s-%s", md.Namespace, md.Name))
	if err != nil {
		return fmt.Errorf("failed to create bootstrap kubeconfig: %w", err)
	}

	token, err := bootstrap.ExtractAPIServerToken(ctx, r.workerClient)
	if err != nil {
		return fmt.Errorf("failed to fetch api-server token: %w", err)
	}

	// We need to create OSC resource as it doesn't exist
	osc, err = resources.GenerateOperatingSystemConfig(
		md,
		osp,
		bootstrapKubeconfig,
		token,
		oscName,
		r.namespace,
		r.caCert,
		r.clusterDNSIPs,
		r.containerRuntime,
		r.externalCloudProvider,
		r.pauseImage,
		r.initialTaints,
		r.nodeHTTPProxy,
		r.nodeNoProxy,
		r.containerRuntimeConfig,
		r.kubeletFeatureGates,
	)
	if err != nil {
		return fmt.Errorf("failed to generate %s osc: %w", oscName, err)
	}

	// Create resource in cluster
	if err := r.Create(ctx, osc); err != nil {
		return fmt.Errorf("failed to create %s osc: %w", oscName, err)
	}
	r.log.Infof("successfully generated provisioning osc: %v", oscName)
	return nil
}

func (r *Reconciler) reconcileSecrets(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	oscName := fmt.Sprintf(resources.OperatingSystemConfigNamePattern, md.Name, md.Namespace)
	osc := &osmv1alpha1.OperatingSystemConfig{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: r.namespace, Name: oscName}, osc); err != nil {
		return fmt.Errorf("failed to list OperatingSystemConfigs: %w", err)
	}

	if err := r.ensureCloudConfigSecret(ctx, osc.Spec.BootstrapConfig, resources.BootstrapCloudConfig, osc.Spec.OSName, osc.Spec.CloudProvider.Name, md.Name, md.Namespace); err != nil {
		return fmt.Errorf("failed to reconcile bootstrapping config secret: %w", err)
	}

	if err := r.ensureCloudConfigSecret(ctx, osc.Spec.ProvisioningConfig, resources.ProvisioningCloudConfig, osc.Spec.OSName, osc.Spec.CloudProvider.Name, md.Name, md.Namespace); err != nil {
		return fmt.Errorf("failed to reconcile provisioning config secret: %w", err)
	}

	return nil
}

func (r *Reconciler) ensureCloudConfigSecret(ctx context.Context, config osmv1alpha1.OSCConfig, secretType resources.CloudConfigSecret, operatingSystem osmv1alpha1.OperatingSystem, cloudProvider osmv1alpha1.CloudProvider, mdName, mdNamespace string) error {
	secretName := fmt.Sprintf(resources.CloudConfigSecretNamePattern, mdName, mdNamespace, secretType)

	// Check if secret already exists, in that case we don't need to do anything since secrets are immutable
	secret := &corev1.Secret{}
	if err := r.workerClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: CloudInitSettingsNamespace}, secret); err == nil {
		// Early return since the object already exists
		return nil
	}

	provisionData, err := r.generator.Generate(&config, operatingSystem, cloudProvider)
	if err != nil {
		return fmt.Errorf("failed to generate %s data", secretType)
	}

	// Generate secret for cloud-config
	secret = resources.GenerateCloudConfigSecret(secretName, CloudInitSettingsNamespace, provisionData)

	// Create resource in cluster
	if err := r.workerClient.Create(ctx, secret); err != nil {
		return fmt.Errorf("failed to create %s %s secret: %w", secretName, secretType, err)
	}
	r.log.Infof("successfully generated %s secret: %v", secretType, secretName)
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
	err := r.workerClient.Update(ctx, md)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
	}

	return reconcile.Result{}, nil
}

// deleteOperatingSystemConfig deletes the OperatingSystemConfig created against a MachineDeployment
func (r *Reconciler) deleteOperatingSystemConfig(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	oscName := fmt.Sprintf(resources.OperatingSystemConfigNamePattern, md.Name, md.Namespace)
	osc := &osmv1alpha1.OperatingSystemConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oscName,
			Namespace: r.namespace,
		},
	}

	if err := r.Delete(ctx, osc); err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete OperatingSystemConfig %s: %v against MachineDeployment %w", oscName, md.Name, err)
	}
	return nil
}

// deleteGeneratedSecrets deletes the secrets created against a MachineDeployment
func (r *Reconciler) deleteGeneratedSecrets(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	// Delete provisioning secret
	provisioningSecretName := fmt.Sprintf(resources.CloudConfigSecretNamePattern, md.Name, md.Namespace, resources.ProvisioningCloudConfig)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      provisioningSecretName,
			Namespace: CloudInitSettingsNamespace,
		},
	}

	if err := r.Delete(ctx, secret); err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete provisioning secret %s against MachineDeployment %s: %w", secret, md.Name, err)
	}

	// Delete bootstrap secret
	bootstrapSecretName := fmt.Sprintf(resources.CloudConfigSecretNamePattern, md.Name, md.Namespace, resources.BootstrapCloudConfig)
	secret.Name = bootstrapSecretName

	if err := r.Delete(ctx, secret); err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete bootstrap secret %s against MachineDeployment %s: %w", secret, md.Name, err)
	}
	return nil
}

// filterMachineDeploymentPredicate will filter machine deployments based on the presence of OSP annotation
func filterMachineDeploymentPredicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetAnnotations()[resources.MachineDeploymentOSPAnnotation] != ""
	})
}
