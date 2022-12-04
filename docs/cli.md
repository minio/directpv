### Install Kubectl plugin

The `directpv` kubectl plugin can be used to manage the lifecycles of volume and drives in the kubernetes cluster

```sh
$ kubectl krew install directpv
```

### Install DirectPV

Using the kubectl plugin, install directpv driver in your kubernetes cluster

```sh
$ kubectl directpv install --help
Install directpv in k8s cluster

Usage:
  directpv install [flags]

Flags:
      --admission-control            turn on DirectPV admission controller
      --apparmor-profile string      set Apparmor profile
  -h, --help                         help for install
  -i, --image string                 DirectPV image (default "directpv:")
      --image-pull-secrets strings   image pull secrets to be set in pod specs
  -n, --node-selector strings        node selector parameters
  -g, --org string                   organization name where DirectPV images are available (default "minio")
  -r, --registry string              registry where DirectPV images are available (default "quay.io")
      --seccomp-profile string       set Seccomp profile
  -t, --tolerations strings          tolerations parameters

Global Flags:
      --dry-run             prints the installation yaml
  -k, --kubeconfig string   path to kubeconfig
      --no-headers          disables table headers
  -o, --output string       output format should be one of wide|json|yaml or empty

```

### Uninstall DirectPV

Using the kubectl plugin, uninstall directpv driver from your kubernetes cluster

```sh
Uninstall directpv in k8s cluster

Usage:
  directpv uninstall [flags]

Flags:
  -h, --help   help for uninstall

Global Flags:
      --dry-run             prints the installation yaml
  -k, --kubeconfig string   path to kubeconfig
      --no-headers          disables table headers
  -o, --output string       output format should be one of wide|json|yaml or emptympty

```

### Drives 

The `kubectl directpv drives` sub-command is used to manage the drives in the kubernetes cluster

```sh
Manage Drives in directpv cluster

Usage:
  directpv drives [command]

Aliases:
  drives, drive, dr

Available Commands:
  access-tier tag/untag directpv drives based on their access-tiers
  format      format drives in the directpv cluster
  list        list drives in the directpv cluster
  purge       purge detached|lost drives in the directpv cluster
  release     release drives from the directpv cluster

Flags:
  -h, --help   help for drives

Global Flags:
      --dry-run             prints the installation yaml
  -k, --kubeconfig string   path to kubeconfig
      --no-headers          disables table headers
  -o, --output string       output format should be one of wide|json|yaml or empty

Use "directpv drives [command] --help" for more information about a command.
```

### List drives in the cluster

```sh
list drives in the directpv cluster

Usage:
  directpv drives list [flags]

Aliases:
  list, ls

Examples:

# List all drives
$ kubectl directpv drives ls

# List all drives (including 'unavailable' drives)
$ kubectl directpv drives ls --all

# Filter all ready drives 
$ kubectl directpv drives ls --status=ready

# Filter all drives from a particular node
$ kubectl directpv drives ls --nodes=direct-1

# Combine multiple filters using multi-arg
$ kubectl directpv drives ls --nodes=direct-1 --nodes=othernode-2 --status=available

# Combine multiple filters using csv
$ kubectl directpv drives ls --nodes=direct-1,othernode-2 --status=ready

# Filter all drives based on access-tier
$ kubectl directpv drives drives ls --access-tier="hot"

# Filter all drives with access-tier being set
$ kubectl directpv drives drives ls --access-tier="*"

# Filter drives by ellipses notation for drive paths and nodes
$ kubectl directpv drives ls --drives='/dev/xvd{a...d}' --nodes='node-{1...4}'


Flags:
      --access-tier strings   match based on access-tier
  -a, --all                   list all drives (including unavailable)
  -d, --drives strings        filter by drive path(s) (also accepts ellipses range notations)
  -h, --help                  help for list
  -n, --nodes strings         filter by node name(s) (also accepts ellipses range notations)
  -s, --status strings        match based on drive status [InUse, Available, Unavailable, Ready, Terminating, Released]

Global Flags:
      --dry-run             prints the installation yaml
  -k, --kubeconfig string   path to kubeconfig
      --no-headers          disables table headers
  -o, --output string       output format should be one of wide|json|yaml or empty

```

**EXAMPLE** When directpv is first installed, the output will look something like this, with drives in `Available` status. `Unavailable` drives will be listed with `--all` flag.

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

This command will format the drives with XFS filesystem and makes them "Ready" for the workloads to schedule volumes in them.

