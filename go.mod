module k8c.io/operating-system-manager

go 1.15

require (
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/go-test/deep v1.0.7
	github.com/kubermatic/machine-controller v1.26.0
	go.uber.org/zap v1.16.0
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	k8c.io/kubermatic/v2 v2.16.2
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.20.2
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/controller-tools v0.4.1

)

replace (
	k8s.io/client-go => k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.5
)
