// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package mount

import (
	"reflect"
	"testing"
)

func testProbeCast1Result() map[string][]MountInfo {
	return map[string][]MountInfo{
		"0:12": {
			{MajorMinor: "0:12", MountPoint: "/sys/kernel/tracing", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tracefs"},
		},
		"0:20": {
			{MajorMinor: "0:20", MountPoint: "/dev/mqueue", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "mqueue"},
		},
		"0:21": {
			{MajorMinor: "0:21", MountPoint: "/sys/fs/selinux", MountOptions: []string{"noexec", "nosuid", "relatime", "rw"}, fsType: "selinuxfs"},
		},
		"0:22": {
			{MajorMinor: "0:22", MountPoint: "/sys", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "sysfs"},
		},
		"0:23": {
			{MajorMinor: "0:23", MountPoint: "/proc", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "proc"},
		},
		"0:24": {
			{MajorMinor: "0:24", MountPoint: "/dev/shm", MountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:25": {
			{MajorMinor: "0:25", MountPoint: "/dev/pts", MountOptions: []string{"noexec", "nosuid", "relatime", "rw"}, fsType: "devpts"},
		},
		"0:26": {
			{MajorMinor: "0:26", MountPoint: "/run", MountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:27": {
			{MajorMinor: "0:27", MountPoint: "/sys/fs/cgroup", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup2"},
		},
		"0:28": {
			{MajorMinor: "0:28", MountPoint: "/sys/fs/pstore", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "pstore"},
		},
		"0:29": {
			{MajorMinor: "0:29", MountPoint: "/sys/firmware/efi/efivars", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "efivarfs"},
		},
		"0:30": {
			{MajorMinor: "0:30", MountPoint: "/sys/fs/bpf", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "bpf"},
		},
		"0:33": {
			{MajorMinor: "0:33", MountPoint: "/proc/sys/fs/binfmt_misc", MountOptions: []string{"relatime", "rw"}, fsType: "autofs"},
		},
		"0:34": {
			{MajorMinor: "0:34", MountPoint: "/dev/hugepages", MountOptions: []string{"relatime", "rw"}, fsType: "hugetlbfs"},
		},
		"0:35": {
			{MajorMinor: "0:35", MountPoint: "/sys/fs/fuse/connections", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "fusectl"},
		},
		"0:36": {
			{MajorMinor: "0:36", MountPoint: "/sys/kernel/config", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "configfs"},
		},
		"0:37": {
			{MajorMinor: "0:37", MountPoint: "/tmp", MountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:39": {
			{MajorMinor: "0:39", MountPoint: "/var/lib/nfs/rpc_pipefs", MountOptions: []string{"relatime", "rw"}, fsType: "rpc_pipefs"},
		},
		"0:45": {
			{MajorMinor: "0:45", MountPoint: "/run/user/1000", MountOptions: []string{"nodev", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:46": {
			{MajorMinor: "0:46", MountPoint: "/run/user/1000/gvfs", MountOptions: []string{"nodev", "nosuid", "relatime", "rw"}, fsType: "fuse", fsSubType: "gvfsd-fuse"},
		},
		"0:5": {
			{MajorMinor: "0:5", MountPoint: "/dev", MountOptions: []string{"noexec", "nosuid", "rw"}, fsType: "devtmpfs"},
		},
		"0:6": {
			{MajorMinor: "0:6", MountPoint: "/sys/kernel/security", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "securityfs"},
		},
		"0:7": {
			{MajorMinor: "0:7", MountPoint: "/sys/kernel/debug", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "debugfs"},
		},
		"259:1": {
			{MajorMinor: "259:1", MountPoint: "/boot/efi", MountOptions: []string{"relatime", "rw"}, fsType: "vfat"},
		},
		"259:2": {
			{MajorMinor: "259:2", MountPoint: "/boot", MountOptions: []string{"relatime", "rw"}, fsType: "ext4"},
		},
		"259:3": {
			{MajorMinor: "259:3", MountPoint: "/", MountOptions: []string{"relatime", "rw"}, fsType: "ext4"},
		},
		"259:4": {
			{MajorMinor: "259:4", MountPoint: "/home", MountOptions: []string{"relatime", "rw"}, fsType: "ext4"},
		},
	}
}

func testProbeCast2Result() map[string][]MountInfo {
	return map[string][]MountInfo{
		"0:11": {
			{MajorMinor: "0:11", MountPoint: "/sys/kernel/tracing", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tracefs"},
		},
		"0:18": {
			{MajorMinor: "0:18", MountPoint: "/dev/mqueue", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "mqueue"},
		},
		"0:19": {
			{MajorMinor: "0:19", MountPoint: "/sys/fs/selinux", MountOptions: []string{"relatime", "rw"}, fsType: "selinuxfs"},
		},
		"0:20": {
			{MajorMinor: "0:20", MountPoint: "/sys/kernel/config", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "configfs"},
		},
		"0:21": {
			{MajorMinor: "0:21", MountPoint: "/sys", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "sysfs"},
		},
		"0:22": {
			{MajorMinor: "0:22", MountPoint: "/proc", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "proc"},
		},
		"0:23": {
			{MajorMinor: "0:23", MountPoint: "/dev/shm", MountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:24": {
			{MajorMinor: "0:24", MountPoint: "/dev/pts", MountOptions: []string{"noexec", "nosuid", "relatime", "rw"}, fsType: "devpts"},
		},
		"0:25": {
			{MajorMinor: "0:25", MountPoint: "/run", MountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
			{MajorMinor: "0:25", MountPoint: "/run/snapd/ns", MountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:26": {
			{MajorMinor: "0:26", MountPoint: "/run/lock", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:27": {
			{MajorMinor: "0:27", MountPoint: "/sys/fs/cgroup", MountOptions: []string{"nodev", "noexec", "nosuid", "ro"}, fsType: "tmpfs"},
		},
		"0:28": {
			{MajorMinor: "0:28", MountPoint: "/sys/fs/cgroup/unified", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup2"},
		},
		"0:29": {
			{MajorMinor: "0:29", MountPoint: "/sys/fs/cgroup/systemd", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:30": {
			{MajorMinor: "0:30", MountPoint: "/sys/fs/pstore", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "pstore"},
		},
		"0:31": {
			{MajorMinor: "0:31", MountPoint: "/sys/fs/bpf", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "bpf"},
		},
		"0:32": {
			{MajorMinor: "0:32", MountPoint: "/sys/fs/cgroup/net_cls,net_prio", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:33": {
			{MajorMinor: "0:33", MountPoint: "/sys/fs/cgroup/cpuset", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:34": {
			{MajorMinor: "0:34", MountPoint: "/sys/fs/cgroup/cpu,cpuacct", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:35": {
			{MajorMinor: "0:35", MountPoint: "/sys/fs/cgroup/perf_event", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:36": {
			{MajorMinor: "0:36", MountPoint: "/sys/fs/cgroup/pids", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:37": {
			{MajorMinor: "0:37", MountPoint: "/sys/fs/cgroup/rdma", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:38": {
			{MajorMinor: "0:38", MountPoint: "/sys/fs/cgroup/blkio", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:39": {
			{MajorMinor: "0:39", MountPoint: "/sys/fs/cgroup/devices", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:4": {
			{MajorMinor: "0:4", MountPoint: "/run/snapd/ns/lxd.mnt", MountOptions: []string{"rw"}, fsType: "nsfs"},
			{MajorMinor: "0:4", MountPoint: "/run/netns/cni-824a6a31-7b0f-585f-3a6a-5e35af9205c2", MountOptions: []string{"rw"}, fsType: "nsfs"},
		},
		"0:40": {
			{MajorMinor: "0:40", MountPoint: "/sys/fs/cgroup/memory", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:41": {
			{MajorMinor: "0:41", MountPoint: "/sys/fs/cgroup/hugetlb", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:42": {
			{MajorMinor: "0:42", MountPoint: "/sys/fs/cgroup/freezer", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:43": {
			{MajorMinor: "0:43", MountPoint: "/proc/sys/fs/binfmt_misc", MountOptions: []string{"relatime", "rw"}, fsType: "autofs"},
		},
		"0:44": {
			{MajorMinor: "0:44", MountPoint: "/dev/hugepages", MountOptions: []string{"relatime", "rw"}, fsType: "hugetlbfs"},
		},
		"0:45": {
			{MajorMinor: "0:45", MountPoint: "/sys/fs/fuse/connections", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "fusectl"},
		},
		"0:48": {
			{MajorMinor: "0:48", MountPoint: "/run/user/1000", MountOptions: []string{"nodev", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:5": {
			{MajorMinor: "0:5", MountPoint: "/dev", MountOptions: []string{"relatime", "rw"}, fsType: "devtmpfs"},
		},
		"0:50": {
			{MajorMinor: "0:50", MountPoint: "/var/lib/kubelet/pods/54589c77-2df4-43e7-a1df-cf90d87c1107/volumes/kubernetes.io~projected/kube-api-access-nqwj8", MountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:51": {
			{MajorMinor: "0:51", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/f9e55b5765965539bc04917b21c3cdc9c32ab8055f0e17fb84c19441d0818451/shm", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:52": {
			{MajorMinor: "0:52", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/f9e55b5765965539bc04917b21c3cdc9c32ab8055f0e17fb84c19441d0818451/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:6": {
			{MajorMinor: "0:6", MountPoint: "/sys/kernel/security", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "securityfs"},
		},
		"0:63": {
			{MajorMinor: "0:63", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/ce0c9df77871f26de88ae3382784f1d5eece115be98cd16e60bbdebbf338d899/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:7": {
			{MajorMinor: "0:7", MountPoint: "/sys/kernel/debug", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "debugfs"},
		},
		"0:72": {
			{MajorMinor: "0:72", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/ad498073d40f2b93f66ea5b2c3a1b93ec736cf7e3558437a0713a0f6be13590e/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"202:1": {
			{MajorMinor: "202:1", MountPoint: "/", MountOptions: []string{"relatime", "rw"}, fsType: "ext4"},
		},
		"202:80": {
			{MajorMinor: "202:80", MountPoint: "/var/lib/direct-csi/mnt/9d619667-0286-436e-ba21-ba4b5870cc39", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"202:96": {
			{MajorMinor: "202:96", MountPoint: "/var/lib/direct-csi/mnt/ccf1082d-21ee-42ac-a480-7a00933e169d", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"253:0": {
			{MajorMinor: "253:0", MountPoint: "/var/lib/direct-csi/mnt/e5cd069d-678b-4024-acc8-039c62379323", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"7:0": {
			{MajorMinor: "7:0", MountPoint: "/snap/amazon-ssm-agent/3552", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:1": {
			{MajorMinor: "7:1", MountPoint: "/snap/core18/1997", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:2": {
			{MajorMinor: "7:2", MountPoint: "/snap/core18/2066", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:3": {
			{MajorMinor: "7:3", MountPoint: "/snap/lxd/20326", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:4": {
			{MajorMinor: "7:4", MountPoint: "/snap/lxd/19647", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:5": {
			{MajorMinor: "7:5", MountPoint: "/snap/snapd/12159", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:6": {
			{MajorMinor: "7:6", MountPoint: "/snap/snapd/12057", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"}},
	}
}

func testProbeCast3Result() map[string][]MountInfo {
	return map[string][]MountInfo{
		"0:101": {
			{MajorMinor: "0:101", MountPoint: "/var/lib/kubelet/pods/3594fe81-84c4-415c-9b28-f9bdb136e0c0/volumes/kubernetes.io~secret/conversion-webhook-certs", MountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:102": {
			{MajorMinor: "0:102", MountPoint: "/var/lib/kubelet/pods/3594fe81-84c4-415c-9b28-f9bdb136e0c0/volumes/kubernetes.io~projected/kube-api-access-5g9f8", MountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:103": {
			{MajorMinor: "0:103", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/7b363bb8ecce707382129064a22fa327e1662129dbac50d240c44f6c4b6a77e3/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:11": {
			{MajorMinor: "0:11", MountPoint: "/sys/kernel/tracing", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tracefs"},
		},
		"0:113": {
			{MajorMinor: "0:113", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/6c7918a2a63b270069791a9b55385c4931067e1eab47e1284056148f0c43b549/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:122": {
			{MajorMinor: "0:122", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/32f034e0946e06e0f2b575644b313679f53cb50240c908a6b9ef874a098c5a77/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:128": {
			{MajorMinor: "0:128", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/75383da21ef3dcb1eb120d5e434b7a59927d05b6e8afe97872d450b25cb313c8/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:137": {
			{MajorMinor: "0:137", MountPoint: "/var/lib/kubelet/pods/f61e15e5-8a08-476a-9104-e0a93f311243/volumes/kubernetes.io~secret/conversion-webhook-certs", MountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:138": {
			{MajorMinor: "0:138", MountPoint: "/var/lib/kubelet/pods/f61e15e5-8a08-476a-9104-e0a93f311243/volumes/kubernetes.io~projected/kube-api-access-xnrhw", MountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:139": {
			{MajorMinor: "0:139", MountPoint: "/var/lib/kubelet/pods/f61e15e5-8a08-476a-9104-e0a93f311243/volumes/kubernetes.io~secret/admission-webhook-certs", MountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:140": {
			{MajorMinor: "0:140", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/d636b609e740a50a2603874c843bc8f5b70d78f91bb5dfafbab0e5e39681e7b9/shm", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:141": {
			{MajorMinor: "0:141", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/d636b609e740a50a2603874c843bc8f5b70d78f91bb5dfafbab0e5e39681e7b9/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:152": {
			{MajorMinor: "0:152", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/8c5f67a5933accd4df38393c83b953de9214ba85d374b83b92cee5784e97cc8d/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:158": {
			{MajorMinor: "0:158", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/35c7f15e24b076a9930d51ffdf3c1cc0a6abbf13b2874e732508805df1967cf3/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:164": {
			{MajorMinor: "0:164", MountPoint: "/var/lib/kubelet/pods/e9d36191-301a-4a72-b061-8ce0bb0d6fcf/volumes/kubernetes.io~projected/kube-api-access-dv9jq", MountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:165": {
			{MajorMinor: "0:165", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/71b7b01fc3163bc784630cb90c33cae4c9de73fcadff4f108899ae4ea8643b9c/shm", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:166": {
			{MajorMinor: "0:166", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/71b7b01fc3163bc784630cb90c33cae4c9de73fcadff4f108899ae4ea8643b9c/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:177": {
			{MajorMinor: "0:177", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/987743f71c3a50963e3a4c9a025bd35d9351a0331c8f9a174748494110a8fe67/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:18": {
			{MajorMinor: "0:18", MountPoint: "/dev/mqueue", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "mqueue"},
		},
		"0:186": {
			{MajorMinor: "0:186", MountPoint: "/var/lib/kubelet/pods/558a704e-918b-4562-8426-dc27deba229d/volumes/kubernetes.io~projected/kube-api-access-st8md", MountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:187": {
			{MajorMinor: "0:187", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/f458def070643977236c7f8a9f71489278d3c999171e1289399aec9fe4d27875/shm", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:188": {
			{MajorMinor: "0:188", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/f458def070643977236c7f8a9f71489278d3c999171e1289399aec9fe4d27875/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:19": {
			{MajorMinor: "0:19", MountPoint: "/sys/fs/selinux", MountOptions: []string{"relatime", "rw"}, fsType: "selinuxfs"},
		},
		"0:199": {
			{MajorMinor: "0:199", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/7548d31ea3936cff4c369f2f9578208fd1965f1187ae437f3def2db7951bd8fb/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:20": {
			{MajorMinor: "0:20", MountPoint: "/sys/kernel/config", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "configfs"},
		},
		"0:21": {
			{MajorMinor: "0:21", MountPoint: "/sys", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "sysfs"},
		},
		"0:22": {
			{MajorMinor: "0:22", MountPoint: "/proc", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "proc"},
		},
		"0:23": {
			{MajorMinor: "0:23", MountPoint: "/dev/shm", MountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:24": {
			{MajorMinor: "0:24", MountPoint: "/dev/pts", MountOptions: []string{"noexec", "nosuid", "relatime", "rw"}, fsType: "devpts"},
		},
		"0:25": {
			{MajorMinor: "0:25", MountPoint: "/run", MountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
			{MajorMinor: "0:25", MountPoint: "/run/snapd/ns", MountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:26": {
			{MajorMinor: "0:26", MountPoint: "/run/lock", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:27": {
			{MajorMinor: "0:27", MountPoint: "/sys/fs/cgroup", MountOptions: []string{"nodev", "noexec", "nosuid", "ro"}, fsType: "tmpfs"},
		},
		"0:28": {
			{MajorMinor: "0:28", MountPoint: "/sys/fs/cgroup/unified", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup2"},
		},
		"0:29": {
			{MajorMinor: "0:29", MountPoint: "/sys/fs/cgroup/systemd", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:30": {
			{MajorMinor: "0:30", MountPoint: "/sys/fs/pstore", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "pstore"},
		},
		"0:31": {
			{MajorMinor: "0:31", MountPoint: "/sys/fs/bpf", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "bpf"},
		},
		"0:32": {
			{MajorMinor: "0:32", MountPoint: "/sys/fs/cgroup/net_cls,net_prio", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:33": {
			{MajorMinor: "0:33", MountPoint: "/sys/fs/cgroup/cpuset", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:34": {
			{MajorMinor: "0:34", MountPoint: "/sys/fs/cgroup/cpu,cpuacct", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:35": {
			{MajorMinor: "0:35", MountPoint: "/sys/fs/cgroup/perf_event", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:36": {
			{MajorMinor: "0:36", MountPoint: "/sys/fs/cgroup/pids", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:37": {
			{MajorMinor: "0:37", MountPoint: "/sys/fs/cgroup/rdma", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:38": {
			{MajorMinor: "0:38", MountPoint: "/sys/fs/cgroup/blkio", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:39": {
			{MajorMinor: "0:39", MountPoint: "/sys/fs/cgroup/devices", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:4": {
			{MajorMinor: "0:4", MountPoint: "/run/snapd/ns/lxd.mnt", MountOptions: []string{"rw"}, fsType: "nsfs"},
			{MajorMinor: "0:4", MountPoint: "/run/netns/cni-824a6a31-7b0f-585f-3a6a-5e35af9205c2", MountOptions: []string{"rw"}, fsType: "nsfs"},
			{MajorMinor: "0:4", MountPoint: "/run/netns/cni-8f09d44f-03c2-922a-6cec-0430c0407f7c", MountOptions: []string{"rw"}, fsType: "nsfs"},
			{MajorMinor: "0:4", MountPoint: "/run/netns/cni-22bf6456-9aa5-d392-ae01-26b012139712", MountOptions: []string{"rw"}, fsType: "nsfs"},
			{MajorMinor: "0:4", MountPoint: "/run/netns/cni-870b0f67-54cb-0a2c-c503-da688e74ac04", MountOptions: []string{"rw"}, fsType: "nsfs"},
			{MajorMinor: "0:4", MountPoint: "/run/netns/cni-3891f233-1797-7edf-8e30-9db5b50786a9", MountOptions: []string{"rw"}, fsType: "nsfs"},
			{MajorMinor: "0:4", MountPoint: "/run/netns/cni-da6e2633-4c09-babd-2511-27c45c0a69f3", MountOptions: []string{"rw"}, fsType: "nsfs"},
		},
		"0:40": {
			{MajorMinor: "0:40", MountPoint: "/sys/fs/cgroup/memory", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:41": {
			{MajorMinor: "0:41", MountPoint: "/sys/fs/cgroup/hugetlb", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:42": {
			{MajorMinor: "0:42", MountPoint: "/sys/fs/cgroup/freezer", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:43": {
			{MajorMinor: "0:43", MountPoint: "/proc/sys/fs/binfmt_misc", MountOptions: []string{"relatime", "rw"}, fsType: "autofs"},
		},
		"0:44": {
			{MajorMinor: "0:44", MountPoint: "/dev/hugepages", MountOptions: []string{"relatime", "rw"}, fsType: "hugetlbfs"},
		},
		"0:45": {
			{MajorMinor: "0:45", MountPoint: "/sys/fs/fuse/connections", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "fusectl"},
		},
		"0:48": {
			{MajorMinor: "0:48", MountPoint: "/run/user/1000", MountOptions: []string{"nodev", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:5": {
			{MajorMinor: "0:5", MountPoint: "/dev", MountOptions: []string{"relatime", "rw"}, fsType: "devtmpfs"},
		},
		"0:50": {
			{MajorMinor: "0:50", MountPoint: "/var/lib/kubelet/pods/54589c77-2df4-43e7-a1df-cf90d87c1107/volumes/kubernetes.io~projected/kube-api-access-nqwj8", MountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:51": {
			{MajorMinor: "0:51", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/f9e55b5765965539bc04917b21c3cdc9c32ab8055f0e17fb84c19441d0818451/shm", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:52": {
			{MajorMinor: "0:52", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/f9e55b5765965539bc04917b21c3cdc9c32ab8055f0e17fb84c19441d0818451/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:6": {
			{MajorMinor: "0:6", MountPoint: "/sys/kernel/security", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "securityfs"},
		},
		"0:63": {
			{MajorMinor: "0:63", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/ce0c9df77871f26de88ae3382784f1d5eece115be98cd16e60bbdebbf338d899/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:7": {
			{MajorMinor: "0:7", MountPoint: "/sys/kernel/debug", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "debugfs"},
		},
		"0:72": {
			{MajorMinor: "0:72", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/ad498073d40f2b93f66ea5b2c3a1b93ec736cf7e3558437a0713a0f6be13590e/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:81": {
			{MajorMinor: "0:81", MountPoint: "/var/lib/kubelet/pods/6c9d90f3-3956-417d-a142-d0b96a7c7422/volumes/kubernetes.io~projected/kube-api-access-2gc2w", MountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:82": {
			{MajorMinor: "0:82", MountPoint: "/var/lib/kubelet/pods/6c9d90f3-3956-417d-a142-d0b96a7c7422/volumes/kubernetes.io~secret/directcsi-conversion-webhook", MountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:83": {
			{MajorMinor: "0:83", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/396ed771035a8a89202957e6c901e72ce3c833f5ae2a5ecdd9e9394abad17eba/shm", MountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:84": {
			{MajorMinor: "0:84", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/396ed771035a8a89202957e6c901e72ce3c833f5ae2a5ecdd9e9394abad17eba/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:95": {
			{MajorMinor: "0:95", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/ef86ec6f2f1c38dc8711586745a4d994be6b3d2546d1d997c5364b40d84c87a2/rootfs", MountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"202:1": {
			{MajorMinor: "202:1", MountPoint: "/", MountOptions: []string{"relatime", "rw"}, fsType: "ext4"},
		},
		"202:80": {
			{MajorMinor: "202:80", MountPoint: "/var/lib/direct-csi/mnt/9d619667-0286-436e-ba21-ba4b5870cc39", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"202:96": {
			{MajorMinor: "202:96", MountPoint: "/var/lib/direct-csi/mnt/ccf1082d-21ee-42ac-a480-7a00933e169d", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"253:0": {
			{MajorMinor: "253:0", MountPoint: "/var/lib/direct-csi/mnt/e5cd069d-678b-4024-acc8-039c62379323", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3fc2e5c6-bbea-4c16-ad4a-eb6e769a5a49/globalmount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-b9c22b37-64f6-49b9-af7d-b43a88548a58/globalmount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-5f59b78e-bea9-445f-b808-805a5db626fb/globalmount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-c77067c7-a42a-4b4c-a9b2-7beaac6dcfee/globalmount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/e9d36191-301a-4a72-b061-8ce0bb0d6fcf/volumes/kubernetes.io~csi/pvc-3fc2e5c6-bbea-4c16-ad4a-eb6e769a5a49/mount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/e9d36191-301a-4a72-b061-8ce0bb0d6fcf/volumes/kubernetes.io~csi/pvc-b9c22b37-64f6-49b9-af7d-b43a88548a58/mount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/e9d36191-301a-4a72-b061-8ce0bb0d6fcf/volumes/kubernetes.io~csi/pvc-5f59b78e-bea9-445f-b808-805a5db626fb/mount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/e9d36191-301a-4a72-b061-8ce0bb0d6fcf/volumes/kubernetes.io~csi/pvc-c77067c7-a42a-4b4c-a9b2-7beaac6dcfee/mount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-14aad478-3e37-4cf0-8010-703e2c94ccdd/globalmount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ae36638-e194-42b5-8229-0d6c38894bde/globalmount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/558a704e-918b-4562-8426-dc27deba229d/volumes/kubernetes.io~csi/pvc-3ae36638-e194-42b5-8229-0d6c38894bde/mount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-21c7d48e-2fc3-43b7-8843-53e34f114f0f/globalmount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-9082e93a-2219-4a01-8625-79916623f491/globalmount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/558a704e-918b-4562-8426-dc27deba229d/volumes/kubernetes.io~csi/pvc-14aad478-3e37-4cf0-8010-703e2c94ccdd/mount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/558a704e-918b-4562-8426-dc27deba229d/volumes/kubernetes.io~csi/pvc-21c7d48e-2fc3-43b7-8843-53e34f114f0f/mount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{MajorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/558a704e-918b-4562-8426-dc27deba229d/volumes/kubernetes.io~csi/pvc-9082e93a-2219-4a01-8625-79916623f491/mount", MountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"7:0": {
			{MajorMinor: "7:0", MountPoint: "/snap/amazon-ssm-agent/3552", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:1": {
			{MajorMinor: "7:1", MountPoint: "/snap/core18/1997", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:2": {
			{MajorMinor: "7:2", MountPoint: "/snap/core18/2066", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:3": {
			{MajorMinor: "7:3", MountPoint: "/snap/lxd/20326", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:4": {
			{MajorMinor: "7:4", MountPoint: "/snap/lxd/19647", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:5": {
			{MajorMinor: "7:5", MountPoint: "/snap/snapd/12159", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:6": {
			{MajorMinor: "7:6", MountPoint: "/snap/snapd/12057", MountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"}},
	}
}

func TestProbe(t *testing.T) {
	testCases := []struct {
		filename       string
		expectedResult map[string][]MountInfo
	}{
		{"mountinfo.testdata1", testProbeCast1Result()},
		{"mountinfo.testdata2", testProbeCast2Result()},
		{"mountinfo.testdata3", testProbeCast3Result()},
	}

	for i, testCase := range testCases {
		result, err := probe(testCase.filename)
		if err != nil {
			t.Fatalf("case %v: unexpected error %v", i+1, err)
		}

		if !reflect.DeepEqual(result, testCase.expectedResult) {
			t.Fatalf("case %v: result: expected: %+v, got: %+v", i+1, testCase.expectedResult, result)
		}
	}
}
