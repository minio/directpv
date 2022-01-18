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

package sys

import (
	"reflect"
	"testing"
)

func testProbeMountsCast1Result() map[string][]MountInfo {
	return map[string][]MountInfo{
		"0:12": {
			{majorMinor: "0:12", MountPoint: "/sys/kernel/tracing", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tracefs"},
		},
		"0:20": {
			{majorMinor: "0:20", MountPoint: "/dev/mqueue", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "mqueue"},
		},
		"0:21": {
			{majorMinor: "0:21", MountPoint: "/sys/fs/selinux", mountOptions: []string{"noexec", "nosuid", "relatime", "rw"}, fsType: "selinuxfs"},
		},
		"0:22": {
			{majorMinor: "0:22", MountPoint: "/sys", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "sysfs"},
		},
		"0:23": {
			{majorMinor: "0:23", MountPoint: "/proc", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "proc"},
		},
		"0:24": {
			{majorMinor: "0:24", MountPoint: "/dev/shm", mountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:25": {
			{majorMinor: "0:25", MountPoint: "/dev/pts", mountOptions: []string{"noexec", "nosuid", "relatime", "rw"}, fsType: "devpts"},
		},
		"0:26": {
			{majorMinor: "0:26", MountPoint: "/run", mountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:27": {
			{majorMinor: "0:27", MountPoint: "/sys/fs/cgroup", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup2"},
		},
		"0:28": {
			{majorMinor: "0:28", MountPoint: "/sys/fs/pstore", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "pstore"},
		},
		"0:29": {
			{majorMinor: "0:29", MountPoint: "/sys/firmware/efi/efivars", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "efivarfs"},
		},
		"0:30": {
			{majorMinor: "0:30", MountPoint: "/sys/fs/bpf", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "bpf"},
		},
		"0:33": {
			{majorMinor: "0:33", MountPoint: "/proc/sys/fs/binfmt_misc", mountOptions: []string{"relatime", "rw"}, fsType: "autofs"},
		},
		"0:34": {
			{majorMinor: "0:34", MountPoint: "/dev/hugepages", mountOptions: []string{"relatime", "rw"}, fsType: "hugetlbfs"},
		},
		"0:35": {
			{majorMinor: "0:35", MountPoint: "/sys/fs/fuse/connections", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "fusectl"},
		},
		"0:36": {
			{majorMinor: "0:36", MountPoint: "/sys/kernel/config", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "configfs"},
		},
		"0:37": {
			{majorMinor: "0:37", MountPoint: "/tmp", mountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:39": {
			{majorMinor: "0:39", MountPoint: "/var/lib/nfs/rpc_pipefs", mountOptions: []string{"relatime", "rw"}, fsType: "rpc_pipefs"},
		},
		"0:45": {
			{majorMinor: "0:45", MountPoint: "/run/user/1000", mountOptions: []string{"nodev", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:46": {
			{majorMinor: "0:46", MountPoint: "/run/user/1000/gvfs", mountOptions: []string{"nodev", "nosuid", "relatime", "rw"}, fsType: "fuse", fsSubType: "gvfsd-fuse"},
		},
		"0:5": {
			{majorMinor: "0:5", MountPoint: "/dev", mountOptions: []string{"noexec", "nosuid", "rw"}, fsType: "devtmpfs"},
		},
		"0:6": {
			{majorMinor: "0:6", MountPoint: "/sys/kernel/security", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "securityfs"},
		},
		"0:7": {
			{majorMinor: "0:7", MountPoint: "/sys/kernel/debug", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "debugfs"},
		},
		"259:1": {
			{majorMinor: "259:1", MountPoint: "/boot/efi", mountOptions: []string{"relatime", "rw"}, fsType: "vfat"},
		},
		"259:2": {
			{majorMinor: "259:2", MountPoint: "/boot", mountOptions: []string{"relatime", "rw"}, fsType: "ext4"},
		},
		"259:3": {
			{majorMinor: "259:3", MountPoint: "/", mountOptions: []string{"relatime", "rw"}, fsType: "ext4"},
		},
		"259:4": {
			{majorMinor: "259:4", MountPoint: "/home", mountOptions: []string{"relatime", "rw"}, fsType: "ext4"},
		},
	}
}

func testProbeMountsCast2Result() map[string][]MountInfo {
	return map[string][]MountInfo{
		"0:11": {
			{majorMinor: "0:11", MountPoint: "/sys/kernel/tracing", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tracefs"},
		},
		"0:18": {
			{majorMinor: "0:18", MountPoint: "/dev/mqueue", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "mqueue"},
		},
		"0:19": {
			{majorMinor: "0:19", MountPoint: "/sys/fs/selinux", mountOptions: []string{"relatime", "rw"}, fsType: "selinuxfs"},
		},
		"0:20": {
			{majorMinor: "0:20", MountPoint: "/sys/kernel/config", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "configfs"},
		},
		"0:21": {
			{majorMinor: "0:21", MountPoint: "/sys", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "sysfs"},
		},
		"0:22": {
			{majorMinor: "0:22", MountPoint: "/proc", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "proc"},
		},
		"0:23": {
			{majorMinor: "0:23", MountPoint: "/dev/shm", mountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:24": {
			{majorMinor: "0:24", MountPoint: "/dev/pts", mountOptions: []string{"noexec", "nosuid", "relatime", "rw"}, fsType: "devpts"},
		},
		"0:25": {
			{majorMinor: "0:25", MountPoint: "/run", mountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
			{majorMinor: "0:25", MountPoint: "/run/snapd/ns", mountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:26": {
			{majorMinor: "0:26", MountPoint: "/run/lock", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:27": {
			{majorMinor: "0:27", MountPoint: "/sys/fs/cgroup", mountOptions: []string{"nodev", "noexec", "nosuid", "ro"}, fsType: "tmpfs"},
		},
		"0:28": {
			{majorMinor: "0:28", MountPoint: "/sys/fs/cgroup/unified", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup2"},
		},
		"0:29": {
			{majorMinor: "0:29", MountPoint: "/sys/fs/cgroup/systemd", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:30": {
			{majorMinor: "0:30", MountPoint: "/sys/fs/pstore", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "pstore"},
		},
		"0:31": {
			{majorMinor: "0:31", MountPoint: "/sys/fs/bpf", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "bpf"},
		},
		"0:32": {
			{majorMinor: "0:32", MountPoint: "/sys/fs/cgroup/net_cls,net_prio", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:33": {
			{majorMinor: "0:33", MountPoint: "/sys/fs/cgroup/cpuset", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:34": {
			{majorMinor: "0:34", MountPoint: "/sys/fs/cgroup/cpu,cpuacct", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:35": {
			{majorMinor: "0:35", MountPoint: "/sys/fs/cgroup/perf_event", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:36": {
			{majorMinor: "0:36", MountPoint: "/sys/fs/cgroup/pids", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:37": {
			{majorMinor: "0:37", MountPoint: "/sys/fs/cgroup/rdma", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:38": {
			{majorMinor: "0:38", MountPoint: "/sys/fs/cgroup/blkio", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:39": {
			{majorMinor: "0:39", MountPoint: "/sys/fs/cgroup/devices", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:4": {
			{majorMinor: "0:4", MountPoint: "/run/snapd/ns/lxd.mnt", mountOptions: []string{"rw"}, fsType: "nsfs"},
			{majorMinor: "0:4", MountPoint: "/run/netns/cni-824a6a31-7b0f-585f-3a6a-5e35af9205c2", mountOptions: []string{"rw"}, fsType: "nsfs"},
		},
		"0:40": {
			{majorMinor: "0:40", MountPoint: "/sys/fs/cgroup/memory", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:41": {
			{majorMinor: "0:41", MountPoint: "/sys/fs/cgroup/hugetlb", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:42": {
			{majorMinor: "0:42", MountPoint: "/sys/fs/cgroup/freezer", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:43": {
			{majorMinor: "0:43", MountPoint: "/proc/sys/fs/binfmt_misc", mountOptions: []string{"relatime", "rw"}, fsType: "autofs"},
		},
		"0:44": {
			{majorMinor: "0:44", MountPoint: "/dev/hugepages", mountOptions: []string{"relatime", "rw"}, fsType: "hugetlbfs"},
		},
		"0:45": {
			{majorMinor: "0:45", MountPoint: "/sys/fs/fuse/connections", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "fusectl"},
		},
		"0:48": {
			{majorMinor: "0:48", MountPoint: "/run/user/1000", mountOptions: []string{"nodev", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:5": {
			{majorMinor: "0:5", MountPoint: "/dev", mountOptions: []string{"relatime", "rw"}, fsType: "devtmpfs"},
		},
		"0:50": {
			{majorMinor: "0:50", MountPoint: "/var/lib/kubelet/pods/54589c77-2df4-43e7-a1df-cf90d87c1107/volumes/kubernetes.io~projected/kube-api-access-nqwj8", mountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:51": {
			{majorMinor: "0:51", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/f9e55b5765965539bc04917b21c3cdc9c32ab8055f0e17fb84c19441d0818451/shm", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:52": {
			{majorMinor: "0:52", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/f9e55b5765965539bc04917b21c3cdc9c32ab8055f0e17fb84c19441d0818451/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:6": {
			{majorMinor: "0:6", MountPoint: "/sys/kernel/security", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "securityfs"},
		},
		"0:63": {
			{majorMinor: "0:63", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/ce0c9df77871f26de88ae3382784f1d5eece115be98cd16e60bbdebbf338d899/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:7": {
			{majorMinor: "0:7", MountPoint: "/sys/kernel/debug", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "debugfs"},
		},
		"0:72": {
			{majorMinor: "0:72", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/ad498073d40f2b93f66ea5b2c3a1b93ec736cf7e3558437a0713a0f6be13590e/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"202:1": {
			{majorMinor: "202:1", MountPoint: "/", mountOptions: []string{"relatime", "rw"}, fsType: "ext4"},
		},
		"202:80": {
			{majorMinor: "202:80", MountPoint: "/var/lib/direct-csi/mnt/9d619667-0286-436e-ba21-ba4b5870cc39", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"202:96": {
			{majorMinor: "202:96", MountPoint: "/var/lib/direct-csi/mnt/ccf1082d-21ee-42ac-a480-7a00933e169d", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"253:0": {
			{majorMinor: "253:0", MountPoint: "/var/lib/direct-csi/mnt/e5cd069d-678b-4024-acc8-039c62379323", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"7:0": {
			{majorMinor: "7:0", MountPoint: "/snap/amazon-ssm-agent/3552", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:1": {
			{majorMinor: "7:1", MountPoint: "/snap/core18/1997", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:2": {
			{majorMinor: "7:2", MountPoint: "/snap/core18/2066", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:3": {
			{majorMinor: "7:3", MountPoint: "/snap/lxd/20326", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:4": {
			{majorMinor: "7:4", MountPoint: "/snap/lxd/19647", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:5": {
			{majorMinor: "7:5", MountPoint: "/snap/snapd/12159", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:6": {
			{majorMinor: "7:6", MountPoint: "/snap/snapd/12057", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"}},
	}
}

func testProbeMountsCast3Result() map[string][]MountInfo {
	return map[string][]MountInfo{
		"0:101": {
			{majorMinor: "0:101", MountPoint: "/var/lib/kubelet/pods/3594fe81-84c4-415c-9b28-f9bdb136e0c0/volumes/kubernetes.io~secret/conversion-webhook-certs", mountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:102": {
			{majorMinor: "0:102", MountPoint: "/var/lib/kubelet/pods/3594fe81-84c4-415c-9b28-f9bdb136e0c0/volumes/kubernetes.io~projected/kube-api-access-5g9f8", mountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:103": {
			{majorMinor: "0:103", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/7b363bb8ecce707382129064a22fa327e1662129dbac50d240c44f6c4b6a77e3/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:11": {
			{majorMinor: "0:11", MountPoint: "/sys/kernel/tracing", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tracefs"},
		},
		"0:113": {
			{majorMinor: "0:113", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/6c7918a2a63b270069791a9b55385c4931067e1eab47e1284056148f0c43b549/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:122": {
			{majorMinor: "0:122", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/32f034e0946e06e0f2b575644b313679f53cb50240c908a6b9ef874a098c5a77/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:128": {
			{majorMinor: "0:128", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/75383da21ef3dcb1eb120d5e434b7a59927d05b6e8afe97872d450b25cb313c8/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:137": {
			{majorMinor: "0:137", MountPoint: "/var/lib/kubelet/pods/f61e15e5-8a08-476a-9104-e0a93f311243/volumes/kubernetes.io~secret/conversion-webhook-certs", mountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:138": {
			{majorMinor: "0:138", MountPoint: "/var/lib/kubelet/pods/f61e15e5-8a08-476a-9104-e0a93f311243/volumes/kubernetes.io~projected/kube-api-access-xnrhw", mountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:139": {
			{majorMinor: "0:139", MountPoint: "/var/lib/kubelet/pods/f61e15e5-8a08-476a-9104-e0a93f311243/volumes/kubernetes.io~secret/admission-webhook-certs", mountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:140": {
			{majorMinor: "0:140", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/d636b609e740a50a2603874c843bc8f5b70d78f91bb5dfafbab0e5e39681e7b9/shm", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:141": {
			{majorMinor: "0:141", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/d636b609e740a50a2603874c843bc8f5b70d78f91bb5dfafbab0e5e39681e7b9/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:152": {
			{majorMinor: "0:152", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/8c5f67a5933accd4df38393c83b953de9214ba85d374b83b92cee5784e97cc8d/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:158": {
			{majorMinor: "0:158", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/35c7f15e24b076a9930d51ffdf3c1cc0a6abbf13b2874e732508805df1967cf3/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:164": {
			{majorMinor: "0:164", MountPoint: "/var/lib/kubelet/pods/e9d36191-301a-4a72-b061-8ce0bb0d6fcf/volumes/kubernetes.io~projected/kube-api-access-dv9jq", mountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:165": {
			{majorMinor: "0:165", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/71b7b01fc3163bc784630cb90c33cae4c9de73fcadff4f108899ae4ea8643b9c/shm", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:166": {
			{majorMinor: "0:166", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/71b7b01fc3163bc784630cb90c33cae4c9de73fcadff4f108899ae4ea8643b9c/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:177": {
			{majorMinor: "0:177", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/987743f71c3a50963e3a4c9a025bd35d9351a0331c8f9a174748494110a8fe67/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:18": {
			{majorMinor: "0:18", MountPoint: "/dev/mqueue", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "mqueue"},
		},
		"0:186": {
			{majorMinor: "0:186", MountPoint: "/var/lib/kubelet/pods/558a704e-918b-4562-8426-dc27deba229d/volumes/kubernetes.io~projected/kube-api-access-st8md", mountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:187": {
			{majorMinor: "0:187", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/f458def070643977236c7f8a9f71489278d3c999171e1289399aec9fe4d27875/shm", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:188": {
			{majorMinor: "0:188", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/f458def070643977236c7f8a9f71489278d3c999171e1289399aec9fe4d27875/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:19": {
			{majorMinor: "0:19", MountPoint: "/sys/fs/selinux", mountOptions: []string{"relatime", "rw"}, fsType: "selinuxfs"},
		},
		"0:199": {
			{majorMinor: "0:199", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/7548d31ea3936cff4c369f2f9578208fd1965f1187ae437f3def2db7951bd8fb/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:20": {
			{majorMinor: "0:20", MountPoint: "/sys/kernel/config", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "configfs"},
		},
		"0:21": {
			{majorMinor: "0:21", MountPoint: "/sys", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "sysfs"},
		},
		"0:22": {
			{majorMinor: "0:22", MountPoint: "/proc", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "proc"},
		},
		"0:23": {
			{majorMinor: "0:23", MountPoint: "/dev/shm", mountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:24": {
			{majorMinor: "0:24", MountPoint: "/dev/pts", mountOptions: []string{"noexec", "nosuid", "relatime", "rw"}, fsType: "devpts"},
		},
		"0:25": {
			{majorMinor: "0:25", MountPoint: "/run", mountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
			{majorMinor: "0:25", MountPoint: "/run/snapd/ns", mountOptions: []string{"nodev", "nosuid", "rw"}, fsType: "tmpfs"},
		},
		"0:26": {
			{majorMinor: "0:26", MountPoint: "/run/lock", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:27": {
			{majorMinor: "0:27", MountPoint: "/sys/fs/cgroup", mountOptions: []string{"nodev", "noexec", "nosuid", "ro"}, fsType: "tmpfs"},
		},
		"0:28": {
			{majorMinor: "0:28", MountPoint: "/sys/fs/cgroup/unified", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup2"},
		},
		"0:29": {
			{majorMinor: "0:29", MountPoint: "/sys/fs/cgroup/systemd", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:30": {
			{majorMinor: "0:30", MountPoint: "/sys/fs/pstore", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "pstore"},
		},
		"0:31": {
			{majorMinor: "0:31", MountPoint: "/sys/fs/bpf", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "bpf"},
		},
		"0:32": {
			{majorMinor: "0:32", MountPoint: "/sys/fs/cgroup/net_cls,net_prio", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:33": {
			{majorMinor: "0:33", MountPoint: "/sys/fs/cgroup/cpuset", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:34": {
			{majorMinor: "0:34", MountPoint: "/sys/fs/cgroup/cpu,cpuacct", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:35": {
			{majorMinor: "0:35", MountPoint: "/sys/fs/cgroup/perf_event", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:36": {
			{majorMinor: "0:36", MountPoint: "/sys/fs/cgroup/pids", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:37": {
			{majorMinor: "0:37", MountPoint: "/sys/fs/cgroup/rdma", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:38": {
			{majorMinor: "0:38", MountPoint: "/sys/fs/cgroup/blkio", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:39": {
			{majorMinor: "0:39", MountPoint: "/sys/fs/cgroup/devices", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:4": {
			{majorMinor: "0:4", MountPoint: "/run/snapd/ns/lxd.mnt", mountOptions: []string{"rw"}, fsType: "nsfs"},
			{majorMinor: "0:4", MountPoint: "/run/netns/cni-824a6a31-7b0f-585f-3a6a-5e35af9205c2", mountOptions: []string{"rw"}, fsType: "nsfs"},
			{majorMinor: "0:4", MountPoint: "/run/netns/cni-8f09d44f-03c2-922a-6cec-0430c0407f7c", mountOptions: []string{"rw"}, fsType: "nsfs"},
			{majorMinor: "0:4", MountPoint: "/run/netns/cni-22bf6456-9aa5-d392-ae01-26b012139712", mountOptions: []string{"rw"}, fsType: "nsfs"},
			{majorMinor: "0:4", MountPoint: "/run/netns/cni-870b0f67-54cb-0a2c-c503-da688e74ac04", mountOptions: []string{"rw"}, fsType: "nsfs"},
			{majorMinor: "0:4", MountPoint: "/run/netns/cni-3891f233-1797-7edf-8e30-9db5b50786a9", mountOptions: []string{"rw"}, fsType: "nsfs"},
			{majorMinor: "0:4", MountPoint: "/run/netns/cni-da6e2633-4c09-babd-2511-27c45c0a69f3", mountOptions: []string{"rw"}, fsType: "nsfs"},
		},
		"0:40": {
			{majorMinor: "0:40", MountPoint: "/sys/fs/cgroup/memory", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:41": {
			{majorMinor: "0:41", MountPoint: "/sys/fs/cgroup/hugetlb", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:42": {
			{majorMinor: "0:42", MountPoint: "/sys/fs/cgroup/freezer", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "cgroup"},
		},
		"0:43": {
			{majorMinor: "0:43", MountPoint: "/proc/sys/fs/binfmt_misc", mountOptions: []string{"relatime", "rw"}, fsType: "autofs"},
		},
		"0:44": {
			{majorMinor: "0:44", MountPoint: "/dev/hugepages", mountOptions: []string{"relatime", "rw"}, fsType: "hugetlbfs"},
		},
		"0:45": {
			{majorMinor: "0:45", MountPoint: "/sys/fs/fuse/connections", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "fusectl"},
		},
		"0:48": {
			{majorMinor: "0:48", MountPoint: "/run/user/1000", mountOptions: []string{"nodev", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:5": {
			{majorMinor: "0:5", MountPoint: "/dev", mountOptions: []string{"relatime", "rw"}, fsType: "devtmpfs"},
		},
		"0:50": {
			{majorMinor: "0:50", MountPoint: "/var/lib/kubelet/pods/54589c77-2df4-43e7-a1df-cf90d87c1107/volumes/kubernetes.io~projected/kube-api-access-nqwj8", mountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:51": {
			{majorMinor: "0:51", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/f9e55b5765965539bc04917b21c3cdc9c32ab8055f0e17fb84c19441d0818451/shm", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:52": {
			{majorMinor: "0:52", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/f9e55b5765965539bc04917b21c3cdc9c32ab8055f0e17fb84c19441d0818451/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:6": {
			{majorMinor: "0:6", MountPoint: "/sys/kernel/security", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "securityfs"},
		},
		"0:63": {
			{majorMinor: "0:63", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/ce0c9df77871f26de88ae3382784f1d5eece115be98cd16e60bbdebbf338d899/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:7": {
			{majorMinor: "0:7", MountPoint: "/sys/kernel/debug", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "debugfs"},
		},
		"0:72": {
			{majorMinor: "0:72", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/ad498073d40f2b93f66ea5b2c3a1b93ec736cf7e3558437a0713a0f6be13590e/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:81": {
			{majorMinor: "0:81", MountPoint: "/var/lib/kubelet/pods/6c9d90f3-3956-417d-a142-d0b96a7c7422/volumes/kubernetes.io~projected/kube-api-access-2gc2w", mountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:82": {
			{majorMinor: "0:82", MountPoint: "/var/lib/kubelet/pods/6c9d90f3-3956-417d-a142-d0b96a7c7422/volumes/kubernetes.io~secret/directcsi-conversion-webhook", mountOptions: []string{"relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:83": {
			{majorMinor: "0:83", MountPoint: "/run/k3s/containerd/io.containerd.grpc.v1.cri/sandboxes/396ed771035a8a89202957e6c901e72ce3c833f5ae2a5ecdd9e9394abad17eba/shm", mountOptions: []string{"nodev", "noexec", "nosuid", "relatime", "rw"}, fsType: "tmpfs"},
		},
		"0:84": {
			{majorMinor: "0:84", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/396ed771035a8a89202957e6c901e72ce3c833f5ae2a5ecdd9e9394abad17eba/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"0:95": {
			{majorMinor: "0:95", MountPoint: "/run/k3s/containerd/io.containerd.runtime.v2.task/k8s.io/ef86ec6f2f1c38dc8711586745a4d994be6b3d2546d1d997c5364b40d84c87a2/rootfs", mountOptions: []string{"relatime", "rw"}, fsType: "overlay"},
		},
		"202:1": {
			{majorMinor: "202:1", MountPoint: "/", mountOptions: []string{"relatime", "rw"}, fsType: "ext4"},
		},
		"202:80": {
			{majorMinor: "202:80", MountPoint: "/var/lib/direct-csi/mnt/9d619667-0286-436e-ba21-ba4b5870cc39", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"202:96": {
			{majorMinor: "202:96", MountPoint: "/var/lib/direct-csi/mnt/ccf1082d-21ee-42ac-a480-7a00933e169d", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"253:0": {
			{majorMinor: "253:0", MountPoint: "/var/lib/direct-csi/mnt/e5cd069d-678b-4024-acc8-039c62379323", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3fc2e5c6-bbea-4c16-ad4a-eb6e769a5a49/globalmount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-b9c22b37-64f6-49b9-af7d-b43a88548a58/globalmount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-5f59b78e-bea9-445f-b808-805a5db626fb/globalmount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-c77067c7-a42a-4b4c-a9b2-7beaac6dcfee/globalmount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/e9d36191-301a-4a72-b061-8ce0bb0d6fcf/volumes/kubernetes.io~csi/pvc-3fc2e5c6-bbea-4c16-ad4a-eb6e769a5a49/mount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/e9d36191-301a-4a72-b061-8ce0bb0d6fcf/volumes/kubernetes.io~csi/pvc-b9c22b37-64f6-49b9-af7d-b43a88548a58/mount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/e9d36191-301a-4a72-b061-8ce0bb0d6fcf/volumes/kubernetes.io~csi/pvc-5f59b78e-bea9-445f-b808-805a5db626fb/mount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/e9d36191-301a-4a72-b061-8ce0bb0d6fcf/volumes/kubernetes.io~csi/pvc-c77067c7-a42a-4b4c-a9b2-7beaac6dcfee/mount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-14aad478-3e37-4cf0-8010-703e2c94ccdd/globalmount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-3ae36638-e194-42b5-8229-0d6c38894bde/globalmount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/558a704e-918b-4562-8426-dc27deba229d/volumes/kubernetes.io~csi/pvc-3ae36638-e194-42b5-8229-0d6c38894bde/mount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-21c7d48e-2fc3-43b7-8843-53e34f114f0f/globalmount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-9082e93a-2219-4a01-8625-79916623f491/globalmount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/558a704e-918b-4562-8426-dc27deba229d/volumes/kubernetes.io~csi/pvc-14aad478-3e37-4cf0-8010-703e2c94ccdd/mount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/558a704e-918b-4562-8426-dc27deba229d/volumes/kubernetes.io~csi/pvc-21c7d48e-2fc3-43b7-8843-53e34f114f0f/mount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
			{majorMinor: "253:0", MountPoint: "/var/lib/kubelet/pods/558a704e-918b-4562-8426-dc27deba229d/volumes/kubernetes.io~csi/pvc-9082e93a-2219-4a01-8625-79916623f491/mount", mountOptions: []string{"relatime", "rw"}, fsType: "xfs"},
		},
		"7:0": {
			{majorMinor: "7:0", MountPoint: "/snap/amazon-ssm-agent/3552", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:1": {
			{majorMinor: "7:1", MountPoint: "/snap/core18/1997", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:2": {
			{majorMinor: "7:2", MountPoint: "/snap/core18/2066", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:3": {
			{majorMinor: "7:3", MountPoint: "/snap/lxd/20326", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:4": {
			{majorMinor: "7:4", MountPoint: "/snap/lxd/19647", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:5": {
			{majorMinor: "7:5", MountPoint: "/snap/snapd/12159", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"},
		},
		"7:6": {
			{majorMinor: "7:6", MountPoint: "/snap/snapd/12057", mountOptions: []string{"nodev", "relatime", "ro"}, fsType: "squashfs"}},
	}
}

func TestProbeMounts(t *testing.T) {
	testCases := []struct {
		filename       string
		expectedResult map[string][]MountInfo
	}{
		{"mountinfo.testdata1", testProbeMountsCast1Result()},
		{"mountinfo.testdata2", testProbeMountsCast2Result()},
		{"mountinfo.testdata3", testProbeMountsCast3Result()},
	}

	for i, testCase := range testCases {
		result, err := probeMounts(testCase.filename)
		if err != nil {
			t.Fatalf("case %v: unexpected error %v", i+1, err)
		}

		if !reflect.DeepEqual(result, testCase.expectedResult) {
			t.Fatalf("case %v: result: expected: %+v, got: %+v", i+1, testCase.expectedResult, result)
		}
	}
}
