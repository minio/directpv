---
title: Driver Specification
---

## Driver Specification

### CSIDriver

| Key               | Value                     |
|-------------------|---------------------------|
| `name`            | `direct-csi-min-io`       |
| `podInfoOnMount`  | `true`                    |
| `attachRequired`  | `false`                   |
| `modes`           | `Persistent`, `Ephemeral` |

### StorageClass

| Key                 | Value                    |
|---------------------|--------------------------|
| `name`              | `direct-csi-min-io`      |
| `provisioner`       | `direct-csi-min-io`      |
| `reclaimPolicy`     | `Retain`                 |
| `volumeBindingMode` | `WaitForFirstConsumer`   |

### DirectCSIDrive CRD

| Key          | Value                 |
| -------------|-----------------------|
| `name`       | `directcsidrive`      |
| `apigroup`   | `direct.csi.min.io`   |

### DirectCSIVolume CRD

| Key          | Value                 |
| -------------|-----------------------|
| `name`       | `directcsivolume`     |
| `apigroup`   | `direct.csi.min.io`   |


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
| `direct.csi.min.io` | `directcsidrives` | `get`, `list`, `watch`, `create`, `update`, `delete` |
| `direct.csi.min.io` | `directcsivolumes` | `get`, `list`, `watch`, `create`, `update`, `delete` |
| `snapshot.storage.k8s.io` | `volumesnapshotcontents` | `get`, `list` |
| `snapshot.storage.k8s.io` | `volumesnapshots` | `get`, `list` |
| `storage.k8s.io` | `csinodes` | `get`, `list`, `watch` |
| `storage.k8s.io` | `storageclasses` | `get`, `list`, `watch` |
| `storage.k8s.io` | `volumeattachments` | `get`, `list`, `watch` |


