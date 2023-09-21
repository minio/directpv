# Drive management

## Prerequisites
* Working DirectPV plugin. To install the plugin, refer to the [plugin installation guide](./installation.md#directpv-plugin-installation).
* Working DirectPV CSI driver in Kubernetes. To install the driver, refer to the [driver installation guide](./installation.md#directpv-csi-driver-installation).

## Add drives
Drives are added to DirectPV to provision volumes. This involves a two step process as shown below.

1. Run `discover` command.
The `discover` command probes eligible drives from DirectPV nodes and stores drive information in a YAML file. You should carefully examine the YAML file and set the `select` field to `yes` or `no` value to indicate drive selection. `select` field is set to `yes` value by default. Below is an example of the `discover` command:

```sh
# Probe and save drive information to drives.yaml file.
$ kubectl directpv discover

 Discovered node 'master' ✔
 Discovered node 'node1' ✔

┌─────────────────────┬────────┬───────┬─────────┬────────────┬──────┬───────────┬─────────────┐
│ ID                  │ NODE   │ DRIVE │ SIZE    │ FILESYSTEM │ MAKE │ AVAILABLE │ DESCRIPTION │
├─────────────────────┼────────┼───────┼─────────┼────────────┼──────┼───────────┼─────────────┤
│ 252:16$ud8mwCjPT... │ master │ vdb   │ 512 MiB │ -          │ -    │ YES       │ -           │
│ 252:16$gGz4UIuBj... │ node1  │ vdb   │ 512 MiB │ -          │ -    │ YES       │ -           │
└─────────────────────┴────────┴───────┴─────────┴────────────┴──────┴───────────┴─────────────┘

Generated 'drives.yaml' successfully.

# Show generated drives.yaml file.
$ cat drives.yaml
version: v1
nodes:
    - name: master
      drives:
        - id: 252:16$ud8mwCjPTH8147TysmiQ2GGLpffqUht6bz7NtHqReJo=
          name: vdb
          size: 536870912
          make: ""
          select: "yes"
    - name: node1
      drives:
        - id: 252:16$gGz4UIuBjQlO1KibOv7bZ+kEDk3UCeBneN/UJdqdQl4=
          name: vdb
          size: 536870912
          make: ""
          select: "yes"

```

2. Run `init` command.
The `init` command creates a request to add the selected drives in the YAML file generated using the `discover` command. As this process wipes out all data on the selected drives, wrong drive selection will lead to permanent data loss. Below is an example of `init` command:

```sh
$ kubectl directpv init drives.yaml

 ███████████████████████████████████████████████████████████████████████████ 100%

 Processed initialization request 'c24e22f5-d582-49ba-a883-2ce56909904e' for node 'master' ✔
 Processed initialization request '7e38a453-88ed-412c-b146-03eef37b23bf' for node 'node1' ✔

┌──────────────────────────────────────┬────────┬───────┬─────────┐
│ REQUEST_ID                           │ NODE   │ DRIVE │ MESSAGE │
├──────────────────────────────────────┼────────┼───────┼─────────┤
│ c24e22f5-d582-49ba-a883-2ce56909904e │ master │ vdb   │ Success │
│ 7e38a453-88ed-412c-b146-03eef37b23bf │ node1  │ vdb   │ Success │
└──────────────────────────────────────┴────────┴───────┴─────────┘
```

Refer to the [discover command](./command-reference.md#discover-command) and the [init command](./command-reference.md#init-command) for more information.

## List drives
To get information of drives from DirectPV, run the `list drives` command. Below is an example:

```sh
$ kubectl directpv list drives
┌────────┬──────┬──────┬─────────┬─────────┬─────────┬────────┐
│ NODE   │ NAME │ MAKE │ SIZE    │ FREE    │ VOLUMES │ STATUS │
├────────┼──────┼──────┼─────────┼─────────┼─────────┼────────┤
│ master │ vdb  │ -    │ 512 MiB │ 506 MiB │ -       │ Ready  │
│ node1  │ vdb  │ -    │ 512 MiB │ 506 MiB │ -       │ Ready  │
└────────┴──────┴──────┴─────────┴─────────┴─────────┴────────┘
```

Refer to the [list drives command](./command-reference.md#drives-command) for more information.

## Label drives
Drives are labeled to set custom tagging which can be used in volume provisioning. Below is an example:
```sh
# Set label 'tier' key to 'hot' value.
$ kubectl directpv label drives tiet=hot

# Remove label 'tier'.
$ kubectl directpv label drives tier-
```

Refer to the [label drives command](./command-reference.md#drives-command-1) for more information.

## Replace drive
Replace a faulty drive with a new drive on a same node. In this process, all volumes in the faulty drive are moved to the new drive then faulty drive is removed from DirectPV. Currently, DirectPV does not support moving data on the volume to the new drive. Use [replace.sh](./tools/replace.sh) script to perform drive replacement. Below is an example:
```sh
# Replace 'sdd' drive by 'sdf' drive on 'node1' node
$ replace.sh sdd sdf node1
```

## Remove drives
Drives that do not contain any volumes can be removed. Below is an example:
```sh
# Remove drive 'vdb' from 'node1' node
$ kubectl directpv remove --drives=vdb --nodes=node1
```

Refer [remove command](./command-reference.md#remove-command) for more information.

## Suspend drives

***CAUTION: THIS IS DANGEROUS OPERATION WHICH LEADS TO DATA LOSS***

By Kubernetes design, [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) workload is active only if all of its pods are in running state. Any faulty drive(s) will prevent the statefulset from starting up. DirectPV provides a workaround to suspend failed drives which will mount the respective volumes on empty `/var/lib/directpv/tmp` directory with read-only access. This can be done by executing the `suspend drives` command. Below is an example:

```sh
> kubectl directpv suspend drives af3b8b4c-73b4-4a74-84b7-1ec30492a6f0
```

Suspended drives can be resumed once they are fixed. Upon resuming, the corresponding volumes will resume using the respective allocated drives. This can be done by using the `resume drives` command. Below is an example:

```sh
> kubectl directpv resume drives af3b8b4c-73b4-4a74-84b7-1ec30492a6f0
```
