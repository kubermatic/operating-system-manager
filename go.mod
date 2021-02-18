module k8c.io/operating-system-manager

go 1.15

require (
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/code-generator v0.20.2
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
	sigs.k8s.io/controller-tools v0.4.1
)

replace k8s.io/client-go => k8s.io/client-go v0.20.2
