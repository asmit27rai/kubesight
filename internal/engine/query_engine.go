package engine

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asmit27rai/kubesight/internal/probabilistic"
	"github.com/asmit27rai/kubesight/internal/sampling"
	"github.com/asmit27rai/kubesight/pkg/metrics"
)

type QueryEngine struct {
	hll        *probabilistic.HyperLogLog
	cms        *probabilistic.CountMinSketch
	bloom      *probabilistic.BloomFilter
	sampler    *sampling.AdaptiveSampler
	samples    map[string][]*metrics.MetricPoint
	mutex      sync.RWMutex
	stats      QueryEngineStats
}

type QueryEngineStats struct {
	TotalQueries    uint64        `json:"total_queries"`
	ApproxQueries   uint64        `json:"approx_queries"`
	AvgLatency      time.Duration `json:"avg_latency"`
	TotalSamples    uint64        `json:"total_samples"`
	ErrorRate       float64       `json:"error_rate"`
	LastUpdateTime  time.Time     `json:"last_update"`
}

func NewQueryEngine(config QueryEngineConfig) *QueryEngine {
	return &QueryEngine{
		hll:     probabilistic.NewHyperLogLog(config.HLLPrecision),
		cms:     probabilistic.NewCountMinSketch(config.CMSWidth, config.CMSDepth),
		bloom:   probabilistic.NewBloomFilter(config.BloomSize, config.BloomHashes),
		sampler: sampling.NewAdaptiveSampler(config.SamplingConfig),
		samples: make(map[string][]*metrics.MetricPoint),
		stats:   QueryEngineStats{LastUpdateTime: time.Now()},
	}
}

type QueryEngineConfig struct {
	HLLPrecision   uint8                      `json:"hll_precision"`
	CMSWidth       uint32                     `json:"cms_width"`
	CMSDepth       uint32                     `json:"cms_depth"`
	BloomSize      uint32                     `json:"bloom_size"`
	BloomHashes    uint32                     `json:"bloom_hashes"`
	SamplingConfig sampling.SamplingConfig    `json:"sampling_config"`
}

func (qe *QueryEngine) ProcessMetric(metric *metrics.MetricPoint) {
	qe.mutex.Lock()
	defer qe.mutex.Unlock()

	if sampled, shouldSample := qe.sampler.Sample(metric); shouldSample && sampled != nil {
		qe.updateDataStructures(sampled)
		
		key := qe.getMetricKey(sampled)
		qe.samples[key] = append(qe.samples[key], sampled)
		
		if len(qe.samples[key]) > 1000 {
			qe.samples[key] = qe.samples[key][len(qe.samples[key])-1000:]
		}
	}

	qe.stats.TotalSamples++
}

func (qe *QueryEngine) ExecuteQuery(request *metrics.QueryRequest) (*metrics.QueryResult, error) {
	startTime := time.Now()
	
	qe.mutex.Lock()
	qe.stats.TotalQueries++
	qe.mutex.Unlock()

	result, err := qe.processQuery(request)
	if err != nil {
		return nil, err
	}

	processingTime := time.Since(startTime)
	
	qe.mutex.Lock()
	qe.stats.AvgLatency = time.Duration((int64(qe.stats.AvgLatency)*int64(qe.stats.TotalQueries-1) + int64(processingTime)) / int64(qe.stats.TotalQueries))
	if result.IsApproximate {
		qe.stats.ApproxQueries++
	}
	qe.mutex.Unlock()

	result.ProcessingTime = processingTime
	result.Timestamp = time.Now()

	return result, nil
}

func (qe *QueryEngine) processQuery(request *metrics.QueryRequest) (*metrics.QueryResult, error) {
	switch request.QueryType {
	case metrics.CountDistinct:
		return qe.executeCountDistinct(request)
	case metrics.Sum:
		return qe.executeSum(request)
	case metrics.Average:
		return qe.executeAverage(request)
	case metrics.Percentile:
		return qe.executePercentile(request)
	case metrics.TopK:
		return qe.executeTopK(request)
	case metrics.Membership:
		return qe.executeMembership(request)
	case metrics.FrequencyCount:
		return qe.executeFrequencyCount(request)
	default:
		return nil, fmt.Errorf("unsupported query type: %s", request.QueryType)
	}
}

func (qe *QueryEngine) executeCountDistinct(request *metrics.QueryRequest) (*metrics.QueryResult, error) {
	qe.mutex.RLock()
	defer qe.mutex.RUnlock()

	count := qe.hll.Count()
	error := qe.hll.EstimateError()

	result := &metrics.ApproximateCountResult{
		Count:          count,
		EstimatedError: error,
	}

	return &metrics.QueryResult{
		ID:            request.ID,
		Query:         request.Query,
		Result:        result,
		Error:         &error,
		SampleSize:    len(qe.getAllSamples()),
		IsApproximate: true,
	}, nil
}

