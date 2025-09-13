package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/asmit27rai/kubesight/pkg/metrics"
)

type MockDataGenerator struct {
	kafkaBrokers   []string
	writer         *kafka.Writer
	generationRate int
	clusterCount   int
	namespaceCount int
	podCount       int

	clusters    []string
	namespaces  []string
	pods        []string
	containers  []string
	metricNames []string
}

func main() {
	log.Println("Starting KubeSight Mock Data Generator...")

	config := parseConfig()

	generator := NewMockDataGenerator(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down mock data generator...")
		cancel()
	}()

	command := "generate"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "generate":
		generator.StartGenerating(ctx)
	case "burst":
		generator.GenerateBurst(ctx, 10000)
	default:
		log.Fatalf("Unknown command: %s. Use 'generate' or 'burst'", command)
	}
}

type Config struct {
	KafkaBrokers   []string
	GenerationRate int
	ClusterCount   int
	NamespaceCount int
	PodCount       int
}

func parseConfig() Config {
	config := Config{
		KafkaBrokers:   []string{"kafka:29092"},
		GenerationRate: 100,
		ClusterCount:   3,
		NamespaceCount: 5,
		PodCount:       20,
	}

	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		config.KafkaBrokers = []string{brokers}
	}

	if rate := os.Getenv("GENERATION_RATE"); rate != "" {
		if r, err := strconv.Atoi(rate); err == nil {
			config.GenerationRate = r
		}
	}

	if clusters := os.Getenv("CLUSTER_COUNT"); clusters != "" {
		if c, err := strconv.Atoi(clusters); err == nil {
			config.ClusterCount = c
		}
	}

	if namespaces := os.Getenv("NAMESPACE_COUNT"); namespaces != "" {
		if n, err := strconv.Atoi(namespaces); err == nil {
			config.NamespaceCount = n
		}
	}

	if pods := os.Getenv("POD_COUNT"); pods != "" {
		if p, err := strconv.Atoi(pods); err == nil {
			config.PodCount = p
		}
	}

	return config
}

func NewMockDataGenerator(config Config) *MockDataGenerator {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.KafkaBrokers...),
		Topic:        "k8s-metrics",
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        true,
		BatchTimeout: 10 * time.Millisecond,
		BatchSize:    100,
	}

	generator := &MockDataGenerator{
		kafkaBrokers:   config.KafkaBrokers,
		writer:         writer,
		generationRate: config.GenerationRate,
		clusterCount:   config.ClusterCount,
		namespaceCount: config.NamespaceCount,
		podCount:       config.PodCount,
	}

	generator.initializeTemplates()

	return generator
}

func (g *MockDataGenerator) initializeTemplates() {
	for i := 0; i < g.clusterCount; i++ {
		g.clusters = append(g.clusters, fmt.Sprintf("cluster-%d", i+1))
	}

	namespaceNames := []string{"default", "kube-system", "monitoring", "ingress", "app"}
	for i := 0; i < g.namespaceCount && i < len(namespaceNames); i++ {
		g.namespaces = append(g.namespaces, namespaceNames[i])
	}
	for i := len(namespaceNames); i < g.namespaceCount; i++ {
		g.namespaces = append(g.namespaces, fmt.Sprintf("namespace-%d", i+1))
	}

	for i := 0; i < g.podCount; i++ {
		g.pods = append(g.pods, fmt.Sprintf("pod-%d", i+1))
	}

	g.containers = []string{"main", "sidecar", "proxy", "init", "worker"}

	g.metricNames = []string{
		"cpu_usage",
		"memory_usage",
		"disk_usage",
		"network_in",
		"network_out",
		"pod_restarts",
		"request_count",
		"response_time",
		"error_rate",
		"disk_io_read",
		"disk_io_write",
		"memory_rss",
		"memory_cache",
		"network_packets_in",
		"network_packets_out",
	}

	log.Printf("Initialized templates: %d clusters, %d namespaces, %d pods, %d metrics",
		len(g.clusters), len(g.namespaces), len(g.pods), len(g.metricNames))
}

func (g *MockDataGenerator) StartGenerating(ctx context.Context) {
	log.Printf("Starting continuous data generation at %d metrics/second", g.generationRate)

	interval := time.Second / time.Duration(g.generationRate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	count := 0
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Generated %d total metrics in %v", count, time.Since(start))
			g.writer.Close()
			return

		case <-ticker.C:
			metric := g.generateRandomMetric()
			if err := g.sendMetric(ctx, metric); err != nil {
				log.Printf("Error sending metric: %v", err)
			} else {
				count++

				if count%1000 == 0 {
					elapsed := time.Since(start)
					rate := float64(count) / elapsed.Seconds()
					log.Printf("Generated %d metrics (%.1f/sec)", count, rate)
				}
			}
		}
	}
}

