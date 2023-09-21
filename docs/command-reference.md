# Command reference

## Prerequisites
* Working DirectPV plugin. To install the plugin, refer to the [plugin installation guide](./installation.md#directpv-plugin-installation).
* Working DirectPV CSI driver in Kubernetes. To install the driver, refer to the [driver installation guide](./installation.md#directpv-csi-driver-installation).

## Note
The DirectPV plugin command is referred to as `kubectl directpv` in this document. If you are using the direct binary instead, replace with `kubectl-directpv` throughout.

## Command changes from DirectCSI
| DirectCSI command                | DirectPV command                                          |
|:---------------------------------|:----------------------------------------------------------|
| `kubectl directcsi drives list`  | `kubectl directpv list drives`                            |
| `kubectl directcsi volumes list` | `kubectl directpv list volumes`                           |
| `kubectl directcsi format`       | `kubectl directpv discover`, then `kubectl directpv init` |

## Global flags
All of the plugin commands can use the following global flags.

| Flag           | Argument | Description                                         |
|:---------------|:---------|:----------------------------------------------------|
| `--kubeconfig` | _string_ | Path to the kubeconfig file to use for CLI requests |
| `--quiet`      | -        | Suppress printing error messages                    |
| `-h`, `--help` | -        | help for directpv                                   |
| `--version`    | -        | version for directpv                                |
                     
## Commands
List of subcommands are below

| Subcommand  | Description                                                                       |
|:------------|:----------------------------------------------------------------------------------|
| `install`   | Install DirectPV in Kubernetes                                                    |
| `discover`  | Discover new drives                                                               |
| `init`      | Initialize the drives                                                             |
| `info`      | Show information about DirectPV installation                                      |
| `list`      | List drives and volumes                                                           |
| `label`     | Set labels to drives and volumes                                                  |
| `cordon`    | Mark drives as unschedulable                                                      |
| `uncordon`  | Mark drives as schedulable                                                        |
| `migrate`   | Migrate drives and volumes from legacy DirectCSI                                  |
| `move`      | Move volumes excluding data from source drive to destination drive on a same node |
| `clean`     | Cleanup stale volumes                                                             |
| `suspend`   | Suspend drives and volumes                                                        |
| `resume`    | Resume suspended drives and volumes                                               |
| `remove`    | Remove unused drives from DirectPV                                                |
| `uninstall` | Uninstall DirectPV in Kubernetes                                                  |

## `install` command
```
Install DirectPV in Kubernetes

USAGE:
  directpv install [flags]

FLAGS:
      --node-selector strings        Select the storage nodes using labels (KEY=VALUE,..)
      --tolerations strings          Set toleration labels on the storage nodes (KEY[=VALUE]:EFFECT,..)
      --registry string              Name of container registry (default "quay.io")
      --org string                   Organization name in the registry (default "minio")
      --image string                 Name of the DirectPV image (default "directpv:v4.0.6")
      --image-pull-secrets strings   Image pull secrets for DirectPV images (SECRET1,..)
      --apparmor-profile string      Set path to Apparmor profile
      --seccomp-profile string       Set path to Seccomp profile
  -o, --output string                Generate installation manifest. One of: yaml|json
      --kube-version string          Select the kubernetes version for manifest generation (default "1.27.0")
      --legacy                       Enable legacy mode (Used with '-o')
      --openshift                    Use OpenShift specific installation
  -h, --help                         help for install

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Install DirectPV
   $ kubectl directpv install

2. Pull images from private registry (eg, private-registry.io/org-name) for DirectPV installation
   $ kubectl directpv install --registry private-registry.io --org org-name

3. Specify '--node-selector' to deploy DirectPV daemonset pods only on selective nodes
   $ kubectl directpv install --node-selector node-label-key=node-label-value

4. Specify '--tolerations' to tolerate and deploy DirectPV daemonset pods on tainted nodes (Example: key=value:NoSchedule)
   $ kubectl directpv install --tolerations key=value:NoSchedule

5. Generate DirectPV installation manifest in YAML
   $ kubectl directpv install -o yaml > directpv-install.yaml

6. Install DirectPV with apparmor profile
   $ kubectl directpv install --apparmor-profile directpv

7. Install DirectPV with seccomp profile
   $ kubectl directpv install --seccomp-profile profiles/seccomp.json
```

## `discover` command
```
Discover new drives

USAGE:
  directpv discover [flags]

FLAGS:
  -n, --nodes strings        discover drives from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings       discover drives by given names; supports ellipses pattern e.g. sd{a...z}
      --all                  If present, include non-formattable devices in the display
      --output-file string   output file to write the init config (default "drives.yaml")
      --timeout duration     specify timeout for the discovery process (default 2m0s)
  -h, --help                 help for discover

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Discover drives
   $ kubectl directpv discover

2. Discover drives from a node
   $ kubectl directpv discover --nodes=node1

3. Discover a drive from all nodes
   $ kubectl directpv discover --drives=nvme1n1

4. Discover all drives from all nodes (including unavailable)
   $ kubectl directpv discover --all

5. Discover specific drives from specific nodes
   $ kubectl directpv discover --nodes=node{1...4} --drives=sd{a...f}
```

## `init` command
```
Initialize the drives

USAGE:
  directpv init drives.yaml [flags]

FLAGS:
      --timeout duration   specify timeout for the initialization process (default 2m0s)
      --dangerous          Perform initialization of drives which will permanently erase existing data
  -h, --help               help for init

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Initialize the drives
   $ kubectl directpv init drives.yaml
```

## `info` command
```
Show information about DirectPV installation

USAGE:
  directpv info [flags]

FLAGS:
  -h, --help   help for info

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages
```

## `list` command
```
List drives and volumes

USAGE:
  directpv list [command]

FLAGS:
  -n, --nodes strings    Filter output by nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings   Filter output by drive names; supports ellipses pattern e.g. sd{a...z}
  -o, --output string    Output format. One of: json|yaml|wide
      --no-headers       When using the default or custom-column output format, don't print headers (default print headers)
  -h, --help             help for list

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

AVAILABLE COMMANDS:
  drives      List drives
  volumes     List volumes

Use "directpv list [command] --help" for more information about this command.
```

### `drives` command
```
List drives

USAGE:
  directpv list drives [DRIVE ...] [flags]

ALIASES:
  drives, drive, dr

FLAGS:
      --status strings   Filter output by drive status; one of: error|lost|moving|ready|removed
      --show-labels      show all labels as the last column (default hide labels column)
      --labels strings   Filter output by drive labels; supports comma separated kv pairs. e.g. tier=hot,region=east
      --all              If present, list all drives
  -h, --help             help for drives

GLOBAL FLAGS:
  -n, --nodes strings       Filter output by nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings      Filter output by drive names; supports ellipses pattern e.g. sd{a...z}
  -o, --output string       Output format. One of: json|yaml|wide
      --no-headers          When using the default or custom-column output format, don't print headers (default print headers)
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. List all ready drives
   $ kubectl directpv list drives

2. List all drives from a node
   $ kubectl directpv list drives --nodes=node1

3. List a drive from all nodes
   $ kubectl directpv list drives --drives=nvme1n1

4. List specific drives from specific nodes
   $ kubectl directpv list drives --nodes=node{1...4} --drives=sd{a...f}

5. List drives are in 'error' status
   $ kubectl directpv list drives --status=error

6. List all drives from all nodes with all information.
   $ kubectl directpv list drives --output wide

7. List drives with labels.
   $ kubectl directpv list drives --show-labels

8. List drives filtered by labels
   $ kubectl directpv list drives --labels tier=hot
```

### `volumes` command
```
List volumes

USAGE:
  directpv list volumes [VOLUME ...] [flags]

ALIASES:
  volumes, volume, vol

FLAGS:
      --drive-id strings         Filter output by drive IDs
      --pod-names strings        Filter output by pod names; supports ellipses pattern e.g. minio-{0...4}
      --pod-namespaces strings   Filter output by pod namespaces; supports ellipses pattern e.g. tenant-{0...3}
      --pvc                      Add PVC names in the output
      --status strings           Filter output by volume status; one of: pending|ready
      --show-labels              show all labels as the last column (default hide labels column)
      --labels strings           Filter output by volume labels; supports comma separated kv pairs. e.g. tier=hot,region=east
      --all                      If present, list all volumes
  -h, --help                     help for volumes

GLOBAL FLAGS:
  -n, --nodes strings       Filter output by nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings      Filter output by drive names; supports ellipses pattern e.g. sd{a...z}
  -o, --output string       Output format. One of: json|yaml|wide
      --no-headers          When using the default or custom-column output format, don't print headers (default print headers)
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. List all ready volumes
   $ kubectl directpv list volumes

2. List volumes served by a node
   $ kubectl directpv list volumes --nodes=node1

3. List volumes served by drives on nodes
   $ kubectl directpv list volumes --nodes=node1,node2 --drives=nvme0n1

4. List volumes by pod name
   $ kubectl directpv list volumes --pod-names=minio-{1...3}

5. List volumes by pod namespace
   $ kubectl directpv list volumes --pod-namespaces=tenant-{1...3}

6. List all volumes from all nodes with all information include PVC name.
   $ kubectl directpv list drives --all --pvc --output wide

7. List volumes in Pending state
   $ kubectl directpv list volumes --status=pending

8. List volumes served by a drive ID
   $ kubectl directpv list volumes --drive-id=b84758b0-866f-4a12-9d00-d8f7da76ceb3

9. List volumes with labels.
   $ kubectl directpv list volumes --show-labels

10. List volumes filtered by labels
   $ kubectl directpv list volumes --labels tier=hot
```

## `label` command
```
Set labels to drives and volumes

USAGE:
  directpv label [command]

FLAGS:
  -n, --nodes strings    If present, filter objects from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings   If present, filter objects by given drive names; supports ellipses pattern e.g. sd{a...z}
      --all              If present, select all objects
      --dry-run          Run in dry run mode
  -h, --help             help for label

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

AVAILABLE COMMANDS:
  drives      Set labels to drives
  volumes     Set labels to volumes

Use "directpv label [command] --help" for more information about this command.
```

### `drives` command
```
Set labels to drives

USAGE:
  directpv label drives k=v|k- [flags]

ALIASES:
  drives, drive, dr

FLAGS:
      --status strings   If present, select drives by status; one of: error|lost|moving|ready|removed
      --ids strings      If present, select by drive ID
      --labels strings   If present, select by drive labels; supports comma separated kv pairs. e.g. tier=hot,region=east
  -h, --help             help for drives

GLOBAL FLAGS:
  -n, --nodes strings       If present, filter objects from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings      If present, filter objects by given drive names; supports ellipses pattern e.g. sd{a...z}
      --all                 If present, select all objects
      --dry-run             Run in dry run mode
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Set 'tier: hot' label to all drives in all nodes
   $ kubectl directpv label drives tier=hot --all

2. Set 'type: fast' to specific drives from a node
   $ kubectl directpv label drives type=fast --nodes=node1 --drives=nvme1n{1...3}

3. Remove 'tier: hot' label from all drives in all nodes
   $ kubectl directpv label drives tier- --all
```

### `volumes` command
```
Set labels to volumes

USAGE:
  directpv label volumes k=v|k- [flags]

ALIASES:
  volumes, volume, vol

FLAGS:
      --drive-id strings         Filter output by drive IDs
      --pod-names strings        Filter output by pod names; supports ellipses pattern e.g. minio-{0...4}
      --pod-namespaces strings   Filter output by pod namespaces; supports ellipses pattern e.g. tenant-{0...3}
      --status strings           Filter output by volume status; one of: pending|ready
      --labels strings           If present, select by volume labels; supports comma separated kv pairs. e.g. tier=hot,region=east
      --ids strings              If present, select by volume ID
  -h, --help                     help for volumes

GLOBAL FLAGS:
  -n, --nodes strings       If present, filter objects from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings      If present, filter objects by given drive names; supports ellipses pattern e.g. sd{a...z}
      --all                 If present, select all objects
      --dry-run             Run in dry run mode
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Set 'tier: hot' label to all volumes in all nodes
   $ kubectl directpv label volumes tier=hot --all

2. Set 'type: fast' to volumes allocated in specific drives from a node
   $ kubectl directpv label volumes type=fast --nodes=node1 --drives=nvme1n{1...3}

3. Remove 'tier: hot' label from all volumes in all nodes
   $ kubectl directpv label volumes tier- --all
```

## `cordon` command
```
Mark drives as unschedulable

USAGE:
  directpv cordon [DRIVE ...] [flags]

FLAGS:
  -n, --nodes strings    If present, select drives from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings   If present, select drives by given names; supports ellipses pattern e.g. sd{a...z}
      --status strings   If present, select drives by drive status; one of: error|lost|moving|ready|removed
      --all              If present, select all drives
      --dry-run          Run in dry run mode
  -h, --help             help for cordon

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Cordon all drives from all nodes
   $ kubectl directpv cordon --all

2. Cordon all drives from a node
   $ kubectl directpv cordon --nodes=node1

3. Cordon a drive from all nodes
   $ kubectl directpv cordon --drives=nvme1n1

4. Cordon specific drives from specific nodes
   $ kubectl directpv cordon --nodes=node{1...4} --drives=sd{a...f}

5. Cordon drives which are in 'error' status
   $ kubectl directpv cordon --status=error
```

## `uncordon` command
```
Mark drives as schedulable

USAGE:
  directpv uncordon [DRIVE ...] [flags]

FLAGS:
  -n, --nodes strings    If present, select drives from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings   If present, select drives by given names; supports ellipses pattern e.g. sd{a...z}
      --status strings   If present, select drives by status; one of: error|lost|moving|ready|removed
      --all              If present, select all drives
      --dry-run          Run in dry run mode
  -h, --help             help for uncordon

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Uncordon all drives from all nodes
   $ kubectl directpv uncordon --all

2. Uncordon all drives from a node
   $ kubectl directpv uncordon --nodes=node1

3. Uncordon a drive from all nodes
   $ kubectl directpv uncordon --drives=nvme1n1

4. Uncordon specific drives from specific nodes
   $ kubectl directpv uncordon --nodes=node{1...4} --drives=sd{a...f}

5. Uncordon drives which are in 'error' status
   $ kubectl directpv uncordon --status=error
```

## `migrate` command
```
Migrate drives and volumes from legacy DirectCSI

USAGE:
  directpv migrate [flags]

FLAGS:
      --dry-run   Run in dry run mode
  -h, --help      help for migrate

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Migrate drives and volumes from legacy DirectCSI
   $ kubectl directpv migrate
```

## `move` command
```
Move volumes excluding data from source drive to destination drive on a same node

USAGE:
  directpv move SRC-DRIVE DEST-DRIVE [flags]

ALIASES:
  move, mv

FLAGS:
  -h, --help   help for move

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Move volumes from drive af3b8b4c-73b4-4a74-84b7-1ec30492a6f0 to drive 834e8f4c-14f4-49b9-9b77-e8ac854108d5
   $ kubectl directpv drives move af3b8b4c-73b4-4a74-84b7-1ec30492a6f0 834e8f4c-14f4-49b9-9b77-e8ac854108d5
```

## `clean` command
```
Cleanup stale volumes

USAGE:
  directpv clean [VOLUME ...] [flags]

FLAGS:
  -n, --nodes strings            If present, select volumes from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings           If present, select volumes by given drive names; supports ellipses pattern e.g. sd{a...z}
      --all                      If present, select all volumes
      --dry-run                  Run in dry run mode
      --drive-id strings         Select volumes by drive IDs
      --pod-names strings        Select volumes by pod names; supports ellipses pattern e.g. minio-{0...4}
      --pod-namespaces strings   Select volumes by pod namespaces; supports ellipses pattern e.g. tenant-{0...3}
  -h, --help                     help for clean

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Cleanup all stale volumes
   $ kubectl directpv clean --all

2. Clean a volume by its ID
   $ kubectl directpv clean pvc-6355041d-f9c6-4bd6-9335-f2bccbe73929

3. Clean volumes served by drive name in all nodes.
   $ kubectl directpv clean --drives=nvme1n1

4. Clean volumes served by drive
   $ kubectl directpv clean --drive-id=78e6486e-22d2-4c93-99d0-00f4e3a8411f

5. Clean volumes served by a node
   $ kubectl directpv clean --nodes=node1

6. Clean volumes by pod name
   $ kubectl directpv clean --pod-names=minio-{1...3}

7. Clean volumes by pod namespace
   $ kubectl directpv clean --pod-namespaces=tenant-{1...3}
```

## `suspend` command
```
Suspend drives and volumes

USAGE:
  directpv suspend [command]

FLAGS:
      --dry-run   Run in dry run mode
  -h, --help      help for suspend

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

AVAILABLE COMMANDS:
  drives      Suspend drives
  volumes     Suspend volumes

Use "directpv suspend [command] --help" for more information about this command.
```

### `drives` command
```
Suspend the drives (CAUTION: This will make the corresponding volumes as read-only)

USAGE:
  directpv suspend drives [DRIVE ...] [flags]

FLAGS:
  -n, --nodes strings    If present, suspend drives from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings   If present, suspend drives by given names; supports ellipses pattern e.g. sd{a...z}
      --dangerous        Suspending the drives will make the corresponding volumes as read-only
  -h, --help             help for drives

GLOBAL FLAGS:
      --dry-run             Run in dry run mode
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Suspend all drives from a node
   $ kubectl directpv suspend drives --nodes=node1

2. Suspend specific drive from specific node
   $ kubectl directpv suspend drives --nodes=node1 --drives=sda

3. Suspend a drive by its DRIVE-ID 'af3b8b4c-73b4-4a74-84b7-1ec30492a6f0'
   $ kubectl directpv suspend drives af3b8b4c-73b4-4a74-84b7-1ec30492a6f0
```

### `volumes` command
```
Suspend the volumes (CAUTION: This will make the corresponding volumes as read-only)

USAGE:
  directpv suspend volumes [VOLUME ...] [flags]

FLAGS:
  -n, --nodes strings            If present, suspend volumes from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings           If present, suspend volumes by given drive names; supports ellipses pattern e.g. sd{a...z}
      --pod-names strings        If present, suspend volumes by given pod names; supports ellipses pattern e.g. minio-{0...4}
      --pod-namespaces strings   If present, suspend volumes by given pod namespaces; supports ellipses pattern e.g. tenant-{0...3}
      --dangerous                Suspending the volumes will make them as read-only
  -h, --help                     help for volumes

GLOBAL FLAGS:
      --dry-run             Run in dry run mode
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Suspend all volumes from a node
   $ kubectl directpv suspend volumes --nodes=node1

2. Suspend specific volume from specific node
   $ kubectl directpv suspend volumes --nodes=node1 --volumes=sda

3. Suspend a volume by its name 'pvc-0700b8c7-85b2-4894-b83a-274484f220d0'
   $ kubectl directpv suspend volumes pvc-0700b8c7-85b2-4894-b83a-274484f220d0
```

## `resume` command
```
Resume suspended drives and volumes

USAGE:
  directpv resume [command]

FLAGS:
      --dry-run   Run in dry run mode
  -h, --help      help for resume

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

AVAILABLE COMMANDS:
  drives      Resume suspended drives
  volumes     Resume suspended volumes

Use "directpv resume [command] --help" for more information about this command.
```

### `drives` command
```
Resume suspended drives

USAGE:
  directpv resume drives [DRIVE ...] [flags]

FLAGS:
  -n, --nodes strings    If present, resume drives from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings   If present, resume drives by given names; supports ellipses pattern e.g. sd{a...z}
  -h, --help             help for drives
      --dry-run          Run in dry run mode

GLOBAL FLAGS:
      --dry-run             Run in dry run mode
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Resume all suspended drives from a node
   $ kubectl directpv resume drives --nodes=node1

2. Resume specific drive from specific node
   $ kubectl directpv resume drives --nodes=node1 --drives=sda

3. Resume a suspended drive by its DRIVE-ID 'af3b8b4c-73b4-4a74-84b7-1ec30492a6f0'
   $ kubectl directpv resume drives af3b8b4c-73b4-4a74-84b7-1ec30492a6f0
```

### `volumes` command
```
Resume suspended volumes

USAGE:
  directpv resume volumes [VOLUME ...] [flags]

FLAGS:
  -n, --nodes strings            If present, resume volumes from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings           If present, resume volumes by given drive names; supports ellipses pattern e.g. sd{a...z}
      --pod-names strings        If present, resume volumes by given pod names; supports ellipses pattern e.g. minio-{0...4}
      --pod-namespaces strings   If present, resume volumes by given pod namespaces; supports ellipses pattern e.g. tenant-{0...3}
  -h, --help                     help for volumes

GLOBAL FLAGS:
      --dry-run             Run in dry run mode
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Resume all volumes from a node
   $ kubectl directpv resume volumes --nodes=node1

2. Resume specific volume from specific node
   $ kubectl directpv resume volumes --nodes=node1 --volumes=sda

3. Resume a volume by its name 'pvc-0700b8c7-85b2-4894-b83a-274484f220d0'
   $ kubectl directpv resume volumes pvc-0700b8c7-85b2-4894-b83a-274484f220d0
```

## `remove` command
```
Remove unused drives from DirectPV

USAGE:
  directpv remove [DRIVE ...] [flags]

FLAGS:
  -n, --nodes strings    If present, select drives from given nodes; supports ellipses pattern e.g. node{1...10}
  -d, --drives strings   If present, select drives by given names; supports ellipses pattern e.g. sd{a...z}
      --status strings   If present, select drives by drive status; one of: error|lost|moving|ready|removed
      --all              If present, select all unused drives
      --dry-run          Run in dry run mode
  -h, --help             help for remove

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages

EXAMPLES:
1. Remove an unused drive from all nodes
   $ kubectl directpv remove --drives=nvme1n1

2. Remove all unused drives from a node
   $ kubectl directpv remove --nodes=node1

3. Remove specific unused drives from specific nodes
   $ kubectl directpv remove --nodes=node{1...4} --drives=sd{a...f}

4. Remove all unused drives from all nodes
   $ kubectl directpv remove --all

5. Remove drives are in 'error' status
   $ kubectl directpv remove --status=error
```

## `uninstall` command
```
Uninstall DirectPV in Kubernetes

USAGE:
  directpv uninstall [flags]

FLAGS:
  -h, --help   help for uninstall

GLOBAL FLAGS:
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests
      --quiet               Suppress printing error messages
```
