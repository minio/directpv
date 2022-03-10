Security Checklist
-------------------

DirectCSI runs with elevated privileges, which are needed for reading block devices and for mount/unmount operations. Here are the list of syscalls employed by DirectCSI.

### Syscalls

| Syscall     | Linux Capability needed     | PodSecurityPolicy    | Compensating Enforcement  |
|-------------|-----------------------------|----------------------|---------------------------|
| mount       | CAP_SYS_ADMIN               | privileged: true     | Seccomp & Apparmor        |
| umount      | CAP_SYS_ADMIN               | privileged: true     | Seccomp & Apparmor        |

The [Apparmor profile](./apparmor.profile) restricts mounts/unmounts to directories specified [here](#file-permissions). In addition, it prevents execution of all binaries in DirectPV pods except ones listed [here](#external-binary-execution).

The [Seccomp profile](./seccomp.json) restricts syscalls to the minimum required by DirectPV.

### Host Access

DirectCSI needs the following host access

| Path       | Notes                                                        |
|------------|--------------------------------------------------------------|
| HostPID    | This is required to detect mount status of managed drives    |

### File Permissions

Here are the list of file permissions needed from the host

| Path                 | Permissions   | Mount Propagation    | Notes                                             |
|----------------------|---------------|----------------------|---------------------------------------------------|
| /sys/                | O_RDONLY      | None                 | For probing block devices on the host             |
| /var/lib/kubelet     | O_WRONLY      | Bidirectional        | Serving directory for volumes destined into pods  |
| /var/lib/direct-csi  | O_RDWR        | Bidirectional        | Working directory for creating devices and mounts |

### External Binary Execution

Inside the container, DirectCSI executes the following utilities for formatting drives and setting up storage quotas. No files on the hosts are executed by DirectCSI

| Exec Path                 | Notes                                            |
|---------------------------|--------------------------------------------------|
| /usr/sbin/mkfs.xfs        | For formatting drives                            |
| /usr/sbin/xfs_quota       | For setting up storage quota                     |

## Security Enforcement
------------------------

 - [Seccomp profile](./seccomp.json) can be used to tightly restrict the access DirectCSI has over the host OS.
 - [AppArmor Profile](./apparmor.profile) can be used to restrict the files and directories that are available to direct-csi.
