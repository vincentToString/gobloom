package bloom

import (
	"math"
	"sync"

	"github.com/bits-and-blooms/bitset"
)

// This type holds bit array + parameters for a Bloom filter.
type BloomFilter struct {
	numBits   uint
	numHashes uint
	bitset    *bitset.BitSet
	mu        sync.RWMutex
}

// Constructor for BloomFilter
func NewBloomFilter(numBits, numHashes uint) *BloomFilter {
	if numBits < 1 {numBits = 1 }
	if numHashes < 1 {numHashes = 1 }
	return &BloomFilter{ // If Condition: For numBits and numHashes, ensure they are at least 1 to avoid panics
		numBits:   numBits,
		numHashes: numHashes,
		bitset:    bitset.New(numBits),
	}
}

// Merge Functionality
// // From creates a new Bloom filter with len(_data_) * 64 bits and _k_ hashing
// // functions. The data slice is not going to be reset.
// func From(data []uint64, k uint) *BloomFilter {
// 	m := uint(len(data) * 64)
// 	return FromWithM(data, m, k)
// }

// // FromWithM creates a new Bloom filter with _m_ length, _k_ hashing functions.
// // The data slice is not going to be reset.
// func FromWithM(data []uint64, m, k uint) *BloomFilter {
// 	return &BloomFilter{m, k, bitset.From(data)}
// }


// Theordically, we can use 2 hash values to create k hashes, but here we are using 4 hash values to create k hashes for better randomness.
func baseHashes(data []byte) [4]uint64{
	var d digest128 // murmur hashing
	hash1, hash2, hash3, hash4 := d.sum256(data)
	return [4]uint64{
		hash1, hash2, hash3, hash4,
	}
}

// With above 4 hash values, we now need to actual bitset location. We would use something simlar to double hashing to get the location
// Logic coming from: https://github.com/bits-and-blooms/bloom/blob/master/bloom.go#L122
// Explaination: 
/*
	// hashes ==> [h0, h1, h2, h3]
	// ii ==> i as a uint64
	firstPart ==> hashes[ii%2] // This gives us either h0 or h1 based on i being even or odd
	secondPart ==> ii * hashes[2+(((ii+(ii%2))%4)/2)] // This gives us either h2 or h3 based on i being even or odd
	// So, we are using the first two hashes to get the base location and then using the second two hashes to get the offset (Which is double hashing)
	We are just adding two parts here to dynamically select two hashes used in the double hashing.
*/
func getLocation(hashes [4]uint64, i uint) uint64{
	ii := uint64(i)
	return hashes[ii%2] + ii*hashes[2+(((ii+(ii%2))%4)/2)]
}

// struct method to apply above location logic on the actual BloomFilter's numBits (modulus part) 
func (f *BloomFilter) location(hashes [4]uint64, i uint) uint {
	return uint(getLocation(hashes, i) % uint64(f.numBits))
}

// calculated numBits and numHashes based on 1. size of the data and 2. false positive rate
func EstimateParameters(dataSize int, fp float64) (numBits uint, numHashes uint){
	// m = (n ln fp) / (ln 2)^2
	numBits = uint(math.Ceil(-float64(dataSize) * math.Log(fp) / math.Pow(math.Log(2), 2)))
	// k = (m ln 2) / n
	numHashes = uint(math.Ceil(float64(numBits) * math.Log(2) / float64(dataSize)))
	return
}

func NewWithEstimatedParams(dataSize int, fp float64) *BloomFilter {
	numBits, numHashes := EstimateParameters(dataSize, fp)
	return NewBloomFilter(numBits, numHashes)
}

// Getters for outside Bloom Package Use

func (f *BloomFilter) NumBits() uint {
	return f.numBits
}

func (f *BloomFilter) NumHashes() uint {
	return f.numHashes
}

func (f *BloomFilter) BitSet() *bitset.BitSet {
	return f.bitset
}

// Add data to the Bloom Filter. Return the fileter(Allowing chaining)
func (f *BloomFilter) Add(data []byte) *BloomFilter{
	f.mu.Lock()
	defer f.mu.Unlock()
	hashes := baseHashes(data)
	for i:= uint(0); i < f.numHashes; i++{
		f.bitset.Set(f.location(hashes, i)) // As we iterate i, we are using different hashes to do the double hashing
	}
	return f
}

// Unknown Merge functionality for now
// func (f *BloomFilter) Merge(g *BloomFilter) error {
// 	// Make sure the m's and k's are the same, otherwise merging has no real use.
// 	if f.m != g.m {
// 		return fmt.Errorf("m's don't match: %d != %d", f.m, g.m)
// 	}

// 	if f.k != g.k {
// 		return fmt.Errorf("k's don't match: %d != %d", f.m, g.m)
// 	}

// 	f.b.InPlaceUnion(g.b)
// 	return nil
// }

// Other unknown functionality for now
// Copy creates a copy of a Bloom filter.
// func (f *BloomFilter) Copy() *BloomFilter {
// 	fc := New(f.m, f.k)
// 	fc.Merge(f) // #nosec
// 	return fc
// }

// // AddString to the Bloom Filter. Returns the filter (allows chaining)
// func (f *BloomFilter) AddString(data string) *BloomFilter {
// 	return f.Add([]byte(data))
// }

// Verify functionality
// Verify checks if the data is in the Bloom filter. Returns true if it is, false otherwise.
// True ==> Can be false postive, meaning the data might actually not be in the filter while returning true
// False ==> Data is DEFINITELY!! not in the filter, never seen before
func (f *BloomFilter) Verify(data []byte) bool{
	f.mu.RLock()
	defer f.mu.RUnlock()
	h := baseHashes(data)
	for i:= uint(0); i < f.numHashes; i++{
		if !f.bitset.Test(f.location(h, i)){
			return false // If the data is seen before, all the bits should already be set.
			// Now if any of the desired bit is not set, then that data has to be definitely not in the filter ==> false
		}
	}
	return true // If all the bits are set, then the data is possibly in the filter ==> true
	// Note: This can be a false positive as there might be other data that has set the exact same k bits in the set.
}

// Unknown functionality for now
// // TestString returns true if the string is in the BloomFilter, false otherwise.
// // If true, the result might be a false positive. If false, the data
// // is definitely not in the set.
// func (f *BloomFilter) TestString(data string) bool {
// 	return f.Test([]byte(data))
// }

// // TestLocations returns true if all locations are set in the BloomFilter, false
// // otherwise.
// func (f *BloomFilter) TestLocations(locs []uint64) bool {
// 	for i := 0; i < len(locs); i++ {
// 		if !f.b.Test(uint(locs[i] % uint64(f.m))) {
// 			return false
// 		}
// 	}
// 	return true
// }





