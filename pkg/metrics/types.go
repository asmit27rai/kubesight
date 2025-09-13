package metrics

import (
	"encoding/json"
	"time"
)

type MetricPoint struct {
	Timestamp     time.Time         `json:"timestamp"`
	ClusterID     string            `json:"cluster_id"`
	Namespace     string            `json:"namespace"`
	PodName       string            `json:"pod_name"`
	ContainerName string            `json:"container_name"`
	MetricName    string            `json:"metric_name"`
	Value         float64           `json:"value"`
	Unit          string            `json:"unit"`
	Labels        map[string]string `json:"labels"`
}

type LogEntry struct {
	Timestamp     time.Time         `json:"timestamp"`
	ClusterID     string            `json:"cluster_id"`
	Namespace     string            `json:"namespace"`
	PodName       string            `json:"pod_name"`
	ContainerName string            `json:"container_name"`
	Level         string            `json:"level"`
	Message       string            `json:"message"`
	Labels        map[string]string `json:"labels"`
}

type KubernetesEvent struct {
	Timestamp time.Time         `json:"timestamp"`
	ClusterID string            `json:"cluster_id"`
	Namespace string            `json:"namespace"`
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Reason    string            `json:"reason"`
	Type      string            `json:"type"`
	Message   string            `json:"message"`
	Count     int32             `json:"count"`
	Labels    map[string]string `json:"labels"`
}

type QueryRequest struct {
	ID         string            `json:"id"`
	Query      string            `json:"query"`
	QueryType  QueryType         `json:"query_type"`
	TimeRange  TimeRange         `json:"time_range"`
	Filters    map[string]string `json:"filters"`
	ErrorBound float64           `json:"error_bound,omitempty"`
	Confidence float64           `json:"confidence,omitempty"`
}

type QueryType string

const (
	CountDistinct  QueryType = "count_distinct"
	Sum            QueryType = "sum"
	Average        QueryType = "average"
	Percentile     QueryType = "percentile"
	TopK           QueryType = "top_k"
	Membership     QueryType = "membership"
	FrequencyCount QueryType = "frequency_count"
)

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type QueryResult struct {
	ID             string        `json:"id"`
	Query          string        `json:"query"`
	Result         interface{}   `json:"result"`
	Error          *float64      `json:"error,omitempty"`
	Confidence     *float64      `json:"confidence,omitempty"`
	SampleSize     int           `json:"sample_size"`
	ProcessingTime time.Duration `json:"processing_time"`
	IsApproximate  bool          `json:"is_approximate"`
	Timestamp      time.Time     `json:"timestamp"`
}

type ApproximateCountResult struct {
	Count          uint64  `json:"count"`
	EstimatedError float64 `json:"estimated_error"`
}

type TopKResult struct {
	Items []TopKItem `json:"items"`
	K     int        `json:"k"`
}

type TopKItem struct {
	Key       string  `json:"key"`
	Count     uint64  `json:"count"`
	Frequency float64 `json:"frequency"`
}

type PercentileResult struct {
	Percentile float64 `json:"percentile"`
	Value      float64 `json:"value"`
	SampleSize int     `json:"sample_size"`
}

type MembershipResult struct {
	Member      bool    `json:"member"`
	Probability float64 `json:"probability"` // Probability of false positive
}

type SystemStats struct {
	Timestamp       time.Time `json:"timestamp"`
	TotalMetrics    uint64    `json:"total_metrics"`
	SampledMetrics  uint64    `json:"sampled_metrics"`
	SamplingRate    float64   `json:"sampling_rate"`
	ProcessingRate  float64   `json:"processing_rate"` // metrics/second
	MemoryUsage     uint64    `json:"memory_usage"`
	QueryLatencyP95 float64   `json:"query_latency_p95"`
	ErrorRate       float64   `json:"error_rate"`
}

func (mp *MetricPoint) IsAnomaly() bool {
	if mp.MetricName == "cpu_usage" && mp.Value > 0.9 {
		return true
	}
	if mp.MetricName == "memory_usage" && mp.Value > 0.85 {
		return true
	}
	if mp.MetricName == "pod_restarts" && mp.Value > 3 {
		return true
	}
	return false
}

func (mp *MetricPoint) ToJSON() (string, error) {
	data, err := json.Marshal(mp)
	return string(data), err
}

func (mp *MetricPoint) FromJSON(data string) error {
	return json.Unmarshal([]byte(data), mp)
}

func (mp *MetricPoint) GetKey() string {
	return mp.ClusterID + "/" + mp.Namespace + "/" + mp.PodName + "/" + mp.MetricName
}
