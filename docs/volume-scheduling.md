# Volume Scheduling

`DirectPV` comes with storage class named `directpv-min-io` with volume binding mode [WaitForFirstConsumer](https://kubernetes.io/docs/concepts/storage/storage-classes/#volume-binding-mode). This mode delays volume binding and provisioning of a `PersistentVolume` until a `Pod` using the `PersistentVolumeClaim` is created. PersistentVolumes will be selected or provisioned conforming to the topology that is specified by the Pod's scheduling constraints. These include, but are not limited to, resource requirements, node selectors, pod affinity and anti-affinity, and taints and tolerations.

## Drive selection

DirectPV CSI controller selects suitable drive for `CreateVolume` request like below
1. Filesystem type and/or access-tier in the request is validated. DirectPV supports `xfs` filesystem only.
2. Each `DirectPVDrive` CRD object is checked whether the requested volume is already present or not. If present, the first drive containing the volume is selected.
3. As no `DirectPVDrive` CRD object has the requested volume, each drive is selected by
   a. By requested capacity
   b. By access-tier if requested
   c. By topology constraints if requested
4. In the process of step (3), if more than one drive is selected, the maximum free capacity drive is picked.
5. If step (4) picks up more than one drive, a drive is randomly selected.
6. Finally the selected drive is updated with requested volume information.
7. If none of them are selected, an appropriate error is returned.
8. If any error in the above steps, Kubernetes retries the request.
9. In case of parallel requests and the same drive is selected, step (6) succeeds for any one of the request and fails for rest of the requests by Kubernetes.

```text
                  +--------------+
               No |   Is Valid   |
     +------------| CreateVolume |
     |            |   request?   |
     |            +==============+
     |                    | Yes
     |            +=======V=======+ Loop In
     |            | Loop each     |-----------------+
     |            | DirectPVDrive |                 |
     |            | CRD           |<--+       +-----V-----+
     |            +===============+   |       |     Is    |      +-------------+
     |                   | Loop Out   |       | requested | Yes  | Return this |
+----V---+        +------V------+     |       |  volume   |----->|    drive    |
| Return |   Yes  | Is no drive |     |       |  present? |      +-------------+
|  error |<-------|   matched?  |     |       +===========+
+--------+        +=============+     |             | No
                         | No         |       +-----V-----+
+-----------+     +------V-------+    |   No  | Match by  |
|   Return  |  No | Is more than |    |<------| capacity? |
| the first |<----|  one drive   |    |       +===========+
|   drive   |     |   matched?   |    |             | Yes
+-----^-----+     +==============+    |     +-------V-------+
      |                  | Yes        |  No |   Match by    |
      |           +------V--------+   |<----|  access-tier  |
      |           | Filter drives |   |     | if requested? |
      |           |  by maximum   |   |     +===============+
      |           | free capacity |   |             | Yes
      |           +---------------+   |     +-------V-------+
      |                  |            |     |   Match by    |
      |           +------V-------+    |  No |   topology    |
      |     No    | Is more than |    |<----|  constraints  |
      +-----------|   one drive  |    |     | if requested? |
                  |    matched?  |    |     +===============+
                  +==============+    |            | Yes
                         | Yes        |      +-----V-----+
                  +------V-----+      |      | Append to |
                  |  Randomly  |      +<-----|  matched  |
                  |   select   |             |   drives  |
                  |  a drive   |             +-----------+
                  +------------+
```
