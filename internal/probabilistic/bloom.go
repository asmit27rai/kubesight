package probabilistic

import (
	"fmt"
	"hash/fnv"
	"math"
	"sync"
)

type BloomFilter struct {
	bits      []bool
	size      uint32
	numHashes uint32
	numItems  uint32
	mutex     sync.RWMutex
}

func NewBloomFilter(size, numHashes uint32) *BloomFilter {
	return &BloomFilter{
		bits:      make([]bool, size),
		size:      size,
		numHashes: numHashes,
		numItems:  0,
	}
}

func NewBloomFilterOptimal(expectedItems uint32, falsePositiveRate float64) *BloomFilter {
	size := uint32(-float64(expectedItems) * math.Log(falsePositiveRate) / (math.Log(2) * math.Log(2)))

	numHashes := uint32((float64(size) / float64(expectedItems)) * math.Log(2))

	if numHashes == 0 {
		numHashes = 1
	}

	return NewBloomFilter(size, numHashes)
}

func (bf *BloomFilter) Add(item []byte) {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	hashes := bf.getHashes(item)

	for _, hash := range hashes {
		index := hash % bf.size
		bf.bits[index] = true
	}

	bf.numItems++
}

func (bf *BloomFilter) Contains(item []byte) bool {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	hashes := bf.getHashes(item)

	for _, hash := range hashes {
		index := hash % bf.size
		if !bf.bits[index] {
			return false
		}
	}

	return true
}

func (bf *BloomFilter) Union(other *BloomFilter) error {
	if bf.size != other.size || bf.numHashes != other.numHashes {
		return fmt.Errorf("cannot union bloom filters with different parameters")
	}

	bf.mutex.Lock()
	other.mutex.RLock()
	defer bf.mutex.Unlock()
	defer other.mutex.RUnlock()

	for i := uint32(0); i < bf.size; i++ {
		bf.bits[i] = bf.bits[i] || other.bits[i]
	}

	bf.numItems += other.numItems

	return nil
}

func (bf *BloomFilter) Clear() {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	for i := range bf.bits {
		bf.bits[i] = false
	}
	bf.numItems = 0
}

func (bf *BloomFilter) FalsePositiveRate() float64 {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	if bf.numItems == 0 {
		return 0.0
	}

	k := float64(bf.numHashes)
	n := float64(bf.numItems)
	m := float64(bf.size)

	return math.Pow(1-math.Exp(-k*n/m), k)
}

func (bf *BloomFilter) EstimateItems() uint32 {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	setBits := bf.countSetBits()

	if setBits == 0 {
		return 0
	}

	m := float64(bf.size)
	k := float64(bf.numHashes)
	x := float64(setBits)

	estimated := -(m / k) * math.Log(1-x/m)

	return uint32(estimated)
}

func (bf *BloomFilter) GetStats() BloomStats {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	setBits := bf.countSetBits()
	loadFactor := float64(setBits) / float64(bf.size)

	return BloomStats{
		Size:              bf.size,
		NumHashes:         bf.numHashes,
		NumItems:          bf.numItems,
		SetBits:           setBits,
		LoadFactor:        loadFactor,
		FalsePositiveRate: bf.FalsePositiveRate(),
		EstimatedItems:    bf.EstimateItems(),
	}
}

func (bf *BloomFilter) getHashes(data []byte) []uint32 {
	hashes := make([]uint32, bf.numHashes)

	hash1 := bf.hash1(data)
	hash2 := bf.hash2(data)

	for i := uint32(0); i < bf.numHashes; i++ {
		hashes[i] = hash1 + i*hash2
	}

	return hashes
}

func (bf *BloomFilter) hash1(data []byte) uint32 {
	hasher := fnv.New32a()
	hasher.Write(data)
	return hasher.Sum32()
}

func (bf *BloomFilter) hash2(data []byte) uint32 {
	hasher := fnv.New32()
	hasher.Write(data)
	hash := hasher.Sum32()
	if hash%2 == 0 {
		hash++
	}
	return hash
}

func (bf *BloomFilter) countSetBits() uint32 {
	count := uint32(0)
	for _, bit := range bf.bits {
		if bit {
			count++
		}
	}
	return count
}

type BloomStats struct {
	Size              uint32  `json:"size"`
	NumHashes         uint32  `json:"num_hashes"`
	NumItems          uint32  `json:"num_items"`
	SetBits           uint32  `json:"set_bits"`
	LoadFactor        float64 `json:"load_factor"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
	EstimatedItems    uint32  `json:"estimated_items"`
}

type BloomFilterTimeWindow struct {
	filters []*BloomFilter
	window  int
	current int
	mutex   sync.RWMutex
}

func NewBloomFilterTimeWindow(windows int, size, numHashes uint32) *BloomFilterTimeWindow {
	filters := make([]*BloomFilter, windows)
	for i := 0; i < windows; i++ {
		filters[i] = NewBloomFilter(size, numHashes)
	}

	return &BloomFilterTimeWindow{
		filters: filters,
		window:  windows,
		current: 0,
	}
}

func (bftw *BloomFilterTimeWindow) Add(item []byte) {
	bftw.mutex.Lock()
	defer bftw.mutex.Unlock()

	bftw.filters[bftw.current].Add(item)
}

func (bftw *BloomFilterTimeWindow) Contains(item []byte) bool {
	bftw.mutex.RLock()
	defer bftw.mutex.RUnlock()

	for _, filter := range bftw.filters {
		if filter.Contains(item) {
			return true
		}
	}
	return false
}

func (bftw *BloomFilterTimeWindow) Rotate() {
	bftw.mutex.Lock()
	defer bftw.mutex.Unlock()

	bftw.current = (bftw.current + 1) % bftw.window
	bftw.filters[bftw.current].Clear()
}
