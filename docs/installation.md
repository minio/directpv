---
title: Installation
---

Installation
-------------

### Prerequisites

| Name         | Version  |
| -------------|----------|
| kubectl      | v1.19+   |
| kubernetes   | v1.19+   |

`ValidatingAdmissionWebhook` should be enabled in the kube-apiserver

### Plugin Installation

DirectCSI plugin can be installed from kubectl krew index using:

```sh
kubectl krew install direct-csi
```

After running this installation:

 - The `direct-csi` plugin will be installed in your krew installation directory (default: `$HOME/.krew`) 
 - Run `kubectl direct-csi` to verify that the installation worked
 - If the error `Error: unknown command "direct-csi" for "kubectl"` is shown, try adding `$HOME/.krew/bin` to your `$PATH`

### Driver Installation

#### 1. Install the driver

```sh
kubectl direct-csi install --crd
```

This will install direct-csi driver in the kubernetes cluster.

**Notes:**

 - direct-csi components are installed in the namespace `direct-csi-min-io`
 - alternate kubeconfig path can be specified using `kubectl direct-csi --kubeconfig /path/to/kubeconfig` 
 - `--crd` flag is only required when installing for the first time
 - the rbac requirements for the driver is [here](./specification.md#driver-rbac)
 - the driver runs in `privileged` mode, which is required for mounting, unmounting and formatting drives

#### 2. List discovered drives

```sh
kubectl direct-csi drives ls
```

This will list all available drives in the kubernetes cluster.

**Notes:**

 - if all drives are not listed, try again in a few seconds
 - drives mounted at `/`(root) are not shown in the list by default. They are marked as `Unavailable`
 - drives marked `Available` can be formatted and managed by `direct-csi`
 - drives marked as `InUse` or `Ready` are already managed by direct-csi
 - using `--all` flag will list all drives, including those marked `Unavailable`

#### 3. Add available drives

```sh
kubectl direct-csi drives format --drives /dev/xvdb,/dev/xvdc --nodes directcsi-1,directcsi-2,directcsi-3,directcsi-4
```

This will format selected drives and mark them `Ready` to be used by direct-csi.

**Notes:**

 - formatting will erase all data on the drives. Double check to make sure that only intended drives are specified 
 - `Unavailable` drives cannot be formatted
 - `--drives` is a string list flag, i.e. a comma separated list of drives can be specified using this flag
 - `--nodes` is a string list flag, i.e. a comma separated list of nodes can be specified using this flag
 - both  `--drives` and `--nodes` understand glob format. i.e. the above command can be shortened to `kubectl direct-csi drives format --drives '/dev/xvd*' --nodes directcsi-*`. Using this can lead to unintended data loss. Use at your own risk. 

#### 4. Verify installation

```sh
kubectl direct-csi info
```

This will show information about the drives formatted and added to direct-csi.

After running this installation:

 - storage class named `direct-csi-min-io` is created
 - `direct-csi-min-io` can be specified in `PodSpec.VolumeClaimTemplates` to provision DirectCSI volumes
 - example statefulset using direct-csi can be found [here](../minio.yaml#L61) 
 - optional: view the [driver specification](./specification.md)
<!-- - view the [usage guide](./usage-guide.md) -->
