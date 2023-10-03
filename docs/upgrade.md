# Upgrade DirectPV installation

## Upgrade DirectPV CSI driver

### Upgrade DirectPV CSI driver v4.x.x

#### Offline upgrade
Follow the below steps for an offline upgrade
1. Uninstall DirectPV CSI driver.
```sh
$ kubectl directpv uninstall
```
2. Upgrade DirectPV plugin by [this documentation](#upgrade-directpv-plugin).
3. Install the latest DirectPV CSI driver by [this documentation](./installation.md#directpv-csi-driver-installation).

#### In-place upgrade
Follow the below steps for an in-place upgrade
1. Upgrade DirectPV plugin by [this documentation](#upgrade-directpv-plugin).
2. Run install script with appropriate node-selector, tolerations, and `KUBELET_DIR_PATH` environment variable. Below is an example:
```sh
$ curl -sfL https://github.com/minio/directpv/raw/master/docs/tools/install.sh | sh - apply
```

### Upgrade legacy DirectCSI CSI driver
Follow the below steps to upgrade to the latest DirectPV CSI driver from a legacy DirectCSI installation.
1. Uninstall DirectCSI driver if you run v3.1.0 or newer version. For other versions, skip this step.
```sh
$ kubectl directcsi uninstall
```
2. Install the latest DirectPV plugin by [this documentation](./installation.md#directpv-plugin-installation) or upgrade existing DirectPV plugin by [this documentation](#upgrade-directpv-plugin).
3. Install the latest DirectPV CSI driver by [this documentation](./installation.md#directpv-csi-driver-installation).
4. Uninstall DirectCSI driver if you run older than v3.1.0 version. For other versions, skip this step.
```sh
$ kubectl directcsi uninstall
```

## Upgrade DirectPV plugin

### Upgrade using `Krew`
To upgrade the plugin, run below command
```sh
$ kubectl krew upgrade directpv
```

### Upgrade of release binary
Refer to the [binary installation documentation](./installation.md#installation-of-release-binary).
