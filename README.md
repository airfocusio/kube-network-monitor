# kube-network-monitor

A Kubernetes DaemonSet that continuously pings any other node in the cluster and exposes the reachability and latency as Prometheus metrics.

## Query

```
histogram_quantile(0.5, sum(rate(network_monitor_latency_bucket[5m])) by (le, source, target))
```
