# DirectPV

_This version `v4.0.x` entered maintenance mode on Jan 01, 2025._

[DirectPV](https://github.com/minio/directpv) is a [CSI](https://kubernetes.io/blog/2019/01/15/container-storage-interface-ga/) driver for [Direct Attached Storage](https://en.wikipedia.org/wiki/Direct-attached_storage). In a simpler sense, it is a distributed persistent volume manager, and not a storage system like SAN or NAS. It is useful to *discover, format, mount, schedule and monitor* drives across servers.

Distributed data stores such as object storage, databases and message queues are designed for direct attached storage, and they handle high availability and data durability by themselves. Running them on traditional SAN or NAS based CSI drivers (Network PV) adds yet another layer of replication/erasure coding and extra network hops in the data path. This additional layer of disaggregation results in increased-complexity and poor performance.

![Architecture Diagram](https://github.com/minio/directpv/blob/master/docs/images/architecture.png?raw=true)

## Quickstart

1. Install DirectPV Krew plugin
```sh
$ kubectl krew install directpv
```

2. Install DirectPV in your kubernetes cluster
```sh
$ kubectl directpv install
```

3. Get information of the installation
```sh
$ kubectl directpv info
```

4. Add drives
```sh
# Probe and save drive information to drives.yaml file.
$ kubectl directpv discover

# Initialize selected drives.
$ kubectl directpv init drives.yaml
```

5. Deploy a demo MinIO server
```sh
$ curl -sfL https://github.com/minio/directpv/raw/master/functests/minio.yaml | kubectl apply -f -
```

## Further information
Refer [detailed documentation](./docs/README.md)

## Join Community
DirectPV is a MinIO project. You can contact the authors over the [slack channel](https://slack.min.io/)

## Unsupported versions
* Versions `v1.x`, `v2.x` and `v3.x` of DirectCSI/DirectPV are marked end-of-life and unsupported.
* DirectPV version `v4.0.x` entered maintenance mode on Jan 01, 2025.

## License
DirectPV is released under GNU AGPLv3 license. Refer the [LICENSE document](https://github.com/minio/directpv/blob/master/LICENSE) for a complete copy of the license.
