# DirectPV CSI driver architecture

DirectPV is implemented as per the [CSI specification](https://github.com/container-storage-interface/spec/blob/master/spec.md). It comes with the below components run as Pods in Kubernetes.
* `Controller`
* `Node server`

When DirectPV contains legacy volumes from `DirectCSI`, the below additional components also run as Pods.
* `Legacy controller `
* `Legacy node server`

## Controller
The Controller runs as `Deployment` Pods named `controller`, which are three replicas located in any Kubernetes nodes. In the three replicas, one instance is elected to serve requests. Each pod contains below running containers:
* `CSI provisioner` - Bridges volume creation and deletion requests from `Persistent Volume Claim` to CSI controller.
* `Controller` - Controller server which honors CSI requests to create, delete and expand volumes.
* `CSI resizer` - Bridges volume expansion requests from `Persistent Volume Claim` to CSI controller.

### Controller server
Controller server runs as container `controller` in a `controller` `Deployment` Pod. It handles below requests:
* `Create volume` - Controller server creates new `DirectPVVolume` CRD after reversing requested storage space on suitable `DirectPVDrive` CRD. For more information, refer to the [Volume scheduling guide](./volume-scheduling.md)
* `Delete volume` - Controller server deletes `DirectPVVolume` CRD for unbound volumes after releasing previously reserved space in `DirectPVDrive` CRD.
* `Expand volume` - Controller server expands `DirectPVVolume` CRD after reversing requested storage space in `DirectPVDrive` CRD.

Below is a workflow diagram
```
┌────────────┐                                               ┌────────────┐
│            │ Create Event ┌─────────────┐ CreateVolume API │            │   ┌────────────────────┐
│            │------------->│     CSI     │----------------->│            │-->│  DirectPVDrive CRD │
│ Persistent │ Delete Event │ Provisioner │ DeleteVolume API │            │   └────────────────────┘
│   Volume   │------------->│             │----------------->│ Controller │
│   Claim    │              └─────────────┘                  │   Server   │
│   (PVC)    │                                               │            │   ┌────────────────────┐
│            │ Update Event ┌─────────────┐ ExpandVolume API │            │-->│ DirectPVVolume CRD │
│            │------------->│ CSI Resizer │----------------->│            │   └────────────────────┘
│            │              └─────────────┘                  └────────────┘
└────────────┘
```

## Legacy controller
Legacy controller runs as `Deployment` Pods named `legacy-controller`, which are three replicas located in any Kubernetes nodes. In the three replicas, one instance is elected to serve requests. Each pod contains below running containers:
* `CSI provisioner` - Bridges legacy volume creation and deletion requests from `Persistent Volume Claim` to CSI controller.
* `Controller` - Honors CSI requests to delete and expand volumes. Create volume request is prohibited i.e. this controller works only for legacy volumes previously created in `DirectCSI`.
* `CSI resizer` - Bridges legacy volume expansion requests from `Persistent Volume Claim` to CSI controller.

### Legacy controller server
Legacy controller server runs as container `controller` in a `legacy-controller` `Deployment` Pod. It handles below requests:
* `Create volume` - Controller server errors out for this request.
* `Delete volume` - Controller server deletes `DirectPVVolume` CRD for unbound volumes after releasing previously reserved space in `DirectPVDrive` CRD.
* `Expand volume` - Controller server expands `DirectPVVolume` CRD after reversing requested storage space in `DirectPVDrive` CRD.

Below is a workflow diagram
```
┌─────────────────────┐                                                   ┌────────────┐
│                     │ Delete Event ┌─────────────────┐ DeleteVolume API │            │   ┌────────────────────┐
│  Persistent Volume  │------------->│ CSI Provisioner │----------------->│            │-->│  DirectPVDrive CRD │
│   Claim (PVC) with  │              └─────────────────┘                  │ Controller │   └────────────────────┘
│  direct-csi-min-io  │                                                   │   Server   │   ┌────────────────────┐
│    Storage Class    │ Update Event ┌─────────────────┐ ExpandVolume API │            │-->│ DirectPVVolume CRD │
│                     │------------->│   CSI Resizer   │----------------->│            │   └────────────────────┘
│                     │              └─────────────────┘                  └────────────┘
└─────────────────────┘
```

## Node server
Node server runs as `DaemonSet` Pods named `node-server` in all or selected Kubernetes nodes. Each node server Pod runs on a node independently. Each pod contains below running containers:
* `Node driver registrar` - Registers node server to kubelet to get CSI RPC calls.
* `Node server` - Honors stage, unstage, publish, unpublish and expand volume RPC requests.
* `Node controller` - Honors CRD events from `DirectPVDrive`, `DirectPVVolume`, `DirectPVNode` and `DirectPVInitRequest`.
* `Liveness probe` - Exposes `/healthz` endpoint to check node server liveness by Kubernetes.

Below is a workflow diagram
```
┌─────────┐                    ┌────────┐                 ┌──────────────────────────────────┐    ┌────────────────────┐
│         │  StageVolume RPC   │        │   StageVolume   │ * Create data directory          │    │                    │
│         │------------------->│        │---------------->│ * Set xfs quota                  │<-->│                    │
│         │                    │        │                 │ * Bind mount staging target path │    │                    │
│         │                    │        │                 └──────────────────────────────────┘    │                    │
│         │ PublishVolume RPC  │        │  PublishVolume  ┌──────────────────────────────────┐    │                    │
│         │------------------->│        │---------------->│ * Bind mount target path         │<-->│                    │
│ Kubelet │                    │  Node  │                 └──────────────────────────────────┘    │ DirectPVDrive CRD  │
│         │UnpublishVolume RPC │ Server │ UnpublishVolume ┌──────────────────────────────────┐    │ DirectPVVolume CRD │
│         │------------------->│        │---------------->│ * Unmount target path            │<-->│                    │
│         │                    │        │                 └──────────────────────────────────┘    │                    │
│         │ UnstageVolume RPC  │        │  UnstageVolume  ┌──────────────────────────────────┐    │                    │
│         │------------------->│        │---------------->│ * Unmount staging target path    │<-->│                    │
│         │                    │        │                 └──────────────────────────────────┘    │                    │
│         │  ExpandVolume RPC  │        │   ExpandVolume  ┌──────────────────────────────────┐    │                    │
│         │------------------->│        │---------------->│ * Set xfs quota                  │<-->│                    │
└─────────┘                    └────────┘                 └──────────────────────────────────┘    └────────────────────┘
```

## Legacy node server
Legacy node server runs as `DaemonSet` Pods named `legacy-node-server` in all or selected Kubernetes nodes. Each legacy node server Pod runs on a node independently. Each pod contains the below running containers:
* `Node driver registrar` - Registers legacy node server to kubelet to get CSI RPC calls.
* `Node server` - Honors stage, unstage, publish, unpublish and expand volume RPC requests.
* `Liveness probe` - Exposes `/healthz` endpoint to check legacy node server liveness by Kubernetes.

Workflow diagram is same as in [node server](#node-server).
