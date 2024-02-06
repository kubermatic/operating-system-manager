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
	mcbootstrap "github.com/kubermatic/machine-controller/pkg/bootstrap"
	"github.com/kubermatic/machine-controller/pkg/containerruntime"
	machinecontrollerutil "github.com/kubermatic/machine-controller/pkg/controller/util"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/operating-system-manager/pkg/bootstrap"
	"k8c.io/operating-system-manager/pkg/controllers/osc/resources"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	"k8c.io/operating-system-manager/pkg/generator"
	kuberneteshelper "k8c.io/operating-system-manager/pkg/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
)

type Reconciler struct {
	ctrlruntimeclient.Client
	workerClient ctrlruntimeclient.Client
	recorder     record.EventRecorder

	log *zap.SugaredLogger

	bootstrappingManager bootstrap.Bootstrap

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
	workerClient ctrlruntimeclient.Client,
	client ctrlruntimeclient.Client,
	bootstrappingManager bootstrap.Bootstrap,
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
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		bootstrappingManager:          bootstrappingManager,
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
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: workerCount})
	if err != nil {
		return err
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &clusterv1alpha1.MachineDeployment{}), &handler.EnqueueRequestForObject{}, filterMachineDeploymentPredicate()); err != nil {
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
	if err := r.checkOSP(ctx, md); err != nil {
		return fmt.Errorf("failed to validate referenced OSP: %w", err)
	}

	// Check if OSC and secret need to be rotated
	if err := r.handleOSCAndSecretsRotation(ctx, md); err != nil {
		return fmt.Errorf("failed to perform rotation for OSC and secrets: %w", err)
	}

	if err := r.reconcileOperatingSystemConfigs(ctx, md); err != nil {
		return fmt.Errorf("failed to reconcile operating system config: %w", err)
	}

	if err := r.reconcileSecrets(ctx, md); err != nil {
		return fmt.Errorf("failed to reconcile secrets: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileOperatingSystemConfigs(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	machineDeploymentKey := fmt.Sprintf("%s-%s", md.Namespace, md.Name)
	// machineDeploymentKey must be no more than 63 characters else it'll fail to create bootstrap token.
	if len(machineDeploymentKey) >= 63 {
		// As a fallback, we just use the name of the machine deployment.
		machineDeploymentKey = md.Name
	}

	bootstrapKubeconfig, bootstrapKubeconfigName, err := r.bootstrappingManager.CreateBootstrapKubeconfig(ctx, machineDeploymentKey)
	if err != nil {
		return fmt.Errorf("failed to create bootstrap kubeconfig: %w", err)
	}

	// Check if OSC already exists, in that case we don't need to do anything since OSC are immutable
	oscName := fmt.Sprintf(resources.OperatingSystemConfigNamePattern, md.Name, md.Namespace)
	osc := &osmv1alpha1.OperatingSystemConfig{}
	if err := r.Get(ctx, types.NamespacedName{Name: oscName, Namespace: r.namespace}, osc); err == nil {
		// Early return since the object already exists
		return nil
	}

	ospName := md.Annotations[resources.MachineDeploymentOSPAnnotation]

	// Check if user has specified custom namespace for OSPs
	ospNamespace := md.Annotations[resources.MachineDeploymentOSPNamespaceAnnotation]
	if len(ospNamespace) == 0 {
		ospNamespace = r.namespace
	}

	osp := &osmv1alpha1.OperatingSystemProfile{}
	if err := r.Get(ctx, types.NamespacedName{Name: ospName, Namespace: ospNamespace}, osp); err != nil {
		return fmt.Errorf("failed to get OperatingSystemProfile %q from namespace %q: %w", ospName, ospNamespace, err)
	}

	provisioner, err := generator.GetProvisioningUtility(osp.Spec.OSName, *md)
	if err != nil {
		return fmt.Errorf("failed to determine provisioning utility: %w", err)
	}

	if osp.Spec.ProvisioningUtility != "" && provisioner != osp.Spec.ProvisioningUtility {
		return fmt.Errorf("specified provisioning utility %q is not supported by the OperatingSystemProfile", osp.Spec.ProvisioningUtility)
	}

	if r.nodeRegistryCredentialsSecret != "" {
		registryCredentials, err := containerruntime.GetContainerdAuthConfig(ctx, r.Client, r.nodeRegistryCredentialsSecret)
		if err != nil {
			return fmt.Errorf("failed to get containerd auth config: %w", err)
		}
		r.containerRuntimeConfig.RegistryCredentials = registryCredentials
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
		bootstrapKubeconfigName,
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

	if osc.Spec.CloudProvider.Name == "edge" {
		if err := r.generateEdgeScript(ctx, md, token, bootstrapKubeconfig); err != nil {
			return fmt.Errorf("failed to generate edge provider bootstrap script: %w", err)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to generate %s osc: %w", oscName, err)
	}

	// Add machine deployment revision to OSC
	revision := md.Annotations[machinecontrollerutil.RevisionAnnotation]
	osc.Annotations = addMachineDeploymentRevision(revision, osc.Annotations)
	osc.Spec.ProvisioningUtility = osp.Spec.ProvisioningUtility

	// Defaults to cloud-init although we should never hit this condition i.e ProvisioningUtility in OSP to be empty.
	if osc.Spec.ProvisioningUtility == "" {
		osc.Spec.ProvisioningUtility = osmv1alpha1.ProvisioningUtilityCloudInit
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
		return fmt.Errorf("failed to get OperatingSystemConfigs %q from namespace %q: %w", oscName, r.namespace, err)
	}

	if err := r.ensureCloudConfigSecret(ctx, osc.Spec.ProvisioningConfig, osc.Spec.ProvisioningUtility, resources.ProvisioningCloudConfig, osc.Spec.OSName, osc.Spec.CloudProvider.Name, md); err != nil {
		return fmt.Errorf("failed to reconcile provisioning config secret: %w", err)
	}

	if err := r.ensureCloudConfigSecret(ctx, osc.Spec.BootstrapConfig, osc.Spec.ProvisioningUtility, mcbootstrap.BootstrapCloudConfig, osc.Spec.OSName, osc.Spec.CloudProvider.Name, md); err != nil {
		return fmt.Errorf("failed to reconcile bootstrapping config secret: %w", err)
	}

	return nil
}

func (r *Reconciler) ensureCloudConfigSecret(ctx context.Context, config osmv1alpha1.OSCConfig, provisioningUtility osmv1alpha1.ProvisioningUtility, secretType mcbootstrap.CloudConfigSecret, operatingSystem osmv1alpha1.OperatingSystem, cloudProvider osmv1alpha1.CloudProvider, md *clusterv1alpha1.MachineDeployment) error {
	secretName := fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, secretType)

	// Check if secret already exists, in that case we don't need to do anything since secrets are immutable
	secret := &corev1.Secret{}
	if err := r.workerClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: mcbootstrap.CloudInitSettingsNamespace}, secret); err == nil {
		// Early return since the object already exists
		return nil
	}

	provisionData, err := r.generator.Generate(&config, provisioningUtility, operatingSystem, cloudProvider, *md, secretType)
	if err != nil {
		return fmt.Errorf("failed to generate %s data with error: %w", secretType, err)
	}

	// Generate secret for cloud-config
	secret = resources.GenerateCloudConfigSecret(secretName, mcbootstrap.CloudInitSettingsNamespace, provisionData)

	// Add machine deployment revision to secret
	revision := md.Annotations[machinecontrollerutil.RevisionAnnotation]
	secret.Annotations = addMachineDeploymentRevision(revision, secret.Annotations)

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
	provisioningSecretName := fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, resources.ProvisioningCloudConfig)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      provisioningSecretName,
			Namespace: mcbootstrap.CloudInitSettingsNamespace,
		},
	}

	if err := r.workerClient.Delete(ctx, secret); err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete provisioning secret %s against MachineDeployment %s: %w", provisioningSecretName, md.Name, err)
	}

	// Delete bootstrap secret
	bootstrapSecretName := fmt.Sprintf(mcbootstrap.CloudConfigSecretNamePattern, md.Name, md.Namespace, mcbootstrap.BootstrapCloudConfig)
	secret.Name = bootstrapSecretName

	if err := r.workerClient.Delete(ctx, secret); err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete bootstrap secret %s against MachineDeployment %s: %w", bootstrapSecretName, md.Name, err)
	}

	// Delete kubelet bootstrapping kubeconfig secret
	machineDeploymentKey := fmt.Sprintf("%s-%s", md.Namespace, md.Name)
	// machineDeploymentKey must be no more than 63 characters else it'll fail to create bootstrap token.
	if len(machineDeploymentKey) >= 63 {
		// As a fallback, we just use the name of the machine deployment.
		machineDeploymentKey = md.Name
	}

	bootstrapConfigName := fmt.Sprintf("%s-kubelet-bootstrap-config", machineDeploymentKey)
	secret.Name = bootstrapConfigName

	if err := r.workerClient.Delete(ctx, secret); err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete kubelet bootstrap config secret %s against MachineDeployment %s: %w", bootstrapConfigName, md.Name, err)
	}

	return nil
}

