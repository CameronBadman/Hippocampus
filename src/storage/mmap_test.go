package storage

import (
	"Hippocampus/src/types"
	"os"
	"testing"
)

func TestMmapStorage(t *testing.T) {
	tmpfile := "test_mmap.bin"
	defer os.Remove(tmpfile)

	// Create and save a tree with regular storage first
	tree := types.NewTree(128)
	for i := 0; i < 1000; i++ {
		vec := make([]float32, 128)
		for j := range vec {
			vec[j] = float32(i*128+j) / 10000.0
		}
		tree.Insert(vec, "test")
	}

	fs := New(tmpfile)
	if err := fs.Save(tree); err != nil {
		t.Fatalf("Failed to save tree: %v", err)
	}

	// Load with mmap
	mmap, err := NewMmapStorage(tmpfile)
	if err != nil {
		t.Fatalf("Failed to create mmap storage: %v", err)
	}
	defer mmap.Close()

	// Verify node count and dimensions
	if mmap.GetNodeCount() != 1000 {
		t.Errorf("Expected 1000 nodes, got %d", mmap.GetNodeCount())
	}

	if mmap.GetDimensions() != 128 {
		t.Errorf("Expected 128 dimensions, got %d", mmap.GetDimensions())
	}

	// Test lazy index loading
	index := mmap.GetOrBuildIndex(0)
	if len(index) != 1000 {
		t.Errorf("Expected index with 1000 entries, got %d", len(index))
	}

	// Verify index is sorted
	for i := 1; i < len(index); i++ {
		valPrev := mmap.GetDimensionValue(index[i-1], 0)
		valCurr := mmap.GetDimensionValue(index[i], 0)
		if valPrev > valCurr {
			t.Errorf("Index not sorted at position %d", i)
		}
	}

	// Test GetNode
	node, err := mmap.GetNode(0)
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	if len(node.Key) != 128 {
		t.Errorf("Expected node with 128 dimensions, got %d", len(node.Key))
	}
}

func BenchmarkMmapIndexLoad(b *testing.B) {
	tmpfile := "bench_mmap.bin"
	defer os.Remove(tmpfile)

	// Create database with 5000 nodes
	tree := types.NewTree(512)
	for i := 0; i < 5000; i++ {
		vec := make([]float32, 512)
		for j := range vec {
			vec[j] = float32(i*512+j) / 1000000.0
		}
		tree.Insert(vec, "test")
	}

	fs := New(tmpfile)
	fs.Save(tree)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mmap, _ := NewMmapStorage(tmpfile)
		// Just creating the storage (offset table only)
		mmap.Close()
	}
}

func BenchmarkMmapLazyIndexBuild(b *testing.B) {
	tmpfile := "bench_mmap_index.bin"
	defer os.Remove(tmpfile)

	// Create database
	tree := types.NewTree(512)
	for i := 0; i < 5000; i++ {
		vec := make([]float32, 512)
		for j := range vec {
			vec[j] = float32(i*512+j) / 1000000.0
		}
		tree.Insert(vec, "test")
	}

	fs := New(tmpfile)
	fs.Save(tree)

	mmap, _ := NewMmapStorage(tmpfile)
	defer mmap.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Build index for one dimension
		dim := i % 512
		mmap.GetOrBuildIndex(dim)
	}
}

func BenchmarkMmapVsRegularLoad(b *testing.B) {
	tmpfile := "bench_compare.bin"
	defer os.Remove(tmpfile)

	// Create database
	tree := types.NewTree(512)
	for i := 0; i < 10000; i++ {
		vec := make([]float32, 512)
		for j := range vec {
			vec[j] = float32(i*512+j) / 1000000.0
		}
		tree.Insert(vec, "test")
	}

	fs := New(tmpfile)
	fs.Save(tree)

	b.Run("Regular Load", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fs := New(tmpfile)
			tree, _ := fs.Load()
			_ = tree
		}
	})

	b.Run("Mmap Load (offset table only)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mmap, _ := NewMmapStorage(tmpfile)
			mmap.Close()
		}
	})
}
