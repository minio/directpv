
Troubleshooting
-------------

### Cleaning up abandoned volumes from a "Terminating" drive

(NOTE: "Terminating" state is deprecated in versions > v3.0.0)

A drive with **Terminating** state indicates that an InUse drive was removed physically from the system and volumes on the drive are unreachable. If a new drive is attached as replacement of the removed drive, it is considered as a new drive and follow steps as mentioned [here](https://github.com/minio/directpv/blob/master/docs/cli.md#format-and-add-drives-to-directpv) to make use of the new drive.

```sh
$ kubectl directpv drives list --status="terminating"
DRIVE      CAPACITY  ALLOCATED  FILESYSTEM  VOLUMES  NODE         ACCESS-TIER  STATUS        
 /dev/xvdb  8.0 GiB   1.0 GiB    xfs         2        directpv-2  -            Terminating
```

In such cases, the corresponding volumes will be indicated as follows,

```sh
$ kubectl directpv volumes list --drives /dev/xvdb --nodes directpv-2 --all
VOLUME                                    CAPACITY  NODE         DRIVE  PODNAME  PODNAMESPACE                                                                                           
 pvc-ea019d52-673b-4715-a8a4-a913dd49166d  512 MiB   directpv-2  xvdb   minio-2  default       *[DRIVE LOST] Please refer https://github.com/minio/directpv/blob/master/docs/troubleshooting.md
 pvc-c973cc37-01bd-4e67-abea-b1905f19fc17  512 MiB   directpv-2  xvdb   minio-2  default       *[DRIVE LOST] Please refer https://github.com/minio/directpv/blob/master/docs/troubleshooting.md
```

To clean up the abandoned volumes and reschedule them, the respective PVCs has to be **deleted**.

```sh
$ kubectl delete pvc minio-data-3-minio-2 minio-data-1-minio-2
persistentvolumeclaim "minio-data-3-minio-2" deleted
persistentvolumeclaim "minio-data-1-minio-2" deleted  
```

The deleted PVCs will be re-created and provisions volumes successfully on the remaining "Ready" or "InUse" drives based on the requested topology specifications.

### Purging the released or failed volumes

`kubectl directpv volumes purge` command can be used to purge the lost, failed or released volumes in the cluster. This command should be used for special cases like one of the following

- When the pods and corresponding PVCs were force deleted. Force deletion might skip few necessary volume cleanups and make them stale.
- When the corresponding drive is removed or detached from the cluster. `kubectl directpv volumes list` would indicate such lost volumes with an error tag.
- The volumes were deleted when directpv pod running in that node was node.
- etc..

Plese check `kubectl directpv volumes purge --help` for more helpers.

(NOTE: The PVs of these stale volumes should be in "released" or "failed" state in-order to purge them)

### Purging the removed / detached drive

After v3.0.0, the removed or detached drive will show up in the drives list with an error message indicating that the drive is removed. If the drive is in InUse state, the corresponding volumes need to be purged first. ie, the corresponding PVCs has to be cleaned-up first.

(NOTE: before deleting the lost PVCs, please [cordon](https://kubernetes.io/docs/concepts/architecture/nodes/) the node, to avoid any PVC conflicts)
