# Container Storage Interface (CSI) driver for Direct Volume Access ![Go](https://github.com/minio/direct-csi/workflows/Go/badge.svg)
This repository provides tools and scripts for building and testing the DIRECT CSI provider.

## Steps to run

```sh
# Install the driver
$> kubectl apply -k github.com/minio/direct-csi

# Create a storage topology
$> cat << EOF > storage_topology.yaml
storage_topology:
   -  name: direct.csi.min.io # This field will later end up being the name of the storage class
      layout: host{1...4}/dev/nvme0n{1...4} 
      fstype: xfs
      mount_options: 
      - ro
      - atime
      resource_limit: 
         storage: 10TiB
         volumes: 1000
      reclaim_policy: Retain|Delete
      selectors:
        nodes: 
        - host{1...4}
EOF

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

## License
Use of DIRECT CSI driver is governed by the AGPLv3 license that can be found in the [LICENSE](./LICENSE) file.
