Drive replacement (applies for versions > v3.2.2)
-------------

These are the steps to move the volumes from one drive to another drive from the **same node**. These steps will be useful during drive replacement.

(NOTE: This will just move the volume references from source to the destination drive without moving the actual data)

## STEP-BY-STEP Guide for drive replacement

### STEP 1: Add new drive

Once the new drive has been attached to the node as a replacement of an older drive, Discover and initialize the newly added drive by executing the following steps

```sh
$ kubectl directpv discover --drives /dev/<drive-name> --nodes <node-name>
```

This command will generate `drives.yaml` which will be used later for initialization. Please refer [Discover CLI](./cli.md#discover-drives) for more helpers on discovery.

Now, initialize the drive using the `drives.yaml` file that was created by the previous command

```sh
$ kubectl directpv init drives.yaml
```

On success, you should see the new drive listed in `kubectl directpv list drives --drives /dev/<drive-name>`

### STEP 2: Get source and destination drive IDs

```sh
$ kubectl directpv list drives --drives <source-drive-name>,<dest-drive-name> --nodes <node-name> -o wide
```

### STEP 3: Cordon source and destination drives

By cordoning, we mark the drives as unschedulable so that no new volumes will be scheduled on these drives

```sh
$ kubectl directpv cordon <source-drive-id> <dest-drive-id>
```

### STEP 4: Get the pods used by the source drive

```sh
$ kubectl directpv list volumes --drives /dev/<source-drive-name> --nodes <node-name>
```

### STEP 5: Cordon the corresponding kubernetes node and delete the pods used by the source drive

Cordon the corresponding k8s node and delete the corresponding workload pods used by the source drive, this will take the workloads offline and also doesn't schedule any new pods on this node.

```sh
$ kubectl cordon <node-name>
```

(NOTE: Avoid force deleting here as force deletion might not unpublish the volumes)

```sh
$ kubectl delete pod/<pods-name> -n <pod-ns>
```

This pod will fall in "pending" state after deleting as the node is cordoned

### STEP 6: Wait for associated DirectPV volumes to be unbounded

There shouldn't be any volumes in "Bounded" state

```sh
$ kubectl directpv list volumes --pod-names=<pod-name> --pod-namespaces=<pod-ns>
```

### STEP 7: Run move command

```sh
$ kubectl directpv move <source-drive-id> <dest-drive-id>
```

After moving, list the drives to see if the volumes got moved to destination 

```
$ kubectl directpv list drives --drives <source-drive-name>,<dest-drive-name> --nodes <node-name> -o wide
```

### STEP 8: Uncordon the destination drive

```sh
$ kubectl directpv uncordon <dest-drive-id>
```

### STEP 9: Uncordon the kubernetes node

```sh
$ kubectl uncordon <node-name>
```

Now, the corresponding workload pods should be coming up as the node is uncordoned. This pod will be using the new drive.

### STEP 10 (Optional): You can remove the old drive which was replaced

This will release the drive as it is not used anymore

```sh
$ kubectl directpv remove <source-drive-id>
```

## Bash script for drive replacement

Alternatively, after the "STEP 2" in the previous section, you can use this [bash script](./tools/replace.sh) to move the drives.

#### Example Usage

```sh
./replace.sh <source-drive-id> <dest-drive-id>
```