func (g *MockDataGenerator) GenerateBurst(ctx context.Context, burstSize int) {
	log.Printf("💥 Generating burst of %d metrics...", burstSize)

	start := time.Now()

	for i := 0; i < burstSize; i++ {
		metric := g.generateRandomMetric()
		if err := g.sendMetric(ctx, metric); err != nil {
			log.Printf("Error sending metric %d: %v", i, err)
		}

		if (i+1)%1000 == 0 {
			log.Printf("📊 Burst progress: %d/%d metrics", i+1, burstSize)
		}

		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	elapsed := time.Since(start)
	rate := float64(burstSize) / elapsed.Seconds()
	log.Printf("Burst complete: %d metrics in %v (%.1f/sec)", burstSize, elapsed, rate)

	g.writer.Close()
}

func (g *MockDataGenerator) generateRandomMetric() *metrics.MetricPoint {
	now := time.Now()

	cluster := g.clusters[rand.Intn(len(g.clusters))]
	namespace := g.namespaces[rand.Intn(len(g.namespaces))]
	pod := g.pods[rand.Intn(len(g.pods))]
	container := g.containers[rand.Intn(len(g.containers))]
	metricName := g.metricNames[rand.Intn(len(g.metricNames))]

	var value float64
	var unit string

	switch metricName {
	case "cpu_usage", "memory_usage", "disk_usage":
		value = rand.Float64()
		if rand.Float32() < 0.05 {
			value = 0.8 + rand.Float64()*0.2
		}
		unit = "percent"

	case "network_in", "network_out", "disk_io_read", "disk_io_write":
		value = rand.Float64() * 1000000
		unit = "bytes_per_sec"

	case "network_packets_in", "network_packets_out":
		value = rand.Float64() * 10000
		unit = "packets_per_sec"

	case "pod_restarts":
		value = float64(rand.Intn(5))
		unit = "count"

	case "request_count":
		value = rand.Float64() * 1000
		unit = "requests_per_sec"

	case "response_time":
		value = rand.Float64() * 500
		if rand.Float32() < 0.02 {
			value = 500 + rand.Float64()*1500
		}
		unit = "milliseconds"

	case "error_rate":
		value = rand.Float64() * 0.05
		unit = "percent"

	case "memory_rss", "memory_cache":
		value = rand.Float64() * 1000000000
		unit = "bytes"

	default:
		value = rand.Float64() * 100
		unit = "generic"
	}

	labels := map[string]string{
		"source":    "mock-generator",
		"generated": "true",
		"version":   "v1.0.0",
	}

	if rand.Float32() < 0.1 {
		labels["anomaly"] = "possible"
	}

	if namespace == "kube-system" {
		labels["system"] = "true"
	}

	return &metrics.MetricPoint{
		Timestamp:     now,
		ClusterID:     cluster,
		Namespace:     namespace,
		PodName:       pod,
		ContainerName: container,
		MetricName:    metricName,
		Value:         value,
		Unit:          unit,
		Labels:        labels,
	}
}

func (g *MockDataGenerator) sendMetric(ctx context.Context, metric *metrics.MetricPoint) error {
	data, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %v", err)
	}

	message := kafka.Message{
		Key:   []byte(metric.GetKey()),
		Value: data,
		Time:  metric.Timestamp,
	}

	return g.writer.WriteMessages(ctx, message)
}

func (g *MockDataGenerator) GenerateSpecificScenario(ctx context.Context, scenario string) {
	log.Printf("Generating scenario: %s", scenario)

	var metrics []*metrics.MetricPoint

	switch scenario {
	case "high_cpu":
		for i := 0; i < 100; i++ {
			metric := g.generateRandomMetric()
			if metric.MetricName == "cpu_usage" {
				metric.Value = 0.9 + rand.Float64()*0.1
				metric.Labels["scenario"] = "high_cpu"
				metrics = append(metrics, metric)
			}
		}

	case "pod_restarts":
		for i := 0; i < 50; i++ {
			metric := g.generateRandomMetric()
			metric.MetricName = "pod_restarts"
			metric.Value = float64(3 + rand.Intn(5))
			metric.Labels["scenario"] = "pod_restarts"
			metrics = append(metrics, metric)
		}

	case "network_spike":
		for i := 0; i < 200; i++ {
			metric := g.generateRandomMetric()
			if metric.MetricName == "network_in" || metric.MetricName == "network_out" {
				metric.Value *= 10
				metric.Labels["scenario"] = "network_spike"
				metrics = append(metrics, metric)
			}
		}
	}

	for _, metric := range metrics {
		if err := g.sendMetric(ctx, metric); err != nil {
			log.Printf("Error sending scenario metric: %v", err)
		}
	}

	log.Printf("Scenario '%s' complete: sent %d metrics", scenario, len(metrics))
}
