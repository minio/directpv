### Install Kubectl plugin

The `direct-csi` kubectl plugin has been provided to manage the lifecycle of direct-csi

```sh
$ kubectl krew install direct-csi
```


### Install DirectCSI

Using the kubectl plugin, install direct-csi driver in your kubernetes cluster

```sh
$ kubectl direct-csi install --help

	-k, --kubeconfig string   path to kubeconfig
	-c, --crd                 register crds along with installation [use it on your first installation]
	-f, --force               delete and recreate CRDs [use it when upgrading direct-csi]
```

### Uninstall DirectCSI

Using the kubectl plugin, uninstall direct-csi driver from your kubernetes cluster

```sh
$ kubectl direct-csi drives uninstall --help

Usage:
  kubectl-direct_csi uninstall [flags]

Flags:
  -c, --crd    unregister direct.csi.min.io group crds
  -h, --help   help for uninstall

Global Flags:
  -k, --kubeconfig string   path to kubeconfig
  -v, --v Level             log level for V logs
```


### Drives 

The kubectl plugin makes it easy to discover drives in your cluster

```sh
$ kubectl direct-csi drives list --help

Flags:
  -d, --drives strings   glob selector for drive paths
  -h, --help             help for list
  -n, --nodes strings    glob selector for node names
  -s, --status strings   glob selector for drive status

Examples:

# Filter all nvme drives in all nodes 
$ kubectl direct-csi drives list --drives='/dev/nvme*'

# Filter all available drives 
$ kubectl direct-csi drives list --status=available

# Filter all drives from a particular node
$ kubectl direct-csi drives list --nodes=directcsi-1

# Combine multiple filters
$ kubectl direct-csi drives list --nodes=directcsi-1 --nodes=othernode-2 --status=ready

# Combine multiple filters using csv
$ kubectl direct-csi drives list --nodes=directcsi-1,othernode-2 --status=ready
```

**EXAMPLE** When direct-csi is first installed, the output will look something like this, with most drives in `Available` status

```sh
$ kubectl direct-csi drives list
 DRIVE      CAPACITY  ALLOCATED  VOLUMES  NODE         STATUS
 /dev/xvdb  10 GiB    -          -        directcsi-1  Available
 /dev/xvdc  10 GiB    -          -        directcsi-1  Available 
 /dev/xvdb  10 GiB    -          -        directcsi-2  Available 
 /dev/xvdc  10 GiB    -          -        directcsi-2  Available 
 /dev/xvdb  10 GiB    -          -        directcsi-3  Available 
 /dev/xvdc  10 GiB    -          -        directcsi-3  Available 
 /dev/xvdb  10 GiB    -          -        directcsi-4  Available 
 /dev/xvdc  10 GiB    -          -        directcsi-4  Available 
```

### Format and add Drives to DirectCSI 

```sh
$ kubectl direct-csi drives format --help
add drives to the DirectCSI cluster

Usage:
  kubectl-direct_csi drives format [flags]

Examples:

# Add all Available drives
$ kubectl direct-csi drives format --all

# Add all nvme drives in all nodes 
$ kubectl direct-csi drives format --drives='/dev/nvme*'

# Add all drives from a particular node
$ kubectl direct-csi drives format --nodes=directcsi-1

# Combine multiple parameters using multi-arg
$ kubectl direct-csi drives format --nodes=directcsi-1 --nodes=othernode-2 --status=ready

# Combine multiple parameters using csv
$ kubectl direct-csi drives format --nodes=directcsi-1,othernode-2 --status=ready

Flags:
  -d, --drives strings      glog selector for drive paths
  -f, --force               force format a drive even if a FS is already present
  -h, --help                help for add
  -n, --nodes strings       glob selector for node names

Global Flags:
  -k, --kubeconfig string   path to kubeconfig
  -v, --v Level             log level for V logs
```

