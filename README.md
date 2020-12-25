# Direct CSI 

![Go](https://github.com/minio/direct-csi/workflows/Go/badge.svg)

Direct CSI is a driver to allocate volumes for pods that require _direct access_ to storage media (eg. MinIO). It maintains a global view of storage in the cluster, and directs pods to run on nodes where volume is provisioned. Each volume is a subdirectory carved out of available drives on the node and mounted into pods.

## Installation

###### Install Direct-CSI cli

```bash
kubectl krew install directcsi
```

###### Install Direct-CSI driver

```bash
kubectl directcsi install --crd 
```

###### Add Drives to DirectCSI pool

Choose drives to be managed by DirectCSI. Refer to [Add Drives](#add-drives) command for more info.

```bash
kubectl directcsi drives add /dev/nvme* --nodes myhost{1...4}
```

## Make a Persistent Volume Claim

Provision a Direct-CSI volume by specifying `volumeClaimTemplates`:

###### Example

```yaml
volumeClaimTemplates:
  - metadata:
    name: myvol1
  spec:
    accessModes: [ "ReadWriteOnce" ]
    resources:
      requests:
        storage: 500G
    storageClassName: direct.csi.min.io
```

## Direct-CSI Command Line Reference

##### Get Info

Show storage summary of the nodes managed by DirectCSI.
```bash
Usage:
  kubectl directcsi info
  
NODENAME       DRIVES
rack1node1     (4/5)
``` 

##### List Drives

List drive status across the storage nodes.

```bash
Usage:
  kubectl directcsi drives list [FLAGS] [DRIVE WILDCARD,...]

[FLAGS]
  --nodes, -n  VALUE      drives from nodes whose name matches WILDCARD. Defaults to '*'
  --all                   list all drives
  --status, -s VALUE      filter by status [in-use, unformatted, new, terminating, unavailable, ready]
```

###### Example


```
# list nvme drives on nodes in rack1 and rack2
$> kubectl directcsi drives list --nodes 'rack1*' '/dev/nvme*' --all
DRIVES                      STATUS      VOLUMES  ALLOCATED      CAPACITY     FREE          FS         MOUNT           MODEL
rack1node1:/dev/nvme1n1     in-use      4        376 GiB        1 TiB        36 GiB        xfs        (internal)      WDC PC SN730 SDBQNTY-986G-2001
rack1node1:/dev/nvme2n1     new         0        0              1 TiB        986 GiB       -          -               WDC PC SN730 SDBQNTY-986G-2001
rack1node2:/dev/nvme1n1     ignore      0        0              1 TiB        986 GiB       xfs        /mnt/dat...     WDC PC SN730 SDBQNTY-986G-2001
rack1node2:/dev/nvme2n1     new         0        0              1 TiB        986 GiB       xfs        -               WDC PC SN730 SDBQNTY-986G-2001
rack1node2:/dev/nvme3n1     offline     14       986            1 TiB        14 GiB        ext4       -               WDC PC SN730 SDBQNTY-986G-2001
```

##### Add Drives

Choose drives to be managed by DirectCSI. Only new drives are allowed.
```bash
Usage:
  kubectl directcsi drives add [FLAGS] [DRIVE WILDCARD,...]

[FLAGS]
  --nodes, -n  VALUE      drives from nodes whose name matches WILDCARD. Defaults to '*'
  --fs, -f  VALUE         filesystem to be formatted. Defaults to 'xfs'
  --force                 overwrite existing filesystem
```

##### Remove Drives

Remove drives from being managed by DirectCSI. Only works on drives that have no bounded volumes.
```bash
Usage:
  kubectl directcsi drives remove [FLAGS] [DRIVE WILDCARD,...]

[FLAGS]
  --nodes, -n  VALUE      drives from nodes whose name matches WILDCARD. Defaults to '*'
```

##### Ignore Drives

Ignore drives from being managed by DirectCSI. Only works on drives that have no bounded volumes.
```bash
Usage:
  kubectl directcsi drives ignore [FLAGS] [DRIVE WILDCARD,...]

[FLAGS]
  --nodes, -n  VALUE      drives from nodes whose name matches WILDCARD. Defaults to '*'
```

##### List Volumes

List all the provisioned volumes
```bash
Usage:
  kubectl directcsi volumes list --drives [DRIVE_WILDCARD,...] --nodes [NODE_NAME,...]

[FLAGS]
  --drives, -d   VALUE     list volumes provisioned from particular drive. Defaults to all
  --nodes, -n  VALUE       list volumes provisioned from drives on particular node. Defaults to all
  --verbose, -v            show detailed volume information 
```

###### Example

```
# list volumes on nvme drives in rack
$> kubectl directcsi volumes list --nodes 'rack1*' --drives '/dev/nvme*'   
VOLUME        NODENAME            DRIVE                CAPACITY     STATUS   
pvc-uuid      rack1node1          /dev/nvme1n1         500 GiB      Bound
pvc-uuid      rack1node2          /dev/nvme1n1         100 GiB      Released
```

##### Purge Volume

Permanently delete the volume and all of its contents.

```bash
Usage:
  kubectl directcsi volumes purge VOLUME
```

## License

Use of `directcsi` driver is governed by the GNU AGPLv3 license that can be found in the [LICENSE](./LICENSE) file.
