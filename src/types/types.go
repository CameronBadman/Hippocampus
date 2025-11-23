package types

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"
)

type Metadata map[string]interface{}

// RadiusMapping maps semantic words to epsilon values
var RadiusMapping = map[string]float32{
	"exact":   0.10, // Very tight matching
	"precise": 0.15,
	"similar": 0.25,
	"related": 0.35,
	"broad":   0.45,
	"fuzzy":   0.60, // Very loose matching
}

// GetRadiusValue returns the epsilon value for a radius word, or default if not set
func GetRadiusValue(radiusWord string, defaultEpsilon float32) float32 {
	if radiusWord == "" {
		return defaultEpsilon
	}
	if val, exists := RadiusMapping[radiusWord]; exists {
		return val
	}
	return defaultEpsilon
}

type Node struct {
	Key       []float32 // Variable dimensions
	Value     string
	Metadata  Metadata  // Flexible metadata
	Timestamp time.Time // When node was created
}

type Tree struct {
	Dimensions int         // Number of dimensions
	Nodes      []Node
	Index      [][]int32   // Variable dimensions
	indexDirty bool        // Track if indices need rebuilding
}

// Filter represents search constraints
type Filter struct {
	Metadata      map[string]interface{} // Key-value filters
	TimestampFrom *time.Time             // Filter by time range
	TimestampTo   *time.Time
}

// MatchesFilter checks if a node matches the filter criteria
func (n *Node) MatchesFilter(f *Filter) bool {
	if f == nil {
		return true
	}

	// Check timestamp filters
	if f.TimestampFrom != nil && n.Timestamp.Before(*f.TimestampFrom) {
		return false
	}
	if f.TimestampTo != nil && n.Timestamp.After(*f.TimestampTo) {
		return false
	}

	// Check metadata filters
	for key, expectedValue := range f.Metadata {
		actualValue, exists := n.Metadata[key]
		if !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

func NewTree(dimensions int) *Tree {
	if dimensions <= 0 {
		dimensions = 512 // Default to 512 for backwards compatibility
	}
	return &Tree{
		Dimensions: dimensions,
		Nodes:      make([]Node, 0, 1000), // Preallocate for 1000 nodes
		Index:      make([][]int32, dimensions),
		indexDirty: false,
	}
}

func (t *Tree) Insert(key []float32, value string) error {
	return t.InsertWithMetadata(key, value, nil)
}

func (t *Tree) InsertWithMetadata(key []float32, value string, metadata Metadata) error {
	if len(key) != t.Dimensions {
		return fmt.Errorf("dimension mismatch: expected %d, got %d", t.Dimensions, len(key))
	}

	nodeIdx := int32(len(t.Nodes))
	// Make a copy of the key to avoid external modifications
	keyCopy := make([]float32, t.Dimensions)
	copy(keyCopy, key)

	node := Node{
		Key:       keyCopy,
		Value:     value,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}
	t.Nodes = append(t.Nodes, node)

	// If indices exist, update them incrementally
	if len(t.Index[0]) > 0 && !t.indexDirty {
		for dim := 0; dim < t.Dimensions; dim++ {
			insertPos := sort.Search(len(t.Index[dim]), func(i int) bool {
				return t.Nodes[t.Index[dim][i]].Key[dim] >= key[dim]
			})
			t.Index[dim] = append(t.Index[dim], 0)
			copy(t.Index[dim][insertPos+1:], t.Index[dim][insertPos:])
			t.Index[dim][insertPos] = nodeIdx
		}
	} else {
		// Mark indices as dirty - will rebuild on next search
		t.indexDirty = true
	}
	return nil
}

func (t *Tree) RebuildIndex() {
	nodeCount := len(t.Nodes)
	for dim := 0; dim < t.Dimensions; dim++ {
		t.Index[dim] = make([]int32, nodeCount)
		for i := 0; i < nodeCount; i++ {
			t.Index[dim][i] = int32(i)
		}
		sort.Slice(t.Index[dim], func(i, j int) bool {
			return t.Nodes[t.Index[dim][i]].Key[dim] < t.Nodes[t.Index[dim][j]].Key[dim]
		})
	}
	t.indexDirty = false
}

// ensureIndex ensures indices are built before search
func (t *Tree) ensureIndex() {
	if t.indexDirty || len(t.Index) == 0 || len(t.Index[0]) == 0 {
		t.RebuildIndex()
	}
}

func (t *Tree) Search(query []float32, epsilon float32, threshold float32, topK int) ([]Node, error) {
	return t.SearchWithFilter(query, epsilon, threshold, topK, nil)
}

func (t *Tree) SearchWithFilter(query []float32, epsilon float32, threshold float32, topK int, filter *Filter) ([]Node, error) {
	if len(query) != t.Dimensions {
		return nil, fmt.Errorf("dimension mismatch: expected %d, got %d", t.Dimensions, len(query))
	}

	if len(t.Nodes) == 0 {
		return nil, nil
	}

	// Ensure indices are built
	t.ensureIndex()

	// Parallel search across dimensions
	candidateSet := t.parallelDimensionSearch(query, epsilon)

	type scoredNode struct {
		node     Node
		distance float32
	}

	// Preallocate candidates slice
	candidates := make([]scoredNode, 0, topK*2)
	maxAllowedDistance := epsilon * float32(math.Sqrt(float64(t.Dimensions))) * (1.0 - threshold)

	for nodeIdx, count := range candidateSet {
		if count == t.Dimensions {
			node := &t.Nodes[nodeIdx]

			// Apply filter first (cheap check before distance calculation)
			if !node.MatchesFilter(filter) {
				continue
			}

			var sumSquares float32
			for dim := 0; dim < t.Dimensions; dim++ {
				diff := query[dim] - node.Key[dim]
				sumSquares += diff * diff
			}
			distance := float32(math.Sqrt(float64(sumSquares)))

			if distance <= maxAllowedDistance {
				candidates = append(candidates, scoredNode{
					node:     *node,
					distance: distance,
				})
			}
		}
	}

	// Sort only if we have more results than needed
	if len(candidates) > topK {
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].distance < candidates[j].distance
		})
	} else if len(candidates) > 1 {
		// For small result sets, still sort
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].distance < candidates[j].distance
		})
	}

	limit := topK
	if len(candidates) < topK {
		limit = len(candidates)
	}

	results := make([]Node, limit)
	for i := 0; i < limit; i++ {
		results[i] = candidates[i].node
	}

	return results, nil
}