**WARNING** - Adding drives to direct-csi will result in them being formatted

 - You can optionally select particular nodes from which the drives should be added using the `--nodes` flag
 - The drives are always formatted with `XFS` filesystem
 - If a parition table or a filesystem is already present on a drive, then `drive format` will fail 
 - You can override this behavior by setting the `--force` flag, which overwrites any parition table or filesystem present on the drive
 - Any drive/paritition mounted at '/' (root) or having the GPT PartUUID of Boot partitions will be marked `Unavailable`. These drives cannot be added even if `--force` flag is set
 

#### Drive Status 

 | Status      | Description                                                                                                  |
 |-------------|--------------------------------------------------------------------------------------------------------------|
 | Available   | These drives are available for DirectCSI to use                                                              |
 | Unavailable | Either a boot partition or drive mounted at '/'. Note: All other drives can be given to direct-csi to manage |
 | InUse       | Drive is currently in use by a volume. i.e. number of volumes > 0                                            |
 | Ready       | Drive is formatted and ready to be used, but no volumes have been assigned on this drive yet                 |
 | Terminating | Drive is currently being deleted                                                                             |


### Volumes 

The kubectl plugin makes it easy to discover volumes in your cluster

```sh
Usage:
  kubectl-direct_csi volumes list [flags]

Aliases:
  list, ls

Examples:

# List all volumes provisioned on nvme drives across all nodes 
$ kubectl direct-csi volumes ls --drives='/dev/nvme*'

# List all staged and published volumes
$ kubectl direct-csi volumes ls --status=staged,published

# List all volumes from a particular node
$ kubectl direct-csi volumes ls --nodes=directcsi-1

# Combine multiple filters using csv
$ kubectl direct-csi vol ls --nodes=directcsi-1,directcsi-2 --status=staged --drives=/dev/nvme0n1


Flags:
  -d, --drives strings   glob prefix match for drive paths
  -h, --help             help for list
  -n, --nodes strings    glob prefix match for node names
  -s, --status strings   glob prefix match for drive status

Global Flags:
  -k, --kubeconfig string   path to kubeconfig
  -v, --v Level             log level for V logs
```

### Verify Installation

 - Check if all the pods are deployed correctly. i.e. they are 'Running'

```sh
$ kubectl -n direct-csi-min-io get pods
NAME                                 READY   STATUS    RESTARTS   AGE
direct-csi-min-io-2hcmc              3/3     Running   0          6s
direct-csi-min-io-79bc887cff-5rxzj   2/2     Running   0          6s
direct-csi-min-io-79bc887cff-hpzzx   2/2     Running   0          6s
direct-csi-min-io-79bc887cff-pf98s   2/2     Running   0          6s
direct-csi-min-io-l2hq8              3/3     Running   0          6s
direct-csi-min-io-mfv4h              3/3     Running   0          6s
direct-csi-min-io-zq486              3/3     Running   0          6s
```

 - Check if directcsidrives and directcsivolumes CRDs are registered

```sh
$ kubectl get crd | grep directcsi
directcsidrives.direct.csi.min.io                     2020-12-23T03:01:13Z
directcsivolumes.direct.csi.min.io                    2020-12-23T03:01:13Z
```

 - Check if DirectCSI drives are discovered. Atleast 1 drive should be in `Ready` or `InUse` for volumes to be provisioned


```sh
$ kubectl direct-csi drives list --all
 DRIVE      CAPACITY  ALLOCATED  VOLUMES  NODE         STATUS
 /dev/xvda  64 GiB    -          -        directcsi-1  Unavailable 
 /dev/xvdb  10 GiB    4.0 GiB    2        directcsi-1  InUse 
 /dev/xvdc  10 GiB    4.0 GiB    2        directcsi-1  InUse 
 /dev/xvda  64 GiB    -          -        directcsi-2  Unavailable 
 /dev/xvdb  10 GiB    4.0 GiB    2        directcsi-2  InUse 
 /dev/xvdc  10 GiB    4.0 GiB    2        directcsi-2  InUse 
 /dev/xvda  64 GiB    -          -        directcsi-3  Unavailable 
 /dev/xvdb  10 GiB    4.0 GiB    2        directcsi-3  InUse 
 /dev/xvdc  10 GiB    4.0 GiB    2        directcsi-3  InUse 
 /dev/xvda  64 GiB    -          -        directcsi-4  Unavailable 
 /dev/xvdb  10 GiB    4.0 GiB    2        directcsi-4  InUse 
 /dev/xvdc  10 GiB    4.0 GiB    2        directcsi-4  InUse 
```

 - Check if volumes are being provisioned correctly

