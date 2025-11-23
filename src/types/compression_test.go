package types

import (
	"math"
	"testing"
)

func TestQuantizeVector(t *testing.T) {
	original := []float32{0.1, 0.5, 0.9, -0.3, 0.0}

	qv := QuantizeVector(original)

	if qv.Dims != len(original) {
		t.Errorf("Expected %d dimensions, got %d", len(original), qv.Dims)
	}

	// Dequantize and check error
	dequantized := qv.Dequantize()

	if len(dequantized) != len(original) {
		t.Errorf("Dequantized vector has wrong length")
	}

	// Check quantization error is small
	totalError := float32(0)
	for i := range original {
		diff := original[i] - dequantized[i]
		totalError += diff * diff
	}
	rmse := float32(math.Sqrt(float64(totalError / float32(len(original)))))

	if rmse > 0.01 {
		t.Errorf("Quantization error too high: %f", rmse)
	}
}

func TestQuantizationErrorMeasurement(t *testing.T) {
	vec := make([]float32, 512)
	for i := range vec {
		vec[i] = float32(i) / 512.0
	}

	qv := QuantizeVector(vec)
	error := QuantizationError(vec, qv)

	// Error should be very small for well-distributed values
	if error > 0.01 {
		t.Errorf("Quantization error too high: %f", error)
	}

	t.Logf("Quantization error for 512-dim vector: %f", error)
}

func TestCompressionRatio(t *testing.T) {
	dims := []int{128, 512, 1536, 3072}

	for _, dim := range dims {
		ratio := CompressionRatio(dim)
		expected := float32(4.0) // Should be approximately 4x

		if math.Abs(float64(ratio-expected)) > 0.5 {
			t.Errorf("Unexpected compression ratio for %d dims: %.2f", dim, ratio)
		}

		t.Logf("%d dimensions: %.2fx compression", dim, ratio)
	}
}

func TestNodeCompressionDecompression(t *testing.T) {
	// Create a node
	vec := make([]float32, 512)
	for i := range vec {
		vec[i] = float32(i) / 512.0
	}

	original := &Node{
		Key:      vec,
		Value:    "test content",
		Metadata: Metadata{"key": "value"},
	}

	// Compress
	compressed := original.Compress()

	// Verify compression
	if compressed.Value != original.Value {
		t.Error("Value corrupted during compression")
	}

	// Decompress
	decompressed := compressed.Decompress()

	// Check vector accuracy
	totalError := float32(0)
	for i := range original.Key {
		diff := original.Key[i] - decompressed.Key[i]
		totalError += diff * diff
	}
	rmse := float32(math.Sqrt(float64(totalError / float32(len(original.Key)))))

	if rmse > 0.01 {
		t.Errorf("Decompression error too high: %f", rmse)
	}

	// Check other fields
	if decompressed.Value != original.Value {
		t.Error("Value corrupted during decompression")
	}
}

func TestApproximateDistance(t *testing.T) {
	vec1 := make([]float32, 512)
	vec2 := make([]float32, 512)

	for i := range vec1 {
		vec1[i] = float32(i) / 512.0
		vec2[i] = float32(i+1) / 512.0
	}

	// Calculate exact distance
	var exactDist float32
	for i := range vec1 {
		diff := vec1[i] - vec2[i]
		exactDist += diff * diff
	}
	exactDist = float32(math.Sqrt(float64(exactDist)))

	// Calculate approximate distance on quantized vectors
	qv1 := QuantizeVector(vec1)
	qv2 := QuantizeVector(vec2)

	approxDist, err := qv1.ApproximateDistance(qv2)
	if err != nil {
		t.Fatalf("Failed to calculate approximate distance: %v", err)
	}

	// Error should be small
	distError := math.Abs(float64(exactDist - approxDist))
	if distError > 0.1 {
		t.Errorf("Distance error too high: exact=%f, approx=%f, error=%f",
			exactDist, approxDist, distError)
	}

	t.Logf("Exact distance: %f, Approximate: %f, Error: %f", exactDist, approxDist, distError)
}

func BenchmarkQuantizeVector(b *testing.B) {
	vec := make([]float32, 512)
	for i := range vec {
		vec[i] = float32(i) / 512.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = QuantizeVector(vec)
	}
}

func BenchmarkDequantizeVector(b *testing.B) {
	vec := make([]float32, 512)
	for i := range vec {
		vec[i] = float32(i) / 512.0
	}
	qv := QuantizeVector(vec)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = qv.Dequantize()
	}
}

func BenchmarkApproximateDistance(b *testing.B) {
	vec1 := make([]float32, 512)
	vec2 := make([]float32, 512)

	for i := range vec1 {
		vec1[i] = float32(i) / 512.0
		vec2[i] = float32(i+1) / 512.0
	}

	qv1 := QuantizeVector(vec1)
	qv2 := QuantizeVector(vec2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qv1.ApproximateDistance(qv2)
	}
}

func BenchmarkCompressionOverhead(b *testing.B) {
	vec := make([]float32, 512)
	for i := range vec {
		vec[i] = float32(i) / 512.0
	}

	node := &Node{Key: vec, Value: "test"}

	b.Run("Compress", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = node.Compress()
		}
	})

	compressed := node.Compress()

	b.Run("Decompress", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = compressed.Decompress()
		}
	})
}
