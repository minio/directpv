Drive replacement (applies for versions >= v4.0.0)
-------------

These are the steps to move the volumes from one drive to another drive from the **same node**. These steps will be useful during drive replacement.

(NOTE: This will just move the volume references from source to the destination drive without moving the actual data)

### STEP 1: Discover and initialize the new drive

Once the new drive has been attached to the node as a replacement of an older drive, Discover and initialize the newly added drive by executing the following steps

```sh
$ kubectl directpv discover --drives /dev/<drive-name>
```

This command will generate `drives.yaml` which will be used later for initialization.

(NOTE: If you are not sure about the new drive name, you can also use `kubectl directpv discover` to see the available drives in the cluster and filter the drives.yaml to filter the right drive)

Now, initialize the drive using the `drives.yaml` file that was created by the previous command

```sh
$ kubectl directpv init drives.yaml
```

On success, you should see the new drive listed in `kubectl directpv list drives --drives /dev/<drive-name>`

### STEP 2: Cordon the both the source and destination drives which are used for this activity

By cordoning, we mark the drives as unschedulable so that no new volumes will be scheduled on these drives

#### STEP 2.1: Get the `DRIVE ID`'s of the source and destination drives by the following command

```sh
$ kubectl directpv list drives --drives <source-drive-name>,<dest-drive-name> --nodes <node-name> -o wide
```

#### STEP 2.2: Cordon both the source and destination drives by their `DRIVE ID`s

```sh
$ kubectl directpv cordon <source-drive-id> <dest-drive-id>
```

### STEP 3: Cordon the k8s node and delete the corresponding workload pod that is running on this node

Cordon the corresponding k8s node and delete the corresponding wokload pod, this will take the workloads offline and also doesn't schedule any new pods on this node.

#### STEP 3.1: Cordon the k8s node

```sh
$ kubectl cordon <node-name>
```

#### STEP 3.2: Delete the workload pod running in that node (This is to take the workload offline for the maintenance activity)

(NOTE: Avoid force deleting here as force deletion might not unpublish the volumes)

```sh
$ kubectl delete pod/<pods-name> -n <pod-ns>
```

This pod will fall in "pending" state after deleting as the node is cordoned

Also, check if the corresponding volumes are not in "Bounded" state (published volumes will be in "Bounded" state)

```sh
# NOTE: Give sometime for k8s to unpublish these volumes and then check with the following command
$ kubectl directpv volumes list --all --pod-names=<pod-name> --pod-namespaces=<pod-ns>
```

### STEP 4: Move the volume references from source drive to destination drive

Get the `DRIVE ID`s of the source and destination drives as mentioned in STEP 2.1

```sh
$ kubectl directpv move <source-drive-id> <dest-drive-id>
```

After moving, list the drives to see if the volumes got moved to destination 

```
$ kubectl directpv list drives --drives <source-drive-name>,<dest-drive-name> --nodes <node-name> -o wide
```

### STEP 5: Uncordon the drives and k8s node

#### STEP 5.1: Uncordon both the source and destination drives by their `DRIVE ID`s

```sh
$ kubectl directpv cordon <source-drive-id> <dest-drive-id>
```

#### STEP 5.2: Uncordon the corresponding k8s node

```sh
$ kubectl uncordon <node-name>
```

Now, the corresponding workload pods should be coming up as the node is uncordoned. This pod will be using the new drive.

### STEP 6 (Optional): You can remove the old drive which was replaced

This will release the drive as it is not used anymore

```sh
$ kubectl directpv remove --drives <source-drive-name> --nodes <node-name>
```
