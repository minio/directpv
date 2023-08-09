# Volume Scheduling

`DirectPV` comes with default storage class `directpv-min-io` or custom storage class having `directpv-min-io` provisioner with volume binding mode [WaitForFirstConsumer](https://kubernetes.io/docs/concepts/storage/storage-classes/#volume-binding-mode). This volume binding mode delays volume binding and provisioning of a `PersistentVolume` until a `Pod` using the `PersistentVolumeClaim` is created. PersistentVolumes will be selected or provisioned conforming to the topology that is specified by the Pod's scheduling constraints. These include, but are not limited to, resource requirements, node selectors, pod affinity and anti-affinity, and taints and tolerations.

## Drive selection algorithm

DirectPV CSI controller selects suitable drive for `CreateVolume` request like below
1. Filesystem type and/or access-tier in the request is validated. DirectPV supports `xfs` filesystem only.
2. Each `DirectPVDrive` CRD object is checked whether the requested volume is already present or not. If present, the first drive containing the volume is selected.
3. As no `DirectPVDrive` CRD object has the requested volume, each drive is selected by
   a. By requested capacity
   b. By access-tier if requested
   c. By topology constraints if requested
   d. By volume claim ID if requested
4. In the process of step (3), if more than one drive is selected, the maximum free capacity drive is picked.
5. If step (4) picks up more than one drive, a drive is randomly selected.
6. Finally the selected drive is updated with requested volume information.
7. If none of them are selected, an appropriate error is returned.
8. If any error in the above steps, Kubernetes retries the request.
9. In case of parallel requests and the same drive is selected, step (6) succeeds for any one of the request and fails for rest of the requests by Kubernetes.

```text
                  ╭╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╮
               No │   Is Valid    │
      +-----------│ CreateVolume  │
      |           │   request?    │
      |           ╰╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╯
      |                   | Yes
      |           ╒═══════V═══════╕ Loop In
      |           │   Loop each   │-----------------+
      |           │ DirectPVDrive │                 |
      |           │      CRD      │<--+     ╭╌╌╌╌╌╌╌V╌╌╌╌╌╌╌╮      ┌─────────────┐
      |           ╘═══════════════╛   |     │  Is requested │ Yes  │ Return this │
      |                   | Loop Out  |     │volume present?│----->│    drive    │
┌─────V─────┐     ╭╌╌╌╌╌╌╌V╌╌╌╌╌╌╌╮   |     ╰╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╯      └─────────────┘
│  Return   │ Yes │  Is no drive  │   |             | No
│   error   │<----│   matched?    │   |     ╭╌╌╌╌╌╌╌V╌╌╌╌╌╌╌╮
└───────────┘     ╰╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╯   |  No │    Match by   │
                          | No        |<----│   capacity?   │
┌───────────┐     ╭╌╌╌╌╌╌╌V╌╌╌╌╌╌╌╮   |     ╰╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╯
│   Return  │  No │ Is more than  │   |             | Yes
│ the first │<----│   one drive   │   |     ╭╌╌╌╌╌╌╌V╌╌╌╌╌╌╌╮
│   drive   │     │    matched?   │   |  No │   Match by    │
└─────^─────┘     ╰╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╯   |<----│  access-tier  │
      |                   | Yes       |     │ if requested? │
      |           ┌───────V───────┐   |     ╰╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╯
      |           │ Filter drives │   |             | Yes
      |           │  by maximum   │   |     ╭╌╌╌╌╌╌╌V╌╌╌╌╌╌╌╮
      |           │ free capacity │   |     │   Match by    │
      |           └───────────────┘   |  No │   topology    │
      |                   |           |<----│  constraints  │
      |           ╭╌╌╌╌╌╌╌V╌╌╌╌╌╌╌╮   |     │ if requested? │
      |     No    │ Is more than  │   |     ╰╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╯
      +-----------│   one drive   │   |             | Yes
                  │    matched?   │   |     ╭╌╌╌╌╌╌╌V╌╌╌╌╌╌╌╮
                  ╰╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╯   |     │   Match by    │
                          | Yes       | Yes │ volume claim  │
                  ┌───────V───────┐   |<----│      ID       │
                  │     Return    │   |     │ if requested? │
                  │    Randomly   │   |     ╰╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╯
                  │ selected drive│   |             | No
                  └───────────────┘   |     ┌───────V───────┐
                                      |     │   Append to   │
                                      +<----│ matched drives│
                                            └───────────────┘
```

## Customizing drive selection
Apart from controlling drive selection based on node selectors, pod affinity and anti-affinity, and taints and tolerations, drive labels are used to instruct DirectPV to pick up specific drives with custom storage class for volume scheduling. Below steps are involved for this process.

* Label selected drives by [label drives](./command-reference.md#drives-command-1) command. Below is an example:
```sh
# Label 'nvme1n1' drive in all nodes as 'fast' value to 'tier' key.
$ kubectl directpv label drives --drives=nvme1n1 tier=fast
```

* Create new storage class with drive labels using [create-storage-class.sh script](../tools/create-storage-class.sh). Below is an example:
```sh
# Create new storage class 'fast-tier-storage' with drive labels 'directpv.min.io/tier: fast'
$ create-storage-class.sh fast-tier-storage 'directpv.min.io/tier: fast'
```

* Use newly created storage class in [volume provisioning](./volume-provisioning.md). Below is an example:
```sh
$ kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: sleep-pvc
spec:
  volumeMode: Filesystem
  storageClassName: fast-tier-storage
  accessModes: [ "ReadWriteOnce" ]
  resources:
    requests:
      storage: 8Mi
EOF
```

### Unique drive selection

The default free capacity based drive selection leads to allocate more than one volume in a single drive for StatefulSet deployments which lacks performance and high availability for application like MinIO object storage. To overcome this behavior, DirectPV provides a way to allocate one volume per drive. This feature needs to be set by having custom storage class with label 'directpv.min.io/volume-claim-id'. Below is an example to create custom storage class using [create-storage-class.sh script](../tools/create-storage-class.sh):

```sh
create-storage-class.sh tenant-1-storage 'directpv.min.io/volume-claim-id: 555e99eb-e255-4407-83e3-fc443bf20f86'
```

This custom storage class has to be used in your StatefulSet deployment. Below is an example to deploy MinIO object storage

```yaml
kind: Service
apiVersion: v1
metadata:
  name: minio
  labels:
    app: minio
spec:
  selector:
    app: minio
  ports:
    - name: minio
      port: 9000

---

apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: minio
  labels:
    app: minio
spec:
  serviceName: "minio"
  replicas: 2
  selector:
    matchLabels:
      app: minio
  template:
    metadata:
      labels:
        app: minio
        directpv.min.io/organization: minio
        directpv.min.io/app: minio-example
        directpv.min.io/tenant: tenant-1
    spec:
      containers:
      - name: minio
        image: minio/minio
        env:
        - name: MINIO_ACCESS_KEY
          value: minio
        - name: MINIO_SECRET_KEY
          value: minio123
        volumeMounts:
        - name: minio-data-1
          mountPath: /data1
        - name: minio-data-2
          mountPath: /data2
        args:
        - "server"
        - "http://minio-{0...1}.minio.default.svc.cluster.local:9000/data{1...2}"
  volumeClaimTemplates:
  - metadata:
      name: minio-data-1
    spec:
      storageClassName: tenant-1-storage
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 16Mi
  - metadata:
      name: minio-data-2
    spec:
      storageClassName: tenant-1-storage
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 16Mi
```