```sh
format drives in the directpv cluster

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
$ kubectl directpv drives format --nodes 'direct-{1...3}'

# Format all drives from a particular node
$ kubectl directpv drives format --nodes=direct-1

# Format all drives based on the access-tier set [hot|cold|warm]
$ kubectl directpv drives format --access-tier=hot

# Combine multiple parameters using multi-arg
$ kubectl directpv drives format --nodes=direct-1 --nodes=othernode-2 --status=available

# Combine multiple parameters using csv
$ kubectl directpv drives format --nodes=direct-1,othernode-2 --status=available

# Combine multiple parameters using ellipses notations
$ kubectl directpv drives format --nodes "direct-{3...4}" --drives "/dev/xvd{b...f}"

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

Global Flags:
      --dry-run             prints the installation yaml
  -k, --kubeconfig string   path to kubeconfig
      --no-headers          disables table headers
  -o, --output string       output format should be one of wide|json|yaml or empty

```

**WARNING** - Adding drives to directpv will result in them being formatted

 - You can optionally select particular nodes from which the drives should be added using the `--nodes` flag
 - The drives are always formatted with `XFS` filesystem
 - If a parition table or a filesystem is already present on a drive, then `drive format` will fail 
 - You can override this behavior by setting the `--force` flag, which overwrites any parition table or filesystem present on the drive
 - Any mounted drives/parititions or having the GPT PartUUID of Boot partitions will be marked `Unavailable`. These drives cannot be added even if `--force` flag is set
 

##### Drive Status 

 | Status      | Description                                                                                                                                               |
 |-------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------|
 | Available   | These drives are available for DirectPV to use                                                                                                            |
 | Unavailable | These drives cannot be used as it does not comply with certain conditions. Please refer the drives states documentaion                                    |
 | InUse       | Drive is currently in use by a volume. i.e. number of volumes > 0                                                                                         |
 | Ready       | Drive is formatted and ready to be used, but no volumes have been assigned on this drive yet                                                              |
 | Terminating | Drive is currently being deleted (Deprecated)                                                                                                             |
 | Released    | Intermediate state when `kubectl drives release` was called on them                                                                                       |

- To know more about the drive states, Please refer [Drive States](drive-states.md)

### Tag/Untag directpv drives based on their access-tiers

```sh
tag/untag directpv drives based on their access-tiers

Usage:
  directpv drives access-tier [command]

Aliases:
  access-tier, accesstier

Available Commands:
  set         tag directpv drive(s) based on their access-tiers [hot,cold,warm]
  unset       remove the access-tier tag from the directpv drive(s)

Flags:
  -h, --help   help for access-tier

Global Flags:
      --dry-run             prints the installation yaml
  -k, --kubeconfig string   path to kubeconfig
      --no-headers          disables table headers
  -o, --output string       output format should be one of wide|json|yaml or empty

Use "directpv drives access-tier [command] --help" for more information about a command.
```

These tags can be used to control scheduling of volumes on selective drives based on the access-tier set on them. For more details, please refer [Scheduling](../scheduling.md)

### Release the "Ready" drives

This command will umount the "Ready" drives and makes them "Available"

```sh
release drives from the directpv cluster

Usage:
  directpv drives release [flags]

Examples:

 # Release all drives in the cluster
 $ kubectl directpv drives release --all
 
 # Release the 'sdf' drives in all nodes
 $ kubectl directpv drives release --drives '/dev/sdf'

 # Release the selective drives using ellipses notation for drive paths
 $ kubectl directpv drives release --drives '/dev/sd{a...z}'
 
 # Release the drives from selective nodes using ellipses notation for node names
 $ kubectl directpv drives release --nodes 'directcsi-{1...3}'
 
 # Release all drives from a particular node
 $ kubectl directpv drives release --nodes=directcsi-1
 
 # Release all drives based on the access-tier set [hot|cold|warm]
 $ kubectl directpv drives release --access-tier=hot
 
 # Combine multiple parameters using multi-arg
 $ kubectl directpv drives release --nodes=direct-1 --nodes=othernode-2 --status=available
 
 # Combine multiple parameters using csv
 $ kubectl directpv drives release --nodes=direct-1,othernode-2 --status=ready
 

Flags:
      --access-tier strings   release based on access-tier set. The possible values are [hot,cold,warm] 
  -a, --all                   release all available drives
  -d, --drives strings        filter by drive path(s) (also accepts ellipses range notations)
  -h, --help                  help for release
  -n, --nodes strings         filter by node name(s) (also accepts ellipses range notations)

Global Flags:
      --dry-run             prints the installation yaml
  -k, --kubeconfig string   path to kubeconfig
      --no-headers          disables table headers
  -o, --output string       output format should be one of wide|json|yaml or empty

```

### Volumes 

### List DirectPV volumes in the cluster

The `kubectl directpv volumes` sub-command is used to manage the volumes in the kubernetes cluster

