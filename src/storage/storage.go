package storage

import (
	"Hippocampus/types"
	"encoding/binary"
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

	if err := binary.Write(f, binary.LittleEndian, int64(len(t.Nodes))); err != nil {
		return err
	}

	for i := range t.Nodes {
		if err := writeNode(f, &t.Nodes[i]); err != nil {
			return err
		}
	}

	for dim := 0; dim < 512; dim++ {
		if err := binary.Write(f, binary.LittleEndian, int32(len(t.Index[dim]))); err != nil {
			return err
		}
		for _, idx := range t.Index[dim] {
			if err := binary.Write(f, binary.LittleEndian, idx); err != nil {
				return err
			}
		}
	}

	return nil
}

func (fs *FileStorage) Load() (*types.Tree, error) {
	f, err := os.Open(fs.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var nodeCount int64
	if err := binary.Read(f, binary.LittleEndian, &nodeCount); err != nil {
		return nil, err
	}

	t := &types.Tree{
		Nodes: make([]types.Node, nodeCount),
		Index: [512][]int32{},
	}

	for i := range t.Nodes {
		if err := readNode(f, &t.Nodes[i]); err != nil {
			return nil, err
		}
	}

	for dim := 0; dim < 512; dim++ {
		var indexLen int32
		if err := binary.Read(f, binary.LittleEndian, &indexLen); err != nil {
			return nil, err
		}
		t.Index[dim] = make([]int32, indexLen)
		for i := range t.Index[dim] {
			if err := binary.Read(f, binary.LittleEndian, &t.Index[dim][i]); err != nil {
				return nil, err
			}
		}
	}

	return t, nil
}

func writeNode(w io.Writer, n *types.Node) error {
	if err := binary.Write(w, binary.LittleEndian, n.Key); err != nil {
		return err
	}

	valueBytes := []byte(n.Value)
	if err := binary.Write(w, binary.LittleEndian, int64(len(valueBytes))); err != nil {
		return err
	}
	_, err := w.Write(valueBytes)
	return err
}

func readNode(r io.Reader, n *types.Node) error {
	if err := binary.Read(r, binary.LittleEndian, &n.Key); err != nil {
		return err
	}

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
