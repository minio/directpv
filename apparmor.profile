#include <tunables/global>

profile directpv flags=(attach_disconnected,mediate_deleted) {
  #include <abstractions/base>

  network,
  file,
  unix,

  # allow all mounts
  mount,
  
  # allow all umounts
  umount,

  deny /bin/** wl,
  deny /boot/** wl,
  deny /etc/** wl,
  deny /home/** wl,
  deny /lib/** wl,
  deny /lib64/** wl,
  deny /media/** wl,
  deny /mnt/** wl,
  deny /opt/** wl,
  deny /proc/** wl,
  deny /root/** wl,
  deny /sbin/** wl,
  deny /srv/** wl,
  deny /sys/** wl,
  deny /run/udev/data/** wl,
  # deny /usr/** wl,

  # allow directpv directory to be writeable
  /var/lib/directpv/** w,
  /var/lib/kubelet/** w,
  /csi/** w,
  /sys/fs/xfs/**/error/metadata/{EIO,ENOSPC}/retry_timeout_seconds rw,

  # only a limited set of binaries can be executed
  /usr/sbin/mkfs ix,
  /usr/sbin/mkfs.xfs ix,
  /directpv ix,

  deny /bin/sh mrwklx,
  deny /bin/bash mrwklx,
  deny /bin/dash mrwklx,
  deny /usr/bin/sh mrwklx,
  deny /usr/bin/bash mrwklx,
  deny /usr/bin/dash mrwklx,

  capability sys_admin,
  capability sys_chroot,
  capability sys_resource,
  capability net_bind_service,
  capability mknod,
  capability kill,
  capability ipc_owner,
  capability fsetid,
  capability fowner,
  capability dac_override,
  capability dac_read_search,
  capability chown,
  capability lease,
  capability setgid,
  capability setuid,
  capability setfcap,

  deny @{PROC}/* w,   # deny write for all files directly in /proc (not in a subdir)
  deny @{PROC}/{[^1-9],[^1-9][^0-9],[^1-9s][^0-9y][^0-9s],[^1-9][^0-9][^0-9][^0-9]*}/** w,
  deny @{PROC}/sys/[^k]** w,  # deny /proc/sys except /proc/sys/k* (effectively /proc/sys/kernel)
  deny @{PROC}/sys/kernel/{?,??,[^s][^h][^m]**} w,  # deny everything except shm* in /proc/sys/kernel/
  deny @{PROC}/sysrq-trigger rwklx,
  deny @{PROC}/mem rwklx,
  deny @{PROC}/kmem rwklx,
  deny @{PROC}/kcore rwklx,
  deny /sys/[^f]*/** wklx,
  deny /sys/f[^s]*/** wklx,
  deny /sys/fs/[^c]*/** wklx,
  deny /sys/fs/c[^g]*/** wklx,
  deny /sys/fs/cg[^r]*/** wklx,
  deny /sys/firmware/efi/efivars/** rwklx,
  deny /sys/kernel/security/** rwklx,
}
