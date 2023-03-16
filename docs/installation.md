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


### Plugin Installation

DirectPV plugin can be installed from kubectl krew index using:

```sh
kubectl krew install directpv
```

After running this installation:

 - The `directpv` plugin will be installed in your krew installation directory (default: `$HOME/.krew`) 
 - Run `kubectl directpv --version` to verify that the installation worked
 - If the error `Error: unknown command "directpv" for "kubectl"` is shown, try adding `$HOME/.krew/bin` to your `$PATH`

### Driver Installation

For installation in production grade environments, ensure that all criteria in the [Production Readiness Checklist](#production-readiness-checklist) are satisfied.

(NOTE: Please refer the [CLI Guide](./cli.md) for more helpers on the following commands.)

#### 1. Install the driver

```sh
kubectl directpv install
```

This will install `directpv-min-io` CSI driver in the kubernetes cluster.

**Notes:**

 - DirectPV components are installed in the namespace `directpv`
 - alternate kubeconfig path can be specified using `kubectl directpv --kubeconfig /path/to/kubeconfig` 
 - the rbac requirements for the driver is [here](./specification.md#driver-rbac)
 - the driver runs in `privileged` mode, which is required for mounting, unmounting and formatting drives
 - The deamonset used by directpv will be using the ports `10443` for metrics and `30443` for readiness handlers. Make sure these ports are open on the nodes.

#### 2. List discovered drives

```sh
kubectl directpv discover
```

This will list all available drives in the kubernetes cluster and will generate an init config file (default: drives.yaml) to initialize these drives.

#### 3. Intitialize the drives

```sh
kubectl directpv init drives.yaml
```

This will initialize selected drives in drives.yaml

**Notes:**

 - initialization will erase all data on the drives. Double check to make sure that only intended drives are specified 

#### 4. Verify installation

```sh
kubectl directpv info
```

This will show information about the drives formatted and added to DirectPV.

After running this installation:

 - storage class named `directpv-min-io` is created
 - `directpv-min-io` can be specified as the `storageClassName` in in `PodSpec.VolumeClaimTemplates` to provision DirectPV volumes
 - example statefulset using `directpv-min-io` can be found [here](../minio.yaml#L61) 
 - optional: view the [driver specification](./specification.md)
<!-- - view the [usage guide](./usage-guide.md) -->

## Air-gapped Installation (Private Registry)

Push the following images to your private registry
 
 - quay.io/minio/csi-node-driver-registrar:v2.6.0
 - quay.io/minio/csi-provisioner:v3.3.0
 - quay.io/minio/livenessprobe:v2.8.0
 - quay.io/minio/directpv:${latest_tag_name}
 
 **Notes:**

 - If your kubernetes version is less than v1.20, you need push `quay.io/minio/csi-provisioner:v2.2.0-go1.18`

Here is a shell script to Copy-Paste into your terminal to do the above steps:
```sh
/bin/bash -e

# set this to private registry URL (the URL should NOT include http or https)
if [ -z $PRIVATE_REGISTRY_URL ]; then "PRIVATE_REGISTRY_URL env var should be set"; fi

images[0]=quay.io/minio/csi-node-driver-registrar:v2.6.0
images[1]=quay.io/minio/csi-provisioner:v3.3.0
images[2]=quay.io/minio/livenessprobe:v2.8.0
images[3]=quay.io/minio/directpv:$(curl -s "https://api.github.com/repos/minio/directpv/releases/latest" | grep tag_name | sed -E 's/.*"([^"]+)".*/\1/')

function privatize(){ echo $1 | sed "s#quay.io#${PRIVATE_REGISTRY_URL}#g"; }
function pull_tag_push(){ docker pull $1 &&  docker tag $1 $2 && docker push $2; }
for image in ${images[*]}; do pull_tag_push $image $(privatize $image); done
```

## Custom Installation

If any other customization is desired,

Step 1: Generate the specification for installing DirectPV
```sh
$ kubectl directpv install -o yaml > directpv-install.yaml
```

Step 2: Make appropriate changes to the resources
```
$ emacs directpv-install.yaml
```

Step 3: Install DirectPV
```
$ kubectl create -f directpv-install.yaml
```

**Notes:**

- Client-side upgrade functionality will not be available for custom installations. You need to execute `kubectl directpv migrate` to migrate the old resources.

## Production Readiness Checklist

Make sure the following check-boxes are ticked before production deployment

 - [ ] If using a private registry, all the images listed in [air-gapped installation](#air-gapped-installation-private-registry) should be available in the private registry
 - [ ] If seccomp is enabled in the system, DirectPV [seccomp policy](../seccomp.json) should be loaded on all nodes. Instructions available [here](https://kubernetes.io/docs/tutorials/clusters/seccomp/)
 - [ ] If apparmor is enabled in the system, DirectPV [apparmor profile](../apparmor.profile) should be loaded on all nodes. Instructions available [here](https://kubernetes.io/docs/tutorials/clusters/apparmor/)
 - [ ] Review and Sign-off the [Security Checklist](../security-checklist.md) for providing elevated privileges to DirectPV.
