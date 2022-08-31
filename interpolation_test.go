package edau

import "math"
import "math/rand"
import "time"
import "testing"

var testPoints []float64
var testLocations []float64
func init() {
	rand.Seed(time.Now().UnixNano())

	testPoints = make([]float64, 16)
	for i := 0; i < 16; i++ {
		testPoints[i] = math.Sin(math.Pi*float64(i)/2)
	}

	testLocations = make([]float64, 16)
	for i := 0; i < 16; i++ {
		testLocations[i] = 7 + rand.Float64()
	}
}

func TestLagrangeN4(t *testing.T) {
	for _, loc := range testLocations {
		samples, target := alignSamplesAndTarget4(testPoints, loc)
		result := InterpLagrangeN(samples, target)
		expect := math.Sin(math.Pi*loc/2)
		diff := math.Abs(result - expect)
		if diff > 0.1 {
			t.Fatalf("TestLagrangeN (4) for %f expected %f but got %f (diff = %f)", loc, expect, result, diff)
		}
	}
}

func TestLagrangeN6(t *testing.T) {
	for _, loc := range testLocations {
		samples, target := alignSamplesAndTarget6(testPoints, loc)
		result := InterpLagrangeN(samples, target)
		expect := math.Sin(math.Pi*loc/2)
		diff := math.Abs(result - expect)
		if diff > 0.05 {
			t.Fatalf("TestLagrangeN (6) for %f expected %f but got %f (diff = %f)", loc, expect, result, diff)
		}
	}
}

func TestInterpLagrange4Pt3Ord(t *testing.T) {
	for _, loc := range testLocations {
		samples, target := alignSamplesAndTarget4(testPoints, loc)
		result := InterpLagrange4Pt3Ord(samples, target)
		expect := math.Sin(math.Pi*loc/2)
		diff := math.Abs(result - expect)
		if diff > 0.1 {
			t.Fatalf("TestInterpLagrange4Pt3Ord for %f expected %f but got %f (diff = %f)", loc, expect, result, diff)
		}
	}
}

func TestInterpLagrange6Pt5Ord(t *testing.T) {
	for _, loc := range testLocations {
		samples, target := alignSamplesAndTarget6(testPoints, loc)
		result := InterpLagrange6Pt5Ord(samples, target)
		expect := math.Sin(math.Pi*loc/2)
		diff := math.Abs(result - expect)
		if diff > 0.05 {
			t.Fatalf("TestInterpLagrange6Pt5Ord for %f expected %f but got %f (diff = %f)", loc, expect, result, diff)
		}
	}
}

func TestInterpHermite4pt3Ord(t *testing.T) {
	for _, loc := range testLocations {
		samples, target := alignSamplesAndTarget4(testPoints, loc)
		result := InterpHermite4Pt3Ord(samples, target)
		expect := math.Sin(math.Pi*loc/2)
		diff := math.Abs(result - expect)
		if diff > 0.1 {
			t.Fatalf("TestInterpHermite4pt3Ord for %f expected %f but got %f (diff = %f)", loc, expect, result, diff)
		}
	}
}

func TestInterpFastHermite6Pt3Ord(t *testing.T) {
	for _, loc := range testLocations {
		samples, target := alignSamplesAndTarget6(testPoints, loc)
		result := InterpHermite6Pt3Ord(samples, target)
		expect := math.Sin(math.Pi*loc/2)
		diff := math.Abs(result - expect)
		if diff > 0.05 {
			t.Fatalf("TestInterpFastHermite6Pt5Ord for %f expected %f but got %f (diff = %f)", loc, expect, result, diff)
		}
	}
}

// benchmarks

func BenchmarkLagrangeN4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, loc := range testLocations {
			samples, target := alignSamplesAndTarget4(testPoints, loc)
			result := InterpLagrangeN(samples, target)
			expect := math.Sin(math.Pi*loc/2)
			diff := math.Abs(result - expect)
			if diff > 0.1 { panic("precision failure") }
		}
   }
}

func BenchmarkLagrangeN6(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, loc := range testLocations {
			samples, target := alignSamplesAndTarget6(testPoints, loc)
			result := InterpLagrangeN(samples, target)
			expect := math.Sin(math.Pi*loc/2)
			diff := math.Abs(result - expect)
			if diff > 0.05 { panic("precision failure") }
		}
   }
}

func BenchmarkLagrange4Pt3Ord(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, loc := range testLocations {
			samples, target := alignSamplesAndTarget4(testPoints, loc)
			result := InterpLagrange4Pt3Ord(samples, target)
			expect := math.Sin(math.Pi*loc/2)
			diff := math.Abs(result - expect)
			if diff > 0.1 { panic("precision failure") }
		}
   }
}

func BenchmarkLagrange6Pt5Ord(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, loc := range testLocations {
			samples, target := alignSamplesAndTarget6(testPoints, loc)
			result := InterpLagrange6Pt5Ord(samples, target)
			expect := math.Sin(math.Pi*loc/2)
			diff := math.Abs(result - expect)
			if diff > 0.05 { panic("precision failure") }
		}
   }
}

func BenchmarkHermite4Pt3Ord(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, loc := range testLocations {
			samples, target := alignSamplesAndTarget4(testPoints, loc)
			result := InterpHermite4Pt3Ord(samples, target)
			expect := math.Sin(math.Pi*loc/2)
			diff := math.Abs(result - expect)
			if diff > 0.1 { panic("precision failure") }
		}
   }
}

func BenchmarkHermite6Pt3Ord(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, loc := range testLocations {
			samples, target := alignSamplesAndTarget6(testPoints, loc)
			result := InterpHermite6Pt3Ord(samples, target)
			expect := math.Sin(math.Pi*loc/2)
			diff := math.Abs(result - expect)
			if diff > 0.05 { panic("precision failure") }
		}
   }
}

// --- helper functions ---

func alignSamplesAndTarget4(samples []float64, targetPosition float64) ([]float64, float64) {
	targetFloorPosition := int(targetPosition)
	targetPosition -= float64(targetFloorPosition - 1) // shift target to align to samples zero-indexing
	targetSamples  := samples[targetFloorPosition - 1 : targetFloorPosition + 3]
	return targetSamples, targetPosition
}

func alignSamplesAndTarget6(samples []float64, targetPosition float64) ([]float64, float64) {
	targetFloorPosition := int(targetPosition)
	targetPosition -= float64(targetFloorPosition - 2) // shift target to align to samples zero-indexing
	targetSamples  := samples[targetFloorPosition - 2 : targetFloorPosition + 4]
	return targetSamples, targetPosition
}