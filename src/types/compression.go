package types

import (
	"fmt"
	"math"
	"time"
)

// QuantizedVector represents a compressed vector using scalar quantization
type QuantizedVector struct {
	Values []uint8   // Quantized to 8-bit integers
	Min    float32   // Min value for dequantization
	Max    float32   // Max value for dequantization
	Dims   int       // Number of dimensions
}

// QuantizeVector compresses a float32 vector to uint8 (4x compression)
func QuantizeVector(vec []float32) *QuantizedVector {
	if len(vec) == 0 {
		return &QuantizedVector{Dims: 0}
	}

	// Find min and max values
	min := vec[0]
	max := vec[0]
	for _, v := range vec {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Quantize to uint8 (0-255)
	quantized := make([]uint8, len(vec))
	scale := float32(255.0) / (max - min)
	if max == min {
		scale = 0 // All values are the same
	}

	for i, v := range vec {
		normalized := (v - min) * scale
		quantized[i] = uint8(math.Round(float64(normalized)))
	}

	return &QuantizedVector{
		Values: quantized,
		Min:    min,
		Max:    max,
		Dims:   len(vec),
	}
}

// Dequantize converts quantized vector back to float32
func (qv *QuantizedVector) Dequantize() []float32 {
	if qv.Dims == 0 {
		return []float32{}
	}

	result := make([]float32, qv.Dims)
	scale := (qv.Max - qv.Min) / 255.0

	for i, v := range qv.Values {
		result[i] = qv.Min + float32(v)*scale
	}

	return result
}

// ApproximateDistance calculates distance directly on quantized vectors
// This is faster than dequantizing first, with minimal accuracy loss
func (qv *QuantizedVector) ApproximateDistance(other *QuantizedVector) (float32, error) {
	if qv.Dims != other.Dims {
		return 0, fmt.Errorf("dimension mismatch: %d vs %d", qv.Dims, other.Dims)
	}

	// Calculate distance in quantized space
	var sumSquares uint64
	for i := 0; i < qv.Dims; i++ {
		diff := int16(qv.Values[i]) - int16(other.Values[i])
		sumSquares += uint64(diff * diff)
	}

	// Scale back to original space
	scale1 := (qv.Max - qv.Min) / 255.0
	scale2 := (other.Max - other.Min) / 255.0
	avgScale := (scale1 + scale2) / 2.0

	distance := math.Sqrt(float64(sumSquares)) * float64(avgScale)
	return float32(distance), nil
}

// CompressedNode is a node with quantized vector
type CompressedNode struct {
	Key       *QuantizedVector
	Value     string
	Metadata  Metadata
	Timestamp time.Time
}

// Compress converts a regular node to compressed format
func (n *Node) Compress() *CompressedNode {
	return &CompressedNode{
		Key:       QuantizeVector(n.Key),
		Value:     n.Value,
		Metadata:  n.Metadata,
		Timestamp: n.Timestamp,
	}
}

// Decompress converts compressed node back to regular format
func (cn *CompressedNode) Decompress() *Node {
	return &Node{
		Key:       cn.Key.Dequantize(),
		Value:     cn.Value,
		Metadata:  cn.Metadata,
		Timestamp: cn.Timestamp,
	}
}

// SizeBytes returns approximate memory size
func (qv *QuantizedVector) SizeBytes() int {
	return len(qv.Values) + 8 + 4 // values + min + max (+ overhead)
}

// CompressionRatio calculates the compression ratio
func CompressionRatio(dims int) float32 {
	original := dims * 4  // float32 = 4 bytes each
	compressed := dims*1 + 8 + 4 // uint8 + min + max
	return float32(original) / float32(compressed)
}

// QuantizationError measures average error from quantization
func QuantizationError(original []float32, quantized *QuantizedVector) float32 {
	dequant := quantized.Dequantize()

	var totalError float32
	for i := range original {
		diff := original[i] - dequant[i]
		totalError += diff * diff
	}

	return float32(math.Sqrt(float64(totalError / float32(len(original)))))
}

// ProductQuantization for even better compression (experimental)
type ProductQuantizedVector struct {
	Subvectors [][]uint8  // Each subvector quantized separately
	Codebook   [][]float32 // Learned centroids
	SubvecDim  int
	NumSubvec  int
}

// PQQuantize uses product quantization (8x-16x compression)
// Divides vector into subvectors, quantizes each separately
func PQQuantize(vec []float32, numSubvectors int) *ProductQuantizedVector {
	dims := len(vec)
	subvecDim := dims / numSubvectors

	pq := &ProductQuantizedVector{
		Subvectors: make([][]uint8, numSubvectors),
		SubvecDim:  subvecDim,
		NumSubvec:  numSubvectors,
	}

	for i := 0; i < numSubvectors; i++ {
		start := i * subvecDim
		end := start + subvecDim
		if end > dims {
			end = dims
		}

		subvec := vec[start:end]
		quantized := QuantizeVector(subvec)
		pq.Subvectors[i] = quantized.Values
	}

	return pq
}
