package probabilistic

import (
	"hash/fnv"
	"math"
	"sync"
	"fmt"
)

type HyperLogLog struct {
	precision uint8
	m         uint32
	buckets   []uint8
	alpha     float64
	mutex     sync.RWMutex
}

func NewHyperLogLog(precision uint8) *HyperLogLog {
	if precision < 4 || precision > 16 {
		precision = 14
	}

	m := uint32(1) << precision
	hll := &HyperLogLog{
		precision: precision,
		m:         m,
		buckets:   make([]uint8, m),
		alpha:     calculateAlpha(m),
	}

	return hll
}

func (hll *HyperLogLog) Add(data []byte) {
	hll.mutex.Lock()
	defer hll.mutex.Unlock()

	hash := hashBytes(data)
	
	bucketIdx := hash >> (64 - hll.precision)
	
	w := hash << hll.precision
	leadingZeros := uint8(1)
	if w != 0 {
		leadingZeros = uint8(countLeadingZeros(w)) + 1
	}

	if leadingZeros > hll.buckets[bucketIdx] {
		hll.buckets[bucketIdx] = leadingZeros
	}
}

func (hll *HyperLogLog) Count() uint64 {
	hll.mutex.RLock()
	defer hll.mutex.RUnlock()

	sum := 0.0
	emptyBuckets := 0

	for _, bucket := range hll.buckets {
		if bucket == 0 {
			emptyBuckets++
		}
		sum += math.Pow(2, -float64(bucket))
	}

	estimate := hll.alpha * math.Pow(float64(hll.m), 2) / sum

	if estimate <= 2.5*float64(hll.m) && emptyBuckets > 0 {
		estimate = float64(hll.m) * math.Log(float64(hll.m)/float64(emptyBuckets))
	}

	if estimate > (1.0/30.0)*math.Pow(2, 32) {
		estimate = -math.Pow(2, 32) * math.Log(1-estimate/math.Pow(2, 32))
	}

	return uint64(estimate)
}

func (hll *HyperLogLog) Merge(other *HyperLogLog) error {
	if hll.precision != other.precision {
		return ErrPrecisionMismatch
	}

	hll.mutex.Lock()
	other.mutex.RLock()
	defer hll.mutex.Unlock()
	defer other.mutex.RUnlock()

	for i := uint32(0); i < hll.m; i++ {
		if other.buckets[i] > hll.buckets[i] {
			hll.buckets[i] = other.buckets[i]
		}
	}

	return nil
}

func (hll *HyperLogLog) Clear() {
	hll.mutex.Lock()
	defer hll.mutex.Unlock()

	for i := range hll.buckets {
		hll.buckets[i] = 0
	}
}

func (hll *HyperLogLog) EstimateError() float64 {
	return 1.04 / math.Sqrt(float64(hll.m))
}

func (hll *HyperLogLog) GetStats() HLLStats {
	hll.mutex.RLock()
	defer hll.mutex.RUnlock()

	emptyBuckets := 0
	maxBucket := uint8(0)

	for _, bucket := range hll.buckets {
		if bucket == 0 {
			emptyBuckets++
		}
		if bucket > maxBucket {
			maxBucket = bucket
		}
	}

	return HLLStats{
		Precision:     hll.precision,
		Buckets:       hll.m,
		EmptyBuckets:  uint32(emptyBuckets),
		MaxBucket:     maxBucket,
		EstimatedError: hll.EstimateError(),
	}
}

type HLLStats struct {
	Precision     uint8   `json:"precision"`
	Buckets       uint32  `json:"buckets"`
	EmptyBuckets  uint32  `json:"empty_buckets"`
	MaxBucket     uint8   `json:"max_bucket"`
	EstimatedError float64 `json:"estimated_error"`
}

func calculateAlpha(m uint32) float64 {
	switch m {
	case 16:
		return 0.673
	case 32:
		return 0.697
	case 64:
		return 0.709
	default:
		return 0.7213 / (1 + 1.079/float64(m))
	}
}

func hashBytes(data []byte) uint64 {
	hasher := fnv.New64a()
	hasher.Write(data)
	return hasher.Sum64()
}

func countLeadingZeros(x uint64) int {
	if x == 0 {
		return 64
	}
	n := 0
	if x <= 0x00000000FFFFFFFF {
		n += 32
		x <<= 32
	}
	if x <= 0x0000FFFFFFFFFFFF {
		n += 16
		x <<= 16
	}
	if x <= 0x00FFFFFFFFFFFFFF {
		n += 8
		x <<= 8
	}
	if x <= 0x0FFFFFFFFFFFFFFF {
		n += 4
		x <<= 4
	}
	if x <= 0x3FFFFFFFFFFFFFFF {
		n += 2
		x <<= 2
	}
	if x <= 0x7FFFFFFFFFFFFFFF {
		n += 1
	}
	return n
}

var (
	ErrPrecisionMismatch = fmt.Errorf("precision mismatch between HyperLogLogs")
)