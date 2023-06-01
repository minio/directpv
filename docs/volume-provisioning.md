# Volume provisioning
Volume provisioning involves making a `PersistentVolumeClaim` using default `directpv-min-io` storage class or custom storage class having `directpv-min-io` provisioner. As the storage classes come with volume binding mode [WaitForFirstConsumer](https://kubernetes.io/docs/concepts/storage/storage-classes/#volume-binding-mode); this mode delays volume binding and provisioning of a `PersistentVolume` until a `Pod` using the `PersistentVolumeClaim` is created. PersistentVolumes will be selected or provisioned conforming to the topology that is specified by the Pod's scheduling constraints. These include, but are not limited to, resource requirements, node selectors, pod affinity and anti-affinity, and taints and tolerations. Pods consuming volumes are scheduled to nodes where volumes were scheduled. This ensures high performance data access to the pod.

DirectPV volume in `Ready` state indicates that the volume is ready for binding to the pod. After binding, `Bound` state is set to the volume.

## Making Persistent volume claim
PV claim must be defined with specific parameters in `PersistentVolumeClaim` specification. These parameters are

| Parameter          | Value                                                                            |
|:-------------------|:---------------------------------------------------------------------------------|
| `volumeMode`       | `Filesystem`                                                                     |
| `storageClassName` | `directpv-min-io` or any storage class name having `directpv-min-io` provisioner |
| `accessModes`      | `[ "ReadWriteOnce" ]`                                                            |

Below is an example claiming `8MiB` storage from `directpv-min-io` storage class for `sleep-pvc` PVC.
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: sleep-pvc
spec:
  volumeMode: Filesystem
  storageClassName: directpv-min-io
  accessModes: [ "ReadWriteOnce" ]
  resources:
    requests:
      storage: 8Mi
```

For `WaitForFirstConsumer` volume binding mode, a pod consuming `sleep-pvc` must be defined. Below is an example which uses `sleep-volume` mounted on `/mnt`
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: sleep-pod
spec:
  volumes:
    - name: sleep-volume
      persistentVolumeClaim:
        claimName: sleep-pvc
  containers:
    - name: sleep-container
      image: example.org/test/sleep:v0.0.1
      volumeMounts:
        - mountPath: "/mnt"
          name: sleep-volume
```

## Making Persistent volume claim in StatefulSet
PV claim must be defined with specific parameters in `volumeClaimTemplates` specification. These parameters are

| Parameter          | Value                                                                            |
|:-------------------|:---------------------------------------------------------------------------------|
| `storageClassName` | `directpv-min-io` or any storage class name having `directpv-min-io` provisioner |
| `accessModes`      | `[ "ReadWriteOnce" ]`                                                            |

Below is an example claiming two `16MiB` storage from `directpv-min-io` storage class for `minio-data-1` and `minio-data-2` PVC to two `minio` pods.

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
      storageClassName: directpv-min-io
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 16Mi
  - metadata:
      name: minio-data-2
    spec:
      storageClassName: directpv-min-io
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 16Mi
```

## Further reads
* [volume scheduling guide](./volume-scheduling.md)
