# Installation
DirectPV comes with two components:
1. DirectPV plugin - installed on client machine.
2. DirectPV CSI driver - installed on Kubernetes cluster.

## DirectPV plugin installation
The plugin needs to be installed to manage DirectPV CSI driver in Kubernetes.

### Prerequisites
* Access to Kubernetes cluster.

### Installation using `Krew`
The latest DirectPV plugin is available in `Krew` repository. Use below steps to install the plugin in your system.
```sh
# Update the plugin list.
$ kubectl krew update

# Install DirectPV plugin.
$ kubectl krew install directpv
```

### Installation of release binary
The plugin binary name starts by `kubectl-directpv` and is available at https://github.com/minio/directpv/releases/latest. Download the binary as per your operating system and architecture. Below is an example for `GNU/Linux` on `amd64` architecture:
```sh
# Download DirectPV plugin.
$ release=$(curl -sfL "https://api.github.com/repos/minio/directpv/releases/latest" | awk '/tag_name/ { print substr($2, 3, length($2)-4) }')
$ curl -fLo kubectl-directpv https://github.com/minio/directpv/releases/download/v${release}/kubectl-directpv_${release}_linux_amd64

# Make the binary executable.
$ chmod a+x kubectl-directpv
```

## DirectPV CSI driver installation
Before starting the installation, it is required to have DirectPV plugin installed on your system. For plugin installation refer [this documentation](#directpv-plugin-installation). If you are not using `krew`, replace `kubectl directpv` by `kubectl-directpv` in below steps.

### Prerequisites
* Kubernetes >= v1.18 on GNU/Linux on amd64.
* If you use private registry, below images must be pushed into your registry. You could use [this helper script](tools/push-images.sh) to do that.
  - quay.io/minio/csi-node-driver-registrar:v2.8.0
  - quay.io/minio/csi-provisioner:v3.5.0 _(for Kubernetes >= v1.20)_
  - quay.io/minio/csi-provisioner:v2.2.0-go1.18 _(for kubernetes < v1.20)_
  - quay.io/minio/livenessprobe:v2.10.0
  - quay.io/minio/csi-resizer:v1.8.0
  - quay.io/minio/directpv:latest
* If `seccomp` is enabled, load [DirectPV seccomp profile](../seccomp.json) on nodes where you want to install DirectPV and use `--seccomp-profile` flag to `kubectl directpv install` command. For more information, refer Kubernetes documentation [here](https://kubernetes.io/docs/tutorials/clusters/seccomp/)
* If `apparmor` is enabled, load [DirectPV apparmor profile](../apparmor.profile) on nodes where you want to install DirectPV and use `--apparmor-profile` flag to `kubectl directpv install` command. For more information, refer to the [Kubernetes documentation](https://kubernetes.io/docs/tutorials/clusters/apparmor/).
* Enabled `ExpandCSIVolumes` [feature gate](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/) for [volume expansion](https://kubernetes-csi.github.io/docs/volume-expansion.html) feature.
* Review the [driver specification documentation](./specification.md)
* For Red Hat Openshift users, refer to the [Openshift specific documentation](./openshift.md) for configuration prior to install DirectPV.

### Default installation
To install DirectPV in all Kubernetes nodes, run
```sh
$ kubectl directpv install
```

### Customized installation
To install DirectPV on selected Kubernetes nodes and/or with tolerations and/or with non-standard `kubelet` directory, use below steps accordingly.

#### Installing on selected nodes
To install DirectPV on selected nodes, use `--node-selector` flag to `install` command. Below is an example
```sh
# Install DirectPV on nodes having label 'group-name' key and 'bigdata' value
$ kubectl directpv install --node-selector group-name=bigdata
```

#### Installing on tainted nodes
To install DirectPV on tainted nodes, use `--toleration` flag to `install` command. Below is an example
```sh
# Install DirectPV on tainted nodes by tolerating 'key1' key, 'Equal' operator for 'value1' value with 'NoSchedule' effect
$ kubectl directpv install --tolerations key1=value1:NoSchedule

# Install DirectPV on tainted nodes by tolerating 'key2' key, 'Exists' operator with 'NoExecute' effect
$ kubectl directpv install --tolerations key2:NoExecute
```

#### Installing on non-standard `kubelet` directory
To install on non-standard `kubelet` directory, set the `KUBELET_DIR_PATH` environment variable and start the installation. Below is an example
```sh
$ export KUBELET_DIR_PATH=/path/to/my/kubelet/dir
$ kubectl directpv install
```

#### Installing on Openshift
To install DirectPV on Openshift with specific configuration, use the `--openshift` flag. Below is an example
```sh
$ kubectl directpv install --openshift
```

#### Installing by generating DirectPV manifests
To install using generated manifests file, run below command
```sh
$ curl -sfL https://github.com/minio/directpv/raw/master/docs/tools/install.sh | sh - apply
```

## What's next
* [Add drives](./drive-management.md#add-drives)
* [Provision volumes](./volume-provisioning.md)

## Further reads
* [Drive management guide](./drive-management.md)
* [Volume management guide](./volume-management.md)
* [Troubleshooting guide](./troubleshooting.md)
