package storage

import (
	"Hippocampus/src/types"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// CompressedStorage stores vectors in compressed format (4x smaller)
type CompressedStorage struct {
	path string
}

func NewCompressed(path string) *CompressedStorage {
	return &CompressedStorage{path: path}
}

func (cs *CompressedStorage) Save(t *types.Tree) error {
	f, err := os.Create(cs.path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write header: dimensions + node count + compression flag
	if err := binary.Write(f, binary.LittleEndian, int32(t.Dimensions)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, int64(len(t.Nodes))); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint8(1)); err != nil { // compressed = 1
		return err
	}

	// Write each node in compressed format
	for i := range t.Nodes {
		if err := writeCompressedNode(f, &t.Nodes[i]); err != nil {
			return err
		}
	}

	return nil
}

func (cs *CompressedStorage) Load() (*types.Tree, error) {
	f, err := os.Open(cs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return types.NewTree(512), nil
		}
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size() == 0 {
		return types.NewTree(512), nil
	}

	// Read header
	var dimensions int32
	if err := binary.Read(f, binary.LittleEndian, &dimensions); err != nil {
		return nil, err
	}

	var nodeCount int64
	if err := binary.Read(f, binary.LittleEndian, &nodeCount); err != nil {
		return nil, err
	}

	var compressed uint8
	if err := binary.Read(f, binary.LittleEndian, &compressed); err != nil {
		// Backwards compatibility: old files don't have compression flag
		compressed = 0
		f.Seek(-1, io.SeekCurrent) // Rewind
	}

	t := types.NewTree(int(dimensions))
	t.Nodes = make([]types.Node, nodeCount)

	for i := range t.Nodes {
		if compressed == 1 {
			if err := readCompressedNode(f, &t.Nodes[i], int(dimensions)); err != nil {
				return nil, err
			}
		} else {
			if err := readNode(f, &t.Nodes[i], int(dimensions)); err != nil {
				return nil, err
			}
		}
	}

	t.RebuildIndex()
	return t, nil
}

func writeCompressedNode(w io.Writer, n *types.Node) error {
	// Quantize vector
	qv := types.QuantizeVector(n.Key)

	// Write compressed vector
	if err := binary.Write(w, binary.LittleEndian, int32(qv.Dims)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, qv.Min); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, qv.Max); err != nil {
		return err
	}
	if _, err := w.Write(qv.Values); err != nil {
		return err
	}

	// Write value (same as before)
	valueBytes := []byte(n.Value)
	if err := binary.Write(w, binary.LittleEndian, int64(len(valueBytes))); err != nil {
		return err
	}
	if _, err := w.Write(valueBytes); err != nil {
		return err
	}

	// Write timestamp
	timestampBytes, err := n.Timestamp.MarshalBinary()
	if err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, int32(len(timestampBytes))); err != nil {
		return err
	}
	if _, err := w.Write(timestampBytes); err != nil {
		return err
	}

	// Write metadata
	var metadataBytes []byte
	if n.Metadata != nil {
		metadataBytes, err = json.Marshal(n.Metadata)
		if err != nil {
			return err
		}
	}
	if err := binary.Write(w, binary.LittleEndian, int32(len(metadataBytes))); err != nil {
		return err
	}
	if len(metadataBytes) > 0 {
		if _, err := w.Write(metadataBytes); err != nil {
			return err
		}
	}

	return nil
}

func readCompressedNode(r io.Reader, n *types.Node, expectedDims int) error {
	// Read compressed vector
	var dims int32
	if err := binary.Read(r, binary.LittleEndian, &dims); err != nil {
		return err
	}

	if int(dims) != expectedDims {
		return fmt.Errorf("dimension mismatch: expected %d, got %d", expectedDims, dims)
	}

	var min, max float32
	if err := binary.Read(r, binary.LittleEndian, &min); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &max); err != nil {
		return err
	}

	values := make([]uint8, dims)
	if _, err := io.ReadFull(r, values); err != nil {
		return err
	}

	// Dequantize
	qv := &types.QuantizedVector{
		Values: values,
		Min:    min,
		Max:    max,
		Dims:   int(dims),
	}
	n.Key = qv.Dequantize()

	// Read value
	var valueLen int64
	if err := binary.Read(r, binary.LittleEndian, &valueLen); err != nil {
		return err
	}
	valueBytes := make([]byte, valueLen)
	if _, err := io.ReadFull(r, valueBytes); err != nil {
		return err
	}
	n.Value = string(valueBytes)

	// Read timestamp
	var timestampLen int32
	if err := binary.Read(r, binary.LittleEndian, &timestampLen); err != nil {
		return err
	}
	if timestampLen > 0 {
		timestampBytes := make([]byte, timestampLen)
		if _, err := io.ReadFull(r, timestampBytes); err != nil {
			return err
		}
		if err := n.Timestamp.UnmarshalBinary(timestampBytes); err != nil {
			n.Timestamp = time.Time{}
		}
	}

	// Read metadata
	var metadataLen int32
	if err := binary.Read(r, binary.LittleEndian, &metadataLen); err != nil {
		return nil // Backwards compatibility
	}
	if metadataLen > 0 {
		metadataBytes := make([]byte, metadataLen)
		if _, err := io.ReadFull(r, metadataBytes); err != nil {
			return err
		}
		n.Metadata = make(types.Metadata)
		if err := json.Unmarshal(metadataBytes, &n.Metadata); err != nil {
			return err
		}
	}

	return nil
}

// CompressionStats analyzes compression effectiveness
func CompressionStats(treePath string) error {
	// Load tree
	fs := New(treePath)
	tree, err := fs.Load()
	if err != nil {
		return err
	}

	if len(tree.Nodes) == 0 {
		fmt.Println("No nodes in tree")
		return nil
	}

	// Calculate original size
	originalSize := int64(0)
	for _, node := range tree.Nodes {
		originalSize += int64(len(node.Key) * 4) // 4 bytes per float32
		originalSize += int64(len(node.Value))
	}

	// Calculate compressed size
	compressedSize := int64(0)
	totalError := float32(0)
	for _, node := range tree.Nodes {
		qv := types.QuantizeVector(node.Key)
		compressedSize += int64(qv.SizeBytes())
		compressedSize += int64(len(node.Value))

		// Measure quantization error
		totalError += types.QuantizationError(node.Key, qv)
	}

	ratio := float64(originalSize) / float64(compressedSize)
	avgError := totalError / float32(len(tree.Nodes))

	fmt.Printf("Compression Analysis:\n")
	fmt.Printf("  Nodes: %d\n", len(tree.Nodes))
	fmt.Printf("  Dimensions: %d\n", tree.Dimensions)
	fmt.Printf("  Original size: %.2f MB\n", float64(originalSize)/(1024*1024))
	fmt.Printf("  Compressed size: %.2f MB\n", float64(compressedSize)/(1024*1024))
	fmt.Printf("  Compression ratio: %.2fx\n", ratio)
	fmt.Printf("  Avg quantization error: %.6f\n", avgError)

	return nil
}
