package bloom

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

// TestNewBloomFilter tests the constructor
func TestNewBloomFilter(t *testing.T) {
	tests := []struct {
		name      string
		numBits   uint
		numHashes uint
		wantBits  uint
		wantHash  uint
	}{
		{"Normal values", 1000, 7, 1000, 7},
		{"Zero bits", 0, 5, 1, 5},
		{"Zero hashes", 100, 0, 100, 1},
		{"Both zero", 0, 0, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf := NewBloomFilter(tt.numBits, tt.numHashes)
			if bf.NumBits() != tt.wantBits {
				t.Errorf("NumBits() = %d, want %d", bf.NumBits(), tt.wantBits)
			}
			if bf.NumHashes() != tt.wantHash {
				t.Errorf("NumHashes() = %d, want %d", bf.NumHashes(), tt.wantHash)
			}
		})
	}
}

// TestEstimateParameters tests parameter calculation
func TestEstimateParameters(t *testing.T) {
	tests := []struct {
		dataSize int
		fp       float64
	}{
		{1000, 0.01},   // 1% FP
		{10000, 0.001}, // 0.1% FP
		{100, 0.0001},  // 0.01% FP
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("n=%d_p=%f", tt.dataSize, tt.fp), func(t *testing.T) {
			m, k := EstimateParameters(tt.dataSize, tt.fp)

			if m == 0 {
				t.Error("numBits should not be 0")
			}
			if k == 0 {
				t.Error("numHashes should not be 0")
			}

			// Verify approximate relationship: k ≈ 0.7 * (m/n)
			expectedK := uint(0.7 * float64(m) / float64(tt.dataSize))
			if k < expectedK-2 || k > expectedK+2 {
				t.Logf("k=%d might be off, expected around %d", k, expectedK)
			}
		})
	}
}

// TestNewWithEstimatedParams tests constructor with estimation
func TestNewWithEstimatedParams(t *testing.T) {
	bf := NewWithEstimatedParams(1000, 0.01)
	if bf.NumBits() == 0 {
		t.Error("NumBits should not be 0")
	}
	if bf.NumHashes() == 0 {
		t.Error("NumHashes should not be 0")
	}
}

// TestAddAndVerify tests basic add and verify operations
func TestAddAndVerify(t *testing.T) {
	bf := NewBloomFilter(1000, 7)

	testData := [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("bloom"),
		[]byte("filter"),
	}

	// Add elements
	for _, data := range testData {
		bf.Add(data)
	}

	// Verify all added elements return true
	for _, data := range testData {
		if !bf.Verify(data) {
			t.Errorf("Verify(%s) = false, want true (element was added)", data)
		}
	}

	// Elements NOT added should likely return false
	// (but may have false positives)
	notAdded := [][]byte{
		[]byte("not_added_1"),
		[]byte("not_added_2"),
		[]byte("xyz"),
	}

	for _, data := range notAdded {
		if bf.Verify(data) {
			t.Logf("Verify(%s) = true (false positive, expected but rare)", data)
		}
	}
}

// TestEmptyFilter tests that empty filter returns false for all queries
func TestEmptyFilter(t *testing.T) {
	bf := NewBloomFilter(1000, 7)

	testData := [][]byte{
		[]byte("test1"),
		[]byte("test2"),
		[]byte("anything"),
	}

	for _, data := range testData {
		if bf.Verify(data) {
			t.Errorf("Verify(%s) = true on empty filter, want false", data)
		}
	}
}

// TestChaining tests that Add returns the filter for chaining
func TestChaining(t *testing.T) {
	bf := NewBloomFilter(1000, 7)

	result := bf.Add([]byte("a")).Add([]byte("b")).Add([]byte("c"))

	if result != bf {
		t.Error("Add() should return the same filter for chaining")
	}

	// Verify all were added
	if !bf.Verify([]byte("a")) || !bf.Verify([]byte("b")) || !bf.Verify([]byte("c")) {
		t.Error("Chained Add() calls did not work correctly")
	}
}

// TestConcurrentAdd tests concurrent writes
func TestConcurrentAdd(t *testing.T) {
	bf := NewBloomFilter(10000, 7)
	var wg sync.WaitGroup
	numGoroutines := 100
	itemsPerGoroutine := 100

	// Concurrent adds
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				data := []byte(fmt.Sprintf("item_%d_%d", id, j))
				bf.Add(data)
			}
		}(i)
	}

	wg.Wait()

	// Verify all items were added
	failures := 0
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < itemsPerGoroutine; j++ {
			data := []byte(fmt.Sprintf("item_%d_%d", i, j))
			if !bf.Verify(data) {
				failures++
			}
		}
	}

	if failures > 0 {
		t.Errorf("Failed to verify %d items after concurrent adds", failures)
	}
}

