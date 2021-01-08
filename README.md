# Direct CSI 

![build](https://github.com/minio/direct-csi/workflows/Go/badge.svg) ![license](https://img.shields.io/badge/license-AGPL%20V3-blue)

Direct CSI is a driver to allocate volumes for pods that require _direct access_ to storage media (eg. MinIO). It maintains a global view of storage in the cluster, and directs pods to run on nodes where volume is provisioned. Each volume is a subdirectory carved out of available drives on the node and mounted into pods.

Visit our [documentation](https://direct-csi.github.io) for more information.

## Installation

###### Install Direct-CSI cli

```bash
kubectl krew install directcsi
```

###### Install Direct-CSI driver

```bash
kubectl directcsi install --crd 
```

###### Add Drives to DirectCSI pool

Choose drives to be managed by DirectCSI. Refer to [Add Drives](./docs/cli.md#add-drives) command for more info.

```bash
kubectl directcsi drives add /dev/nvme* --nodes myhost{1...4}
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

## References

 - [Documentation](https://direct-csi.github.io)
 - [CLI reference](./docs/cli.md)
 - [Architecture](./docs/arch.md) 
 - [Drives and Volumes](./docs/drives.md)

## License

Use of `directcsi` driver is governed by the GNU AGPLv3 license that can be found in the [LICENSE](./LICENSE) file.
