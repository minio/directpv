DirectPV
----------

DirectPV is a CSI driver for [Direct Attached Storage](https://en.wikipedia.org/wiki/Direct-attached_storage). In a simpler sense, it is a distributed persistent volume manager, and not a storage system like SAN or NAS. It is useful to *discover, format, mount, schedule and monitor* drives across servers. Since Kubernetes `hostPath` and `local` PVs are statically provisioned and limited in functionality, DirectPV was created to address this limitation. 

Distributed data stores such as object storage, databases and message queues are designed for direct attached storage, and they handle high availability and data durability by themselves. Running them on traditional SAN or NAS based CSI drivers (Network PV) adds yet another layer of replication/erasure coding and extra network hops in the data path. This additional layer of disaggregation results in increased-complexity and poor performance.

![Architecture Diagram](https://github.com/minio/directpv/blob/master/docs/images/architecture.png?raw=true)

### Installation

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
# Discover drives
$ kubectl directpv discover --output-file drives.yaml

# Review drives.yaml for drive selections and initialize those drives
$ kubectl directpv init drives.yaml
```

5. Get list of added drives
```sh
$ kubectl directpv list drives
```

6. Deploy a demo MinIO server
```sh
$ kubectl apply -f functests/minio.yaml
```

For air-gapped setups and advanced installations, please refer to the [Installation Guide](./docs/installation.md).

### Upgrade from DirectPV v3.2.1

Firstly, it is required to uninstall older version of DirectPV. Once it is uninstalled, follow [Installation instructions](#Installation) to install the latest DirectPV. In this process, all existing drives and volumes will be migrated automatically.

### Security

Please review the [security checklist](./security-checklist.md) before deploying to production.

**Important**: Report security issues to security@min.io. Please do not report security issues here.

### Additional Resources

- [Developer Guide](./docs/development-and-testing.md)
- [Installation Guide](./docs/installation.md)
- [Monitoring & Metrics](./docs/metrics.md)
- [Security Guide](./docs/security.md)
- [Troubleshooting Guide](./docs/troubleshooting.md)

### Join Community

DirectPV is a MinIO project. You can contact the authors over the slack channel:

- [MinIO Slack](https://slack.min.io/)

### License

DirectPV is released under GNU AGPLv3 license. Please refer to the LICENSE document for a complete copy of the license.