// TestConcurrentReadWrite tests concurrent reads and writes
func TestConcurrentReadWrite(t *testing.T) {
	bf := NewBloomFilter(10000, 7)
	var wg sync.WaitGroup
	numReaders := 50
	numWriters := 50
	duration := 100 // iterations

	// Writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < duration; j++ {
				data := []byte(fmt.Sprintf("writer_%d_%d", id, j))
				bf.Add(data)
			}
		}(i)
	}

	// Readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < duration; j++ {
				data := []byte(fmt.Sprintf("reader_%d_%d", id, j))
				bf.Verify(data) // Don't care about result, just testing for races
			}
		}(i)
	}

	wg.Wait()
	// If we get here without panic, concurrency is working
}

// TestFalsePositiveRate tests approximate FP rate
func TestFalsePositiveRate(t *testing.T) {
	n := 10000          // number of items
	p := 0.01           // target FP rate (1%)
	bf := NewWithEstimatedParams(n, p)

	// Add n items
	added := make(map[string]bool)
	for i := 0; i < n; i++ {
		data := []byte(fmt.Sprintf("item_%d", i))
		bf.Add(data)
		added[string(data)] = true
	}

	// Test with different items
	testSize := 10000
	falsePositives := 0

	for i := 0; i < testSize; i++ {
		data := []byte(fmt.Sprintf("test_%d", i))
		if !added[string(data)] && bf.Verify(data) {
			falsePositives++
		}
	}

	observedFP := float64(falsePositives) / float64(testSize)

	t.Logf("Filter params: m=%d, k=%d", bf.NumBits(), bf.NumHashes())
	t.Logf("Added %d items, tested %d new items", n, testSize)
	t.Logf("False positives: %d / %d = %.4f%%", falsePositives, testSize, observedFP*100)
	t.Logf("Expected FP rate: %.4f%%", p*100)

	// Allow 3x tolerance (FP rate is statistical)
	if observedFP > p*3 {
		t.Errorf("FP rate too high: %.4f%% (expected ~%.4f%%)", observedFP*100, p*100)
	}
}

// TestBaseHashes tests that hash function produces different values
func TestBaseHashes(t *testing.T) {
	data := []byte("test")
	hashes := baseHashes(data)

	// Check all 4 hashes are different
	seen := make(map[uint64]bool)
	for i, h := range hashes {
		if seen[h] {
			t.Errorf("Hash %d is duplicate: %d", i, h)
		}
		seen[h] = true
		if h == 0 {
			t.Logf("Warning: hash %d is zero", i)
		}
	}

	// Different input should produce different hashes
	hashes2 := baseHashes([]byte("different"))
	if hashes == hashes2 {
		t.Error("Different inputs produced identical hashes")
	}
}

// TestGetLocation tests location generation
func TestGetLocation(t *testing.T) {
	hashes := [4]uint64{100, 200, 300, 400}

	// Generate multiple locations
	locations := make(map[uint64]bool)
	for i := uint(0); i < 10; i++ {
		loc := getLocation(hashes, i)
		locations[loc] = true
	}

	// Should generate different locations
	if len(locations) < 8 {
		t.Errorf("getLocation generated only %d unique locations out of 10", len(locations))
	}
}

// BenchmarkAdd benchmarks the Add operation
func BenchmarkAdd(b *testing.B) {
	bf := NewBloomFilter(100000, 7)
	data := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		data[i] = []byte(fmt.Sprintf("item_%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf.Add(data[i])
	}
}

// BenchmarkVerify benchmarks the Verify operation
func BenchmarkVerify(b *testing.B) {
	bf := NewBloomFilter(100000, 7)
	// Pre-populate
	for i := 0; i < 10000; i++ {
		bf.Add([]byte(fmt.Sprintf("item_%d", i)))
	}

	data := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		data[i] = []byte(fmt.Sprintf("test_%d", rand.Intn(20000)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf.Verify(data[i])
	}
}

// BenchmarkConcurrentAdd benchmarks concurrent Add operations
func BenchmarkConcurrentAdd(b *testing.B) {
	bf := NewBloomFilter(1000000, 7)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			data := []byte(fmt.Sprintf("item_%d", i))
			bf.Add(data)
			i++
		}
	})
}
