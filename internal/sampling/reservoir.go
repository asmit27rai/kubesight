package sampling

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/asmit27rai/kubesight/pkg/metrics"
)

type ReservoirSampler struct {
	capacity int
	samples  []*metrics.MetricPoint
	count    uint64
	mutex    sync.RWMutex
	rng      *rand.Rand
}

func NewReservoirSampler(capacity int) *ReservoirSampler {
	return &ReservoirSampler{
		capacity: capacity,
		samples:  make([]*metrics.MetricPoint, 0, capacity),
		count:    0,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (rs *ReservoirSampler) Add(metric *metrics.MetricPoint) *metrics.MetricPoint {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	rs.count++

	if len(rs.samples) < rs.capacity {
		sample := *metric
		rs.samples = append(rs.samples, &sample)
		return &sample
	}

	randomIndex := rs.rng.Intn(int(rs.count))

	if randomIndex < rs.capacity {
		sample := *metric
		rs.samples[randomIndex] = &sample
		return &sample
	}

	return nil
}

func (rs *ReservoirSampler) GetSamples() []*metrics.MetricPoint {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	result := make([]*metrics.MetricPoint, len(rs.samples))
	for i, sample := range rs.samples {
		copied := *sample
		result[i] = &copied
	}

	return result
}

func (rs *ReservoirSampler) Size() int {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	return len(rs.samples)
}

func (rs *ReservoirSampler) Count() uint64 {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	return rs.count
}

func (rs *ReservoirSampler) Clear() {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	rs.samples = rs.samples[:0]
	rs.count = 0
}

func (rs *ReservoirSampler) GetRandomSample() *metrics.MetricPoint {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	if len(rs.samples) == 0 {
		return nil
	}

	idx := rs.rng.Intn(len(rs.samples))
	sample := *rs.samples[idx]
	return &sample
}

type WeightedReservoirSampler struct {
	capacity int
	samples  []WeightedSample
	mutex    sync.RWMutex
	rng      *rand.Rand
}

type WeightedSample struct {
	Metric *metrics.MetricPoint
	Weight float64
	Key    float64
}

func NewWeightedReservoirSampler(capacity int) *WeightedReservoirSampler {
	return &WeightedReservoirSampler{
		capacity: capacity,
		samples:  make([]WeightedSample, 0, capacity),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (wrs *WeightedReservoirSampler) AddWeighted(metric *metrics.MetricPoint, weight float64) {
	if weight <= 0 {
		return
	}

	wrs.mutex.Lock()
	defer wrs.mutex.Unlock()

	u := wrs.rng.Float64()
	key := math.Pow(u, 1.0/weight)

	sample := WeightedSample{
		Metric: metric,
		Weight: weight,
		Key:    key,
	}

	if len(wrs.samples) < wrs.capacity {
		wrs.samples = append(wrs.samples, sample)
		return
	}

	minIdx := 0
	minKey := wrs.samples[0].Key
	for i := 1; i < len(wrs.samples); i++ {
		if wrs.samples[i].Key < minKey {
			minKey = wrs.samples[i].Key
			minIdx = i
		}
	}

	if key > minKey {
		wrs.samples[minIdx] = sample
	}
}

func (wrs *WeightedReservoirSampler) GetWeightedSamples() []WeightedSample {
	wrs.mutex.RLock()
	defer wrs.mutex.RUnlock()

	result := make([]WeightedSample, len(wrs.samples))
	copy(result, wrs.samples)
	return result
}

type StratifiedSampler struct {
	strata        map[string]*ReservoirSampler
	totalCapacity int
	allocation    StratificationStrategy
	mutex         sync.RWMutex
}

type StratificationStrategy int

const (
	ProportionalAllocation StratificationStrategy = iota
	EqualAllocation
	OptimalAllocation
)

func NewStratifiedSampler(totalCapacity int, strategy StratificationStrategy) *StratifiedSampler {
	return &StratifiedSampler{
		strata:        make(map[string]*ReservoirSampler),
		totalCapacity: totalCapacity,
		allocation:    strategy,
	}
}

func (ss *StratifiedSampler) AddToStratum(stratum string, metric *metrics.MetricPoint) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if _, exists := ss.strata[stratum]; !exists {
		capacity := ss.calculateStratumCapacity()
		ss.strata[stratum] = NewReservoirSampler(capacity)
	}

	ss.strata[stratum].Add(metric)
}

func (ss *StratifiedSampler) GetStratumSamples(stratum string) []*metrics.MetricPoint {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	if reservoir, exists := ss.strata[stratum]; exists {
		return reservoir.GetSamples()
	}
	return nil
}

func (ss *StratifiedSampler) GetAllStrata() []string {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	strata := make([]string, 0, len(ss.strata))
	for stratum := range ss.strata {
		strata = append(strata, stratum)
	}
	return strata
}

func (ss *StratifiedSampler) GetAllSamples() map[string][]*metrics.MetricPoint {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	result := make(map[string][]*metrics.MetricPoint)
	for stratum, reservoir := range ss.strata {
		result[stratum] = reservoir.GetSamples()
	}
	return result
}

func (ss *StratifiedSampler) calculateStratumCapacity() int {
	switch ss.allocation {
	case EqualAllocation:
		return ss.totalCapacity / max(1, len(ss.strata)+1)
	case OptimalAllocation:
		return ss.totalCapacity / max(1, len(ss.strata)+1)
	default:
		return ss.totalCapacity / max(1, len(ss.strata)+1)
	}
}

type TimeBasedSampler struct {
	windowSize    time.Duration
	windows       map[int64]*ReservoirSampler
	maxWindows    int
	samplerConfig ReservoirConfig
	mutex         sync.RWMutex
}

type ReservoirConfig struct {
	Capacity int
}

func NewTimeBasedSampler(windowSize time.Duration, maxWindows int, config ReservoirConfig) *TimeBasedSampler {
	return &TimeBasedSampler{
		windowSize:    windowSize,
		windows:       make(map[int64]*ReservoirSampler),
		maxWindows:    maxWindows,
		samplerConfig: config,
	}
}

func (tbs *TimeBasedSampler) Add(metric *metrics.MetricPoint) {
	tbs.mutex.Lock()
	defer tbs.mutex.Unlock()

	windowKey := metric.Timestamp.Unix() / int64(tbs.windowSize.Seconds())

	if _, exists := tbs.windows[windowKey]; !exists {
		tbs.windows[windowKey] = NewReservoirSampler(tbs.samplerConfig.Capacity)

		if len(tbs.windows) > tbs.maxWindows {
			tbs.cleanupOldWindows()
		}
	}

	tbs.windows[windowKey].Add(metric)
}

func (tbs *TimeBasedSampler) GetWindowSamples(timestamp time.Time) []*metrics.MetricPoint {
	tbs.mutex.RLock()
	defer tbs.mutex.RUnlock()

	windowKey := timestamp.Unix() / int64(tbs.windowSize.Seconds())
	if reservoir, exists := tbs.windows[windowKey]; exists {
		return reservoir.GetSamples()
	}
	return nil
}

func (tbs *TimeBasedSampler) GetRecentSamples(numWindows int) []*metrics.MetricPoint {
	tbs.mutex.RLock()
	defer tbs.mutex.RUnlock()

	keys := make([]int64, 0, len(tbs.windows))
	for key := range tbs.windows {
		keys = append(keys, key)
	}

	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] < keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	var result []*metrics.MetricPoint
	for i := 0; i < min(numWindows, len(keys)); i++ {
		if reservoir, exists := tbs.windows[keys[i]]; exists {
			result = append(result, reservoir.GetSamples()...)
		}
	}

	return result
}

func (tbs *TimeBasedSampler) cleanupOldWindows() {
	keys := make([]int64, 0, len(tbs.windows))
	for key := range tbs.windows {
		keys = append(keys, key)
	}

	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] < keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	toRemove := len(keys) - tbs.maxWindows + 1
	for i := 0; i < toRemove && i < len(keys); i++ {
		delete(tbs.windows, keys[len(keys)-1-i])
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