```sh
list volumes in the directpv cluster

Usage:
  directpv volumes list [flags]

Aliases:
  list, ls

Examples:

# List all staged and published volumes
$ kubectl directpv volumes ls --status=staged,published

# List all volumes from a particular node
$ kubectl directpv volumes ls --nodes=direct-1

# Combine multiple filters using csv
$ kubectl directpv vol ls --nodes=direct-1,direct-2 --status=staged --drives=/dev/nvme0n1

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

Global Flags:
      --dry-run             prints the installation yaml
  -k, --kubeconfig string   path to kubeconfig
      --no-headers          disables table headers
  -o, --output string       output format should be one of wide|json|yaml or empty

```

### Purge volumes in the directpv cluster

The `kubectl directpv volumes purge` will purge the released or failed volumes in the cluster

```sh
purge released|failed volumes in the directpv cluster. This command has to be cautiously used as it may lead to data loss.

Usage:
  directpv volumes purge [flags]

Examples:

# Purge all released|failed volumes in the cluster
$ kubectl directpv volumes purge --all

# Purge the volume by its name(id)
$ kubectl directpv volumes purge <volume-name>

# Purge all released|failed volumes from a particular node
$ kubectl directpv volumes purge --nodes=direct-1

# Combine multiple filters using csv
$ kubectl directpv volumes purge --nodes=direct-1,direct-2 --drives=/dev/nvme0n1

# Purge all released|failed volumes by pod name
$ kubectl directpv volumes purge --pod-name=minio-{1...3}

# Purge all released|failed volumes by pod namespace
$ kubectl directpv volumes purge --pod-namespace=tenant-{1...3}

# Purge all released|failed volumes based on drive and volume ellipses
$ kubectl directpv volumes purge --drives '/dev/xvd{a...d} --nodes 'node-{1...4}''


Flags:
  -a, --all                     purge all released|failed volumes
  -d, --drives strings          filter by drive path(s) (also accepts ellipses range notations)
  -h, --help                    help for purge
  -n, --nodes strings           filter by node name(s) (also accepts ellipses range notations)
      --pod-name strings        filter by pod name(s) (also accepts ellipses range notations)
      --pod-namespace strings   filter by pod namespace(s) (also accepts ellipses range notations)

Global Flags:
      --dry-run             prints the installation yaml
  -k, --kubeconfig string   path to kubeconfig
      --no-headers          disables table headers
  -o, --output string       output format should be one of wide|json|yaml or empty

```

### Verify Installation

(Note: `minikube` was used for the following demonstration) 

 - Check if all the directpv pods are deployed correctly. i.e. they are 'Running'

```sh
$ kubectl get pods -n direct-csi-min-io
NAME                                 READY   STATUS    RESTARTS   AGE
direct-csi-min-io-5ccf67d545-8zbwp   2/2     Running   0          168m
direct-csi-min-io-5ccf67d545-95kf8   2/2     Running   0          168m
direct-csi-min-io-5ccf67d545-jvbr5   2/2     Running   0          168m
direct-csi-min-io-fktxr              4/4     Running   0          168m
```

- Check if directcsidrives and directcsivolumes CRDs are registered

```sh
$ kubectl get crd | grep directcsi
directcsidrives.direct.csi.min.io    2022-06-03T07:10:11Z
directcsivolumes.direct.csi.min.io   2022-06-03T07:10:11Z
```

 - Check if DirectPV drives are discovered


```sh
$ kubectl directpv drives list --all
 DRIVE           CAPACITY  ALLOCATED  FILESYSTEM  VOLUMES  NODE                             ACCESS-TIER  STATUS        
 /dev/dm-0       800 MiB   -          xfs         -        control-plane.minikube.internal  -            Available     
 /dev/dm-1       800 MiB   -          xfs         -        control-plane.minikube.internal  -            Available     
 /dev/dm-2       800 MiB   -          xfs         -        control-plane.minikube.internal  -            Available     
 /dev/dm-3       800 MiB   -          xfs         -        control-plane.minikube.internal  -            Available     
 /dev/nvme0n1    238 GiB   -          -           -        control-plane.minikube.internal  -            Unavailable   
 /dev/nvme0n1p1  512 MiB   -          vfat        -        control-plane.minikube.internal  -            Unavailable   
 /dev/nvme0n1p2  238 GiB   -          ext4        -        control-plane.minikube.internal  -            Unavailable   
 /dev/sda        -         -          -           -        control-plane.minikube.internal  -            Unavailable
```

- Format the drives to make them "Ready" using `kubectl directpv drives format` command

```sh
$ kubectl directpv drives list
DRIVE      CAPACITY  ALLOCATED  FILESYSTEM  VOLUMES  NODE                             ACCESS-TIER  STATUS   
 /dev/dm-0  800 MiB   -          xfs         -        control-plane.minikube.internal  -            Ready    
 /dev/dm-1  800 MiB   -          xfs         -        control-plane.minikube.internal  -            Ready    
 /dev/dm-2  800 MiB   -          xfs         -        control-plane.minikube.internal  -            Ready    
 /dev/dm-3  800 MiB   -          xfs         -        control-plane.minikube.internal  -            Ready
```

