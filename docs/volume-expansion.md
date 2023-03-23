Volume Expansion (applies for versions > v3.2.2)
----------------

DirectPV supports volume expansion [CSI feature](https://kubernetes-csi.github.io/docs/volume-expansion.html) from versions above v3.2.2. With this support, the DirectPV provisioned volumes can be expanded to the requested size claimed by the PVC. DirectPV supports "online" volume expansion where the workloads will not have any downtimes during the expansion process.

Volume expansion requires `ExpandCSIVolumes` feature gate to be enabled (enabled by default in k8s v1.16 and above). For more details on the feature gates, please check [here](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/).

To expand the volume, edit the storage size in the corresponding PVC spec

```yaml
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: <new-size> # Set the new size here
  storageClassName: directpv-min-io
  volumeMode: Filesystem
  volumeName: pvc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

DirectPV would expand the volume and reflect the new size in `kubectl directpv list volumes pvc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx -o wide`. The corresponding PVC would remain in "Bounded" state with the expanded size.

(NOTE: As DirectPV supports "online" volume expansion, workload pods doesn't require restarts)
