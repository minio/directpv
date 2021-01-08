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


### Discover Drives 

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
$ kubectl direct-csi drives list --drives=/dev/nvme*

# Filter all new drives 
$ kubectl direct-csi drives list --status=new

# Filter all drives from a particular node
$ kubectl direct-csi drives list --nodes=directcsi-1

# Combine multiple filters
$ kubectl direct-csi drives list --nodes=directcsi-1 --nodes=othernode-2 --status=new

# Combine multiple filters using csv
$ kubectl direct-csi drives list --nodes=directcsi-1,othernode-2 --status=new
```

**EXAMPLE** When direct-csi is first installed, the output will look something like this, with most drives in `new` status

```sh
$ kubectl direct-csi drives list
 SERVER       DRIVES     STATUS       VOLUMES  CAPACITY      ALLOCATED  FREE          FS   MOUNT
 directcsi-1  /dev/xvda  unavailable  0        60.737418 GB  0 B        0 B           ext4 /
 directcsi-1  /dev/xvdb  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-1  /dev/xvdc  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-2  /dev/xvda  unavailable  0        0 B           0 B        0 B           ext4 /
 directcsi-2  /dev/xvdb  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-2  /dev/xvdc  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-3  /dev/xvda  unavailable  0        0 B           0 B        0 B           ext4 /
 directcsi-3  /dev/xvdb  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-3  /dev/xvdc  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-4  /dev/xvda  unavailable  0        0 B           0 B        0 B           ext4 /
 directcsi-4  /dev/xvdb  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-4  /dev/xvdc  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -

(0/12) Drives added to DirectCSI
```

### Add Drives for DirectCSI 

```sh
$ kubectl direct-csi drives add --help
add drives to the DirectCSI cluster

Usage:
  kubectl-direct_csi drives add [flags]

Examples:

# Add all available drives
$ kubectl direct-csi drives add

# Add all nvme drives in all nodes 
$ kubectl direct-csi drives add --drives=/dev/nvme*

# Add all new drives
$ kubectl direct-csi drives add --status=new

# Add all drives from a particular node
$ kubectl direct-csi drives add --nodes=directcsi-1

# Combine multiple parameters using multi-arg
$ kubectl direct-csi drives add --nodes=directcsi-1 --nodes=othernode-2 --status=new

# Combine multiple parameters using csv
$ kubectl direct-csi drives add --nodes=directcsi-1,othernode-2 --status=new

Flags:
  -d, --drives strings      glog selector for drive paths
  -f, --force               force format a drive even if a FS is already present
  -h, --help                help for add
  -m, --mountOpts strings   csv list of mount options
  -n, --nodes strings       glob selector for node names
  -s, --status strings      glob selector for drive status

Global Flags:
  -k, --kubeconfig string   path to kubeconfig
  -v, --v Level             log level for V logs
```

**WARNING** - Adding drives to direct-csi will result in them being formatted

 - The drive arg, the last argument in the commands above takes input in [ellipses format](./ellipses.md)
 - You can optionally select particular nodes from which the drives should be added using the `--nodes` flag
 - The drives are always formatted with `XFS` filesystem
 - If a parition table or a filesystem is present, then `drive add` fails
 - You can override this behavior by setting the `--force` flag, which overwrites any parition table or filesystem present on the drive
 - Any drive/paritition mounted at '/' (root) or having the GPT PartUUID of Boot partitions will show `unavailable` status. These drives cannot be added even if `--force` flag is set
 

#### Drive Status 

 | Status      | Description                                                                                                  |
 |-------------|--------------------------------------------------------------------------------------------------------------|
 | new         | These drives have not been assigned for DirectCSI to use                                                     |
 | unavailable | Either a boot partition or drive mounted at '/'. Note: All other drives can be given to direct-csi to manage |
 | in-use      | Drive is currently in use by a volume. i.e. number of volumes > 0                                            |
 | ready       | Drive is formatted and ready to be used, but no volumes have been assigned on this drive yet                 |
 | terminating | Drive is currently being deleted                                                                             |


 - Add drives that you would like DirectCSI to manage

```sh
$ kubectl direct-csi drives add --drives=/dev/xvdb,/dev/xvdc
 SERVER       DRIVES     STATUS  VOLUMES  CAPACITY      ALLOCATED  FREE          FS   MOUNT
 directcsi-1  /dev/xvdb  ready   0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/19de...
 directcsi-1  /dev/xvdc  ready   0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/1a44...
 directcsi-2  /dev/xvdb  ready   0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/1b1c...
 directcsi-2  /dev/xvdc  ready   0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/2d27...
 directcsi-3  /dev/xvdb  ready   0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/738f...
 directcsi-3  /dev/xvdc  ready   0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/8af6...
 directcsi-4  /dev/xvdb  ready   0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/cb8b...
 directcsi-4  /dev/xvdc  ready   0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/f67b...
 
