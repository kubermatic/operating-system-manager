# Kubermatic operating-system-manager

Operating System Manager is responsible for creating and managing the required configurations for worker nodes in a Kubernetes cluster. It decouples operating system configurations into dedicated and isolable resources for better modularity and maintainability.

These isolated and extensible resources allow a high degree of customization. This is useful for hybrid, edge, and air-gapped environments.

Configurations for worker nodes comprise of set of scripts used to prepare the node, install packages, configure networking, storage etc. These configurations prepare the nodes for running `kubelet`.

## Overview

### Problem Statement

[Machine-Controller](https://github.com/kubermatic/machine-controller) is used to manage the worker nodes in KubeOne clusters. It depends on user-data plugins to generate the required configurations for worker nodes. Each operating system requires its own user-data plugin. These configs are then injected into the worker nodes using provisioning utilities such as [cloud-init](https://cloud-init.io) or [ignition](https://coreos.github.io/ignition). Eventually the nodes are bootstrapped.

This has been the norm in KubeOne till KubeOne v1.4 and it works as expected. Although over time, it has been observed that this workflow has certain limitations.

#### Machine Controller Limitations

- Machine Controller expects **ALL** the supported user-data plugins to exist and be ready. User might only be interested in a subset of the available operating systems. For example, user might only want to work with `ubuntu`.
- The user-data plugins have templates defined [in-code](https://github.com/kubermatic/machine-controller/blob/v1.53.0/pkg/userdata/ubuntu/provider.go#L136). Which is not ideal since code changes are required to update those templates. Then those code changes need to become a part of the subsequent releases for machine-controller and KubeOne. So we need a complete release cycle to ship those changes to customers.
- Managing configs for multiple cloud providers, OS flavors and OS versions, adds a lot of complexity and redundancy in machine-controller.
- Since the templates are defined in-code, there is no way for an end user to customize them to suit their use-cases.
- Each cloud provider sets some sort of limits for the size of `user-data`, machine won't be created in case of non-compliance. For example, at the time of writing this, AWS has set a [hard limit of 16KB](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-add-user-data.html).
- Better support for air-gapped environments is required.

Operating System Manager was created to overcome these limitations.

### Solution

Operating System Manager was created to solve the above mentioned issues. It decouples operating system configurations into dedicated and isolable resources for better modularity and maintainability.

## Architecture

OSM introduces the following new resources which are Kubernetes Custom Resource Definitions:

### OperatingSystemProfile

A resource that contains scripts for bootstrapping and provisioning the worker nodes, along with information about what operating systems and versions are supported for given scripts. Additionally, OSPs support templating so you can include some information from MachineDeployment or the OSM deployment itself.

Default OSPs for supported operating systems are provided/installed automatically by KubeOne. End users can create custom OSPs as well to fit their own use-cases. OSPs are immutable by design and any modifications to an existing OSP requires a version bump in `.spec.version`.

Its dedicated controller runs in the seed cluster, in user cluster namespace, and operates on the OperatingSystemProfile custom resource. It is responsible for installing the default OSPs in user-cluster namespace.

### OperatingSystemConfig

Immutable resource that contains the **actual configurations** that are going to be used to bootstrap and provision the worker nodes. It is a subset of OperatingSystemProfile. OperatingSystemProfile is a template while OperatingSystemConfig is an instance rendered with data from OperatingSystemProfile, MachineDeployment, and flags provided at OSM command-line level.

OperatingSystemConfigs have a 1-to-1 relation with the MachineDeployment. A dedicated controller watches the MachineDeployments and generates the OSCs in `kube-system` and secrets in `cloud-init-settings` namespaces in the cluster. Machine Controller then waits for the bootstrapping- and provisioning-secrets to become available. Once they are ready, it will extract the configurations from those secrets and pass them as `user-data` to the to-be-provisioned machines.

Its dedicated controller runs in the seed cluster, in user cluster namespace, and is responsible for generating the OSCs in seed and secrets in `cloud-init-settings` namespace in the user cluster.

For each MachineDeployment we have two types of configurations, which are stored in secrets:

1. **Bootstrap**: Configuration used for initially setting up the machine and fetching the provisioning configuration.
2. **Provisioning**: Configuration with the actual `cloud-config` that is used to provision the worker machine.

![Architecture](./docs/images/architecture-osm.png)

### Air-gapped Environment

This controller was designed by keeping air-gapped environments in mind. Customers can use their own VM images by creating custom OSP profiles to provision nodes in a cluster that doesn't have outbound internet access.

More work is being done to make it even easier to use OSM in air-gapped environments.

## Support

Information about supported OS versions can be found [here](./docs/compatibility-matrix.md).

## Deploy OSM

- Install [cert-manager](https://cert-manager.io/) for generating certificates used by webhooks since they serve using HTTPS

```terminal
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.7.1/cert-manager.yaml
```

- Run `kubectl create namespace cloud-init-settings` to create namespace where secrets against OSC are stored
- Run `kubectl apply -f deploy/crd/` to install CRDs
- Run `kubectl apply -f deploy/` to deploy OSM

## Development

### Local Development

To run OSM locally:

- Either use a [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) cluster or actual cluster and make sure that the correct context is loaded
- Run `kubectl apply -f deploy/crds` to install CRDs
- Create relevant OperatingSystemProfile resources. Check [sample](./deploy/osps/default) for reference.
- Run `make run`

### Testing

Simply run `make test`

## Troubleshooting

If you encounter issues [file an issue][1] or talk to us on the [#kubermatic channel][6] on the [Kubermatic Slack][7].

## Contributing

Thanks for taking the time to join our community and start contributing!

Feedback and discussion are available on [the mailing list][5].

### Before you start

- Please familiarize yourself with the [Code of Conduct][4] before contributing.
- See [CONTRIBUTING.md][2] for instructions on the developer certificate of origin that we require.
- Read how [we're using ZenHub][8] for project and roadmap planning

### Pull requests

- We welcome pull requests. Feel free to dig through the [issues][1] and jump in.

## Changelog

See [the list of releases][3] to find out about feature changes.

[1]: https://github.com/kubermatic/operating-system-manager/issues
[2]: https://github.com/kubermatic/operating-system-manager/blob/main/CONTRIBUTING.md
[3]: https://github.com/kubermatic/operating-system-manager/releases
[4]: https://github.com/kubermatic/operating-system-manager/blob/main/CODE_OF_CONDUCT.md
[5]: https://groups.google.com/forum/#!forum/kubermatic-dev
[6]: https://kubermatic.slack.com/messages/kubermatic
[7]: http://slack.kubermatic.io/
[8]: https://github.com/kubermatic/operating-system-manager/blob/main/Zenhub.md