func (qe *QueryEngine) executeSum(request *metrics.QueryRequest) (*metrics.QueryResult, error) {
	samples := qe.getFilteredSamples(request)
	
	if len(samples) == 0 {
		return &metrics.QueryResult{
			ID:            request.ID,
			Query:         request.Query,
			Result:        0.0,
			SampleSize:    0,
			IsApproximate: false,
		}, nil
	}

	sum := 0.0
	for _, sample := range samples {
		sum += sample.Value
	}

	samplingRate := qe.sampler.GetEffectiveSamplingRate()
	estimatedSum := sum / samplingRate

	sampleVariance := qe.calculateVariance(samples)
	n := float64(len(samples))
	standardError := math.Sqrt(sampleVariance/n) / samplingRate
	
	errorBound := 1.96 * standardError
	confidence := 0.95

	return &metrics.QueryResult{
		ID:            request.ID,
		Query:         request.Query,
		Result:        estimatedSum,
		Error:         &errorBound,
		Confidence:    &confidence,
		SampleSize:    len(samples),
		IsApproximate: true,
	}, nil
}

func (qe *QueryEngine) executeAverage(request *metrics.QueryRequest) (*metrics.QueryResult, error) {
	samples := qe.getFilteredSamples(request)
	
	if len(samples) == 0 {
		return &metrics.QueryResult{
			ID:            request.ID,
			Query:         request.Query,
			Result:        0.0,
			SampleSize:    0,
			IsApproximate: false,
		}, nil
	}

	sum := 0.0
	for _, sample := range samples {
		sum += sample.Value
	}
	
	average := sum / float64(len(samples))
	
	variance := qe.calculateVariance(samples)
	standardError := math.Sqrt(variance / float64(len(samples)))
	confidence := 0.95

	return &metrics.QueryResult{
		ID:            request.ID,
		Query:         request.Query,
		Result:        average,
		Error:         &standardError,
		Confidence:    &confidence,
		SampleSize:    len(samples),
		IsApproximate: len(samples) < 1000,
	}, nil
}

func (qe *QueryEngine) executePercentile(request *metrics.QueryRequest) (*metrics.QueryResult, error) {
	samples := qe.getFilteredSamples(request)
	
	if len(samples) == 0 {
		return &metrics.QueryResult{
			ID:            request.ID,
			Query:         request.Query,
			Result:        nil,
			SampleSize:    0,
			IsApproximate: false,
		}, nil
	}

	percentileValue := qe.extractPercentileValue(request.Query)
	if percentileValue < 0 || percentileValue > 100 {
		return nil, fmt.Errorf("invalid percentile value: %f", percentileValue)
	}

	values := make([]float64, len(samples))
	for i, sample := range samples {
		values[i] = sample.Value
	}
	sort.Float64s(values)

	index := (percentileValue / 100.0) * float64(len(values)-1)
	lowerIndex := int(math.Floor(index))
	upperIndex := int(math.Ceil(index))
	
	var percentileResult float64
	if lowerIndex == upperIndex {
		percentileResult = values[lowerIndex]
	} else {
		weight := index - float64(lowerIndex)
		percentileResult = values[lowerIndex]*(1-weight) + values[upperIndex]*weight
	}

	result := &metrics.PercentileResult{
		Percentile: percentileValue,
		Value:      percentileResult,
		SampleSize: len(samples),
	}

	return &metrics.QueryResult{
		ID:            request.ID,
		Query:         request.Query,
		Result:        result,
		SampleSize:    len(samples),
		IsApproximate: true,
	}, nil
}

func (qe *QueryEngine) executeTopK(request *metrics.QueryRequest) (*metrics.QueryResult, error) {
	qe.mutex.RLock()
	defer qe.mutex.RUnlock()

	k := qe.extractKValue(request.Query)
	if k <= 0 {
		return nil, fmt.Errorf("invalid K value: %d", k)
	}

	heavyHitters := qe.cms.TopK(k)
	
	items := make([]metrics.TopKItem, len(heavyHitters))
	for i, hh := range heavyHitters {
		items[i] = metrics.TopKItem{
			Key:       fmt.Sprintf("bucket_%d", hh.Bucket),
			Count:     uint64(hh.Count),
			Frequency: hh.Frequency,
		}
	}

	result := &metrics.TopKResult{
		Items: items,
		K:     k,
	}

	return &metrics.QueryResult{
		ID:            request.ID,
		Query:         request.Query,
		Result:        result,
		SampleSize:    int(qe.cms.GetStats().TotalCount),
		IsApproximate: true,
	}, nil
}

