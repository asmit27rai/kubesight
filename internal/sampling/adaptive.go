package sampling

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/asmit27rai/kubesight/pkg/metrics"
)

type AdaptiveSampler struct {
	config          SamplingConfig
	reservoirs      map[string]*ReservoirSampler
	statistics      map[string]*WindowStats
	anomalyDetector *AnomalyDetector
	mutex           sync.RWMutex
	rng             *rand.Rand
	totalProcessed  uint64
	totalSampled    uint64
}

type SamplingConfig struct {
	BaseRate       float64            `json:"base_rate"`
	AnomalyRate    float64            `json:"anomaly_rate"`
	WindowSize     time.Duration      `json:"window_size"`
	ReservoirSize  int                `json:"reservoir_size"`
	StratumWeights map[string]float64 `json:"stratum_weights"`
}

func NewAdaptiveSampler(config SamplingConfig) *AdaptiveSampler {
	return &AdaptiveSampler{
		config:          config,
		reservoirs:      make(map[string]*ReservoirSampler),
		statistics:      make(map[string]*WindowStats),
		anomalyDetector: NewAnomalyDetector(),
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
		totalProcessed:  0,
		totalSampled:    0,
	}
}

func (as *AdaptiveSampler) ShouldSample(metric *metrics.MetricPoint) bool {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	as.totalProcessed++

	samplingRate := as.calculateSamplingRate(metric)

	shouldSample := as.rng.Float64() < samplingRate
	if shouldSample {
		as.totalSampled++
	}

	return shouldSample
}

func (as *AdaptiveSampler) Sample(metric *metrics.MetricPoint) (*metrics.MetricPoint, bool) {
	if !as.ShouldSample(metric) {
		return nil, false
	}

	as.updateStatistics(metric)

	stratum := as.getStratum(metric)
	reservoir := as.getOrCreateReservoir(stratum)

	sampled := reservoir.Add(metric)

	return sampled, sampled != nil
}

func (as *AdaptiveSampler) GetEffectiveSamplingRate() float64 {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	if as.totalProcessed == 0 {
		return as.config.BaseRate
	}

	return float64(as.totalSampled) / float64(as.totalProcessed)
}

func (as *AdaptiveSampler) GetSamples(stratum string) []*metrics.MetricPoint {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	if reservoir, exists := as.reservoirs[stratum]; exists {
		return reservoir.GetSamples()
	}
	return nil
}

func (as *AdaptiveSampler) GetAllSamples() map[string][]*metrics.MetricPoint {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	result := make(map[string][]*metrics.MetricPoint)
	for stratum, reservoir := range as.reservoirs {
		result[stratum] = reservoir.GetSamples()
	}
	return result
}

func (as *AdaptiveSampler) GetStats() SamplingStats {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	return SamplingStats{
		TotalProcessed:        as.totalProcessed,
		TotalSampled:          as.totalSampled,
		EffectiveSamplingRate: as.GetEffectiveSamplingRate(),
		ActiveReservoirs:      len(as.reservoirs),
		BaseRate:              as.config.BaseRate,
		AnomalyRate:           as.config.AnomalyRate,
	}
}

type SamplingStats struct {
	TotalProcessed        uint64  `json:"total_processed"`
	TotalSampled          uint64  `json:"total_sampled"`
	EffectiveSamplingRate float64 `json:"effective_sampling_rate"`
	ActiveReservoirs      int     `json:"active_reservoirs"`
	BaseRate              float64 `json:"base_rate"`
	AnomalyRate           float64 `json:"anomaly_rate"`
}

func (as *AdaptiveSampler) calculateSamplingRate(metric *metrics.MetricPoint) float64 {
	baseRate := as.config.BaseRate

	if as.anomalyDetector.IsAnomaly(metric) {
		baseRate = math.Max(baseRate, as.config.AnomalyRate)
	}

	stratum := as.getStratum(metric)
	if weight, exists := as.config.StratumWeights[stratum]; exists {
		baseRate *= weight
	}

	if stats, exists := as.statistics[stratum]; exists {
		variance := stats.GetVariance()
		baseRate *= (1.0 + variance/100.0)
	}

	if metric.MetricName == "cpu_usage" || metric.MetricName == "memory_usage" {
		if metric.Value > 0.8 {
			baseRate *= 2.0
		}
	}

	return math.Min(math.Max(baseRate, 0.001), 1.0)
}

