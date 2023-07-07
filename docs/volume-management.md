# Volume management

## Prerequisites
* Working DirectPV plugin. To install the plugin, refer to the [plugin installation guide](./installation.md#directpv-plugin-installation).
* Working DirectPV CSI driver in Kubernetes. To install the driver, refer to the [driver installation guide](./installation.md#directpv-csi-driver-installation).
* Added drives in DirectPV. Refer to the [drive management guide](./drive-management.md).

## Add volume
Refer to the [volume provisioning guide](./volume-provisioning.md).

## List volume
To get information of volumes from DirectPV, run the `list volumes` command. Below is an example:

```sh
$ kubectl directpv list drives
┌────────┬──────┬──────┬─────────┬─────────┬─────────┬────────┐
│ NODE   │ NAME │ MAKE │ SIZE    │ FREE    │ VOLUMES │ STATUS │
├────────┼──────┼──────┼─────────┼─────────┼─────────┼────────┤
│ master │ vdb  │ -    │ 512 MiB │ 506 MiB │ -       │ Ready  │
│ node1  │ vdb  │ -    │ 512 MiB │ 506 MiB │ -       │ Ready  │
└────────┴──────┴──────┴─────────┴─────────┴─────────┴────────┘
```

Refer to the [list volumes command](./command-reference.md#volumes-command) for more information.

## Expand volume
DirectPV supports online volume expansion which does not require restart of pods using those volumes. This is automatically done after setting expanded size to `Persistent Volume Claim`. Below is an example:
```sh
# Get 'minio-data-1-minio-0' Persistent volume claim.
$ kubectl get pvc minio-data-1-minio-0 -o yaml > minio-data-1-minio-0.yaml

$ cat minio-data-1-minio-0.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  annotations:
    pv.kubernetes.io/bind-completed: "yes"
    pv.kubernetes.io/bound-by-controller: "yes"
    volume.beta.kubernetes.io/storage-provisioner: directpv-min-io
    volume.kubernetes.io/selected-node: master
    volume.kubernetes.io/storage-provisioner: directpv-min-io
  creationTimestamp: "2023-06-08T04:46:02Z"
  finalizers:
  - kubernetes.io/pvc-protection
  labels:
    app: minio
  name: minio-data-1-minio-0
  namespace: default
  resourceVersion: "76360"
  uid: d7fad69a-d267-43c0-9baf-19fd5f65bdb5
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 16Mi
  storageClassName: directpv-min-io
  volumeMode: Filesystem
  volumeName: pvc-d7fad69a-d267-43c0-9baf-19fd5f65bdb5
status:
  accessModes:
  - ReadWriteOnce
  capacity:
    storage: 16Mi
  phase: Bound

# Edit 'minio-data-1-minio-0' PVC to increase the size from 16MiB to 64MiB.
$ cat minio-data-1-minio-0.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  annotations:
    pv.kubernetes.io/bind-completed: "yes"
    pv.kubernetes.io/bound-by-controller: "yes"
    volume.beta.kubernetes.io/storage-provisioner: directpv-min-io
    volume.kubernetes.io/selected-node: master
    volume.kubernetes.io/storage-provisioner: directpv-min-io
  creationTimestamp: "2023-06-08T04:46:02Z"
  finalizers:
  - kubernetes.io/pvc-protection
  labels:
    app: minio
  name: minio-data-1-minio-0
  namespace: default
  resourceVersion: "76360"
  uid: d7fad69a-d267-43c0-9baf-19fd5f65bdb5
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 64Mi   # <--- increase size here
  storageClassName: directpv-min-io
  volumeMode: Filesystem
  volumeName: pvc-d7fad69a-d267-43c0-9baf-19fd5f65bdb5
status:
  accessModes:
  - ReadWriteOnce
  capacity:
    storage: 16Mi
  phase: Bound

# Apply changes
$ kubectl apply -f minio-data-1-minio-0.yaml

# After successful expansion, you will see updated YAML
$ kubectl get pvc minio-data-1-minio-0 -o yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  annotations:
    pv.kubernetes.io/bind-completed: "yes"
    pv.kubernetes.io/bound-by-controller: "yes"
    volume.beta.kubernetes.io/storage-provisioner: directpv-min-io
    volume.kubernetes.io/selected-node: master
    volume.kubernetes.io/storage-provisioner: directpv-min-io
  creationTimestamp: "2023-06-08T04:46:02Z"
  finalizers:
  - kubernetes.io/pvc-protection
  labels:
    app: minio
  name: minio-data-1-minio-0
  namespace: default
  resourceVersion: "76651"
  uid: d7fad69a-d267-43c0-9baf-19fd5f65bdb5
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 64Mi
  storageClassName: directpv-min-io
  volumeMode: Filesystem
  volumeName: pvc-d7fad69a-d267-43c0-9baf-19fd5f65bdb5
status:
  accessModes:
  - ReadWriteOnce
  capacity:
    storage: 64Mi # <--- increased size here
  phase: Bound
```

## Delete volume
***CAUTION: THIS IS DANGEROUS OPERATION WHICH LEADS TO DATA LOSS***

Volume can be deleted only if it is in `Ready` state (that is, no pod is using it). Run the `kubectl delete pvc` command which triggers DirectPV volume deletion. As removing a volume leads to data loss, double check what volume you are deleting. Below is an example:
```sh
# Delete `sleep-pvc` volume
kubectl delete pvc sleep-pvc
```

## Clean stale volumes
When Pods and/or Persistent Volume Claims are deleted forcefully, associated DirectPV volumes might be left undeleted and they becomes stale. These stale volumes are removed by running `clean` command. Below is an example:
```sh
$ kubectl directpv clean --all
```

Refer [clean command](./command-reference.md#clean-command) for more information.
