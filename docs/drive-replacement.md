Drive replacement (applies for versions > v3.2.2)
-------------

These are the steps to move the volumes from one drive to another drive from the **same node**. These steps will be useful during drive replacement.

(NOTE: This will just move the volume references from source to the destination drive without moving the actual data)

## STEP-BY-STEP Guide for drive replacement

### STEP 1: Add new drive

Once the new drive has been attached to the node as a replacement of an older drive, Discover and initialize the newly added drive by executing the following steps

```sh
$ kubectl directpv discover --drives /dev/<drive-name>
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
$ kubectl directpv volumes list --all --drives /dev/<source-drive-name> --nodes <node-name>
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
$ kubectl directpv volumes list --all --pod-names=<pod-name> --pod-namespaces=<pod-ns>
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

Alternatively, after the "STEP 1" in the previous section, you can use the following bash script to move the drives.

```sh
#!/bin/bash

set -e

# usage: get_drive_id <node> <drive-name>
function get_drive_id() {
    kubectl get directpvdrives \
            --selector="directpv.min.io/node==${1},directpv.min.io/drive-name==${2}" \
            -o go-template='{{range .items}}{{.metadata.name}}{{end}}'
}

# usage: get_volumes <drive-id>
function get_volumes() {
    kubectl get directpvvolumes \
            --selector="directpv.min.io/drive=${1}" \
            -o go-template='{{range .items}}{{.metadata.name}}{{ " " | print }}{{end}}'
}

# usage: get_pod_name <volume>
function get_pod_name() {
    kubectl get directpvvolumes "${1}" \
            -o go-template='{{range $k,$v := .metadata.labels}}{{if eq $k "directpv.min.io/pod.name"}}{{$v}}{{end}}{{end}}'
}

# usage: get_pod_namespace <volume>
function get_pod_namespace() {
    kubectl get directpvvolumes "${1}" \
            -o go-template='{{range $k,$v := .metadata.labels}}{{if eq $k "directpv.min.io/pod.namespace"}}{{$v}}{{end}}{{end}}'
}

if [[ $# -eq 4 ]]; then
    echo "usage: replace <NODE> <SRC-DRIVE> <DEST-DRIVE>"
    exit 255
fi

node=$1

src_drive=${2#"/dev/"}
dest_drive=${3#"/dev/"}

echo -e "\n----------REPLACING DRIVE ${src_drive} with ${dest_drive} IN NODE ${node}----------\n"

# Get source drive ID
src_drive_id=$(get_drive_id "${node}" "${src_drive}")
if [ -z "${src_drive_id}" ]; then
    echo "source drive ${src_drive} on node ${node} not found"
    exit 1
fi

# Get destination drive ID
dest_drive_id=$(get_drive_id "${node}" "${dest_drive}")
if [ -z "${dest_drive_id}" ]; then
    echo "destination drive ${dest_drive} on node ${node} not found"
    exit 1
fi

# Cordon source and destination drives
if ! kubectl directpv cordon "${src_drive_id}" "${dest_drive_id}"; then
    echo "unable to cordon drives"
    exit 1
fi

echo -e "\n--> CORDONED THE SOURCE [${src_drive_id}] AND DESTINATION [${dest_drive_id}] DRIVES SUCCESSFULLY ✓\n"

# Cordon kubernetes node
if ! kubectl cordon "${node}"; then
    echo "unable to cordon node ${node}"
    exit 1
fi

echo -e "\n--> CORDONED THE NODE ${node} SUCCESSFULLY ✓\n"


volumes=( $(get_volumes "${src_drive_id}") )

if (( ${#volumes[@]} == 0 )); then
    echo "no volumes are found in source drive ${src_drive_id}"
    exit 1
fi

for volume in "${volumes[@]}"; do
    pod_name=$(get_pod_name "${volume}")
    pod_namespace=$(get_pod_namespace "${volume}")

    if ! kubectl delete pod "${pod_name}" --namespace "${pod_namespace}"; then
        echo "unable to delete pod ${pod_name} using volume ${volume}"
        exit 1
    fi
done

echo -e "\n--> DELETED THE PODS FROM THE SOURCE DRIVE ✓"


# Wait for associated DirectPV volumes to be unbound
while kubectl directpv list volumes --no-headers "${volumes[@]}" | grep -q Bounded; do
    echo "...waiting for volumes to be unbound"
done

echo -e "\n--> MOVING VOLUMES NOW...\n"

# Run move command
kubectl directpv move "${src_drive_id}" "${dest_drive_id}"

echo -e "\n--> MOVED THE VOLUMES ✓\n"

# Uncordon destination drive
kubectl directpv uncordon "${dest_drive_id}"

echo -e "\n--> UNCORDONED THE DESTINATION DRIVE ${dest_drive_id} ✓\n"

# Uncordon kubernetes node
kubectl uncordon "${node}"

echo -e "\n--> UNCORDONED THE NODE ${node} ✓"

echo -e "\n----------COMPLETED!!!----------\n"
```

#### Example Usage

```sh
./replace.sh <node-name> /dev/<source-device-name> /dev/<destination-device-name>
```