func (as *AdaptiveSampler) getStratum(metric *metrics.MetricPoint) string {
	return metric.ClusterID + "/" + metric.Namespace + "/" + metric.MetricName
}

func (as *AdaptiveSampler) getOrCreateReservoir(stratum string) *ReservoirSampler {
	if reservoir, exists := as.reservoirs[stratum]; exists {
		return reservoir
	}

	reservoir := NewReservoirSampler(as.config.ReservoirSize)
	as.reservoirs[stratum] = reservoir
	return reservoir
}

func (as *AdaptiveSampler) updateStatistics(metric *metrics.MetricPoint) {
	stratum := as.getStratum(metric)

	if _, exists := as.statistics[stratum]; !exists {
		as.statistics[stratum] = NewWindowStats(as.config.WindowSize)
	}

	as.statistics[stratum].Add(metric.Value, metric.Timestamp)
}

type WindowStats struct {
	values     []float64
	timestamps []time.Time
	windowSize time.Duration
	sum        float64
	sumSquares float64
	mutex      sync.RWMutex
}

func NewWindowStats(windowSize time.Duration) *WindowStats {
	return &WindowStats{
		values:     make([]float64, 0),
		timestamps: make([]time.Time, 0),
		windowSize: windowSize,
	}
}

func (ws *WindowStats) Add(value float64, timestamp time.Time) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	ws.values = append(ws.values, value)
	ws.timestamps = append(ws.timestamps, timestamp)
	ws.sum += value
	ws.sumSquares += value * value

	ws.cleanup(timestamp)
}

func (ws *WindowStats) GetMean() float64 {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	if len(ws.values) == 0 {
		return 0
	}
	return ws.sum / float64(len(ws.values))
}

func (ws *WindowStats) GetVariance() float64 {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	if len(ws.values) < 2 {
		return 0
	}

	n := float64(len(ws.values))
	mean := ws.sum / n
	return (ws.sumSquares / n) - (mean * mean)
}

func (ws *WindowStats) cleanup(currentTime time.Time) {
	cutoff := currentTime.Add(-ws.windowSize)

	keepFrom := 0
	for i, ts := range ws.timestamps {
		if ts.After(cutoff) {
			keepFrom = i
			break
		}
	}

	if keepFrom > 0 {
		for i := 0; i < keepFrom; i++ {
			ws.sum -= ws.values[i]
			ws.sumSquares -= ws.values[i] * ws.values[i]
		}

		ws.values = ws.values[keepFrom:]
		ws.timestamps = ws.timestamps[keepFrom:]
	}
}

type AnomalyDetector struct {
	thresholds map[string]AnomalyThreshold
	mutex      sync.RWMutex
}

type AnomalyThreshold struct {
	MetricName string  `json:"metric_name"`
	UpperBound float64 `json:"upper_bound"`
	LowerBound float64 `json:"lower_bound"`
	ZScore     float64 `json:"z_score"`
}

func NewAnomalyDetector() *AnomalyDetector {
	detector := &AnomalyDetector{
		thresholds: make(map[string]AnomalyThreshold),
	}

	detector.setDefaultThresholds()
	return detector
}

func (ad *AnomalyDetector) IsAnomaly(metric *metrics.MetricPoint) bool {
	ad.mutex.RLock()
	defer ad.mutex.RUnlock()

	if metric.IsAnomaly() {
		return true
	}

	if threshold, exists := ad.thresholds[metric.MetricName]; exists {
		return metric.Value > threshold.UpperBound || metric.Value < threshold.LowerBound
	}

	return false
}

func (ad *AnomalyDetector) setDefaultThresholds() {
	ad.thresholds = map[string]AnomalyThreshold{
		"cpu_usage": {
			MetricName: "cpu_usage",
			UpperBound: 0.9,
			LowerBound: 0.0,
			ZScore:     3.0,
		},
		"memory_usage": {
			MetricName: "memory_usage",
			UpperBound: 0.85,
			LowerBound: 0.0,
			ZScore:     3.0,
		},
		"disk_usage": {
			MetricName: "disk_usage",
			UpperBound: 0.9,
			LowerBound: 0.0,
			ZScore:     2.5,
		},
		"network_latency": {
			MetricName: "network_latency",
			UpperBound: 1000.0, // ms
			LowerBound: 0.0,
			ZScore:     3.0,
		},
	}
}
