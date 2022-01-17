DirectPV
----------

DirectPV is a CSI driver for [Direct Attached Storage](https://en.wikipedia.org/wiki/Direct-attached_storage). In a simpler sense, it is a distributed persistent volume manager, and not a storage system like SAN or NAS. It is useful to *discover, format, mount, schedule and monitor* drives across servers. Since Kubernetes `hostPath` and `local` PVs are statically provisioned and limited in functionality, DirectPV was created to address this limitation. 

Distributed data stores such as object storage, databases and message queues are designed for direct attached storage, and they handle high availability and data durability by themselves. Running them on traditional SAN or NAS based CSI drivers (Network PV) adds yet another layer of replication/erasure coding and extra network hops in the data path. This additional layer of disaggregation results in increased-complexity and poor performance.

![Architecture Diagram](https://github.com/minio/directpv/blob/master/docs/images/architecture.png?raw=true)

### Architecture

DirectPV is designed to be lightweight and scalable to 10s of 1000s of drives. It is made up of three components - **Controller, Node Driver, UI**

![DirectPV Architecture](https://github.com/minio/directpv/blob/master/docs/images/directpv_architecture.png?raw=true)

##### Controller

When a volume claim is made, the controller provisions volumes uniformly from a pool free drives. DirectPV is aware of pod's affinity constraints, and allocates volumes from drives local to pods. Note that only one active instance of controller runs per cluster.

##### Node Driver

Node Driver implements the volume management functions such as discovery, format, mount, and monitoring of drives on the nodes. One instance of node driver runs on each of the storage servers. 

##### UI

Storage Administrators can use the kubectl CLI plugin to select, manage and monitor drives. Web based UI is currently under development. 

### Installation

```sh
# Install kubectl directpv plugin
kubectl krew install directpv

# Use the plugin to install directpv in your kubernetes cluster
kubectl directpv install

# Ensure directpv has successfully started
kubectl directpv info

# List available drives in your cluster
kubectl directpv drives ls

# Select drives that directpv should manage and format
kubectl directpv drives format --drives /dev/sd{a...f} --nodes directpv-{1...4}

# 'directpv' can now be specified as the storageclass in PodSpec.VolumeClaimTemplates
```

For air-gapped setups and advanced installations, please refer to the [Installation Guide](./docs/installation.md).

### Upgrade

DirectPV version upgrades are seameless and transparent. Simply uninstall an existing version of directpv and install with a newer version to upgrade.

```
# Uninstall directpv
kubectl directpv uninstall 

# Download latest krew plugin
kubectl krew upgrade directpv

# Install using new plugin
kubectl directpv install
```

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

- [MinIO Slack](https://join.slack.com/t/minio/shared_invite/zt-wjdzimbo-apoPb9jEi5ssl2iedx6MoA)

### License

DirectPV is released under GNU AGPLv3 license. Please refer to the LICENSE document for a complete copy of the license.
