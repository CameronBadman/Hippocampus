package memory

import (
	"container/heap"
	"math"
)

type SearchResult struct {
	ID string
	Text string
	Similarity float32
}

type MaxHeap []SearchResult

func (h MaxHeap) Len() int {return len(h)}
func (h MaxHeap) Less(i, j int) bool {return h[i].Similarity > h[j].Similarity}
func (h MaxHeap) Swap(i, j int) {h[i], h[j] = h[j], h[i]}

func (h *MaxHeap) Push(x any){
	*h = append(*h, x.(SearchResult))
}

func (h *MaxHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func CosineSimilarity(a, b []float32) float32{
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	return dot / (float32(math.Sqrt(float64(normA)))) * float32(math.Sqrt(float64(normB)))
}

func SemanticSearch(memories []Memory, query []float32, topK int) []SearchResult{
	h := &MaxHeap{}
	heap.Init(h)

	for _, m := range memories{
		sim := CosineSimilarity(query, m.Embedding)
		heap.Push(h, SearchResult{ID: m.ID, Text: m.Text, Similarity: sim})
	}

	var results []SearchResult
	for i := 0; i < topK && h.Len() > 0; i++ {
		results = append(results, heap.Pop(h).(SearchResult))
	}

	return results
}