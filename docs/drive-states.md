Drive States (valid for versions <= v3.2.2)
-------------

![states](https://user-images.githubusercontent.com/5410427/169809465-f25d4714-e360-409c-b286-b8a1a6e31b9e.jpg)

This documentation explains the drive states and its transitions. The drive states are as follows

- Unavailable
- Available
- Ready
- InUse
- Released
- Terminating (Deprecated)

### Available

An "Available" drive should satisfy the following conditions

- The drive shouldn't be mounted
- The drive size should not be less than 16 MiB
- The drive should not be a swap drive
- The drive should not be a hidden drive
- The drive should not be a read-only drive
- The drive should not be a "removable" drive
- The drive shouldn't be a partition-parent (parent)
- The drive cannot have any holders

An availabe drive can become unavailable if any of the above conditions are reversed.

### Unavailable

If a drive doesn't comply with the above availability conditions, it is considered to be "Unavailable". An unavailable drive can be made available if actions are taken to satisfy the above conditions.

### Ready

An available drive can be made "Ready" if `kubectl directpv drives format` was called on it. This will format and mount the drive to make it Ready for the workloads to use it.

A "Ready" drive can be made "Available" by releasing it. `kubectl directpv drives release` is the command to release the "Ready" drives. This will umount the drives and release them from directpv.

### InUse

An "InUse" drive indicates that the drive holds volumes and is currently being used by the workload. `kubectl directpv volumes list --drives /dev/<path>` will list the volumes scheduled on this drive. Deleting all the corresponding PVCs will clean-up the volumes and make the drive back to "Ready" state. Please note that deleting the PVCs will lead to "data loss" and the volumes will be umounted from the drive.

### Released

"Released" is an intermediate state which is set on the drive when `kubectl direcpv drives release` was called on it. This will release the drive by umounting the drive mount and making the drive "Available" again.

### Terminating

This drive state is deprecated in v3.0.0, this state indicates that an "InUse" drive is detached from the node and requires some handling to cleanup the volumes present in it. The cleanup steps are explained in [Troubleshooting](./troubleshooting.md).

After v3.0.0, a detached drive will be displayed in `kubectl directpv drives list` with an error message tied to it indicating that the drive is lost.
