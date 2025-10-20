package types

import (
	"math"
	"sort"
)

type Node struct {
	Key   [512]float32
	Value string
}

type Tree struct {
	Nodes []Node
	Index [512][]int32
}

func NewTree() *Tree {
	return &Tree{
		Nodes: make([]Node, 0),
		Index: [512][]int32{},
	}
}

func (t *Tree) Insert(key [512]float32, value string) {
	nodeIdx := int32(len(t.Nodes))
	node := Node{
		Key:   key,
		Value: value,
	}
	t.Nodes = append(t.Nodes, node)
	for dim := 0; dim < 512; dim++ {
		insertPos := sort.Search(len(t.Index[dim]), func(i int) bool {
			return t.Nodes[t.Index[dim][i]].Key[dim] >= key[dim]
		})
		t.Index[dim] = append(t.Index[dim], 0)
		copy(t.Index[dim][insertPos+1:], t.Index[dim][insertPos:])
		t.Index[dim][insertPos] = nodeIdx
	}
}

func (t *Tree) RebuildIndex() {
	for dim := 0; dim < 512; dim++ {
		t.Index[dim] = make([]int32, len(t.Nodes))
		for i := range t.Nodes {
			t.Index[dim][i] = int32(i)
		}
		sort.Slice(t.Index[dim], func(i, j int) bool {
			return t.Nodes[t.Index[dim][i]].Key[dim] < t.Nodes[t.Index[dim][j]].Key[dim]
		})
	}
}

func (t *Tree) Search(query [512]float32, epsilon float32, threshold float32, topK int) []Node {
	if len(t.Nodes) == 0 {
		return nil
	}
	
	candidateSet := make(map[int32]int)
	for dim := 0; dim < 512; dim++ {
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
	
	candidates := make([]scoredNode, 0)
	for nodeIdx, count := range candidateSet {
		if count == 512 {
			var sumSquares float32
			for dim := 0; dim < 512; dim++ {
				diff := query[dim] - t.Nodes[nodeIdx].Key[dim]
				sumSquares += diff * diff
			}
			distance := float32(math.Sqrt(float64(sumSquares)))
			
			maxAllowedDistance := epsilon * float32(math.Sqrt(512)) * (1.0 - threshold)
			
			if distance <= maxAllowedDistance {
				candidates = append(candidates, scoredNode{
					node:     t.Nodes[nodeIdx],
					distance: distance,
				})
			}
		}
	}
	
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].distance < candidates[j].distance
	})
	
	limit := topK
	if len(candidates) < topK {
		limit = len(candidates)
	}
	
	results := make([]Node, limit)
	for i := 0; i < limit; i++ {
		results[i] = candidates[i].node
	}
	
	return results
}
