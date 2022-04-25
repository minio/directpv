
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
- The volumes were deleted when directpv pod running in that node was down.
- etc..

Plese check `kubectl directpv volumes purge --help` for more helpers.

(NOTE: The PVs of these stale volumes should be in "released" or "failed" state in-order to purge them)

### Purging the removed / detached drive

After v3.0.0, the removed or detached InUse drives will show up in the drives list with an error message indicating that the drive is lost. In such cases, the corresponding volumes need to be purged first. ie, the corresponding PVCs has to be cleaned-up first and then the drive can be purged.

(NOTE: before deleting the lost PVCs, please [cordon](https://kubernetes.io/docs/concepts/architecture/nodes/) the node, to avoid any PVC conflicts)

Here is an example with a step-by-step procedure to handle drive replacements,

STEP 1: After detaching the InUse drive and replacing it with a fresh drive, the new drive will show up in the list as "Available"

```sh
[root@control-plane ~] kubectl directpv drives list
 DRIVE      CAPACITY  ALLOCATED  FILESYSTEM  VOLUMES  NODE                             ACCESS-TIER  STATUS
 /dev/vdb   512 MiB   83 MiB     xfs         8        control-plane.minikube.internal  -            InUse
 /dev/vdc   512 MiB   -          xfs         8        control-plane.minikube.internal  -            InUse*       drive is lost or corrupted
 /dev/vdd   512 MiB   -          -           -        control-plane.minikube.internal  -            Available
```

here the drive `/dev/vdc` is detached and new drive `/dev/vdd` is attached to the node.

STEP 2: Format the newly attached drive to make it "Ready" for workloads to utilize it

```sh
[root@control-plane ~] kubectl directpv drives format --drives /dev/vdd --nodes control-plane.minikube.internal
[root@control-plane ~] kubectl directpv drives list
 DRIVE     CAPACITY  ALLOCATED  FILESYSTEM  VOLUMES  NODE                             ACCESS-TIER  STATUS
 /dev/vdb  512 MiB   83 MiB     xfs         8        control-plane.minikube.internal  -            InUse
 /dev/vdc  512 MiB   -          xfs         8        control-plane.minikube.internal  -            InUse*  drive is lost or corrupted
 /dev/vdd  512 MiB   -          xfs         -        control-plane.minikube.internal  -            Ready
```

STEP 3: Cordon the node to stop kubernetes to schedule any workloads during the maintenance

```sh
[root@control-plane ~] kubectl cordon control-plane.minikube.internal
node/control-plane.minikube.internal cordoned
```

STEP 4: Check the lost volumes from the detached drive

```sh
[root@control-plane ~] kubectl directpv volumes list --all --drives /dev/vdc --nodes control-plane.minikube.internal --pvc
 VOLUME                                    CAPACITY  NODE                             DRIVE  PODNAME  PODNAMESPACE               PVC
 pvc-2b261763-bc5b-4a84-9d6d-33588e008dee  10 MiB    control-plane.minikube.internal  vdc    minio-2  default       *Drive Lost  minio-data-2-minio-2
 pvc-3bc3302a-2954-4f43-88f4-a081f68f9818  10 MiB    control-plane.minikube.internal  vdc    minio-3  default       *Drive Lost  minio-data-3-minio-3
 pvc-9a131013-3501-4540-bd10-7fa1a0f81bbf  10 MiB    control-plane.minikube.internal  vdc    minio-1  default       *Drive Lost  minio-data-3-minio-1
 pvc-a443e955-a77a-4b06-9649-49d7d17504cd  10 MiB    control-plane.minikube.internal  vdc    minio-3  default       *Drive Lost  minio-data-2-minio-3
 pvc-b80ea783-ba2a-44b9-8a16-77fe1f7a8537  10 MiB    control-plane.minikube.internal  vdc    minio-1  default       *Drive Lost  minio-data-2-minio-1
 pvc-b8b13ce8-10db-4274-908e-3480249a05e5  10 MiB    control-plane.minikube.internal  vdc    minio-0  default       *Drive Lost  minio-data-3-minio-0
 pvc-ba0b1c56-14cb-4a58-ba1c-4a2b9427ca18  10 MiB    control-plane.minikube.internal  vdc    minio-0  default       *Drive Lost  minio-data-1-minio-0
 pvc-e4f15027-0bc2-44c4-a271-b54141ebc42a  10 MiB    control-plane.minikube.internal  vdc    minio-2  default       *Drive Lost  minio-data-3-minio-2
```

STEP 5: Delete the corresponding pods and PVCs of the lost volumes

```sh
[root@control-plane ~] kubectl delete pods minio-0 minio-1 minio-2 minio-3 -n default
pod "minio-0" deleted
pod "minio-1" deleted
pod "minio-2" deleted
pod "minio-3" deleted
[root@control-plane ~]
```

verify the PVCs to be deleted

```sh
root@control-plane ~] kubectl directpv volumes list --lost --drives /dev/vdc --nodes control-plane.minikube.internal --pvc | awk '{print $10}' | paste -s -d " " -
 minio-data-2-minio-2 minio-data-3-minio-3 minio-data-3-minio-1 minio-data-2-minio-3 minio-data-2-minio-1 minio-data-3-minio-0 minio-data-1-minio-0 minio-data-3-minio-2
```

you can use the following one-liner to delete the lost PVCs

```sh
[root@control-plane ~] kubectl directpv volumes list --lost --drives /dev/vdc --nodes control-plane.minikube.internal --pvc | awk '{print $10}' | paste -s -d " " - | xargs kubectl delete pvc
persistentvolumeclaim "minio-data-2-minio-2" deleted
persistentvolumeclaim "minio-data-3-minio-3" deleted
persistentvolumeclaim "minio-data-3-minio-1" deleted
persistentvolumeclaim "minio-data-2-minio-3" deleted
persistentvolumeclaim "minio-data-2-minio-1" deleted
persistentvolumeclaim "minio-data-3-minio-0" deleted
persistentvolumeclaim "minio-data-1-minio-0" deleted
persistentvolumeclaim "minio-data-3-minio-2" deleted
[root@control-plane ~]
```

wait till the lost volumes are purged successfully

```sh
[root@control-plane ~] kubectl directpv drives list
 DRIVE     CAPACITY  ALLOCATED  FILESYSTEM  VOLUMES  NODE                             ACCESS-TIER  STATUS
 /dev/vdb  512 MiB   83 MiB     xfs         8        control-plane.minikube.internal  -            InUse
 /dev/vdc  512 MiB   -          xfs         -        control-plane.minikube.internal  -            Ready*  drive is lost or corrupted
 /dev/vdd  512 MiB   -          xfs         -        control-plane.minikube.internal  -            Ready
```

you can now purge the lost drive

```sh
[root@control-plane ~] kubectl-directpv drives purge --drives /dev/vdc --nodes control-plane.minikube.internal
[root@control-plane ~] kubectl directpv drives list
 DRIVE     CAPACITY  ALLOCATED  FILESYSTEM  VOLUMES  NODE                             ACCESS-TIER  STATUS
 /dev/vdb  512 MiB   83 MiB     xfs         8        control-plane.minikube.internal  -            InUse
 /dev/vdd  512 MiB   -          xfs         -        control-plane.minikube.internal  -            Ready
```

STEP 6: Uncordon the node to resume scheduling

```sh
[root@control-plane ~] kubectl uncordon control-plane.minikube.internal
node/control-plane.minikube.internal uncordoned
```

STEP 7: New PVCs will be created and volumes will be allocated on the new drive based on the more free capacity approach.

(NOTE: here, you might want to restart the "pending" pod(s) once if there is a pod-PVC race conflict)

```sh
[root@control-plane ~] kubectl get pods -o wide
NAME      READY   STATUS    RESTARTS   AGE     IP           NODE                              NOMINATED NODE   READINESS GATES
minio-0   1/1     Running   0          2m25s   172.17.0.7   control-plane.minikube.internal   <none>           <none>
minio-1   1/1     Running   0          2m17s   172.17.0.6   control-plane.minikube.internal   <none>           <none>
minio-2   1/1     Running   0          2m8s    172.17.0.8   control-plane.minikube.internal   <none>           <none>
minio-3   1/1     Running   0          117s    172.17.0.9   control-plane.minikube.internal   <none>           <none>
[root@control-plane ~] kubectl directpv drives list
 DRIVE     CAPACITY  ALLOCATED  FILESYSTEM  VOLUMES  NODE                             ACCESS-TIER  STATUS
 /dev/vdb  512 MiB   83 MiB     xfs         8        control-plane.minikube.internal  -            InUse
 /dev/vdd  512 MiB   83 MiB     xfs         8        control-plane.minikube.internal  -            InUse
```

### FS attribute mismatch errors in direct-csi pod logs

If the device FS attributes are not updated in `/run/udev/data/b<maj>:<min>` file by the udev service, the following warnings will show up in directpv pods in `direct-csi-min-io` namespace.

```log
W0615 11:17:08.484072   19851 utils.go:130] [name] ID_FS_TYPE not found in /run/udev/data/b200:2. Please refer https://github.com/minio/directpv/blob/master/docs/troubleshooting.md#troubleshooting
```

```log
W0615 11:17:08.484123   19851 utils.go:139] [name] ID_FS_UUID not found in /run/udev/data/b200:2. Please refer https://github.com/minio/directpv/blob/master/docs/troubleshooting.md#troubleshooting
```

The following command will trigger the udev service to sync the attribute values in `/run/udev/data/b<maj:min>`

```bash
sudo udevadm control --reload-rules && sudo udevadm trigger
```

(Note: Also verify if the systemd-udevd services are running on the host)

### Volume mount errors

If the respective volume mounts are gone or accidentally umounted, you may see events in corresponding pods and PVCs(If external health monitor is deployed) indicating that the volume mounts weren't mounted properly.

For example,

```log
Warning  VolumeConditionAbnormal  46s (x13 over 19m)  kubelet            Volume minio-data-1: container path is not mounted
```

```log
  Warning  VolumeConditionAbnormal  29s                csi-pv-monitor-controller-direct-csi-min-io                                                staging path is not mounted
```

In such cases, restarting the corresponding pods may re-create the missing volume mounts.
