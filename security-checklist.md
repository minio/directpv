Security Checklist
-------------------

DirectCSI runs with elevated privileges, which are needed for reading block devices and for mount/unmount operations. Here are the list of syscalls employed by DirectCSI. 

### Syscalls

| Syscall     | Linux Capability needed     |
|-------------|-----------------------------|
| mount       | CAP_SYS_ADMIN               |
| umount      | CAP_SYS_ADMIN               |

In highly secure environments - seccomp, apparmor or selinux can be used to run DirectCSI without providing unbounded privileged access. More information about it can be found [here](#security-enforcement) 


### File Permissions

Here are the list of file permissions needed on the host OS for DirectCSI

| Path             | Permissions   |   Notes                                                                                   |
|------------------|---------------|-------------------------------------------------------------------------------------------|
| /sys/devices     | O_RDONLY      |  For probing block devices on the host                                                    |
| /var/lib         | O_RDWR        |  For creating DirectCSI work directory - /var/lib/direct-csi                              |
| /var/lib/kubelet | O_WRONLY      |  For creating volume mounts that will subsequently be served to application containers    |


### External Binary Execution

Inside the container, DirectCSI execs the following utilities for formatting drives and setting up storage quotas. No files on the hosts are executed by DirectCSI

| Exec Path                 | Notes                           |
----------------------------|---------------------------------|
| /usr/sbin/mkfs.xfs        | For formatting drives           |
| /usr/sbin/xfs_quota       | For setting up storage quota    |


## Security Enforcement
------------------------

 - [SECCOMP profile](./seccomp.json) below can be used to tightly restrict the access DirectCSI has over the host OS.
 - [AppArmor Profile](./apparmor.profile) is coming soon!
 - [SELinux Policy](./selinux.policy) is coming soon!


