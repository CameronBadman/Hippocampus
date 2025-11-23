package storage

import (
	"Hippocampus/src/types"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type FileStorage struct {
	path string
}

func New(path string) *FileStorage {
	return &FileStorage{path: path}
}

func (fs *FileStorage) Save(t *types.Tree) error {
	f, err := os.Create(fs.path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write dimensions first
	if err := binary.Write(f, binary.LittleEndian, int32(t.Dimensions)); err != nil {
		return err
	}

	// Write node count
	if err := binary.Write(f, binary.LittleEndian, int64(len(t.Nodes))); err != nil {
		return err
	}

	for i := range t.Nodes {
		if err := writeNode(f, &t.Nodes[i]); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileStorage) Load() (*types.Tree, error) {
	f, err := os.Open(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return types.NewTree(512), nil // Default dimensions
		}
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size() == 0 {
		return types.NewTree(512), nil // Default dimensions
	}

	// Read dimensions first
	var dimensions int32
	if err := binary.Read(f, binary.LittleEndian, &dimensions); err != nil {
		return nil, err
	}

	// Read node count
	var nodeCount int64
	if err := binary.Read(f, binary.LittleEndian, &nodeCount); err != nil {
		return nil, err
	}

	t := types.NewTree(int(dimensions))
	t.Nodes = make([]types.Node, nodeCount)

	for i := range t.Nodes {
		if err := readNode(f, &t.Nodes[i], int(dimensions)); err != nil {
			return nil, err
		}
	}

	t.RebuildIndex()

	return t, nil
}

func writeNode(w io.Writer, n *types.Node) error {
	// Write dimension count
	if err := binary.Write(w, binary.LittleEndian, int32(len(n.Key))); err != nil {
		return err
	}

	// Write vector
	for _, val := range n.Key {
		if err := binary.Write(w, binary.LittleEndian, val); err != nil {
			return err
		}
	}

	// Write value
	valueBytes := []byte(n.Value)
	if err := binary.Write(w, binary.LittleEndian, int64(len(valueBytes))); err != nil {
		return err
	}

	_, err := w.Write(valueBytes)
	return err
}

func readNode(r io.Reader, n *types.Node, dimensions int) error {
	// Read dimension count (for validation)
	var dimCount int32
	if err := binary.Read(r, binary.LittleEndian, &dimCount); err != nil {
		return err
	}

	if int(dimCount) != dimensions {
		return fmt.Errorf("dimension mismatch in file: expected %d, got %d", dimensions, dimCount)
	}

	// Read vector
	n.Key = make([]float32, dimensions)
	for i := 0; i < dimensions; i++ {
		if err := binary.Read(r, binary.LittleEndian, &n.Key[i]); err != nil {
			return err
		}
	}

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
	return nil
}
