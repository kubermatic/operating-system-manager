# [Experimental] Kubermatic operating-system-manager

Operating System Manager is responsible for creating and managing the required configurations for worker nodes in a kubernetes cluster.

## Project Status

This project is experimental and currently a work-in-progress. **This is not supposed to be used in production environments**.

## Overview

### Problem Statement

[Machine-Controller](https://github.com/kubermatic/machine-controller) can be used to create and manage worker nodes in a kubernetes clusters. For each supported operating system(based on the cloud provider), a specific plugin is used to generate cloud configs. These configs are then injected in the worker nodes using either [cloud-init](https://cloud-init.io/) or (ignition)[https://coreos.github.io/ignition/] based on the operating system. Finally the nodes are bootstrapped.

Currently this workflow has the following limitations/issues:

- Machine Controller expects **ALL** the supported user-data plugins to exist and be ready. User might only be interested in a subset of the available operating systems. For example, user might only want to work with `ubuntu`.
- The user-data plugins have templates defined [in-code](https://github.com/kubermatic/machine-controller/blob/master/pkg/userdata/ubuntu/provider.go#L133). Which is not ideal because code changes are required to update those templates.
- Managing configs for multiple cloud providers, OS flavors and OS versions, adds a lot of complexity and redundancy in machine-controller.
- Since the templates are defined in-code, there is no way for an end user to customize them to suit their use-cases.
- Each cloud provider sets some sort of limits for the size of `user-data`, machine won't be created in case of non-compliance. For example, at the time of writing this, AWS has set a [hard limit of 16KB](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-add-user-data.html).
- Better support for air-gapped environments is required.

### Solution

Operating System Manager was created to solve the above mentioned issues. It decouples operating system configurations into dedicated and isolable resources for better modularity and maintainability.

## Architecture

OSM introduces the following resources:


### OperatingSystemProfile

Templatized resource that represents the details of each operating system. OSPs are immutable and default OSPs for supported operating systems are provided/installed automatically by kubermatic. End users can create custom OSPs as well to fit their own use-cases.

Its dedicated controller runs in the **seed** cluster, in user cluster namespace, and operates on the `OperatingSystemProfile` custom resource. It is responsible for installing the default OSPs in user-cluster namespace.

### OperatingSystemConfig

Immutable resource that contains the actual configurations that are going to be used to bootstrap and provision the worker nodes. It is a subset of OperatingSystemProfile, rendered using OperatingSystemProfile, MachineDeployment and flags

Its dedicated controller runs in the **seed** cluster, in user cluster namespace, and is responsible for generating the OSCs in **seed** and secrets in `cloud-init-settings` namespace in the user cluster.


For each cluster there are at least two OSC objects:

1. **Bootstrap**: OSC used for initial configuration of machine and to fetch the provisioning OSC object.
2. **Provisioning**: OSC with the actual cloud-config that provision the worker node.

OSCs are processed by controllers to eventually generate **secrets inside each user cluster**. These secrets are then consumed by worker nodes.

![Architecture](./docs/images/architecture-osm.png)

### Air-gapped Environment

This controller was designed by keeping air-gapped environments in mind. Customers can use their own VM images by creating custom OSP profiles to provision nodes in a cluster that doesn't have outbound internet access.

More work is being done to make it even easier to use OSM in air-gapped environments.

## Support

Information about supported OS versions can be found [here](./docs/compatibility-matrix.md).

## Deploy OSM

[TBD]

_The code and sample YAML files in the master branch of the operating-system-manager repository are under active development and are not guaranteed to be stable. Use them at your own risk!_

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
[2]: https://github.com/kubermatic/operating-system-manager/blob/master/CONTRIBUTING.md
[3]: https://github.com/kubermatic/operating-system-manager/releases
[4]: https://github.com/kubermatic/operating-system-manager/blob/master/CODE_OF_CONDUCT.md
[5]: https://groups.google.com/forum/#!forum/kubermatic-dev
[6]: https://kubermatic.slack.com/messages/kubermatic
[7]: http://slack.kubermatic.io/
[8]: https://github.com/kubermatic/operating-system-manager/blob/master/Zenhub.md
