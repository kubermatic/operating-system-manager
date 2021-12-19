module k8c.io/operating-system-manager

go 1.16

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/go-test/deep v1.0.7
	github.com/kinvolk/container-linux-config-transpiler v0.9.1
	github.com/kubermatic/machine-controller v1.36.1
	github.com/onsi/ginkgo v1.15.0
	github.com/open-policy-agent/frameworks/constraint v0.0.0-20210802220920-c000ec35322e // indirect
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	go.uber.org/zap v1.16.0
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8c.io/kubermatic/v2 v2.16.2
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.20.4
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/yaml v1.2.0

)

replace k8s.io/client-go => k8s.io/client-go v0.20.2
