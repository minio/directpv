# Monitoring using Prometheus

DirectPV nodes export Prometheus compatible metrics data via port `10443`. The metrics data includes
* directpv_stats_bytes_used
* directpv_stats_bytes_total
and categorized by labels `tenant`, `volumeID` and `node`.

To scrape data in Prometheus, each node must be accessible by port `10443`. A simple example is below

1. Make node server metrics port accessible by localhost:8080
```
$ kubectl -n directpv port-forward node-server-4nd6q 8080:10443
```

2. Add below YAML configuration into Prometheus configuration.
```yaml
scrape_configs:
  - job_name: 'directpv-monitor'
    # Override the global default and scrape targets from this job every 5 seconds.
    scrape_interval: 5s
    static_configs:
      - targets: ['localhost:8080']
        labels:
          group: 'production'
```

3. Run `directpv_stats_bytes_total{node="node-3"}` promQL in Prometheus web interface.

Below is an example comprehensive YAML configuration.

```yaml
global:
  scrape_interval: 15s
  external_labels:
    monitor: 'directpv-monitor'

scrape_configs:

- job_name: 'directpv-metrics'
  scheme: http
  metrics_path: /directpv/metrics
  authorization:
    credentials_file: /var/run/secrets/kubernetes.io/serviceaccount/token

  kubernetes_sd_configs:
  - role: pod

  relabel_configs:
  - source_labels: [__meta_kubernetes_namespace]
    regex: "directpv-(.+)"
    action: keep
  - source_labels: [__meta_kubernetes_pod_controller_kind]
    regex: "DaemonSet"
    action: keep
  - source_labels: [__meta_kubernetes_pod_container_port_name]
    regex: "healthz"
    action: drop
    target_label: kubernetes_port_name

- job_name: 'kubernetes-cadvisor'
  scheme: https
  metrics_path: /metrics/cadvisor
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    insecure_skip_verify: true
  authorization:
    credentials_file: /var/run/secrets/kubernetes.io/serviceaccount/token

  kubernetes_sd_configs:
  - role: node

  relabel_configs:
  - action: labelmap
    regex: __meta_kubernetes_node_label_(.+)
  - source_labels: [__meta_kubernetes_namespace]
    action: replace
    target_label: kubernetes_namespace
  - source_labels: [__meta_kubernetes_service_name]
    action: replace
    target_label: kubernetes_name
```
