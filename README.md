DirectPV
----------

DirectPV is a CSI driver for [Direct Attached Storage](https://en.wikipedia.org/wiki/Direct-attached_storage). In a simpler sense, it is a distributed persistent volume manager, and not a storage system like SAN or NAS. It is useful to *discover, format, mount, schedule and monitor* drives across servers. Since Kubernetes `hostPath` and `local` PVs are statically provisioned and limited in functionality, DirectPV was created to address this limitation. 

Distributed data stores such as object storage, databases and message queues are designed for direct attached storage, and they handle high availability and data durability by themselves. Running them on traditional SAN or NAS based CSI drivers (Network PV) adds yet another layer of replication/erasure coding and extra network hops in the data path. This additional layer of disaggregation results in increased-complexity and poor performance.

![Architecture Diagram](https://github.com/minio/directpv/blob/master/docs/images/architecture.png?raw=true)

### Quickstart guide

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

4. Discover and add drives for volume scheduling.
```sh
# Discover drives to check the available devices in the cluster to initialize
# The following command will create an init config file (default: drives.yaml) which will be used for initialization
$ kubectl directpv discover

# Review the drives.yaml for drive selections and initialize those drives
$ kubectl directpv init drives.yaml
```

(NOTE: XFS is the filesystem used for formatting the drives here)

5. Get list of added drives
```sh
$ kubectl directpv list drives
```

6. Deploy a demo MinIO server

DirectPV enforces node constraints where it allocates storage based on the worker node where a pod deploys. If the pod deploys to a worker node with no or insufficient DirectPV-managed drives, DirectPV cannot allocate storage to that pod. DirectPV does not allocate storage from one node to a pod on another node.

Modify the YAML to reflect the node and storage distribution of your Kubernetes cluster.

```sh
# This should create MinIO pods and PVCs using the `directpv-min-io` storage class
$ kubectl apply -f functests/minio.yaml
```

For air-gapped setups and advanced installations, please refer to the [Installation Guide](./docs/installation.md).

### Upgrade from DirectPV v3.2.x

Firstly, it is required to uninstall older version of DirectPV. Once it is uninstalled, follow [Installation instructions](#Installation) to install the latest DirectPV. In this process, all existing drives and volumes will be migrated automatically.

For migrating from older versions < v3.2.0, Please refer the [Upgrade Guide](./docs/upgrade.md)

### Security

Please review the [security checklist](./security-checklist.md) before deploying to production.

**Important**: Report security issues to security@min.io. Please do not report security issues here.

### Additional Resources

- [Installation Guide](./docs/installation.md)
- [Upgrade Guide](./docs/upgrade.md)
- [CLI Guide](./docs/cli.md)
- [Security Guide](./docs/security.md)
- [Scheduling Guide](./docs/scheduling.md)
- [Drive Replacement Guide](./docs/drive-replacement.md)
- [Volume Expansion](./docs/volume-expansion.md)
- [Driver Specification](./docs/specification.md)
- [Monitoring & Metrics](./docs/metrics.md)
- [Developer Guide](./docs/development-and-testing.md)

### Join Community

DirectPV is a MinIO project. You can contact the authors over the slack channel:

- [MinIO Slack](https://slack.min.io/)

### License

DirectPV is released under GNU AGPLv3 license. Please refer to the LICENSE document for a complete copy of the license.
