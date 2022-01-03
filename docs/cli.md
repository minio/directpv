### Install Kubectl plugin

The `directpv` kubectl plugin has been provided to manage the lifecycle of directpv

```sh
$ kubectl krew install directpv
```


### Install DirectPV

Using the kubectl plugin, install directpv driver in your kubernetes cluster

```sh
$ kubectl directpv install --help

	-k, --kubeconfig string   path to kubeconfig
	-c, --crd                 register crds along with installation [use it on your first installation]
	-f, --force               delete and recreate CRDs [use it when upgrading directpv]
```

### Uninstall DirectPV

Using the kubectl plugin, uninstall directpv driver from your kubernetes cluster

```sh
$ kubectl directpv drives uninstall --help

Usage:
  kubectl-directpv uninstall [flags]

Flags:
  -c, --crd    unregister direct.pv.min.io group crds
  -h, --help   help for uninstall

Global Flags:
  -k, --kubeconfig string   path to kubeconfig
  -v, --v Level             log level for V logs
```


### Drives 

The kubectl plugin makes it easy to discover drives in your cluster

```sh
list drives in the DirectPV cluster

Usage:
  directpv drives list [flags]

Aliases:
  list, ls

Examples:

# Filter drives by ellipses notation for drive paths and nodes
$ kubectl directpv drives ls --drives='/dev/xvd{a...d}' --nodes='node-{1...4}'

# Filter all ready drives 
$ kubectl directpv drives ls --status=ready

# Filter all drives from a particular node
$ kubectl directpv drives ls --nodes=directpv-1

# Combine multiple filters using multi-arg
$ kubectl directpv drives ls --nodes=directpv-1 --nodes=othernode-2 --status=available

# Combine multiple filters using csv
$ kubectl directpv drives ls --nodes=directpv-1,othernode-2 --status=ready

# Filter all drives based on access-tier
$ kubectl directpv drives drives ls --access-tier="hot"

# Filter all drives with access-tier being set
$ kubectl directpv drives drives ls --access-tier="*"


Flags:
      --access-tier strings   match based on access-tier
  -a, --all                   list all drives (including unavailable)
  -d, --drives strings        filter by drive path(s) (also accepts ellipses range notations)
  -h, --help                  help for list
  -n, --nodes strings         filter by node name(s) (also accepts ellipses range notations)
  -s, --status strings        match based on drive status [InUse, Available, Unavailable, Ready, Terminating, Released]
```

**EXAMPLE** When directpv is first installed, the output will look something like this, with most drives in `Available` status

```sh
$ kubectl directpv drives list
 DRIVE      CAPACITY  ALLOCATED  VOLUMES  NODE         STATUS
 /dev/xvdb  10 GiB    -          -        directpv-1  Available
 /dev/xvdc  10 GiB    -          -        directpv-1  Available 
 /dev/xvdb  10 GiB    -          -        directpv-2  Available 
 /dev/xvdc  10 GiB    -          -        directpv-2  Available 
 /dev/xvdb  10 GiB    -          -        directpv-3  Available 
 /dev/xvdc  10 GiB    -          -        directpv-3  Available 
 /dev/xvdb  10 GiB    -          -        directpv-4  Available 
 /dev/xvdc  10 GiB    -          -        directpv-4  Available 
```

### Format and add Drives to DirectPV

```sh
format drives in the DirectPV cluster

Usage:
  directpv drives format [flags]

Examples:

# Format all available drives in the cluster
$ kubectl directpv drives format --all

# Format the 'sdf' drives in all nodes
$ kubectl directpv drives format --drives '/dev/sdf'

# Format the selective drives using ellipses notation for drive paths
$ kubectl directpv drives format --drives '/dev/sd{a...z}'

# Format the drives from selective nodes using ellipses notation for node names
$ kubectl directpv drives format --nodes 'directpv-{1...3}'

# Format all drives from a particular node
$ kubectl directpv drives format --nodes=directpv-1

# Format all drives based on the access-tier set [hot|cold|warm]
$ kubectl directpv drives format --access-tier=hot

# Combine multiple parameters using multi-arg
$ kubectl directpv drives format --nodes=directpv-1 --nodes=othernode-2 --status=available

# Combine multiple parameters using csv
$ kubectl directpv drives format --nodes=directpv-1,othernode-2 --status=available

# Combine multiple parameters using ellipses notations
$ kubectl directpv drives format --nodes "directpv-{3...4}" --drives "/dev/xvd{b...f}"

# Format a drive by it's drive-id
$ kubectl directpv drives format <drive_id>

# Format more than one drive by their drive-ids
$ kubectl directpv drives format <drive_id_1> <drive_id_2>


Flags:
      --access-tier strings   format based on access-tier set. The possible values are hot|cold|warm
  -a, --all                   format all available drives
  -d, --drives strings        filter by drive path(s) (also accepts ellipses range notations)
  -f, --force                 force format a drive even if a FS is already present
  -h, --help                  help for format
  -n, --nodes strings         filter by node name(s) (also accepts ellipses range notations)
```

