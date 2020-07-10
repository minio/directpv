# Container Storage Interface (CSI) driver for JBODs ![Go](https://github.com/minio/jbod-csi-driver/workflows/Go/badge.svg)
This repository provides tools and scripts for building and testing the JBOD CSI provider.

## Steps to run

```sh
# create the namespace for the driver
$> kubectl apply -f ns.yaml

# create a rbac role for the driver
$> kubectl apply -f rbac.yaml

# create the controller and node driver
$> kubectl apply -f deploy.yaml

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
