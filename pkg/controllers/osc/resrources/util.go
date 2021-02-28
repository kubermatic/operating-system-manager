package resrources

type CloudInitSecret string

const (
	BootstrapCloudInit    CloudInitSecret = "bootstrap"
	ProvisioningCloudInit                 = "provisioning"
)

const (
	MachineDeploymentOSPAnnotation = "k8c.io/operating-system-profile"
)
