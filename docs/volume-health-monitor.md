
Volume Health Monitor
-------------

This is a CSI feature introduced as an Alpha feature in Kubernetes v1.19 and a second Alpha was done in v1.21. This feature is to detect "abnormal" volume conditions and report them as events on PVCs and Pods. A DirectPV volume will be considered as "abnormal" if the respective volume mounts are not present.

Feature gate `CSIVolumeHealth` needs to be enabled for the node side monitoring to take effect.

### Enable volume health monitoring

Apply the following manifests to enable volume monitoring for the DirectPV volumes

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-external-health-monitor-controller
  # replace with non-default namespace name
  namespace: default

---
# Health monitor controller must be able to work with PVs, PVCs, Nodes and Pods
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: external-health-monitor-controller-runner
rules:
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["get", "list", "watch", "create", "patch"]

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-external-health-monitor-controller-role
subjects:
  - kind: ServiceAccount
    name: csi-external-health-monitor-controller
    # replace with non-default namespace name
    namespace: default
roleRef:
  kind: ClusterRole
  name: external-health-monitor-controller-runner
  apiGroup: rbac.authorization.k8s.io

---
# Health monitor controller must be able to work with configmaps or leases in the current namespace
# if (and only if) leadership election is enabled
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  # replace with non-default namespace name
  namespace: default
  name: external-health-monitor-controller-cfg
rules:
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "watch", "list", "delete", "update", "create"]

---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-external-health-monitor-controller-role-cfg
  # replace with non-default namespace name
  namespace: default
subjects:
  - kind: ServiceAccount
    name: csi-external-health-monitor-controller
    # replace with non-default namespace name
    namespace: default
roleRef:
  kind: Role
  name: external-health-monitor-controller-cfg
  apiGroup: rbac.authorization.k8s.io
```

```yaml
kind: Deployment
apiVersion: apps/v1
metadata:
  name: csi-external-health-monitor-controller
spec:
  replicas: 3
  selector:
    matchLabels:
      external-health-monitor-controller: directpv-health-monitor
  template:
    metadata:
      labels:
        external-health-monitor-controller: directpv-health-monitor
    spec:
      serviceAccount: csi-external-health-monitor-controller
      containers:
        - name: csi-external-health-monitor-controller
          image: gcr.io/k8s-staging-sig-storage/csi-external-health-monitor-controller:v0.5.0
          imagePullPolicy: Always
          args:
            - "--v=5"
            - "--csi-address=unix:///csi/csi.sock"
            - "--leader-election"
            - "--http-endpoint=:8080"
          volumeMounts:
          - mountPath: /csi
            mountPropagation: None
            name: socket-dir
      volumes:
      - hostPath:
          path: /var/lib/kubelet/plugins/direct-csi-min-io-controller
          type: DirectoryOrCreate
        name: socket-dir
```

### References
 - [External health monitor](https://github.com/kubernetes-csi/external-health-monitor)
 - [Volume health monitoring](https://kubernetes.io/docs/concepts/storage/volume-health-monitoring)
