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

DirectPV plugin can be installed from kubectl krew index using:

```sh
kubectl krew install directpv
```

After running this installation:

 - The `directpv` plugin will be installed in your krew installation directory (default: `$HOME/.krew`) 
 - Run `kubectl directpv` to verify that the installation worked
 - If the error `Error: unknown command "directpv" for "kubectl"` is shown, try adding `$HOME/.krew/bin` to your `$PATH`

### Driver Installation

For installation in production grade environments, ensure that all criteria in the [Production Readiness Checklist](#production-readiness-checklist) are satisfied.

#### 1. Install the driver

```sh
kubectl directpv install --crd
```

This will install directpv driver in the kubernetes cluster.

**Notes:**

 - directpv components are installed in the namespace `directpv-min-io`
 - alternate kubeconfig path can be specified using `kubectl directpv --kubeconfig /path/to/kubeconfig` 
 - `--crd` flag is only required when installing for the first time
 - the rbac requirements for the driver is [here](./specification.md#driver-rbac)
 - the driver runs in `privileged` mode, which is required for mounting, unmounting and formatting drives

#### 2. List discovered drives

```sh
kubectl directpv drives ls
```

This will list all available drives in the kubernetes cluster.

**Notes:**

 - if all drives are not listed, try again in a few seconds
 - drives mounted at `/`(root) are not shown in the list by default. They are marked as `Unavailable`
 - drives marked `Available` can be formatted and managed by `directpv`
 - drives marked as `InUse` or `Ready` are already managed by directpv
 - using `--all` flag will list all drives, including those marked `Unavailable`

#### 3. Add available drives

```sh
kubectl directpv drives format --drives /dev/xvdb,/dev/xvdc --nodes directpv-1,directpv-2,directpv-3,directpv-4
```

This will format selected drives and mark them `Ready` to be used by directpv.

**Notes:**

 - formatting will erase all data on the drives. Double check to make sure that only intended drives are specified 
 - `Unavailable` drives cannot be formatted
 - `--drives` is a string list flag, i.e. a comma separated list of drives can be specified using this flag
 - `--nodes` is a string list flag, i.e. a comma separated list of nodes can be specified using this flag
 - both  `--drives` and `--nodes` understand glob format. i.e. the above command can be shortened to `kubectl directpv drives format --drives '/dev/xvd*' --nodes directpv-*`. Using this can lead to unintended data loss. Use at your own risk. 

#### 4. Verify installation

```sh
kubectl directpv info
```

This will show information about the drives formatted and added to directpv.

After running this installation:

 - storage class named `directpv-min-io` is created
 - `directpv-min-io` can be specified in `PodSpec.VolumeClaimTemplates` to provision DirectPV volumes
 - example statefulset using directpv can be found [here](../minio.yaml#L61) 
 - optional: view the [driver specification](./specification.md)
<!-- - view the [usage guide](./usage-guide.md) -->

## Air-gapped Installation (Private Registry)

Push the following images to your private registry
 
 - quay.io/minio/csi-node-driver-registrar:v2.2.0-go1.17
 - quay.io/minio/csi-provisioner:v2.2.0-go1.17
 - quay.io/minio/livenessprobe:v2.2.0-go1.17
 - quay.io/minio/directpv:${latest_tag_name}

Here is a shell script to Copy-Paste into your terminal to do the above steps:
```sh
/bin/bash -e

# set this to private registry URL (the URL should NOT include http or https)
if [ -z $PRIVATE_REGISTRY_URL ]; then "PRIVATE_REGISTRY_URL env var should be set"; fi

images[0]=quay.io/minio/csi-node-driver-registrar:v2.2.0-go1.17
images[1]=quay.io/minio/csi-provisioner:v2.2.0-go1.17
images[2]=quay.io/minio/livenessprobe:v2.2.0-go1.17
images[3]=quay.io/minio/directpv:$(curl -s "https://api.github.com/repos/minio/directpv/releases/latest" | grep tag_name | sed -E 's/.*"([^"]+)".*/\1/')

function privatize(){ echo $1 | sed "s#quay.io#${PRIVATE_REGISTRY_URL}#g"; }
function pull_tag_push(){ docker pull $1 &&  docker tag $1 $2 && docker push $2; }
for image in ${images[*]}; do pull_tag_push $image $(privatize $image); done
```

## Custom Installation

If any other customization is desired,

Step 1: Generate the specification for installing directpv
```sh
$ kubectl directpv install --dry-run > directpv-install.yaml
```

Step 2: Make appropriate changes to the resources
```
$ emacs directpv-install.yaml
```

Step 3: Install DirectPV
```
$ kubectl create -f directpv-install.yaml
```

Client-side upgrade functionality will not be available for custom installations.

## Production Readiness Checklist

Make sure the following check-boxes are ticked before production deployment

 - [ ] If using a private registry, all the images listed in [air-gapped installation](#air-gapped-installation-private-registry) should be available in the private registry
 - [ ] If seccomp is enabled in the system, directpv [seccomp policy](../seccomp.json) should be loaded on all nodes. Instructions available [here](https://kubernetes.io/docs/tutorials/clusters/seccomp/)
 - [ ] If apparmor is enabled in the system, directpv [apparmor profile](../apparmor.profile) should be loaded on all nodes. Instructions available [here](https://kubernetes.io/docs/tutorials/clusters/apparmor/)
 - [ ] Review and Sign-off the [Security Checklist](../security-checklist.md) for providing elevated privileges to directpv.
