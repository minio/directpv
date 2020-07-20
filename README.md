# Container Storage Interface (CSI) driver for DIRECTs ![Go](https://github.com/minio/direct-csi-driver/workflows/Go/badge.svg)
This repository provides tools and scripts for building and testing the DIRECT CSI provider.

## Steps to run

```sh
# set the environment variables
$> cat << EOF > default.env
DIRECT_CSI_DRIVER_PATHS=/var/lib/direct-csi-driver/data{1...4}
DIRECT_CSI_DRIVER_COMMON_CONTAINER_ROOT=/var/lib/direct-csi-driver
DIRECT_CSI_DRIVER_COMMON_HOST_ROOT=/var/lib/direct-csi-driver
EOF

$> export $(cat default.env)

# create the namespace for the driver
$> kubectl apply -k github.com/minio/direct-csi-driver

# utilize the volume in your application
#
#   ------------------------------------------------------------------------------------------------
#   volumeClaimTemplates: # This is the specification in which you reference the StorageClass
#     - metadata:
#       name: direct-csi-driver-min-io-volume
#     spec:
#       accessModes: [ "ReadWriteOnce" ]
#       resources:
#         requests:
#           storage: 10Gi
#       storageClassName: direct.csi.driver.min.io # This field references the existing StorageClass
#    -----------------------------------------------------------------------------------------------
#
# Example application in test-app.yaml
$> kubectl create -f test-app.yaml
```

## License
Use of DIRECT CSI driver is governed by the AGPLv3 license that can be found in the [LICENSE](./LICENSE) file.
