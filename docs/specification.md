---
title: Driver Specification
---

## Driver Specification

### PVDriver

| Key               | Value                     |
|-------------------|---------------------------|
| `name`            | `direct-pv-min-io`       |
| `podInfoOnMount`  | `true`                    |
| `attachRequired`  | `false`                   |
| `modes`           | `Persistent`, `Ephemeral` |

### StorageClass

| Key                 | Value                    |
|---------------------|--------------------------|
| `name`              | `direct-pv-min-io`      |
| `provisioner`       | `direct-pv-min-io`      |
| `reclaimPolicy`     | `Retain`                 |
| `volumeBindingMode` | `WaitForFirstConsumer`   |

### DirectPVDrive CRD

| Key          | Value                 |
| -------------|-----------------------|
| `name`       | `directpvdrive`      |
| `apigroup`   | `direct.pv.min.io`   |

### DirectPVVolume CRD

| Key          | Value                 |
| -------------|-----------------------|
| `name`       | `directpvvolume`     |
| `apigroup`   | `direct.pv.min.io`   |


### Driver RBAC 

| apiGroup | Resources  | Verbs | 
| ---------|------------|-------|
|  (core)  | `endpoints` | `get`, `list`, `watch`, `create`, `update`, `delete` |
|  (core)  | `events` | `list`, `watch`, `create`, `update` |
|  (core)  | `nodes`   | `get`, `list`, `watch` |
|  (core)  | `persistentvolumes`   | `get`, `list`, `watch`, `create`, `delete`|
|  (core)  | `persistentvolumeclaims`   | `get`, `list`, `watch`|
| `apiextensions.k8s.io` | `customresourcedefinitions` | `get`, `list`, `watch`, `create`, `update`, `delete` |
| `coordination.k8s.io` | `leases` | `get`, `list`, `watch`, `update`, `delete`, `create` |
| `direct.pv.min.io` | `directpvdrives` | `get`, `list`, `watch`, `create`, `update`, `delete` |
| `direct.pv.min.io` | `directpvvolumes` | `get`, `list`, `watch`, `create`, `update`, `delete` |
| `snapshot.storage.k8s.io` | `volumesnapshotcontents` | `get`, `list` |
| `snapshot.storage.k8s.io` | `volumesnapshots` | `get`, `list` |
| `storage.k8s.io` | `pvnodes` | `get`, `list`, `watch` |
| `storage.k8s.io` | `storageclasses` | `get`, `list`, `watch` |
| `storage.k8s.io` | `volumeattachments` | `get`, `list`, `watch` |


