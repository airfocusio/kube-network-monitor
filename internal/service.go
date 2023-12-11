package internal

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ServiceOpts struct {
	SelfNodeName string
	Interval     time.Duration
}

type Service struct {
	mu         sync.Mutex
	Opts       ServiceOpts
	Ctx        context.Context
	Config     *rest.Config
	Clientset  *kubernetes.Clientset
	Nodes      []ServiceNode
	Prometheus map[string]*ServicePrometheus
}

type ServicePrometheus struct {
	Reachable       prometheus.Gauge
	Latency         prometheus.Histogram
	PacketsSent     prometheus.Counter
	PacketsReceived prometheus.Counter
	PacketsLost     prometheus.Counter
}

func (sp ServicePrometheus) Register() error {
	if err := prometheus.Register(sp.Reachable); err != nil {
		return err
	}
	if err := prometheus.Register(sp.Latency); err != nil {
		return err
	}
	if err := prometheus.Register(sp.PacketsSent); err != nil {
		return err
	}
	if err := prometheus.Register(sp.PacketsReceived); err != nil {
		return err
	}
	if err := prometheus.Register(sp.PacketsLost); err != nil {
		return err
	}
	return nil
}

func (sp ServicePrometheus) Unregister() {
	prometheus.Unregister(sp.Reachable)
	prometheus.Unregister(sp.Latency)
	prometheus.Unregister(sp.PacketsSent)
	prometheus.Unregister(sp.PacketsReceived)
	prometheus.Unregister(sp.PacketsLost)
}

type ServiceNode struct {
	Name       string
	InternalIP net.IP
}

const prometheusNamespace = "network_monitor"

func NewService(opts ServiceOpts) (*Service, error) {
	service := &Service{Opts: opts}
	service.Ctx = context.Background()

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes rest config: %w", err)
	}
	service.Config = config

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes rest clientset: %w", err)
	}
	service.Clientset = clientset
	service.Nodes = []ServiceNode{}
	service.Prometheus = map[string]*ServicePrometheus{}

	return service, nil
}

func (s *Service) Run(stop <-chan os.Signal) error {
	errs := make(chan error)

	go func() {
		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		if err := http.ListenAndServe(fmt.Sprintf(":%d", 1024), sm); err != nil {
			errs <- fmt.Errorf("unable to start http stat server: %w", err)
			return
		}

		errs <- nil
	}()

	go func() {
		for {
			Debug.Printf("Updating nodes...\n")
			if err := s.UpdateNodes(); err != nil {
				Error.Printf("Updating nodes failed: %v\n", err)
			}
			time.Sleep(time.Minute)
		}
	}()

	go func() {
		for {
			time.Sleep(s.Opts.Interval)
			Debug.Printf("Pinging...\n")
			if err := s.PingNodes(); err != nil {
				Error.Printf("Pinging failed: %v\n", err)
			}
		}
	}()

	select {
	case <-stop:
		return nil
	case err := <-errs:
		return err
	}
}

func (s *Service) PrometheusForTarget(target string) (*ServicePrometheus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if prom, ok := s.Prometheus[target]; ok {
		return prom, nil
	}

	promLabels := prometheus.Labels{
		"source": s.Opts.SelfNodeName,
		"target": target,
	}
	prom := &ServicePrometheus{
		Reachable: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   prometheusNamespace,
			Name:        "reachable",
			ConstLabels: promLabels,
		}),
		Latency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace:   prometheusNamespace,
			Name:        "latency",
			Buckets:     prometheus.ExponentialBuckets(0.000125, 2, 14),
			ConstLabels: promLabels,
		}),
		PacketsSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   prometheusNamespace,
			Name:        "packets_sent",
			ConstLabels: promLabels,
		}),
		PacketsReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   prometheusNamespace,
			Name:        "packets_received",
			ConstLabels: promLabels,
		}),
		PacketsLost: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   prometheusNamespace,
			Name:        "packets_lost",
			ConstLabels: promLabels,
		}),
	}

	Info.Printf("Adding target %s\n", target)
	if err := prom.Register(); err != nil {
		return nil, err
	}

	s.Prometheus[target] = prom
	return prom, nil
}

func (s *Service) CleanupPrometheusTargets() {
	s.mu.Lock()
	defer s.mu.Unlock()

	prometheus := map[string]*ServicePrometheus{}

	for target, prom := range s.Prometheus {
		exists := false
		for _, node := range s.Nodes {
			if node.Name == target {
				exists = true
				break
			}
		}

		if exists {
			prometheus[target] = prom
		} else {
			Info.Printf("Removing target %s\n", target)
			prom.Unregister()
		}
	}

	s.Prometheus = prometheus
}

func (s *Service) PingNodes() error {
	s.CleanupPrometheusTargets()

	wg := sync.WaitGroup{}
	for _, node := range s.Nodes {
		node := node
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats, err := pingIPOnce(node.InternalIP)
			if err != nil {
				Error.Printf("Unable to ping %s: %v\n", node.InternalIP.String(), err)
				return
			}
			packetsLost := stats.PacketsSent - stats.PacketsRecv
			if packetsLost > 0 {
				Info.Printf("Pinging %s lost %d of %d packets", node.InternalIP.String(), packetsLost, stats.PacketsSent)
			}
			prom, err := s.PrometheusForTarget(node.Name)
			if err != nil {
				Error.Printf("Unable to register prometheus metrics: %v\n", err)
				return
			}

			if stats.PacketsRecv > 0 {
				prom.Reachable.Set(1.0)
				prom.Latency.Observe(stats.AvgRtt.Seconds())
			} else {
				prom.Reachable.Set(0.0)
			}
			prom.PacketsSent.Add(float64(stats.PacketsSent))
			prom.PacketsReceived.Add(float64(stats.PacketsRecv))
			prom.PacketsLost.Add(float64(packetsLost))
		}()
	}
	wg.Wait()
	return nil
}

func (s *Service) UpdateNodes() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(s.Ctx, time.Second)
	defer cancel()

	nodeList, err := s.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	nodes := []ServiceNode{}
	for _, node := range nodeList.Items {
		name := node.ObjectMeta.Name
		internalIP := net.IP{}

		if name == s.Opts.SelfNodeName {
			continue
		}

		ignore := false
		for _, taint := range node.Spec.Taints {
			if taint.Key == "node.kubernetes.io/not-ready" {
				ignore = true
				continue
			}
			if taint.Key == "node.kubernetes.io/unschedulable" {
				ignore = true
				continue
			}
		}
		if ignore {
			continue
		}

		for _, address := range node.Status.Addresses {
			if address.Type == "InternalIP" {
				internalIP = net.ParseIP(address.Address)
			}
		}

		if internalIP.Equal(net.IP{}) {
			Error.Printf("Unable to detect internal IP for node %s\n", name)
			continue
		}

		nodes = append(nodes, ServiceNode{
			Name:       name,
			InternalIP: internalIP,
		})
	}
	s.Nodes = nodes

	return nil
}
