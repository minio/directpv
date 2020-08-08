# Container Storage Interface (CSI) driver for Direct Volume Access ![Go](https://github.com/minio/direct-csi/workflows/Go/badge.svg)
This repository provides tools and scripts for building and testing the DIRECT CSI provider.

## Steps to run

```sh
# Install the driver
$> DIRECT_CSI_DRIVES=data{1...4} DIRECT_CSI_DRIVES_DIR=/mnt KUBELET_DIR_PATH=/var/lib/kubelet kubectl apply -k github.com/minio/direct-csi

# utilize the volume in your application
#
#   ------------------------------------------------------------------------------------------------
#   volumeClaimTemplates: # This is the specification in which you reference the StorageClass
#     - metadata:
#       name: direct.csi-min-io-volume
#     spec:
#       accessModes: [ "ReadWriteOnce" ]
#       resources:
#         requests:
#           storage: 10Gi
#       storageClassName: direct.csi.min.io # This field references the existing StorageClass
#    -----------------------------------------------------------------------------------------------
#
# Deploy MinIO backed by direct.csi.min.io
$> kubectl create -f minio.yaml
```

If you are using microk8s `KUBELET_DIR_PATH` should be changed to `/var/snap/microk8s/common/var/lib/kubelet`

## License
Use of DIRECT CSI driver is governed by the AGPLv3 license that can be found in the [LICENSE](./LICENSE) file.