// parallelDimensionSearch searches dimensions in parallel for candidates
func (t *Tree) parallelDimensionSearch(query []float32, epsilon float32) map[int32]int {
	numWorkers := runtime.NumCPU()
	if numWorkers > t.Dimensions {
		numWorkers = t.Dimensions
	}

	candidateSet := make(map[int32]int, len(t.Nodes)/10)
	var mu sync.Mutex
	var wg sync.WaitGroup

	dimsPerWorker := (t.Dimensions + numWorkers - 1) / numWorkers

	for worker := 0; worker < numWorkers; worker++ {
		wg.Add(1)
		start := worker * dimsPerWorker
		end := start + dimsPerWorker
		if end > t.Dimensions {
			end = t.Dimensions
		}

		go func(startDim, endDim int) {
			defer wg.Done()

			localCandidates := make(map[int32]int, len(t.Nodes)/10)

			for dim := startDim; dim < endDim; dim++ {
				minVal := query[dim] - epsilon
				maxVal := query[dim] + epsilon

				startIdx := sort.Search(len(t.Index[dim]), func(i int) bool {
					return t.Nodes[t.Index[dim][i]].Key[dim] >= minVal
				})

				endIdx := sort.Search(len(t.Index[dim]), func(i int) bool {
					return t.Nodes[t.Index[dim][i]].Key[dim] > maxVal
				})

				for i := startIdx; i < endIdx; i++ {
					nodeIdx := t.Index[dim][i]
					localCandidates[nodeIdx]++
				}
			}

			// Merge local results into global candidate set
			mu.Lock()
			for nodeIdx, count := range localCandidates {
				candidateSet[nodeIdx] += count
			}
			mu.Unlock()
		}(start, end)
	}

	wg.Wait()
	return candidateSet
}

// BatchInsert inserts multiple nodes efficiently
func (t *Tree) BatchInsert(items []struct {
	Key      []float32
	Value    string
	Metadata Metadata
}) error {
	// Validate all dimensions first
	for i, item := range items {
		if len(item.Key) != t.Dimensions {
			return fmt.Errorf("item %d: dimension mismatch: expected %d, got %d", i, t.Dimensions, len(item.Key))
		}
	}

	// Add all nodes
	for _, item := range items {
		keyCopy := make([]float32, t.Dimensions)
		copy(keyCopy, item.Key)

		node := Node{
			Key:       keyCopy,
			Value:     item.Value,
			Metadata:  item.Metadata,
			Timestamp: time.Now(),
		}
		t.Nodes = append(t.Nodes, node)
	}

	// Rebuild indices once at the end
	t.RebuildIndex()
	return nil
}
