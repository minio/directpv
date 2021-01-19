---
title: Architecture
---

Architecture
-------------

### Components

DirectCSI is made up of 4 components:

| Component         | Description                                          |
|-------------------|------------------------------------------------------|
| CSI Driver        | Performs mounting, unmounting of provisioned volumes |
| CSI Controller    | Schedules volumes on nodes                           |
| Drive Controller  | Formats and manages drive lifecycle                  |
| Volume Controller | manages volume lifecycle                             |

The 4 components run as two different pods. 

| Name                          | Components                                           | Description                        |
|-------------------------------|------------------------------------------------------|------------------------------------|
| DirectCSI Node Driver         | CSI Driver, Driver Controller, Volume Controller     | runs on every node as a DaemonSet  |
| DirectCSI Central Controller  | CSI Controller                                       | runs as a deployment               |


### Scalability

Since the node driver runs on every node, the load on it is constrained to operations specific to that node. 

The central controller needs to be scaled up as the number of drives managed by DirectCSI is increased. By default, 3 replicas of central controller are run. As a rule of thumb, having as many central controller instances as etcd nodes is a good working solution for achieving high scale.

### Availability

If node driver is down, then volume mounting, unmounting, formatting and cleanup will not proceed for volumes and drives on that node. In order to restore operations, bring node driver to running status.

In central controller is down, then volume scheduling and deletion will not proceed for all volumes and drives in the direct-csi cluster. In order to restore operations, bring the central controller to running status.

Security is covered [here](./security.md)