**WARNING** - Adding drives to directpv will result in them being formatted

 - You can optionally select particular nodes from which the drives should be added using the `--nodes` flag
 - The drives are always formatted with `XFS` filesystem
 - If a parition table or a filesystem is already present on a drive, then `drive format` will fail 
 - You can override this behavior by setting the `--force` flag, which overwrites any parition table or filesystem present on the drive
 - Any drive/paritition mounted at '/' (root) or having the GPT PartUUID of Boot partitions will be marked `Unavailable`. These drives cannot be added even if `--force` flag is set
 

#### Drive Status 

 | Status      | Description                                                                                                  |
 |-------------|--------------------------------------------------------------------------------------------------------------|
 | Available   | These drives are available for DirectPV to use                                                              |
 | Unavailable | Either a boot partition or drive mounted at '/'. Note: All other drives can be given to directpv to manage |
 | InUse       | Drive is currently in use by a volume. i.e. number of volumes > 0                                            |
 | Ready       | Drive is formatted and ready to be used, but no volumes have been assigned on this drive yet                 |
 | Terminating | Drive is currently being deleted                                                                             |


### Volumes 

The kubectl plugin makes it easy to discover volumes in your cluster

```sh
list volumes in the DirectPV cluster

Usage:
  directpv volumes list [flags]

Aliases:
  list, ls

Examples:


# List all staged and published volumes
$ kubectl directpv volumes ls --status=staged,published

# List all volumes from a particular node
$ kubectl directpv volumes ls --nodes=directpv-1

# Combine multiple filters using csv
$ kubectl directpv vol ls --nodes=directpv-1,directpv-2 --status=staged --drives=/dev/nvme0n1

# List all published volumes by pod name
$ kubectl directpv volumes ls --status=published --pod-name=minio-{1...3}

# List all published volumes by pod namespace
$ kubectl directpv volumes ls --status=published --pod-namespace=tenant-{1...3}

# List all volumes provisioned based on drive and volume ellipses
$ kubectl directpv volumes ls --drives '/dev/xvd{a...d} --nodes 'node-{1...4}''


Flags:
  -a, --all                     list all volumes (including non-provisioned)
  -d, --drives strings          filter by drive path(s) (also accepts ellipses range notations)
  -h, --help                    help for list
  -n, --nodes strings           filter by node name(s) (also accepts ellipses range notations)
      --pod-name strings        filter by pod name(s) (also accepts ellipses range notations)
      --pod-namespace strings   filter by pod namespace(s) (also accepts ellipses range notations)
  -s, --status strings          match based on volume status. The possible values are [staged,published]
```

### Verify Installation

 - Check if all the pods are deployed correctly. i.e. they are 'Running'

```sh
$ kubectl -n directpv-min-io get pods
NAME                                 READY   STATUS    RESTARTS   AGE
direct-pv-min-io-2hcmc              3/3     Running   0          6s
direct-pv-min-io-79bc887cff-5rxzj   2/2     Running   0          6s
direct-pv-min-io-79bc887cff-hpzzx   2/2     Running   0          6s
direct-pv-min-io-79bc887cff-pf98s   2/2     Running   0          6s
direct-pv-min-io-l2hq8              3/3     Running   0          6s
direct-pv-min-io-mfv4h              3/3     Running   0          6s
direct-pv-min-io-zq486              3/3     Running   0          6s
```

 - Check if directpvdrives and directpvvolumes CRDs are registered

```sh
$ kubectl get crd | grep directpv
directpvdrives.direct.pv.min.io                     2020-12-23T03:01:13Z
directpvvolumes.direct.pv.min.io                    2020-12-23T03:01:13Z
```

 - Check if Directpv drives are discovered. Atleast 1 drive should be in `Ready` or `InUse` for volumes to be provisioned


