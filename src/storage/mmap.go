package storage

import (
	"Hippocampus/src/types"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// MmapStorage provides memory-mapped file access with lazy index loading
type MmapStorage struct {
	path       string
	file       *os.File
	mmap       []byte
	dimensions int
	nodeCount  int

	// Lazy-loaded data
	nodeOffsets []int64                 // Byte offset of each node in mmap
	indices     map[int]*DimensionIndex // Lazily loaded indices (dimension -> index)
	indexMutex  sync.RWMutex

	// Write buffer for new inserts
	writeBuffer []types.Node
	bufferMutex sync.Mutex
}

// DimensionIndex holds sorted indices for one dimension
type DimensionIndex struct {
	sortedIndices []int32  // Node indices sorted by this dimension
	loaded        bool
}

// NewMmapStorage creates a memory-mapped storage
func NewMmapStorage(path string) (*MmapStorage, error) {
	// Open or create file
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	storage := &MmapStorage{
		path:        path,
		file:        file,
		indices:     make(map[int]*DimensionIndex),
		writeBuffer: make([]types.Node, 0, 1000),
	}

	// If file is empty, just return empty storage
	if stat.Size() == 0 {
		return storage, nil
	}

	// Memory-map the file (read-only for now)
	mmapData, err := unix.Mmap(
		int(file.Fd()),
		0,
		int(stat.Size()),
		unix.PROT_READ,
		unix.MAP_SHARED,
	)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to mmap: %w", err)
	}

	storage.mmap = mmapData

	// Read header quickly
	if len(mmapData) < 12 {
		return nil, fmt.Errorf("file too small for header")
	}

	storage.dimensions = int(binary.LittleEndian.Uint32(mmapData[0:4]))
	storage.nodeCount = int(binary.LittleEndian.Uint64(mmapData[4:12]))

	// Build offset table (fast: just scanning for positions, not loading data)
	if err := storage.buildOffsetTable(); err != nil {
		unix.Munmap(mmapData)
		file.Close()
		return nil, fmt.Errorf("failed to build offset table: %w", err)
	}

	return storage, nil
}

// buildOffsetTable scans the file to find byte offsets of each node
// This is fast because we don't actually read the vector data
func (m *MmapStorage) buildOffsetTable() error {
	m.nodeOffsets = make([]int64, m.nodeCount)
	offset := int64(12) // Skip header

	for i := 0; i < m.nodeCount; i++ {
		m.nodeOffsets[i] = offset

		// Skip past this node's data
		// Read dimension count
		if offset+4 > int64(len(m.mmap)) {
			return fmt.Errorf("unexpected EOF at node %d", i)
		}
		dims := binary.LittleEndian.Uint32(m.mmap[offset : offset+4])
		offset += 4

		// Skip vector data
		offset += int64(dims) * 4

		// Skip value (read length, skip string)
		if offset+8 > int64(len(m.mmap)) {
			return fmt.Errorf("unexpected EOF at node %d value", i)
		}
		valueLen := binary.LittleEndian.Uint64(m.mmap[offset : offset+8])
		offset += 8 + int64(valueLen)

		// Skip timestamp
		if offset+4 > int64(len(m.mmap)) {
			return fmt.Errorf("unexpected EOF at node %d timestamp", i)
		}
		timestampLen := binary.LittleEndian.Uint32(m.mmap[offset : offset+4])
		offset += 4 + int64(timestampLen)

		// Skip metadata
		if offset+4 > int64(len(m.mmap)) {
			return fmt.Errorf("unexpected EOF at node %d metadata", i)
		}
		metadataLen := binary.LittleEndian.Uint32(m.mmap[offset : offset+4])
		offset += 4 + int64(metadataLen)
	}

	return nil
}

// GetDimensionValue reads a single dimension value from a node (lazy, no copy)
func (m *MmapStorage) GetDimensionValue(nodeIdx int32, dim int) float32 {
	if int(nodeIdx) >= m.nodeCount {
		return 0
	}

	offset := m.nodeOffsets[nodeIdx]
	// Skip dimension count (4 bytes)
	offset += 4
	// Jump to specific dimension
	offset += int64(dim) * 4

	bits := binary.LittleEndian.Uint32(m.mmap[offset : offset+4])
	return *(*float32)(unsafe.Pointer(&bits))
}

