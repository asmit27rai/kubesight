package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/asmit27rai/kubesight/internal/engine"
	"github.com/asmit27rai/kubesight/pkg/metrics"
)

type Processor struct {
	config      ProcessorConfig
	readers     map[string]*kafka.Reader
	queryEngine *engine.QueryEngine
	stats       ProcessorStats
}

type ProcessorConfig struct {
	KafkaBrokers []string
	Topics       Topics
	QueryEngine  *engine.QueryEngine
	BatchSize    int
	BatchTimeout time.Duration
}

type Topics struct {
	Metrics string
	Logs    string
	Events  string
}

type ProcessorStats struct {
	MessagesProcessed uint64
	ProcessingErrors  uint64
	LastProcessedTime time.Time
	ProcessingRate    float64
}

func NewProcessor(config ProcessorConfig) (*Processor, error) {
	if len(config.KafkaBrokers) == 0 {
		return nil, fmt.Errorf("no Kafka brokers specified")
	}

	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.BatchTimeout <= 0 {
		config.BatchTimeout = 5 * time.Second
	}

	processor := &Processor{
		config:      config,
		readers:     make(map[string]*kafka.Reader),
		queryEngine: config.QueryEngine,
		stats:       ProcessorStats{LastProcessedTime: time.Now()},
	}

	processor.initializeReaders()

	return processor, nil
}

func (p *Processor) Start(ctx context.Context) error {
	log.Println("🚀 Starting stream processor...")

	errCh := make(chan error, len(p.readers))

	for topic, reader := range p.readers {
		go func(topic string, reader *kafka.Reader) {
			log.Printf("📡 Starting consumer for topic: %s", topic)
			errCh <- p.processStream(ctx, topic, reader)
		}(topic, reader)
	}

	go p.reportStatistics(ctx)

	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("Stream processing error: %v", err)
			return err
		}
	case <-ctx.Done():
		log.Println("Stream processor shutting down...")
	}

	for topic, reader := range p.readers {
		log.Printf("Closing reader for topic: %s", topic)
		reader.Close()
	}

	return nil
}

func (p *Processor) initializeReaders() {
	readerConfig := kafka.ReaderConfig{
		Brokers:        p.config.KafkaBrokers,
		GroupID:        "kubesight-query-engine",
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	}

	if p.config.Topics.Metrics != "" {
		metricsConfig := readerConfig
		metricsConfig.Topic = p.config.Topics.Metrics
		p.readers["metrics"] = kafka.NewReader(metricsConfig)
	}

	if p.config.Topics.Logs != "" {
		logsConfig := readerConfig
		logsConfig.Topic = p.config.Topics.Logs
		p.readers["logs"] = kafka.NewReader(logsConfig)
	}

	if p.config.Topics.Events != "" {
		eventsConfig := readerConfig
		eventsConfig.Topic = p.config.Topics.Events
		p.readers["events"] = kafka.NewReader(eventsConfig)
	}

	log.Printf("Initialized %d Kafka readers", len(p.readers))
}

func (p *Processor) processStream(ctx context.Context, topic string, reader *kafka.Reader) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			message, err := reader.ReadMessage(ctx)
			cancel()

			if err != nil {
				if err == context.DeadlineExceeded {
					continue
				}
				log.Printf("Error reading from topic %s: %v", topic, err)
				p.stats.ProcessingErrors++
				continue
			}

			if err := p.processMessage(topic, message); err != nil {
				log.Printf("Error processing message from topic %s: %v", topic, err)
				p.stats.ProcessingErrors++
			} else {
				p.stats.MessagesProcessed++
				p.stats.LastProcessedTime = time.Now()
			}
		}
	}
}

func (p *Processor) processMessage(topic string, message kafka.Message) error {
	switch topic {
	case "metrics":
		return p.processMetricMessage(message)
	case "logs":
		return p.processLogMessage(message)
	case "events":
		return p.processEventMessage(message)
	default:
		return fmt.Errorf("unknown topic: %s", topic)
	}
}

func (p *Processor) processMetricMessage(message kafka.Message) error {
	var metric metrics.MetricPoint

	if err := json.Unmarshal(message.Value, &metric); err != nil {
		return fmt.Errorf("failed to unmarshal metric: %v", err)
	}

	if err := p.validateMetric(&metric); err != nil {
		return fmt.Errorf("invalid metric: %v", err)
	}

	p.queryEngine.ProcessMetric(&metric)

	return nil
}

func (p *Processor) processLogMessage(message kafka.Message) error {
	var logEntry metrics.LogEntry

	if err := json.Unmarshal(message.Value, &logEntry); err != nil {
		return fmt.Errorf("failed to unmarshal log entry: %v", err)
	}

	log.Printf("Processed log entry: %s/%s - %s", logEntry.Namespace, logEntry.PodName, logEntry.Level)

	return nil
}

