---
title: Architecture
---

Architecture
-------------

### Components

DirectCSI is made up of 5 components:

| Component         | Description                                                                           |
|-------------------|---------------------------------------------------------------------------------------|
| CSI Driver        | Performs mounting, unmounting of provisioned volumes                                  |
| CSI Controller    | Responsible for scheduling and detaching volumes on the nodes                         |
| Drive Controller  | Formats and manages drive lifecycle                                                   |
| Volume Controller | Manages volume lifecycle                                                              |
| Drive Discovery   | Discovers and monitors the drives and their states on the nodes                       |

The 4 components run as two different pods. 

| Name                          | Components                                                            | Description                        |
|-------------------------------|-----------------------------------------------------------------------|------------------------------------|
| DirectCSI Node Driver         | CSI Driver, Driver Controller, Volume Controller, Drive Discovery     | runs on every node as a DaemonSet  |
| DirectCSI Central Controller  | CSI Controller                                                        | runs as a deployment               |


### Scalability

Since the node driver runs on every node, the load on it is constrained to operations specific to that node. 

The central controller needs to be scaled up as the number of drives managed by DirectCSI is increased. By default, 3 replicas of central controller are run. As a rule of thumb, having as many central controller instances as etcd nodes is a good working solution for achieving high scale.

### Availability

If node driver is down, then volume mounting, unmounting, formatting and cleanup will not proceed for volumes and drives on that node. In order to restore operations, bring node driver to running status.

In central controller is down, then volume scheduling and deletion will not proceed for all volumes and drives in the direct-csi cluster. In order to restore operations, bring the central controller to running status.

Security is covered [here](./security.md)

### Node Driver

This runs on every node as a Daemonset in 'direct-csi-min-io' namespace. Each pod consists of four containers

#### Node driver registrar

This is a kubernetes csi side-car container which registers the `direct-csi-min-io` CSI driver with kubelet. This registration is necessary for kubelet to issue CSI RPC calls like `NodeGetInfo`, `NodeStageVolume`, `NodePublishVolume` to the corresponding nodes.

For more details, please refer [node-driver-registrar](https://github.com/kubernetes-csi/node-driver-registrar).

#### Livenessprobe

This is a kubernetes csi side-car container which exposes an HTTP `/healthz` endpoint as a liveness hook. This endpoint will be used by kubernetes for csi-driver liveness checks.

For more details. please refer [livenessprobe](https://github.com/kubernetes-csi/livenessprobe)

#### Dynamic drive discovery

This container uses `directpv` binary with `--dynamic-drive-handler` flag enabled. This container is responsible for discovering and managing the drives in the node.

The devices will be discovered from `/run/data/udev/` directory and dynamically listens for udev events for any add, change and remove uevents. Apart from dynamically listening, there is a periodic 30sec sync which checks and syncs the drive states.

For any change, the directcsidrive object will be synced to match the local state. A new directcsidrive object will be created when a new device is detected during sync or when an "Add" uevent occurs. If an inuse/ready drive gets corrupted or lost, it will be tagged with a error condition on the drive. If an Available/Unavailable drive is lost, it will be deleted.

#### Direct CSI

This container acts as a node plugin and implements the following node service RPCs.

- [NodeGetInfo](https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetinfo)
- [NodeGetCapabilities](https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetinfoNodeGetCapabilities)
- [NodeGetVolumeStats](https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetvolumestats)
- [NodeStageVolume](https://github.com/container-storage-interface/spec/blob/master/spec.md#nodestagevolume)
- [NodePublishVolume](https://github.com/container-storage-interface/spec/blob/master/spec.md#nodepublishvolume)
- [NodeUnstageVolume](https://github.com/container-storage-interface/spec/blob/master/spec.md#nodeunstagevolume)
- [NodeUnpublishVolume](https://github.com/container-storage-interface/spec/blob/master/spec.md#nodeunpublishvolume)

This container is responsible for bind-mounting and umounting volumes on the responding nodes. Monitoring volumes is a WIP and will be added soon. Please refer [csi spec](https://github.com/container-storage-interface/spec) for more details on the CSI volume lifecycle.

Apart from this, there are also drive and volume controllers in place.

#### Drive Controller

Drive controller manages the directcsidrive object lifecycle. This actively listens for drive object (post-hook) events like Add, Update and Delete. The drive controller is responsible for the following

_Formatting a drive_ :-

If `.Spec.RequestedFormat` is set on the drive object, it indicates the `kubectl directpv drives format` was called on it and this drive will be formatted.

_Releasing a drive_ :-

`kubectl directpv drives release` is a special command to release a "Ready" drive in directpv cluster by umounting the drives and making it "Available". If `.Status.DriveStatus` is set to "Released", it indicates that `kubectl directpv drives release` was called on the drive and it will be released.

_Checking the primary mount of the drive_ :-

Drive controller also checks for the primary drive mounts. If an "InUse" or "Ready" drive is not mounted or if it has unexpected mount options set, the drive will be remounted with correct mountpoint and mount options.

_Tagging the lost drives_ :-

If a drive is not found on the host, it will be tagged as "lost" with an error message attached to the drive object and its respective volume objects.

Overall, drive controller validates and tries to sync the host state of the drive to match the expected state of the drive. For example, it mounts the "Ready" and "InUse" drives if their primary mount is not present in host.

For more details on the drive states, please refer [Drive States](./drive-states.md).

#### Volume Controller

Volume controller manages the directcsivolume object lifecycle. This actively listens for volume object (post-hook) events like Add, Update and Delete. The volume controller is responsible for the following

_Releasing/Purging deleted volumes and free-ing up its space on the drive_ :-

When a volume is deleted (PVC deletion) or purged (using `kubectl directpv drives purge` command), the corresponding volume object will be in terminating state (with deletion timestamp set on it). The volume controller will look for such deleted volume objects and releases them by freeing up the disk space and unsetting the finalizers.


### Central Controller

This runs as a deployment in 'direct-csi-min-io' namespace with default replica count 3.

(Note: The central controller does not do any device level interactions in the host)

Each pod consist of two continers

#### CSI Provisioner

This is a kubernetes csi side-car container which is responsible for sending volume provisioning (CreateVolume) and volume deletion (DeleteVolume) requests to csi drivers.

For more details, please refer [external-provisioner](https://github.com/kubernetes-csi/external-provisioner).

#### Direct CSI

This container acts as a central controller and implements the following RPCs

- [CreateVolume](https://github.com/container-storage-interface/spec/blob/master/spec.md#createvolume)
- [DeleteVolume](https://github.com/container-storage-interface/spec/blob/master/spec.md#deletevolume)

This container is responsible for selecting a suitable drive for a volume scheduling request. The selection algorithm looks for range and topology specifications provided in the CreateVolume request and selects a drive based on its free capacity.

(Note: kube-scheduler is responsible for selecting a node for a pod, central controller will just select a suitable drive in the requested node based on the specifications provided in the create volume request)
