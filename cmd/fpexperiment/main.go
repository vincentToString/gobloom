package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"bloomGo"
)

func main() {
	// Command-line flags
	n := flag.Int("n", 10000, "Number of elements to insert")
	p := flag.Float64("p", 0.01, "Target false positive rate")
	q := flag.Int("q", 100000, "Number of test queries (different keys)")
	seed := flag.Int64("seed", time.Now().UnixNano(), "Random seed for reproducibility")
	flag.Parse()

	fmt.Println("=== Bloom Filter False Positive Rate Experiment ===")
	fmt.Println()

	// Set random seed
	rand.Seed(*seed)
	fmt.Printf("Random seed: %d\n", *seed)
	fmt.Println()

	// Calculate parameters
	fmt.Println("--- Configuration ---")
	fmt.Printf("Expected elements (n): %d\n", *n)
	fmt.Printf("Target FP rate (p): %.6f (%.4f%%)\n", *p, *p*100)
	fmt.Printf("Test queries (q): %d\n", *q)
	fmt.Println()

	// Create bloom filter with estimated parameters
	m, k := bloom.EstimateParameters(*n, *p)
	bf := bloom.NewBloomFilter(m, k)

	fmt.Println("--- Filter Parameters ---")
	fmt.Printf("Number of bits (m): %d\n", m)
	fmt.Printf("Number of hash functions (k): %d\n", k)
	fmt.Printf("Memory usage: %d bytes (%.2f KB)\n", m/8, float64(m)/8/1024)
	fmt.Printf("Bits per element: %.2f\n", float64(m)/float64(*n))
	fmt.Println()

	// Insert n random keys
	fmt.Println("--- Insertion Phase ---")
	inserted := make(map[string]bool, *n)
	startInsert := time.Now()

	for i := 0; i < *n; i++ {
		key := generateRandomKey(16) // 16-byte random keys
		bf.Add(key)
		inserted[string(key)] = true
	}

	insertDuration := time.Since(startInsert)
	fmt.Printf("Inserted %d elements in %v\n", *n, insertDuration)
	fmt.Printf("Average insertion time: %.2f ns/op\n", float64(insertDuration.Nanoseconds())/float64(*n))
	fmt.Println()

	// Test with q different random keys
	fmt.Println("--- Testing Phase ---")
	falsePositives := 0
	trueNegatives := 0
	startTest := time.Now()

	for i := 0; i < *q; i++ {
		key := generateRandomKey(16)
		result := bf.Verify(key)

		// Check if this is a false positive
		_, wasInserted := inserted[string(key)]

		if result && !wasInserted {
			falsePositives++
		} else if !result && !wasInserted {
			trueNegatives++
		}
	}

	testDuration := time.Since(startTest)
	fmt.Printf("Tested %d queries in %v\n", *q, testDuration)
	fmt.Printf("Average query time: %.2f ns/op\n", float64(testDuration.Nanoseconds())/float64(*q))
	fmt.Println()

	// Calculate results
	fmt.Println("--- Results ---")
	fmt.Printf("False positives: %d / %d\n", falsePositives, *q)
	fmt.Printf("True negatives: %d / %d\n", trueNegatives, *q)

	observedFP := float64(falsePositives) / float64(*q)
	theoreticalFP := *p

	fmt.Println()
	fmt.Println("--- False Positive Rate Analysis ---")
	fmt.Printf("Observed FP rate:     %.6f (%.4f%%)\n", observedFP, observedFP*100)
	fmt.Printf("Theoretical FP rate:  %.6f (%.4f%%)\n", theoreticalFP, theoreticalFP*100)

	ratio := observedFP / theoreticalFP
	fmt.Printf("Ratio (observed/theoretical): %.2fx\n", ratio)

	// Determine if result is acceptable
	fmt.Println()
	if observedFP <= theoreticalFP {
		fmt.Printf("✓ PASS: Observed FP rate is at or below target\n")
	} else if observedFP <= theoreticalFP*1.5 {
		fmt.Printf("~ ACCEPTABLE: Observed FP rate is within 1.5x of target (statistical variation)\n")
	} else {
		fmt.Printf("✗ FAIL: Observed FP rate significantly exceeds target\n")
	}

	// Additional statistics
	fmt.Println()
	fmt.Println("--- Summary ---")
	fmt.Printf("Filter size: %d bits (%d bytes, %.2f KB, %.2f MB)\n",
		m, m/8, float64(m)/8/1024, float64(m)/8/1024/1024)
	fmt.Printf("Space efficiency: %.2f bits/element\n", float64(m)/float64(*n))
	fmt.Printf("Hash functions: %d\n", k)
	fmt.Printf("Load factor: %.4f\n", float64(*n)*float64(k)/float64(m))

	// Theoretical FP rate formula
	actualFP := calculateTheoreticalFP(*n, int(m), int(k))
	fmt.Println()
	fmt.Printf("Theoretical FP (from actual m, k): %.6f (%.4f%%)\n", actualFP, actualFP*100)

	fmt.Println()
	fmt.Println("=== Experiment Complete ===")
}

// generateRandomKey creates a random byte slice of given length
func generateRandomKey(length int) []byte {
	key := make([]byte, length)
	for i := range key {
		key[i] = byte(rand.Intn(256))
	}
	return key
}

// calculateTheoreticalFP computes the theoretical false positive rate
// Formula: (1 - e^(-kn/m))^k
func calculateTheoreticalFP(n, m, k int) float64 {
	// Avoid division by zero
	if m == 0 {
		return 1.0
	}

	exponent := -float64(k) * float64(n) / float64(m)
	base := 1.0 - exp(exponent)

	// Calculate base^k
	result := 1.0
	for i := 0; i < k; i++ {
		result *= base
	}

	return result
}

// exp approximates e^x using Taylor series
func exp(x float64) float64 {
	sum := 1.0
	term := 1.0
	for i := 1; i < 100; i++ {
		term *= x / float64(i)
		sum += term
		if term < 1e-10 && term > -1e-10 {
			break
		}
	}
	return sum
}