```sh
$ kubectl directpv drives list --all
 DRIVE      CAPACITY  ALLOCATED  VOLUMES  NODE         STATUS
 /dev/xvda  64 GiB    -          -        directpv-1  Unavailable 
 /dev/xvdb  10 GiB    4.0 GiB    2        directpv-1  InUse 
 /dev/xvdc  10 GiB    4.0 GiB    2        directpv-1  InUse 
 /dev/xvda  64 GiB    -          -        directpv-2  Unavailable 
 /dev/xvdb  10 GiB    4.0 GiB    2        directpv-2  InUse 
 /dev/xvdc  10 GiB    4.0 GiB    2        directpv-2  InUse 
 /dev/xvda  64 GiB    -          -        directpv-3  Unavailable 
 /dev/xvdb  10 GiB    4.0 GiB    2        directpv-3  InUse 
 /dev/xvdc  10 GiB    4.0 GiB    2        directpv-3  InUse 
 /dev/xvda  64 GiB    -          -        directpv-4  Unavailable 
 /dev/xvdb  10 GiB    4.0 GiB    2        directpv-4  InUse 
 /dev/xvdc  10 GiB    4.0 GiB    2        directpv-4  InUse 
```

 - Check if volumes are being provisioned correctly

```sh
# Create a volumeClaimTemplate that refers to direct-pv-min-io driver
$ cat minio.yaml | grep -C 10 direct.pv.min.io
  volumeClaimTemplates: # This is the specification in which you reference the StorageClass
  - metadata:
      name: minio-data-1
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 2Gi
      storageClassName: directpv-min-io # This field references the existing StorageClass
```


 - Check if volumes are being provisioned correctly

```
# Create the pods and check if PVCs are bound
$ kubectl get pvc | grep minio-data
minio-data-1-minio-0   Bound    pvc-a7c76893-f3d9-472e-9a43-ce5775df61fa   2Gi        RWO            directpv-min-io   13h
minio-data-1-minio-1   Bound    pvc-51232f07-30fa-48eb-ac4f-5d529f82fbcf   2Gi        RWO            directpv-min-io   13h
minio-data-1-minio-2   Bound    pvc-77915ec1-9c7a-4423-8879-4f4f19a12ca8   2Gi        RWO            directpv-min-io   13h
minio-data-1-minio-3   Bound    pvc-13b788ab-75ab-441c-a6a6-46af32929702   2Gi        RWO            directpv-min-io   13h

# List the volumes to see if they are provisioned
$ kubectl directpv vol ls
 VOLUME                                    CAPACITY  NODE         DRIVE
 pvc-590cf951-af53-4d6e-b3d1-3a5b07e88a83  2.0 GiB   directpv-1  xvdc
 pvc-a8171e3d-f4bd-48ef-9ea0-55f68ecaa664  2.0 GiB   directpv-1  xvdc
 pvc-b881c91b-64f3-479e-b1ef-ce8b87cc27ea  2.0 GiB   directpv-3  xvdc
 pvc-ec810adf-753a-47b1-8768-4d564fe360fc  2.0 GiB   directpv-3  xvdc
 pvc-d9979e3f-79c5-475e-a9a7-0545b497a692  2.0 GiB   directpv-2  xvdb
 pvc-1a2f599b-1a63-48da-a8b4-cc36a3ab7a41  2.0 GiB   directpv-2  xvdb
 pvc-7af9bcb4-1d0c-4ad3-b561-ee12d55c0395  2.0 GiB   directpv-1  xvdb
 pvc-95460617-dcc6-4b5f-b9cb-ccb2da504309  2.0 GiB   directpv-1  xvdb
 pvc-efded452-c5b9-45b9-bcda-11a584bee119  2.0 GiB   directpv-4  xvdc
 pvc-fbdfa236-c764-49ad-9b65-1d71366104a8  2.0 GiB   directpv-4  xvdc
 pvc-50b81c44-568d-4325-a63e-7f8abd25705c  2.0 GiB   directpv-2  xvdc
 pvc-c4c21bb2-503e-454a-87ac-c748e62e4d3d  2.0 GiB   directpv-2  xvdc
 pvc-1aea229e-4364-4b96-8c71-869ab627f520  2.0 GiB   directpv-3  xvdb
 pvc-7de5d796-28fa-4fcd-9fac-8e3861d9134c  2.0 GiB   directpv-3  xvdb
 pvc-c13a41f8-5bf0-4f45-84f1-10b3534e14d1  2.0 GiB   directpv-4  xvdb
 pvc-1ee1a06e-0b09-45c2-89c3-e596fb1d6cab  2.0 GiB   directpv-4  xvdb
```
