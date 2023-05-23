FAQ
----

#### What type of disks are recommended for DirectPV?

DirectPV is specifically meant for [Direct Attached Storage](https://en.wikipedia.org/wiki/Direct-attached_storage) such as hard drives, solid-state drives and JBODs etc. 

Avoid using DirectPV with SAN and NAS based storage options, as they inherently involve extra network hops in the data path. This leads to poor performance and increased complexity.

#### How DirectPV is different from LocalPV and HostPath?

Hostpath volumes are ephemeral and are tied to the lifecyle of pods. Hostpath volumes are lost when a pod is restarted or deleted, resulting in the loss of any stored data. 

DirectPV volumes are persistent through node and pod reboots. The lifecycle of a DirectPV volume is managed by the associated [Persistent Volume Claims](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).

LocalPVs are statically provisioned from a local storage resource on the nodes where persistent volumes are required, and must be created prior to the workloads using them. User management is required down to creating the actual object resource for the local PV.

DirectPV also creates statically provisioned volume resources, but does not require user intervention for the creation of the object resource. Instead, DirectPV dynamically provisions the persistent volume in response to a PVC requesting a DirectPV storage volume. This significantly reduces complexity of management.

#### How are the disks selected for a Pod?

DirectPV operates only on those disks which it [explicitly manages](./cli.md#initialize-the-available-drives-present-in-the-cluster)

	1. DirectPV selects managed disks local to the node where the pod is scheduled. This provides direct access for the pods to the disks. 

	2. DirectPV runs a selection algorithm to choose a disk for a volume creation request

	3. DirectPV then creates a sub-directory for the volume with quota set on the sub-directory for the requested volume size

	4. DirectPV publishes the volume to the pod.

 To know more on the selection algorithm, please refer [here](./volume-scheduling.md).

#### What does drive initialization do?

DirectPV command `kubectl directpv init` command will prepare the drives by formatting them with XFS filesystem and mounting them in a desired path (/var/lib/directpv/mnt/<uuid>). Upon success, these drives will be ready for volumes to be scheduled and you can see the initialized drives in `kubectl directpv list drives`.

#### What are the conditions that a drive must satisfy to be used for DirectPV?

A drive must meet the following requirements for it to be used for initialization

- The size of the drive should not be less then 512MiB.
- The drive must not be hidden. (Check /sys/class/block/<device>/hidden, It should not be "1")
- The drive must not be a read-only.
- The drive must not be partitioned.
- The drive must not be held by other devices. (Check /sys/class/block/<device>/holders, It should not have any entries)
- The drive must not be mounted.
- The drive must not have swap-on enabled.
- The drive must not be a CDROM.

DirectPV can support any storage resource presenting itself as a block device. While this includes LVMs, LUKs, LUNs, and similar virtualized drives, we do not recommend using DirectPV with those in production environments.

#### What does error "no drive found for requested topology" mean?

This error can be seen in PVC and pod description when 

- Kubernetes scheduled the pod on a node where DirectPV doesn't have any drives initialized. This can be fixed by using necessary selectors in the workload to make k8s schedule the pods on the correct set of nodes. Please refer [here](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/) to know more on assigning the pod to specific nodes.
- The requested topology specifications like size, zone etc cannot be satisified by any of initialized drives in the selected node. This can be fixed by changing the topology parameter in the workloads to suitable value.

#### Is `direct-csi-min-io` storage class still supported?

No, the support for `direct-csi-min-io` is removed from DirectPV version v4.0.0. No new volumes will be provisioned using `direct-csi-min-io` storage class. However, the existing volumes which are already provisioned by `direct-csi-min-io` storage class will still be managed by the latest DirectPV versions.

Alternatively, you can use `directpv-min-io` storage class for provisioning new volumes. And there are no functional or behavioral differences between 'direct-csi-min-io' and 'directpv-min-io'.

#### Does DirectPV support volume snapshotting?

No, DirectPV doesn't support CSI snapshotting. DirectPV is specifically meant for use cases like MinIO where the data availability and resiliency is taken care by the application itself. Additionally, with the AWS S3 versioning APIs and internal healing, snapshots isn't required.

#### Does DirectPV support _ReadWriteMany_?

No, DirectPV doesn't support _ReadWriteMany_. The workloads using DirectPV run local to the node and are provisioned from local disks in the node. This makes the workloads to direcly access the data path without any additional network hops unlike remote volumes (network PVs). The additional network hops may lead to poor performance and increases the complexity. So, DirectPV doesn't support _ReadWriteMany_ and only supports _ReadWriteOnce_.
