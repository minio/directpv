# Direct CSI 

![build](https://github.com/minio/direct-csi/workflows/Go/badge.svg) ![license](https://img.shields.io/badge/license-AGPL%20V3-blue)

Direct CSI is a driver to allocate volumes for pods that require _direct access_ to storage media (eg. MinIO). It maintains a global view of storage in the cluster, and directs pods to run on nodes where volume is provisioned. Each volume is a subdirectory carved out of available drives on the node and mounted into pods.

## Getting Started

###### Install Direct-CSI plugin

```bash
curl -sfL get.direct-csi.com | sh -
```

This will install direct-csi plugin

###### Install Direct-CSI driver

Use the plugin to install the driver

```bash
kubectl direct-csi install --crd 
```

###### Format Drives intended for DirectCSI

Choose drives to be managed by DirectCSI. It is first formatted before use. Direct-CSI automatically ignores any root ('/') mounts

```bash
kubectl direct-csi drives format /dev/xvd* --nodes myhost{1...4}
```

## Make a Persistent Volume Claim

Provision a Direct-CSI volume by specifying `volumeClaimTemplates`:

###### Example

```yaml
volumeClaimTemplates:
  - metadata:
    name: myvol1
  spec:
    accessModes: [ "ReadWriteOnce" ]
    resources:
      requests:
        storage: 500G
    storageClassName: direct.csi.min.io
```

## License

Use of `direct-csi` driver is governed by the GNU AGPLv3 license that can be found in the [LICENSE](./LICENSE) file.
