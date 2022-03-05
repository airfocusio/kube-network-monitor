# kube-network-monitor

A Kubernetes DaemonSet that continuously pings any other node in the cluster and exposes the reachability and latency as Prometheus metrics.

## Metrics

* `network_monitor_reachable`: Gauge that is either `1` (at least some packets make it through) or `0`
* `network_monitor_latency`: Histogram of round trip times
* `network_monitor_packets_sent`: Counter of sent packets
* `network_monitor_packets_received`: Counter of received packets
* `network_monitor_packets_lost`: Counter of lost packets

## Labels

* `source`: Node name of source
* `target`: Node name of target

## Query

```
histogram_quantile(0.5, sum(rate(network_monitor_latency_bucket[5m])) by (le, source, target))
```