func (p *Processor) processEventMessage(message kafka.Message) error {
	var event metrics.KubernetesEvent

	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal kubernetes event: %v", err)
	}

	eventMetric := &metrics.MetricPoint{
		Timestamp:     event.Timestamp,
		ClusterID:     event.ClusterID,
		Namespace:     event.Namespace,
		PodName:       event.Name,
		ContainerName: "",
		MetricName:    fmt.Sprintf("k8s_event_%s", event.Reason),
		Value:         float64(event.Count),
		Unit:          "count",
		Labels: map[string]string{
			"event_type":   event.Type,
			"event_reason": event.Reason,
			"kind":         event.Kind,
		},
	}

	p.queryEngine.ProcessMetric(eventMetric)

	return nil
}

func (p *Processor) validateMetric(metric *metrics.MetricPoint) error {
	if metric.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	if metric.ClusterID == "" {
		return fmt.Errorf("cluster_id is required")
	}
	if metric.MetricName == "" {
		return fmt.Errorf("metric_name is required")
	}
	if metric.Value < 0 {
		if metric.MetricName != "network_in" && metric.MetricName != "network_out" {
			return fmt.Errorf("negative values not allowed for metric: %s", metric.MetricName)
		}
	}
	return nil
}

func (p *Processor) reportStatistics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	var lastMessageCount uint64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentCount := p.stats.MessagesProcessed
			p.stats.ProcessingRate = float64(currentCount-lastMessageCount) / 30.0
			lastMessageCount = currentCount

			log.Printf("Stream Processor Stats: Messages: %d, Errors: %d, Rate: %.2f msg/s",
				p.stats.MessagesProcessed,
				p.stats.ProcessingErrors,
				p.stats.ProcessingRate)
		}
	}
}

func (p *Processor) GetStats() ProcessorStats {
	return p.stats
}

type MockDataGenerator struct {
	writer     *kafka.Writer
	stopCh     chan struct{}
	interval   time.Duration
	clusterIDs []string
	namespaces []string
	metrics    []string
	pods       []string
}

func NewMockDataGenerator(brokers []string, topic string) *MockDataGenerator {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &MockDataGenerator{
		writer:     writer,
		stopCh:     make(chan struct{}),
		interval:   time.Second,
		clusterIDs: []string{"prod-cluster", "staging-cluster", "dev-cluster"},
		namespaces: []string{"default", "kube-system", "monitoring", "ingress"},
		metrics:    []string{"cpu_usage", "memory_usage", "disk_usage", "network_in", "network_out"},
		pods:       []string{"pod-1", "pod-2", "pod-3", "pod-4", "pod-5"},
	}
}

func (mdg *MockDataGenerator) Start(ctx context.Context) {
	log.Println("Starting mock data generator...")

	ticker := time.NewTicker(mdg.interval)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-mdg.stopCh:
			return
		case <-ticker.C:
			metric := mdg.generateMetric()
			if err := mdg.sendMetric(ctx, metric); err != nil {
				log.Printf("Failed to send mock metric: %v", err)
			} else {
				count++
				if count%100 == 0 {
					log.Printf("Generated %d mock metrics", count)
				}
			}
		}
	}
}

func (mdg *MockDataGenerator) Stop() {
	close(mdg.stopCh)
	mdg.writer.Close()
}

func (mdg *MockDataGenerator) generateMetric() *metrics.MetricPoint {

	now := time.Now()
	metric := &metrics.MetricPoint{
		Timestamp:     now,
		ClusterID:     mdg.clusterIDs[rand.Intn(len(mdg.clusterIDs))],
		Namespace:     mdg.namespaces[rand.Intn(len(mdg.namespaces))],
		PodName:       mdg.pods[rand.Intn(len(mdg.pods))],
		ContainerName: "container-1",
		MetricName:    mdg.metrics[rand.Intn(len(mdg.metrics))],
		Value:         rand.Float64(),
		Unit:          "percent",
		Labels: map[string]string{
			"generated": "true",
			"source":    "mock-generator",
		},
	}

	if rand.Float32() < 0.05 {
		metric.Value = 0.9 + rand.Float64()*0.1
		metric.Labels["anomaly"] = "true"
	}

	return metric
}

func (mdg *MockDataGenerator) sendMetric(ctx context.Context, metric *metrics.MetricPoint) error {
	data, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	message := kafka.Message{
		Key:   []byte(metric.GetKey()),
		Value: data,
		Time:  metric.Timestamp,
	}

	return mdg.writer.WriteMessages(ctx, message)
}
