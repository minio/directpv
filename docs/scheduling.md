---
title: Scheduling
---

Scheduling guidelines
-------------

### Access-tier based volume scheduling

In addition to scheduling based on resource constraints (available space) and node topology (affinity/anti-affinity etc.), it is possible to further influence the scheduling of workloads to specific volumes based on "access-tiers". DirectCSI pre-defines 3 access tiers:

- Hot
- Warm
- Cold

By default, direct-csi drives are not associated with any access-tier. An admin can associate drives to access tiers. Further instructions on the configuration is provided in the following sections.

#### Step 1: Set access-tier tag on the drives

```
kubectl direct-csi drives access-tier set hot|cold|warm [FLAGS]
```

#### Step 2: Format the tiered drives (Incase of fresh/available drives)

```
kubectl direct-csi drives format --access-tier=hot|cold|warm
```

#### Step 3: Set the 'direct-csi-min-io/access-tier' parameter in storage class definition

Create a storage class with the following parameter set

```
parameters:
  direct-csi-min-io/access-tier: warm|hot|cold
```

#### Step 4: Deploy the workload with the corresponding storage class name set

You will see volumes placed on the tiered drives only. You can verify this by the following set of commands

```
kubectl direct-csi volumes ls --access-tier=warm|hot|cold
kubectl direct-csi drives ls --access-tier=warm|hot|cold
```