func (r *Reconciler) handleOSCAndSecretsRotation(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	oscName := fmt.Sprintf(resources.OperatingSystemConfigNamePattern, md.Name, md.Namespace)
	osc := &osmv1alpha1.OperatingSystemConfig{}
	if err := r.Get(ctx, types.NamespacedName{Name: oscName, Namespace: r.namespace}, osc); err != nil {
		if kerrors.IsNotFound(err) {
			// OSC doesn't exist and we need to create it.
			return nil
		}
		return err
	}

	// OSC already exists, we need to check if the template in machine deployment was updated. If it's updated then we need to rotate
	// the OSC and secrets.
	currentRevision := md.Annotations[machinecontrollerutil.RevisionAnnotation]
	existingRevision := osc.Annotations[mcbootstrap.MachineDeploymentRevision]

	if currentRevision == existingRevision {
		// Rotation is not required.
		return nil
	}

	// Delete the existing OSC and let the controller re-create them.
	if err := r.Delete(ctx, osc); err != nil {
		return fmt.Errorf("failed to delete OperatingSystemConfig %s: %v against MachineDeployment %w", oscName, md.Name, err)
	}

	// Delete the existing secrets and let the controller re-create them.
	return r.deleteGeneratedSecrets(ctx, md)
}

func (r *Reconciler) checkOSP(ctx context.Context, md *clusterv1alpha1.MachineDeployment) error {
	err := validateMachineDeployment(ctx, md, r.Client, r.namespace)

	if err != nil {
		r.recorder.Event(md, corev1.EventTypeWarning, "OperatingSystemProfileError", err.Error())
	}

	return err
}

