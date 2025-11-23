package types

import (
	"testing"
)

func TestNewTree(t *testing.T) {
	tests := []struct {
		name string
		dims int
		want int
	}{
		{"default dimensions", 0, 512},
		{"custom dimensions", 128, 128},
		{"large dimensions", 1536, 1536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewTree(tt.dims)
			if tree.Dimensions != tt.want {
				t.Errorf("NewTree(%d).Dimensions = %d, want %d", tt.dims, tree.Dimensions, tt.want)
			}
			if len(tree.Index) != tt.want {
				t.Errorf("NewTree(%d) created %d indices, want %d", tt.dims, len(tree.Index), tt.want)
			}
		})
	}
}

func TestTreeInsert(t *testing.T) {
	tree := NewTree(3)

	// Insert first node
	vec1 := []float32{0.1, 0.2, 0.3}
	err := tree.Insert(vec1, "first")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if len(tree.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(tree.Nodes))
	}

	// Insert second node
	vec2 := []float32{0.4, 0.5, 0.6}
	err = tree.Insert(vec2, "second")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if len(tree.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(tree.Nodes))
	}

	// Test dimension mismatch
	vecWrong := []float32{0.1, 0.2}
	err = tree.Insert(vecWrong, "wrong")
	if err == nil {
		t.Error("Expected dimension mismatch error, got nil")
	}
}

func TestTreeSearch(t *testing.T) {
	tree := NewTree(3)

	// Insert test data
	tree.Insert([]float32{0.1, 0.2, 0.3}, "first")
	tree.Insert([]float32{0.1, 0.3, 0.2}, "second")
	tree.Insert([]float32{0.9, 0.1, 0.05}, "third")

	// Search near first node
	results, err := tree.Search([]float32{0.1, 0.25, 0.25}, 0.2, 0.5, 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) < 1 {
		t.Errorf("Expected at least 1 result, got %d", len(results))
	}

	// Results should include similar vectors
	found := false
	for _, node := range results {
		if node.Value == "first" || node.Value == "second" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'first' or 'second', but didn't")
	}

	// Test dimension mismatch in search
	_, err = tree.Search([]float32{0.1, 0.2}, 0.2, 0.5, 2)
	if err == nil {
		t.Error("Expected dimension mismatch error in search, got nil")
	}
}

func TestTreeEmptySearch(t *testing.T) {
	tree := NewTree(3)

	results, err := tree.Search([]float32{0.1, 0.2, 0.3}, 0.3, 0.5, 5)
	if err != nil {
		t.Fatalf("Search on empty tree failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results from empty tree, got %d", len(results))
	}
}

func BenchmarkInsert(b *testing.B) {
	tree := NewTree(128)
	vec := make([]float32, 128)
	for i := range vec {
		vec[i] = float32(i) / 128.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Insert(vec, "test")
	}
}

func BenchmarkSearch(b *testing.B) {
	tree := NewTree(128)
	vec := make([]float32, 128)

	// Insert 1000 nodes
	for i := 0; i < 1000; i++ {
		for j := range vec {
			vec[j] = float32(i+j) / 1000.0
		}
		tree.Insert(vec, "test")
	}

	query := make([]float32, 128)
	for i := range query {
		query[i] = 0.5
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Search(query, 0.3, 0.5, 5)
	}
}

func BenchmarkSearchLarge(b *testing.B) {
	tree := NewTree(512)
	vec := make([]float32, 512)

	// Insert 5000 nodes
	for i := 0; i < 5000; i++ {
		for j := range vec {
			vec[j] = float32(i*512+j) / 2560000.0
		}
		tree.Insert(vec, "test")
	}

	query := make([]float32, 512)
	for i := range query {
		query[i] = 0.5
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Search(query, 0.3, 0.5, 5)
	}
}