// GetNode loads a full node from mmap (only when needed)
func (m *MmapStorage) GetNode(nodeIdx int32) (types.Node, error) {
	if int(nodeIdx) >= m.nodeCount {
		return types.Node{}, fmt.Errorf("node index out of range")
	}

	offset := m.nodeOffsets[nodeIdx]

	// Read dimension count
	dims := int(binary.LittleEndian.Uint32(m.mmap[offset : offset+4]))
	offset += 4

	// Read vector
	key := make([]float32, dims)
	for i := 0; i < dims; i++ {
		bits := binary.LittleEndian.Uint32(m.mmap[offset : offset+4])
		key[i] = *(*float32)(unsafe.Pointer(&bits))
		offset += 4
	}

	// Read value
	valueLen := binary.LittleEndian.Uint64(m.mmap[offset : offset+8])
	offset += 8
	value := string(m.mmap[offset : offset+int64(valueLen)])
	offset += int64(valueLen)

	// Read timestamp
	timestampLen := binary.LittleEndian.Uint32(m.mmap[offset : offset+4])
	offset += 4
	var timestamp time.Time
	if timestampLen > 0 {
		timestampBytes := m.mmap[offset : offset+int64(timestampLen)]
		timestamp.UnmarshalBinary(timestampBytes)
	}
	offset += int64(timestampLen)

	// Read metadata
	metadataLen := binary.LittleEndian.Uint32(m.mmap[offset : offset+4])
	offset += 4
	var metadata types.Metadata
	if metadataLen > 0 {
		metadataBytes := m.mmap[offset : offset+int64(metadataLen)]
		metadata = make(types.Metadata)
		json.Unmarshal(metadataBytes, &metadata)
	}

	return types.Node{
		Key:       key,
		Value:     value,
		Metadata:  metadata,
		Timestamp: timestamp,
	}, nil
}

// GetOrBuildIndex returns the index for a dimension, building it lazily if needed
func (m *MmapStorage) GetOrBuildIndex(dim int) []int32 {
	// Fast path: already loaded
	m.indexMutex.RLock()
	if idx, exists := m.indices[dim]; exists && idx.loaded {
		result := idx.sortedIndices
		m.indexMutex.RUnlock()
		return result
	}
	m.indexMutex.RUnlock()

	// Slow path: build index
	m.indexMutex.Lock()
	defer m.indexMutex.Unlock()

	// Double-check after lock
	if idx, exists := m.indices[dim]; exists && idx.loaded {
		return idx.sortedIndices
	}

	// Build sorted indices for this dimension
	indices := make([]int32, m.nodeCount)
	for i := 0; i < m.nodeCount; i++ {
		indices[i] = int32(i)
	}

	// Sort by dimension values
	sort.Slice(indices, func(i, j int) bool {
		valI := m.GetDimensionValue(indices[i], dim)
		valJ := m.GetDimensionValue(indices[j], dim)
		return valI < valJ
	})

	m.indices[dim] = &DimensionIndex{
		sortedIndices: indices,
		loaded:        true,
	}

	return indices
}

// LoadAsTree loads the entire tree into memory (for compatibility)
func (m *MmapStorage) LoadAsTree() (*types.Tree, error) {
	tree := types.NewTree(m.dimensions)

	// Load all nodes
	tree.Nodes = make([]types.Node, m.nodeCount)
	for i := 0; i < m.nodeCount; i++ {
		node, err := m.GetNode(int32(i))
		if err != nil {
			return nil, err
		}
		tree.Nodes[i] = node
	}

	// Rebuild indices
	tree.RebuildIndex()

	return tree, nil
}

// Close unmaps and closes the file
func (m *MmapStorage) Close() error {
	if m.mmap != nil {
		if err := unix.Munmap(m.mmap); err != nil {
			return err
		}
		m.mmap = nil
	}
	if m.file != nil {
		return m.file.Close()
	}
	return nil
}

// GetNodeCount returns the number of nodes
func (m *MmapStorage) GetNodeCount() int {
	return m.nodeCount
}

// GetDimensions returns the dimensionality
func (m *MmapStorage) GetDimensions() int {
	return m.dimensions
}