func (r *Reconciler) generateEdgeScript(ctx context.Context, md *clusterv1alpha1.MachineDeployment, token string, config *api.Config) error {
	var clusterName string
	for key := range config.Clusters {
		clusterName = key
		break
	}

	serverURL := config.Clusters[clusterName].Server
	bootstrapSecretName := fmt.Sprintf("%s-%s-bootstrap-config", md.Name, md.Namespace)
	script := fmt.Sprintf("curl -s -k -v --header 'Authorization: Bearer %s' %s/api/v1/namespaces/cloud-init-settings/secrets/%s | jq '.data[\"cloud-config\"]' -r| base64 -d > /etc/cloud/cloud.cfg.d/%s.cfg \ncloud-init --file /etc/cloud/cloud.cfg.d/%s.cfg init\n",
		token, serverURL, bootstrapSecretName, bootstrapSecretName, bootstrapSecretName)

	scriptSecretName := fmt.Sprintf("edge-provider-script-%s-%s", md.Name, md.Namespace)
	secret := &corev1.Secret{}
	if err := r.workerClient.Get(ctx, types.NamespacedName{Name: scriptSecretName, Namespace: bootstrap.CloudInitNamespace}, secret); err != nil {
		if !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to get %s secret in namespace %s: %w", scriptSecretName, bootstrap.CloudInitNamespace, err)
		}
	}

	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	secret.Name = scriptSecretName
	secret.Namespace = bootstrap.CloudInitNamespace
	secret.Data["fetch-bootstrap-script"] = []byte(script)

	return r.workerClient.Create(ctx, secret)
}

// filterMachineDeploymentPredicate will filter machine deployments based on the presence of OSP annotation
func filterMachineDeploymentPredicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(o ctrlruntimeclient.Object) bool {
		return o.GetAnnotations()[resources.MachineDeploymentOSPAnnotation] != ""
	})
}

func addMachineDeploymentRevision(revision string, annotations map[string]string) map[string]string {
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[mcbootstrap.MachineDeploymentRevision] = revision
	return annotations
}

func validateMachineDeployment(ctx context.Context, md *clusterv1alpha1.MachineDeployment, client ctrlruntimeclient.Client, namespace string) error {
	ospName := md.Annotations[resources.MachineDeploymentOSPAnnotation]
	// Ignoring request since no OperatingSystemProfile found
	if len(ospName) == 0 {
		// Returning here as MD without this annotation shouldn't be validated
		return nil
	}

	ospNamespace := md.Annotations[resources.MachineDeploymentOSPNamespaceAnnotation]
	if len(ospNamespace) == 0 {
		ospNamespace = namespace
	}

	osp := &osmv1alpha1.OperatingSystemProfile{}
	err := client.Get(ctx, types.NamespacedName{Name: ospName, Namespace: ospNamespace}, osp)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return fmt.Errorf("OperatingSystemProfile %q not found", ospName)
		}

		return fmt.Errorf("failed to fetch OperatingSystemProfile %q: %w", ospName, err)
	}

	// Get providerConfig from machineDeployment
	providerConfig := providerconfigtypes.Config{}
	err = json.Unmarshal(md.Spec.Template.Spec.ProviderSpec.Value.Raw, &providerConfig)
	if err != nil {
		return fmt.Errorf("failed to decode provider config: %w", err)
	}

	// Ensure that OSP supports the operating system
	if osp.Spec.OSName != osmv1alpha1.OperatingSystem(providerConfig.OperatingSystem) {
		return fmt.Errorf("OperatingSystemProfile %q does not support operating system %q", osp.Name, providerConfig.OperatingSystem)
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
		return fmt.Errorf("OperatingSystemProfile %q does not support cloud provider %q", osp.Name, providerConfig.OperatingSystem)
	}

	return nil
}
