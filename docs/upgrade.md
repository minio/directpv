---
title: Upgrade
---

Version Upgrade
---------------

### Guidelines to upgrade to the latest DirectPV version

DirectPV version upgrades are seameless and transparent. The resources will be upgraded automatically when you run the latest version over the existing resources. The latest version of DirectPV should be available in [krew](https://github.com/kubernetes-sigs/krew-index). For more details on the installation, Please refer the [Installation guidelines](./installation.md).

NOTE: For the users who don't prefer krew, Please find the latest images in [releases](https://github.com/minio/directpv/releases).


#### Upgrade from versions < v3.2.x

If you are on DirectPV version < 3.2.x, it is recommended to upgrade to v3.2.2 and then to the latest

Please follow https://github.com/minio/directpv/blob/master/docs/upgrade.md#upgrade-to-v300 for the upgrade steps from legacy versions


#### Upgrade from v3.2.x

In the latest version of DirectPV, the CSI sidecar images have been updated. If private registry is used for images, please make sure the following images are available in your registry before upgrade.

```
quay.io/minio/csi-node-driver-registrar:v2.6.3
quay.io/minio/csi-provisioner:v3.4.0
quay.io/minio/livenessprobe:v2.9.0
quay.io/minio/csi-resizer:v1.7.0
```

**Notes:**

- If your kubernetes version is less than v1.20, you need push `quay.io/minio/csi-provisioner:v2.2.0-go1.18`

If you are on DirectPV versions < v4.0.0 and if you are using any custom storage classes for controlling volume scheduling based on access-tiers as explained [here](https://github.com/minio/directpv/blob/master/docs/scheduling.md), you need to make the following change to these custom storage classes.

You need to change `direct.csi.min.io/access-tier: <your_access_tier_value>` to `directpv.min.io/access-tier: <your_access_tier_value>` in the respective storage class parameters section.
