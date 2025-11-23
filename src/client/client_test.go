package client

import (
	"os"
	"testing"
)

func TestClientInsertAndSearch(t *testing.T) {
	// Create temp file
	tmpfile := "test_client.bin"
	defer os.Remove(tmpfile)

	// Create client
	c, err := New(tmpfile, 3)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	c.SetVerbose(false)

	// Insert vectors
	err = c.Insert([]float32{0.1, 0.2, 0.3}, "first memory")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	err = c.Insert([]float32{0.1, 0.3, 0.2}, "second memory")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	err = c.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Search
	results, err := c.Search([]float32{0.1, 0.25, 0.25}, 0.2, 0.5, 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results, got none")
	}

	// Check node count
	count, err := c.GetNodeCount()
	if err != nil {
		t.Fatalf("GetNodeCount failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 nodes, got %d", count)
	}
}

func TestClientPersistence(t *testing.T) {
	tmpfile := "test_persistence.bin"
	defer os.Remove(tmpfile)

	// Create client and insert data
	c1, err := New(tmpfile, 3)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	c1.SetVerbose(false)

	c1.Insert([]float32{0.1, 0.2, 0.3}, "persistent memory")
	c1.Flush()

	// Create new client and load data
	c2, err := New(tmpfile, 3)
	if err != nil {
		t.Fatalf("Failed to create second client: %v", err)
	}
	c2.SetVerbose(false)

	count, err := c2.GetNodeCount()
	if err != nil {
		t.Fatalf("GetNodeCount failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 persisted node, got %d", count)
	}

	// Search should still work
	results, err := c2.Search([]float32{0.1, 0.2, 0.3}, 0.1, 0.5, 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 || results[0] != "persistent memory" {
		t.Errorf("Expected to find 'persistent memory', got %v", results)
	}
}

func TestClientDimensionMismatch(t *testing.T) {
	tmpfile := "test_dims.bin"
	defer os.Remove(tmpfile)

	c, err := New(tmpfile, 3)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	c.SetVerbose(false)

	// Try to insert wrong dimensions
	err = c.Insert([]float32{0.1, 0.2}, "wrong dims")
	if err == nil {
		t.Error("Expected dimension mismatch error, got nil")
	}

	// Try to search with wrong dimensions
	_, err = c.Search([]float32{0.1, 0.2, 0.3, 0.4}, 0.3, 0.5, 5)
	if err == nil {
		t.Error("Expected dimension mismatch error in search, got nil")
	}
}

func TestClientEmptyDatabase(t *testing.T) {
	tmpfile := "test_empty.bin"
	defer os.Remove(tmpfile)

	c, err := New(tmpfile, 128)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	c.SetVerbose(false)

	count, err := c.GetNodeCount()
	if err != nil {
		t.Fatalf("GetNodeCount failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 nodes in empty database, got %d", count)
	}

	// Search should return empty results
	vec := make([]float32, 128)
	results, err := c.Search(vec, 0.3, 0.5, 5)
	if err != nil {
		t.Fatalf("Search on empty database failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results from empty database, got %d", len(results))
	}
}

func BenchmarkClientInsert(b *testing.B) {
	tmpfile := "bench_insert.bin"
	defer os.Remove(tmpfile)

	c, _ := New(tmpfile, 128)
	c.SetVerbose(false)

	vec := make([]float32, 128)
	for i := range vec {
		vec[i] = float32(i) / 128.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Insert(vec, "benchmark")
	}
}

func BenchmarkClientSearch(b *testing.B) {
	tmpfile := "bench_search.bin"
	defer os.Remove(tmpfile)

	c, _ := New(tmpfile, 128)
	c.SetVerbose(false)

	// Insert 1000 vectors
	vec := make([]float32, 128)
	for i := 0; i < 1000; i++ {
		for j := range vec {
			vec[j] = float32(i+j) / 1000.0
		}
		c.Insert(vec, "test")
	}

	query := make([]float32, 128)
	for i := range query {
		query[i] = 0.5
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Search(query, 0.3, 0.5, 5)
	}
}
