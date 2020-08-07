# Direct-CSI ![Go](https://github.com/minio/direct-csi/workflows/Go/badge.svg)
Container Storage Interface (CSI) driver for Direct Attached Storage (DAS).

## Use case

This CSI driver is designed for any of the following usecases.

#### Distributed Storage (Databases & object storage)

Distributed storage services require direct access to the local drives for performance, availability and simplicity. As the industry moves towards software defined storage, running these distributed databases and storage systems like Elastic Search, Cassandra, MinIO etc. on top of SAN or NAS - i.e. networked storage is like turtles all the way down. 

These distributed data services handle resiliency and availability at the software layer, which is why they require local drives. Utilizing networked drives for these applications defeats that purpose  

#### Persistent Local Cache 

AI/ML and analytics applications such as Apache Spark can leverage local SSDs as a high performance caching tier using `direct-csi` driver.

## Steps to run

#### Install the driver

```sh
$> kubectl apply -k github.com/minio/direct-csi
```

#### Create a storage topology
```sh
$> cat << EOF > storage_topology.yaml
apiVersion: v1
kind: StorageTopology
name: direct.csi.min.io # This field will later end up being the name of the storage class
layout: 
  nodeLabels:
    key: value
  path: /dev/nvme0n{1...8}
fstype: xfs
mount_options:
  - rw
  - noatime
resourceLimit:
  storage: 10TiB
  volumes: 100
  reclaimPolicy: Retain|Delete
EOF
```

#### Utilize the volume in your application
```sh
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
```

#### Deploy MinIO backed by direct.csi.min.io
```sh
$> kubectl create -f minio.yaml
```

## Direct CSI vs. PersistentLocalVolumes vs. HostPath Volumes

TBD


## License
Use of DIRECT CSI driver is governed by the AGPLv3 license that can be found in the [LICENSE](./LICENSE) file.