func (qe *QueryEngine) executeMembership(request *metrics.QueryRequest) (*metrics.QueryResult, error) {
	qe.mutex.RLock()
	defer qe.mutex.RUnlock()

	item := qe.extractMembershipItem(request.Query)
	if item == "" {
		return nil, fmt.Errorf("no item specified for membership test")
	}

	isMember := qe.bloom.Contains([]byte(item))
	falsePositiveRate := qe.bloom.FalsePositiveRate()

	result := &metrics.MembershipResult{
		Member:      isMember,
		Probability: falsePositiveRate,
	}

	return &metrics.QueryResult{
		ID:            request.ID,
		Query:         request.Query,
		Result:        result,
		SampleSize:    int(qe.bloom.GetStats().NumItems),
		IsApproximate: true,
	}, nil
}

func (qe *QueryEngine) executeFrequencyCount(request *metrics.QueryRequest) (*metrics.QueryResult, error) {
	qe.mutex.RLock()
	defer qe.mutex.RUnlock()

	item := qe.extractFrequencyItem(request.Query)
	if item == "" {
		return nil, fmt.Errorf("no item specified for frequency count")
	}

	count := qe.cms.Estimate([]byte(item))

	return &metrics.QueryResult{
		ID:            request.ID,
		Query:         request.Query,
		Result:        count,
		SampleSize:    int(qe.cms.GetStats().TotalCount),
		IsApproximate: true,
	}, nil
}


func (qe *QueryEngine) updateDataStructures(metric *metrics.MetricPoint) {
	key := qe.getMetricKey(metric)
	qe.hll.Add([]byte(key))

	qe.cms.Update([]byte(key), 1)

	qe.bloom.Add([]byte(key))
}

func (qe *QueryEngine) getMetricKey(metric *metrics.MetricPoint) string {
	return fmt.Sprintf("%s/%s/%s/%s", 
		metric.ClusterID, metric.Namespace, metric.PodName, metric.MetricName)
}

func (qe *QueryEngine) getFilteredSamples(request *metrics.QueryRequest) []*metrics.MetricPoint {
	qe.mutex.RLock()
	defer qe.mutex.RUnlock()

	var allSamples []*metrics.MetricPoint
	for _, samples := range qe.samples {
		allSamples = append(allSamples, samples...)
	}

	var filtered []*metrics.MetricPoint
	for _, sample := range allSamples {
		if qe.matchesFilters(sample, request) {
			filtered = append(filtered, sample)
		}
	}

	return filtered
}

func (qe *QueryEngine) matchesFilters(metric *metrics.MetricPoint, request *metrics.QueryRequest) bool {
	if !request.TimeRange.Start.IsZero() && metric.Timestamp.Before(request.TimeRange.Start) {
		return false
	}
	if !request.TimeRange.End.IsZero() && metric.Timestamp.After(request.TimeRange.End) {
		return false
	}

	for key, value := range request.Filters {
		switch key {
		case "cluster_id":
			if metric.ClusterID != value {
				return false
			}
		case "namespace":
			if metric.Namespace != value {
				return false
			}
		case "metric_name":
			if metric.MetricName != value {
				return false
			}
		case "pod_name":
			if metric.PodName != value {
				return false
			}
		}
	}

	return true
}

func (qe *QueryEngine) getAllSamples() []*metrics.MetricPoint {
	var all []*metrics.MetricPoint
	for _, samples := range qe.samples {
		all = append(all, samples...)
	}
	return all
}

func (qe *QueryEngine) calculateVariance(samples []*metrics.MetricPoint) float64 {
	if len(samples) < 2 {
		return 0
	}

	sum := 0.0
	for _, sample := range samples {
		sum += sample.Value
	}
	mean := sum / float64(len(samples))

	sumSquares := 0.0
	for _, sample := range samples {
		diff := sample.Value - mean
		sumSquares += diff * diff
	}

	return sumSquares / float64(len(samples)-1)
}


func (qe *QueryEngine) extractPercentileValue(query string) float64 {
	if strings.Contains(query, "PERCENTILE") {
		start := strings.Index(query, "(") + 1
		end := strings.Index(query, ")")
		if start > 0 && end > start {
			if val, err := strconv.ParseFloat(query[start:end], 64); err == nil {
				return val
			}
		}
	}
	return 95.0
}

func (qe *QueryEngine) extractKValue(query string) int {
	if strings.Contains(query, "TOP_K") {
		start := strings.Index(query, "(") + 1
		end := strings.Index(query, ")")
		if start > 0 && end > start {
			if val, err := strconv.Atoi(query[start:end]); err == nil {
				return val
			}
		}
	}
	return 10
}

func (qe *QueryEngine) extractMembershipItem(query string) string {
	start := strings.Index(query, "'") + 1
	end := strings.LastIndex(query, "'")
	if start > 0 && end > start {
		return query[start:end]
	}
	return ""
}

func (qe *QueryEngine) extractFrequencyItem(query string) string {
	start := strings.Index(query, "'") + 1
	end := strings.LastIndex(query, "'")
	if start > 0 && end > start {
		return query[start:end]
	}
	return ""
}

func (qe *QueryEngine) GetStats() QueryEngineStats {
	qe.mutex.RLock()
	defer qe.mutex.RUnlock()
	return qe.stats
}