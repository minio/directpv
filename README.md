# Container Storage Interface (CSI) driver for JBODs ![Go](https://github.com/minio/jbod-csi-driver/workflows/Go/badge.svg)
This repository provides tools and scripts for building and testing the JBOD CSI provider.

## Steps to run

```sh
# set the environment variables
$> cat << EOF > defautl.env
JBOD_CSI_DRIVER_PATHS=/var/lib/jbod-csi-driver/data{1...4}
JBOD_CSI_DRIVER_COMMON_CONTAINER_ROOT=/var/lib/jbod-csi-driver
JBOD_CSI_DRIVER_COMMON_HOST_ROOT=/var/lib/jbod-csi-driver
EOF

# create the namespace for the driver
$> kubectl apply -k github.com/minio/jbod-csi-driver

# utilize the volume in your application
#
#   ------------------------------------------------------------------------------------------------
#   volumeClaimTemplates: # This is the specification in which you reference the StorageClass
#     - metadata:
#       name: jbod-csi-driver-min-io-volume
#     spec:
#       accessModes: [ "ReadWriteOnce" ]
#       resources:
#         requests:
#           storage: 10Gi
#       storageClassName: jbod.csi.driver.min.io # This field references the existing StorageClass
#    -----------------------------------------------------------------------------------------------
#
# Example application in test-app.yaml
$> kubectl create -f test-app.yaml
```

## License
Use of JBOD CSI driver is governed by the AGPLv3 license that can be found in the [LICENSE](./LICENSE) file.
