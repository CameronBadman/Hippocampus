package types

import "sort"

type Node struct {
	Key   [512]float32
	Value string
}

type Tree struct {
	Nodes []Node
	Index [512][]int32
}
type SearchResult struct {
	Node  Node
	Score float32 // smaller = closer
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

func (t *Tree) Search(query [512]float32, epsilon float32, topK int) []SearchResult {
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
	results := make([]SearchResult, 0)
	for nodeIdx, count := range candidateSet {
		if count == 512 {
			d := squaredDistance(query, t.Nodes[nodeIdx].Key)
            results = append(results, SearchResult{
                Node:  t.Nodes[nodeIdx],
                Score: d,
            })
		}
	}
	sort.Slice(results, func(i, j int) bool {
        return results[i].Score < results[j].Score
    })
    if topK > 0 && len(results) > topK {
        results = results[:topK]
    }
	return results
}

func squaredDistance(a, b [512]float32) float32 {
	var sum float32
	for i := 0; i < 512; i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return sum
}
