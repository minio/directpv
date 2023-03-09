---
title: Driver Specification
---

## Driver Specification

### CSIDriver

| Key                 | Value                     |
|---------------------|---------------------------|
| `name`              | `directpv-min-io`         |
| `fsGroupPolicy`     | `ReadWriteOnceWithFSType` |
| `requiresRepublish` | `false`                   |
| `podInfoOnMount`    | `true`                    |
| `attachRequired`    | `false`                   |
| `storageCapacity`   | `false`                   |
| `modes`             | `Persistent`, `Ephemeral` |

### StorageClass

| Key                    | Value                    |
|------------------------|--------------------------|
| `name`                 | `directpv-min-io`        |
| `provisioner`          | `directpv-min-io`        |
| `reclaimPolicy`        | `Delete`                 |
| `allowVolumeExpansion` | `false`                  |
| `volumeBindingMode`    | `WaitForFirstConsumer`   |

### DirectPVDrives CRD

| Key          | Value                 |
| -------------|-----------------------|
| `name`       | `directpvdrives`      |
| `apigroup`   | `directpv.min.io`     |

### DirectPVVolumes CRD

| Key          | Value                 |
| -------------|-----------------------|
| `name`       | `directpvvolumes`     |
| `apigroup`   | `directpv.min.io`     |

### DirectPVNodes CRD

| Key          | Value                 |
| -------------|-----------------------|
| `name`       | `directpvnodes`       |
| `apigroup`   | `directpv.min.io`     |

### DirectPVInitRequests CRD

| Key          | Value                  |
| -------------|------------------------|
| `name`       | `directpvinitrequests` |
| `apigroup`   | `directpv.min.io`      |

### Driver RBAC 

| apiGroup                  | Resources                   | Verbs                                                | 
| --------------------------|-----------------------------|------------------------------------------------------|
|  (core)                   | `endpoints`                 | `get`, `list`, `watch`, `create`, `update`, `delete` |
|  (core)                   | `events`                    | `list`, `watch`, `create`, `update`, `patch`         |
|  (core)                   | `nodes`                     | `get`, `list`, `watch`                               |
|  (core)                   | `persistentvolumes`         | `get`, `list`, `watch`, `create`, `delete`           |
|  (core)                   | `persistentvolumeclaims`    | `get`, `list`, `watch`, `update`                     |
|  (core)                   | `pods,pod`                  | `get`, `list`, `watch`                               |
|  `policy`                 | `podsecuritypolicies`       | `use`                                                |
| `apiextensions.k8s.io`    | `customresourcedefinitions` | `get`, `list`, `watch`, `create`, `update`, `delete` |
| `coordination.k8s.io`     | `leases`                    | `get`, `list`, `watch`, `update`, `delete`, `create` |
| `directpv.min.io`         | `directpvdrives`            | `get`, `list`, `watch`, `create`, `update`, `delete` |
| `directpv.min.io`         | `directpvvolumes`           | `get`, `list`, `watch`, `create`, `update`, `delete` |
| `directpv.min.io`         | `directpvnodes`             | `get`, `list`, `watch`, `create`, `update`, `delete` |
| `directpv.min.io`         | `directpvinitrequests`      | `get`, `list`, `watch`, `create`, `update`, `delete` |
| `snapshot.storage.k8s.io` | `volumesnapshotcontents`    | `get`, `list`                                        |
| `snapshot.storage.k8s.io` | `volumesnapshots`           | `get`, `list`                                        |
| `storage.k8s.io`          | `csinodes`                  | `get`, `list`, `watch`                               |
| `storage.k8s.io`          | `storageclasses`            | `get`, `list`, `watch`                               |
| `storage.k8s.io`          | `volumeattachments`         | `get`, `list`, `watch`                               |


The service account binded to the above clusterrole is `directpv-min-io` in `directpv` namespace and the corresponding clusterrolebinding is `directpv-min-io`
