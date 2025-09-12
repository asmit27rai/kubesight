package probabilistic

import (
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"sync"
)

type CountMinSketch struct {
	width   uint32
	depth   uint32
	count   [][]uint32
	hashA   []uint32
	hashB   []uint32
	mutex   sync.RWMutex
	total   uint64
}

func NewCountMinSketch(width, depth uint32) *CountMinSketch {
	cms := &CountMinSketch{
		width: width,
		depth: depth,
		count: make([][]uint32, depth),
		hashA: make([]uint32, depth),
		hashB: make([]uint32, depth),
		total: 0,
	}

	for i := uint32(0); i < depth; i++ {
		cms.count[i] = make([]uint32, width)
	}

	for i := uint32(0); i < depth; i++ {
		cms.hashA[i] = uint32(i*2 + 1)
		cms.hashB[i] = uint32(i*3 + 7)
	}

	return cms
}

func NewCountMinSketchFromErrorRate(errorRate, confidence float64) *CountMinSketch {
	width := uint32(math.Ceil(math.E / errorRate))
	depth := uint32(math.Ceil(math.Log(1 / confidence)))
	return NewCountMinSketch(width, depth)
}

func (cms *CountMinSketch) Update(item []byte, count uint32) {
	cms.mutex.Lock()
	defer cms.mutex.Unlock()

	hash := cms.hash(item)

	for i := uint32(0); i < cms.depth; i++ {
		bucket := cms.getBucket(hash, i)
		cms.count[i][bucket] += count
	}

	cms.total += uint64(count)
}

func (cms *CountMinSketch) Estimate(item []byte) uint32 {
	cms.mutex.RLock()
	defer cms.mutex.RUnlock()

	hash := cms.hash(item)
	minCount := uint32(math.MaxUint32)

	for i := uint32(0); i < cms.depth; i++ {
		bucket := cms.getBucket(hash, i)
		if cms.count[i][bucket] < minCount {
			minCount = cms.count[i][bucket]
		}
	}

	return minCount
}

func (cms *CountMinSketch) HeavyHitters(threshold float64) []HeavyHitterItem {
	cms.mutex.RLock()
	defer cms.mutex.RUnlock()

	minThreshold := uint32(threshold * float64(cms.total))
	
	candidates := make(map[uint32]uint32)
	
	for i := uint32(0); i < cms.width; i++ {
		if cms.count[0][i] >= minThreshold {
			candidates[i] = cms.count[0][i]
		}
	}

	var results []HeavyHitterItem
	for bucket, count := range candidates {
		results = append(results, HeavyHitterItem{
			Bucket:    bucket,
			Count:     count,
			Frequency: float64(count) / float64(cms.total),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Frequency > results[j].Frequency
	})

	return results
}

func (cms *CountMinSketch) TopK(k int) []HeavyHitterItem {
	heavyHitters := cms.HeavyHitters(0.0)
	
	if len(heavyHitters) > k {
		return heavyHitters[:k]
	}
	
	return heavyHitters
}

func (cms *CountMinSketch) Merge(other *CountMinSketch) error {
	if cms.width != other.width || cms.depth != other.depth {
		return fmt.Errorf("dimension mismatch: cannot merge CMS of different dimensions")
	}

	cms.mutex.Lock()
	other.mutex.RLock()
	defer cms.mutex.Unlock()
	defer other.mutex.RUnlock()

	for i := uint32(0); i < cms.depth; i++ {
		for j := uint32(0); j < cms.width; j++ {
			cms.count[i][j] += other.count[i][j]
		}
	}

	cms.total += other.total

	return nil
}

func (cms *CountMinSketch) Clear() {
	cms.mutex.Lock()
	defer cms.mutex.Unlock()

	for i := uint32(0); i < cms.depth; i++ {
		for j := uint32(0); j < cms.width; j++ {
			cms.count[i][j] = 0
		}
	}
	cms.total = 0
}

func (cms *CountMinSketch) GetStats() CMSStats {
	cms.mutex.RLock()
	defer cms.mutex.RUnlock()

	totalCells := cms.width * cms.depth
	nonZeroCells := uint32(0)
	maxCount := uint32(0)

	for i := uint32(0); i < cms.depth; i++ {
		for j := uint32(0); j < cms.width; j++ {
			if cms.count[i][j] > 0 {
				nonZeroCells++
			}
			if cms.count[i][j] > maxCount {
				maxCount = cms.count[i][j]
			}
		}
	}

	return CMSStats{
		Width:        cms.width,
		Depth:        cms.depth,
		TotalCells:   totalCells,
		NonZeroCells: nonZeroCells,
		MaxCount:     maxCount,
		TotalCount:   cms.total,
		LoadFactor:   float64(nonZeroCells) / float64(totalCells),
	}
}

func (cms *CountMinSketch) hash(data []byte) uint64 {
	hasher := fnv.New64a()
	hasher.Write(data)
	return hasher.Sum64()
}

func (cms *CountMinSketch) getBucket(hash uint64, row uint32) uint32 {
	a := uint64(cms.hashA[row])
	b := uint64(cms.hashB[row])
	return uint32((a*hash + b) % uint64(cms.width))
}

type HeavyHitterItem struct {
	Bucket    uint32  `json:"bucket"`
	Count     uint32  `json:"count"`
	Frequency float64 `json:"frequency"`
}

type CMSStats struct {
	Width        uint32  `json:"width"`
	Depth        uint32  `json:"depth"`
	TotalCells   uint32  `json:"total_cells"`
	NonZeroCells uint32  `json:"non_zero_cells"`
	MaxCount     uint32  `json:"max_count"`
	TotalCount   uint64  `json:"total_count"`
	LoadFactor   float64 `json:"load_factor"`
}