package types

import (
	"fmt"
	"math"
	"sort"
)

type Node struct {
	Key   []float32 // Variable dimensions
	Value string
}

type Tree struct {
	Dimensions int         // Number of dimensions
	Nodes      []Node
	Index      [][]int32   // Variable dimensions
	indexDirty bool        // Track if indices need rebuilding
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
	if len(key) != t.Dimensions {
		return fmt.Errorf("dimension mismatch: expected %d, got %d", t.Dimensions, len(key))
	}

	nodeIdx := int32(len(t.Nodes))
	// Make a copy of the key to avoid external modifications
	keyCopy := make([]float32, t.Dimensions)
	copy(keyCopy, key)

	node := Node{
		Key:   keyCopy,
		Value: value,
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
	if len(query) != t.Dimensions {
		return nil, fmt.Errorf("dimension mismatch: expected %d, got %d", t.Dimensions, len(query))
	}

	if len(t.Nodes) == 0 {
		return nil, nil
	}

	// Ensure indices are built
	t.ensureIndex()

	// Preallocate candidate set with estimated size
	candidateSet := make(map[int32]int, len(t.Nodes)/10)

	for dim := 0; dim < t.Dimensions; dim++ {
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
			candidateSet[nodeIdx]++
		}
	}

	type scoredNode struct {
		node     Node
		distance float32
	}

	// Preallocate candidates slice
	candidates := make([]scoredNode, 0, topK*2)
	maxAllowedDistance := epsilon * float32(math.Sqrt(float64(t.Dimensions))) * (1.0 - threshold)

	for nodeIdx, count := range candidateSet {
		if count == t.Dimensions {
			var sumSquares float32
			for dim := 0; dim < t.Dimensions; dim++ {
				diff := query[dim] - t.Nodes[nodeIdx].Key[dim]
				sumSquares += diff * diff
			}
			distance := float32(math.Sqrt(float64(sumSquares)))

			if distance <= maxAllowedDistance {
				candidates = append(candidates, scoredNode{
					node:     t.Nodes[nodeIdx],
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
