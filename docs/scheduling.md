---
title: Scheduling
---

Scheduling guidelines
-------------

### Volume scheduling based on drive labels

In addition to scheduling based on resource constraints (available space) and node topology (affinity/anti-affinity etc.), it is possible to further influence the scheduling of workloads to specific volumes based on drive lavels. The DirectPV drives can be labeled based on its classsification and such labels can be used in the storage class parameters to control scheduling to pick up the chosen drives for volumes. 

By default, the DirectPV drives will not have any user defined labels set on them. Use `kubectl directpv label drives` command to set user defined labels to DirectPV drives.

**Notes:**

- This applies only for creating new volumes as this is a schedule time process.
- In the following example, please replace the place holders `<label-key>`,`<label-value>` and `<drive-name>` with appropriate values based on the classification you chose

#### Step 1: Set the label on the DirectPV drive(s)

```sh
kubectl directpv label drives <label-key>=<label-value> --drives /dev/<drive-name>
```

To Verify if the labels are properly set, list the drives with `--show-labels` flag

```sh
kubectl directpv list drives --drives /dev/<drive-name> --show-labels
```

#### Step 2: Set the 'directpv-min-io/<label-key>: <label-value>' parameter in storage class definition

Create a storage class with the following parameter set

```yaml
parameters:
  directpv.min.io/<label-key>: <label-value>
```

For example, create a new storage class with the `parameters` section

(NOTE: Please refer the exiting storage class `kubectl get storageclass directpv-min-io -n directpv -o yaml` to compare and check if all the fields are present on the new storage class)

```yaml
allowVolumeExpansion: false
allowedTopologies:
- matchLabelExpressions:
  - key: directpv.min.io/identity
    values:
    - directpv-min-io
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  finalizers:
  - foregroundDeletion
  labels:
    application-name: directpv.min.io
    application-type: CSIDriver
    directpv.min.io/created-by: kubectl-directpv
    directpv.min.io/version: v1beta1
  name: directpv-min-io-new # Please choose any storage class name of your choice
  resourceVersion: "511457"
  uid: e93d8dab-b182-482f-b8eb-c69d4a1ec62d
parameters:
  fstype: xfs
  directpv.min.io/<label-key>: <label-value>
provisioner: directpv-min-io
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
```

#### Step 3: Deploy the workload with the new storage class name set

You will see volumes placed on the labeled drives only. You can verify this by the following command

```sh
kubectl directpv list drives --labels <label-key>:<label-value>
kubectl directpv list volumes
```