(8/8) Drives added
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

 - Check if DirectCSI drives are discovered

```sh
$ kubectl direct-csi drives list
 SERVER       DRIVES     STATUS       VOLUMES  CAPACITY      ALLOCATED  FREE          FS   MOUNT
 directcsi-1  /dev/xvda  unavailable  0        0 B           0 B        0 B           ext4 /
 directcsi-1  /dev/xvdb  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-1  /dev/xvdc  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-2  /dev/xvda  unavailable  0        0 B           0 B        0 B           ext4 /
 directcsi-2  /dev/xvdb  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-2  /dev/xvdc  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-3  /dev/xvda  unavailable  0        0 B           0 B        0 B           ext4 /
 directcsi-3  /dev/xvdb  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-3  /dev/xvdc  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-4  /dev/xvda  unavailable  0        0 B           0 B        0 B           ext4 /
 directcsi-4  /dev/xvdb  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -
 directcsi-4  /dev/xvdc  new          0        10.737418 GB  0 B        10.618384 GB  xfs  -

(0/12) Drives available
```

 - Check if drives have been added to DirectCSI. Atleast 1 drive should be available for volumes to be provisioned

```sh
$ kubectl direct-csi drives list
 SERVER       DRIVES     STATUS       VOLUMES  CAPACITY      ALLOCATED  FREE          FS   MOUNT
 directcsi-1  /dev/xvda  unavailable  0        0 B           0 B        0 B           ext4 /
 directcsi-1  /dev/xvdb  ready        0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/1a44
 directcsi-1  /dev/xvdc  ready        0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/1a44
 directcsi-2  /dev/xvda  unavailable  0        0 B           0 B        0 B           ext4 /
 directcsi-2  /dev/xvdb  ready        0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/1b1c
 directcsi-2  /dev/xvdc  ready        0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/2d27
 directcsi-3  /dev/xvda  unavailable  0        0 B           0 B        0 B           ext4 /
 directcsi-3  /dev/xvdb  ready        0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/738f
 directcsi-3  /dev/xvdc  ready        0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/8af6
 directcsi-4  /dev/xvda  unavailable  0        0 B           0 B        0 B           ext4 /
 directcsi-4  /dev/xvdb  ready        0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/cb8b
 directcsi-4  /dev/xvdc  ready        0        10.737418 GB  0 B        10.618384 GB  xfs  /var/lib/direct-csi/mnt/f67b

(8/12) Drives available
```

 - Check if volumes are provisioned correctly

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

# Create the pods and check if PVCs are bound
$ kubectl get pvc | grep minio-data
minio-data-1-minio-0   Bound    pvc-a7c76893-f3d9-472e-9a43-ce5775df61fa   2Gi        RWO            direct-csi-min-io   13h
minio-data-1-minio-1   Bound    pvc-51232f07-30fa-48eb-ac4f-5d529f82fbcf   2Gi        RWO            direct-csi-min-io   13h
minio-data-1-minio-2   Bound    pvc-77915ec1-9c7a-4423-8879-4f4f19a12ca8   2Gi        RWO            direct-csi-min-io   13h
minio-data-1-minio-3   Bound    pvc-13b788ab-75ab-441c-a6a6-46af32929702   2Gi        RWO            direct-csi-min-io   13h
```

