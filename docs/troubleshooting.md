
Troubleshooting
-------------

### Cleaning up abandoned volumes from a "Terminating" drive

A drive with **Terminating** state indicates that an InUse drive was removed physically from the system and volumes on the drive are unreachable. If a new drive is attached as replacement of the removed drive, it is considered as a new drive and follow steps as mentioned [here](https://github.com/minio/direct-csi/blob/master/docs/cli.md#format-and-add-drives-to-directcsi) to make use of the new drive.

```sh
$ kubectl direct-csi drives list --status="terminating"
DRIVE      CAPACITY  ALLOCATED  FILESYSTEM  VOLUMES  NODE         ACCESS-TIER  STATUS        
 /dev/xvdb  8.0 GiB   1.0 GiB    xfs         2        directcsi-2  -            Terminating
```

In such cases, the corresponding volumes will be indicated as follows,

```sh
$ kubectl direct-csi volumes list --drives /dev/xvdb --nodes directcsi-2 --all
VOLUME                                    CAPACITY  NODE         DRIVE  PODNAME  PODNAMESPACE                                                                                           
 pvc-ea019d52-673b-4715-a8a4-a913dd49166d  512 MiB   directcsi-2  xvdb   minio-2  default       *[DRIVE LOST] Please refer https://github.com/minio/direct-csi/blob/master/docs/scheduling.md 
 pvc-c973cc37-01bd-4e67-abea-b1905f19fc17  512 MiB   directcsi-2  xvdb   minio-2  default       *[DRIVE LOST] Please refer https://github.com/minio/direct-csi/blob/master/docs/scheduling.md
```

To clean up the abandoned volumes and reschedule them, the respective PVCs has to be **deleted**.

```sh
$ kubectl delete pvc minio-data-3-minio-2 minio-data-1-minio-2
persistentvolumeclaim "minio-data-3-minio-2" deleted
persistentvolumeclaim "minio-data-1-minio-2" deleted  
```

The deleted PVCs will be re-created and provisions volumes successfully on the remaining "Ready" or "InUse" drives based on the requested topology specifications.
