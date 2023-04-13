Drain a node
-------------

Draining will forcefully remove the DirectPV resources from a node. This [bash script](./tools/drain.sh) can be used for draining and has to be executed cautiously as it is an irreversible operation and may incur data loss.

You can consider draining a node in the following circumstances

#### When a node is detached from kubernetes

If a node which was used by DirectPV is detached from kubernetes, the DirectPV resources from that node will remain intact until the resources are drained.

#### When DirectPV is unselected to run on a specific node

If a node which was used by DirectPV is decided to be a "non-storage" node.

For example, If DirectPV is decided to not run on a specific node by changing the node-selectors like the following example

```sh
$ kubectl directpv uninstall
$ kubectl directpv install --node-selector node-label-key=node-label-value
```

the resources from the detached node can then be cleaned up by

```sh
./drain.sh <node-name>
```
