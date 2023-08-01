# Frequently Asked Questions
* [DirectPV installation fails in my Kubernetes. Why?](#directpv-installation-fails-in-my-kubernetes-why)
* [After upgrading DirectPV to v4.x.x, I do not find `direct-csi-min-io` storage class. Why?](#after-upgrading-directpv-to-v4xx-i-do-not-find-direct-csi-min-io-storage-class-why)
* [In the YAML output of `discover` command, I do not find my storage drive(s). Why?](#in-the-yaml-output-of-discover-command-i-do-not-find-my-storage-drives-why)
* [Do you support SAN, NAS, iSCSI, network drives etc.,?](#do-you-support-san-nas-iscsi-network-drives-etc)
* [Do you support LVM, Linux RAID, Hardware RAID, Software RAID etc.,?](#do-you-support-lvm-linux-raid-hardware-raid-software-raid-etc)
* [Is LUKS device supported?](#is-luks-device-supported)
* [I am already using Local Persistent Volumes (Local PV) for storage. Why do I need DirectPV?](#i-am-already-using-local-persistent-volumes-local-pv-for-storage-why-do-i-need-directpv)
* [I see `no drive found ...` error message in my Persistent Volume Claim. Why?](#i-see-no-drive-found--error-message-in-my-persistent-volume-claim-why)
* [I see Persistent Volume Claim is created, but respective DirectPV volume is not created. Why?](#i-see-persistent-volume-claim-is-created-but-respective-directpv-volume-is-not-created-why)
* [I see volume consuming Pod still in `Pending` state. Why?](#i-see-volume-consuming-pod-still-in-pending-state-why)
* [I see `volume XXXXX is not yet staged, but requested with YYYYY` error. Why?](#i-see-volume-xxxxx-is-not-yet-staged-but-requested-with-yyyyy-error-why)
* [I see ```unable to find device by FSUUID xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx; either device is removed or run command `sudo udevadm control --reload-rules && sudo udevadm trigger` on the host to reload``` error. Why?](#i-see-unable-to-find-device-by-fsuuid-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx-either-device-is-removed-or-run-command-sudo-udevadm-control---reload-rules--sudo-udevadm-trigger-on-the-host-to-reload-error-why)

### DirectPV installation fails in my Kubernetes. Why?
You need to have necessary privileges and permissions to perform installation. Go though the [specifications documentation](./specifications.md). For Red Hat OpenShift, refer to the [OpenShift specific documentation](./openshift.md). 

### After upgrading DirectPV to v4.x.x, I do not find `direct-csi-min-io` storage class. Why?
Legacy DirectCSI is deprecated including storage class `direct-csi-min-io` and it is no longer supported. Previously created volumes continue to work normally. For new volume requests, use the `directpv-min-io` storage class.

### In the YAML output of `discover` command, I do not find my storage drive(s). Why?
DirectPV ignores drives that meet any of the below conditions:
* The size of the drive is less than 512MiB.
* The drive is hidden.
* The drive is read-only.
* The drive is parititioned.
* The drive is held by other devices.
* The drive is mounted or in use by DirectPV already.
* The drive is in-use swap partition.
* The drive is a CDROM.

Check the last column of the `discover --all` command output to see what condition(s) exclude the drive. Resolve the conditions and try again.

### Do you support SAN, NAS, iSCSI, network drives etc.,?
DirectPV is meant for high performance local volumes with Direct Attached Storage. We do not recommend any remote drives, as remote drives may lead to poor performance.

### Do you support LVM, Linux RAID, Hardware RAID, Software RAID etc.,?
It works, but we strongly recommend to use raw devices for better performance.

### Is LUKS device supported?
Yes

### I am already using Local Persistent Volumes (Local PV) for storage. Why do I need DirectPV?
Local Persistent Volumes are statically created `PersistentVolume` which requires administrative skills. Whereas DirectPV dynamically provisions volumes on-demand; they are persistent through pod/node restarts. The lifecycle of DirectPV volumes are managd by associated Persistent Volume Claims (PVCs) which simplifies volume management.

### I see `no drive found ...` error message in my Persistent Volume Claim. Why?
Below is the reason and solution
| Reason                                                       | Solution                                           |
|:-------------------------------------------------------------|:---------------------------------------------------|
| Volume claim is made without adding any drive into DirectPV. | Please add drives.                                 |
| No drive has free space for requested size.                  | 1. Please add new drives. 2. Remove stale volumes. |
| Requested topology is not met.                               | Please modify your Persistent Volume Claim.        |
| Requested drive is not found on the requested node.          | Please modify Persistent Volume Claim.             |
| Requested node is not DirectPV node.                         | Please modify Persistent Volume Claim.             |

### I see Persistent Volume Claim is created, but respective DirectPV volume is not created. Why?
DirectPV comes with [WaitForFirstConsumer](https://kubernetes.io/docs/concepts/storage/storage-classes/#volume-binding-mode) volume binding mode i.e. Pod consuming volume must be scheduled first.

### I see volume consuming Pod still in `Pending` state. Why?
* If you haven't created the respective Persistent Volume Claim, create it.
* You may be facing Kubernetes scheduling problem. Please refer to the Kubernetes documentation [on scheduling](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/)

### I see `volume XXXXX is not yet staged, but requested with YYYYY` error. Why?
According to CSI specification, `Kubelet` should call `StageVolume` RPC first, then `PublishVolume` RPC next. In a rare event, `StageVolume` RPC is not fired/called, but `PublishVolume` RPC is called. Please restart your Kubelet and report this issue to your Kubernetes provider.

### I see ```unable to find device by FSUUID xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx; either device is removed or run command `sudo udevadm control --reload-rules && sudo udevadm trigger` on the host to reload``` error. Why?
In a rare event, `Udev` in your system missed updating `/dev` directory. Please run command `sudo udevadm control --reload-rules && sudo udevadm trigger` and report this issue to your OS vendor.