- Deploy a workload using "directpv-min-io" as the storageClassName in the volumeClaimTemplate 

Please refer `minio.yaml` in the project root directory for a sample MinIO deployment yaml using `directpv-min-io` storage class.

```sh
# Create a volumeClaimTemplate that refers to directpv-min-io storageClass
$ cat minio.yaml | grep -B 10 directpv-min-io
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 100Mi    
      storageClassName: directpv-min-io # This field references the existing StorageClass
  - metadata:
      name: minio-data-2
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 100Mi  
      storageClassName: directpv-min-io # This field references the existing StorageClass
  - metadata:
      name: minio-data-3
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 100Mi
      storageClassName: directpv-min-io # This field references the existing StorageClass
  - metadata:
      name: minio-data-4
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 100Mi 
      storageClassName: directpv-min-io # This field references the existing StorageClass

```

Apply the yaml to deploy the workload

 - Check if the pods are up and the volumes are being provisioned correctly

```
$ kubectl get pvc 
NAME                   STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS      AGE
minio-data-1-minio-0   Bound    pvc-3a3a0a81-20ae-49f1-bb10-ace2b88e4df9   100Mi      RWO            directpv-min-io   25s
minio-data-2-minio-0   Bound    pvc-427ec139-6c39-4e88-9e98-15d99d08b00c   100Mi      RWO            directpv-min-io   25s
minio-data-3-minio-0   Bound    pvc-e6853d87-216e-4f90-b2a8-23afa4c77bc0   100Mi      RWO            directpv-min-io   25s
minio-data-4-minio-0   Bound    pvc-c8de2f50-99d6-4ae6-a2c2-de23328e1358   100Mi      RWO            directpv-min-io   25s

# List the volumes to see if they are provisioned
$ kubectl directpv volumes ls
  VOLUME                                    CAPACITY  NODE                             DRIVE  PODNAME  PODNAMESPACE   
 pvc-1bfc8b70-4a88-47ca-a093-bff4721df7b7  100 MiB   control-plane.minikube.internal  dm-3   minio-2  default        
 pvc-3a3a0a81-20ae-49f1-bb10-ace2b88e4df9  100 MiB   control-plane.minikube.internal  dm-1   minio-0  default        
 pvc-427ec139-6c39-4e88-9e98-15d99d08b00c  100 MiB   control-plane.minikube.internal  dm-3   minio-0  default        
 pvc-66d32683-b41b-4cac-9894-fde96aa21b6b  100 MiB   control-plane.minikube.internal  dm-1   minio-1  default        
 pvc-69742f84-bdbd-4e29-bd3e-10e9cafeedee  100 MiB   control-plane.minikube.internal  dm-3   minio-1  default        
 pvc-7c5e4220-8003-4cdb-b92c-4eddef5944ae  100 MiB   control-plane.minikube.internal  dm-2   minio-3  default        
 pvc-a27fc1dc-0ad6-4092-b56a-3a8ff745d763  100 MiB   control-plane.minikube.internal  dm-1   minio-2  default        
 pvc-b11357e3-0be7-4ccf-ac8c-e9cd1cfa369a  100 MiB   control-plane.minikube.internal  dm-2   minio-2  default        
 pvc-b91651ec-b765-4396-8228-36182bec54f5  100 MiB   control-plane.minikube.internal  dm-3   minio-3  default        
 pvc-bd2c7951-3982-433b-af10-24caa89c2cdf  100 MiB   control-plane.minikube.internal  dm-2   minio-1  default        
 pvc-c8de2f50-99d6-4ae6-a2c2-de23328e1358  100 MiB   control-plane.minikube.internal  dm-0   minio-0  default        
 pvc-d4191545-ed06-4077-963a-f88eebc07ece  100 MiB   control-plane.minikube.internal  dm-0   minio-1  default        
 pvc-e6853d87-216e-4f90-b2a8-23afa4c77bc0  100 MiB   control-plane.minikube.internal  dm-2   minio-0  default        
 pvc-ea980526-22da-45c6-ad77-2a5e3f9d1028  100 MiB   control-plane.minikube.internal  dm-0   minio-3  default        
 pvc-eb694cdf-8407-4687-acea-c307f2ab4f77  100 MiB   control-plane.minikube.internal  dm-1   minio-3  default        
 pvc-fad3002a-60a4-42c9-93a9-f5e59e845b1b  100 MiB   control-plane.minikube.internal  dm-0   minio-2  default     
```