```sh
# Create a volumeClaimTemplate that refers to direct-csi-min-io driver
$ cat minio.yaml | grep -C 10 direct.csi.min.io
  volumeClaimTemplates: # This is the specification in which you reference the StorageClass
  - metadata:
      name: minio-data-1
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 2Gi
      storageClassName: direct-csi-min-io # This field references the existing StorageClass
```


 - Check if volumes are being provisioned correctly

```
# Create the pods and check if PVCs are bound
$ kubectl get pvc | grep minio-data
minio-data-1-minio-0   Bound    pvc-a7c76893-f3d9-472e-9a43-ce5775df61fa   2Gi        RWO            direct-csi-min-io   13h
minio-data-1-minio-1   Bound    pvc-51232f07-30fa-48eb-ac4f-5d529f82fbcf   2Gi        RWO            direct-csi-min-io   13h
minio-data-1-minio-2   Bound    pvc-77915ec1-9c7a-4423-8879-4f4f19a12ca8   2Gi        RWO            direct-csi-min-io   13h
minio-data-1-minio-3   Bound    pvc-13b788ab-75ab-441c-a6a6-46af32929702   2Gi        RWO            direct-csi-min-io   13h

# List the volumes to see if they are provisioned
$ kubectl direct-csi vol ls
 VOLUME                                    CAPACITY  NODE         DRIVE
 pvc-590cf951-af53-4d6e-b3d1-3a5b07e88a83  2.0 GiB   directcsi-1  xvdc
 pvc-a8171e3d-f4bd-48ef-9ea0-55f68ecaa664  2.0 GiB   directcsi-1  xvdc
 pvc-b881c91b-64f3-479e-b1ef-ce8b87cc27ea  2.0 GiB   directcsi-3  xvdc
 pvc-ec810adf-753a-47b1-8768-4d564fe360fc  2.0 GiB   directcsi-3  xvdc
 pvc-d9979e3f-79c5-475e-a9a7-0545b497a692  2.0 GiB   directcsi-2  xvdb
 pvc-1a2f599b-1a63-48da-a8b4-cc36a3ab7a41  2.0 GiB   directcsi-2  xvdb
 pvc-7af9bcb4-1d0c-4ad3-b561-ee12d55c0395  2.0 GiB   directcsi-1  xvdb
 pvc-95460617-dcc6-4b5f-b9cb-ccb2da504309  2.0 GiB   directcsi-1  xvdb
 pvc-efded452-c5b9-45b9-bcda-11a584bee119  2.0 GiB   directcsi-4  xvdc
 pvc-fbdfa236-c764-49ad-9b65-1d71366104a8  2.0 GiB   directcsi-4  xvdc
 pvc-50b81c44-568d-4325-a63e-7f8abd25705c  2.0 GiB   directcsi-2  xvdc
 pvc-c4c21bb2-503e-454a-87ac-c748e62e4d3d  2.0 GiB   directcsi-2  xvdc
 pvc-1aea229e-4364-4b96-8c71-869ab627f520  2.0 GiB   directcsi-3  xvdb
 pvc-7de5d796-28fa-4fcd-9fac-8e3861d9134c  2.0 GiB   directcsi-3  xvdb
 pvc-c13a41f8-5bf0-4f45-84f1-10b3534e14d1  2.0 GiB   directcsi-4  xvdb
 pvc-1ee1a06e-0b09-45c2-89c3-e596fb1d6cab  2.0 GiB   directcsi-4  xvdb
```
