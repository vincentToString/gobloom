# gobloom

A high-performance, thread-safe Bloom filter implementation in Go, inspired by [bits-and-blooms/bloom](https://github.com/bits-and-blooms/bloom).

## Features

- **High throughput**: 20.6M bloom checks/sec (single thread) - easily handles >1M req/sec
- **Thread-safe**: Uses `sync.RWMutex` for concurrent reads and writes
- **Zero-allocation hashing**: Custom MurmurHash3 implementation (0 allocs/op)
- **Optimal parameter calculation**: Automatic sizing based on expected elements and false positive rate
- **Low latency**: ~48ns per Verify, ~58ns per Add (0 heap allocations)

## Installation

```bash
go get github.com/yourusername/gobloom
```

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "github.com/yourusername/gobloom"
)

func main() {
    // Create filter with 1000 bits and 7 hash functions
    bf := bloom.NewBloomFilter(1000, 7)

    // Add elements
    bf.Add([]byte("hello"))
    bf.Add([]byte("world"))

    // Check membership
    fmt.Println(bf.Verify([]byte("hello"))) // true
    fmt.Println(bf.Verify([]byte("nope")))  // false (probably)
}
```

### Automatic Parameter Estimation

```go
// Create filter for 10,000 elements with 1% false positive rate
bf := bloom.NewWithEstimatedParams(10000, 0.01)

// Add your data
for i := 0; i < 10000; i++ {
    bf.Add([]byte(fmt.Sprintf("item_%d", i)))
}
```

### Method Chaining

```go
bf := bloom.NewBloomFilter(1000, 7)
bf.Add([]byte("a")).Add([]byte("b")).Add([]byte("c"))
```

## Memory Usage

### Formulas

Given:
- `n` = number of elements to insert
- `p` = desired false positive rate (e.g., 0.01 for 1%)

The optimal parameters are:

**Number of bits (m):**
```
m = -(n × ln(p)) / (ln(2))²
m ≈ -n × ln(p) / 0.4804
```

**Number of hash functions (k):**
```
k = (m / n) × ln(2)
k ≈ 0.693 × (m / n)
```

### Memory Examples

| Elements (n) | FP Rate (p) | Bits (m) | Hashes (k) | Memory  |
|--------------|-------------|----------|------------|---------|
| 1,000        | 1% (0.01)   | 9,586    | 7          | 1.2 KB  |
| 10,000       | 1% (0.01)   | 95,851   | 7          | 12 KB   |
| 100,000      | 1% (0.01)   | 958,506  | 7          | 117 KB  |
| 1,000,000    | 1% (0.01)   | 9,585,059| 7          | 1.2 MB  |
| 10,000       | 0.1% (0.001)| 143,776  | 10         | 18 KB   |
| 100,000      | 0.01% (0.0001)| 1,917,011| 14        | 234 KB  |

**Formula for bytes:**
```
Bytes = m / 8
```

### Bits per Element

Regardless of element size, the memory usage per element is:

```
bits_per_element = -ln(p) / (ln(2))²
```

Examples:
- `p = 0.01` (1% FP): **~9.6 bits/element**
- `p = 0.001` (0.1% FP): **~14.4 bits/element**
- `p = 0.0001` (0.01% FP): **~19.2 bits/element**

## Concurrency

This implementation is **fully thread-safe**:

- **Multiple readers**: `Verify()` uses read locks (`RLock`), allowing concurrent queries
- **Exclusive writes**: `Add()` uses write locks (`Lock`), ensuring safe concurrent insertions
- **No race conditions**: All bitset operations are protected

### Concurrency Performance

From benchmarks on Intel i7-12700KF:
- **Single-threaded Add**: ~57 ns/op
- **Concurrent Add**: ~180 ns/op (expected overhead from locking)
- **Verify**: ~48 ns/op (benefits from RWMutex read concurrency)

## Implementation Details

### Hashing Strategy

- Uses **MurmurHash3** with 256-bit output (4 × 64-bit hashes)
- Implements **enhanced double hashing** for k hash functions
- Zero heap allocations in hash computation

### Double Hashing Formula

```
location(i) = (h[i%2] + i × h[2 + ((i+(i%2))%4)/2]) mod m
```

This alternates between hash pairs to generate k independent locations.

## Testing

Run tests:
```bash
go test -v
```

Run benchmarks:
```bash
go test -bench=. -benchmem
```

## Performance

### Benchmark Results (Intel i7-12700KF)

```
BenchmarkVerify-20    70396921    48.44 ns/op    0 B/op    0 allocs/op
BenchmarkAdd-20       62697343    57.74 ns/op    0 B/op    0 allocs/op
```

### Throughput

- **Verify**: **20.6 million ops/sec** (single thread)
- **Add**: 17.3 million ops/sec (single thread)
- **Zero allocations**: No GC pressure, predictable latency

**Can this handle 1M bloom checks/sec?** YES, easily - it's **20.6× faster** than that requirement.

For detailed benchmarks and performance analysis, see [BENCHMARKS.md](BENCHMARKS.md).

## API Reference

### Types

```go
type BloomFilter struct {
    // contains filtered or unexported fields
}
```

### Functions

```go
// Create new filter with specific parameters
func NewBloomFilter(numBits, numHashes uint) *BloomFilter

// Create filter optimized for n elements and p false positive rate
func NewWithEstimatedParams(dataSize int, fp float64) *BloomFilter

// Calculate optimal parameters
func EstimateParameters(dataSize int, fp float64) (numBits uint, numHashes uint)
```

### Methods

```go
// Add element to filter (thread-safe)
func (f *BloomFilter) Add(data []byte) *BloomFilter

// Check if element might be in filter (thread-safe)
func (f *BloomFilter) Verify(data []byte) bool

// Getters
func (f *BloomFilter) NumBits() uint
func (f *BloomFilter) NumHashes() uint
func (f *BloomFilter) BitSet() *bitset.BitSet
```

## False Positive Guarantees

From test results with n=10,000 and p=0.01:
- **Observed FP rate**: 0.91%
- **Expected FP rate**: 1.00%
- **Parameters used**: m=95,851 bits, k=7 hashes

The implementation consistently achieves false positive rates **at or below** the theoretical target.

## License

Educational project inspired by [bits-and-blooms/bloom](https://github.com/bits-and-blooms/bloom).

MurmurHash3 implementation adapted from Sébastien Paolacci's work (BSD 3-Clause License).

## References

- [Bloom Filter (Wikipedia)](https://en.wikipedia.org/wiki/Bloom_filter)
- [bits-and-blooms/bloom](https://github.com/bits-and-blooms/bloom)
- [MurmurHash3](https://github.com/aappleby/smhasher)
