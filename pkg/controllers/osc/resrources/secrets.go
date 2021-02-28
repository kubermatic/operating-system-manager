package resrources

import (
	"fmt"
	"k8c.io/operating-system-manager/pkg/resources/reconciling"
	corev1 "k8s.io/api/core/v1"
)

// CloudInitSecretCreator returns a function to create a secret that contains the cloud-init configurations.
func CloudInitSecretCreator(mdName string, oscType CloudInitSecret, data []byte) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		secretName := fmt.Sprintf("%s-osc-%s", mdName, oscType)
		return secretName, func(sec *corev1.Secret) (*corev1.Secret, error) {
			if sec.Data == nil {
				sec.Data = map[string][]byte{}
			}

			sec.Data["cloud-init"] = data

			return sec, nil
		}
	}
}
