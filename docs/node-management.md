# Node management

## Prerequisites
* Working DirectPV plugin. To install the plugin, refer [installation guide](./installation.md#directpv-plugin-installation).
* Working DirectPV CSI driver in Kubernetes. To install the driver, refer [installation guide](./installation.md#directpv-csi-driver-installation).

## Add node
After Adding a node into DirectPV DaemonSet, run DirectPV plugin `discover` command. Below is an example
```sh
$ kubectl directpv discover
```

## List node
Run DirectPV plugin `info` command to get list of nodes. Below is an example
```sh
$ kubectl directpv info
┌──────────┬──────────┬───────────┬─────────┬────────┐
│ NODE     │ CAPACITY │ ALLOCATED │ VOLUMES │ DRIVES │
├──────────┼──────────┼───────────┼─────────┼────────┤
│ • master │ 512 MiB  │ 32 MiB    │ 2       │ 1      │
│ • node1  │ 512 MiB  │ 32 MiB    │ 2       │ 1      │
└──────────┴──────────┴───────────┴─────────┴────────┘

64 MiB/1.0 GiB used, 4 volumes, 2 drives

```

Refer [info command](./command-reference.md#info-command) for more information.

## Delete node
***CAUTION: THIS IS DANGEROUS OPERATION WHICH LEADS TO DATA LOSS***

Before removing a node make sure no volumes or drives on the node are in us, then remove the node from DirectPV DaemonSet and run [remove-node.sh](./tools/remove-node.sh) script.
