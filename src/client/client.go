package client

import (
	"Hippocampus/src/storage"
	hippotypes "Hippocampus/src/types"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Storage    storage.FileStorage
	Dimensions int

	// In-memory cache
	cachedTree *hippotypes.Tree
	dirty      bool
	verbose    bool
}

// New creates a new client without any embedding provider dependencies
func New(binaryPath string, dimensions int) (*Client, error) {
	if dimensions <= 0 {
		dimensions = 512 // Default to 512 for backwards compatibility
	}

	return &Client{
		Storage:    *storage.New(binaryPath),
		Dimensions: dimensions,
		cachedTree: nil,
		dirty:      false,
		verbose:    true,
	}, nil
}

// SetVerbose controls whether the client prints timing and progress info
func (client *Client) SetVerbose(verbose bool) {
	client.verbose = verbose
}

// getTree returns the in-memory tree, loading from disk if needed
func (client *Client) getTree() (*hippotypes.Tree, error) {
	if client.cachedTree == nil {
		tree, err := client.Storage.Load()
		if err != nil {
			return nil, err
		}

		// If tree is empty, initialize with client's dimensions
		if len(tree.Nodes) == 0 {
			tree.Dimensions = client.Dimensions
			tree.Index = make([][]int32, client.Dimensions)
		} else if tree.Dimensions != client.Dimensions {
			// Update dimensions from loaded tree if it differs and tree has data
			if client.verbose {
				fmt.Printf("Note: Tree dimensions (%d) differ from client dimensions (%d), using tree dimensions\n",
					tree.Dimensions, client.Dimensions)
			}
			client.Dimensions = tree.Dimensions
		}
		client.cachedTree = tree
	}
	return client.cachedTree, nil
}

// Flush writes the cached tree to disk if dirty
func (client *Client) Flush() error {
	if client.dirty && client.cachedTree != nil {
		if err := client.Storage.Save(client.cachedTree); err != nil {
			return err
		}
		client.dirty = false
	}
	return nil
}

// Insert adds a vector and its associated text to the database
func (client *Client) Insert(embedding []float32, text string) error {
	return client.InsertWithMetadata(embedding, text, nil)
}

// InsertWithMetadata adds a vector with metadata to the database
func (client *Client) InsertWithMetadata(embedding []float32, text string, metadata hippotypes.Metadata) error {
	if len(embedding) != client.Dimensions {
		return fmt.Errorf("dimension mismatch: expected %d, got %d", client.Dimensions, len(embedding))
	}

	insertStart := time.Now()

	tree, err := client.getTree()
	if err != nil {
		return fmt.Errorf("tree loading error: %w", err)
	}

	// Ensure tree has correct dimensions
	if tree.Dimensions == 0 {
		tree.Dimensions = client.Dimensions
		tree.Index = make([][]int32, client.Dimensions)
	}

	if err := tree.InsertWithMetadata(embedding, text, metadata); err != nil {
		return fmt.Errorf("insert error: %w", err)
	}
	client.dirty = true

	insertDuration := time.Since(insertStart)

	// Periodic flush every 100 inserts
	var flushDuration time.Duration
	if len(tree.Nodes)%100 == 0 {
		flushStart := time.Now()
		if err := client.Flush(); err != nil {
			return fmt.Errorf("flush error: %w", err)
		}
		flushDuration = time.Since(flushStart)
	}

	if client.verbose {
		fmt.Printf("Successfully inserted (total nodes: %d)\n", len(tree.Nodes))
		fmt.Printf("TIMING:INSERT:%.3fms:FLUSH:%.3fms\n",
			insertDuration.Seconds()*1000,
			flushDuration.Seconds()*1000)
	}
	return nil
}

// Search finds similar vectors in the database
func (client *Client) Search(embedding []float32, epsilon float32, threshold float32, topK int) ([]string, error) {
	if len(embedding) != client.Dimensions {
		return nil, fmt.Errorf("dimension mismatch: expected %d, got %d", client.Dimensions, len(embedding))
	}

	searchStart := time.Now()

	tree, err := client.getTree()
	if err != nil {
		return nil, fmt.Errorf("tree loading error: %w", err)
	}

	results, err := tree.Search(embedding, epsilon, threshold, topK)
	if err != nil {
		return nil, fmt.Errorf("search error: %w", err)
	}

	searchDuration := time.Since(searchStart)

	values := make([]string, len(results))
	for i, node := range results {
		values[i] = node.Value
	}

	if client.verbose {
		fmt.Printf("\nFound %d results (top %d, threshold %.2f):\n", len(results), topK, threshold)
		for _, value := range values {
			fmt.Printf("  %s\n", value)
		}
		fmt.Printf("TIMING:SEARCH:%.6fms\n", searchDuration.Seconds()*1000)
	}

	return values, nil
}

// InsertCSV bulk inserts from a CSV file
// CSV format: embedding_vector (comma-separated floats), text
// Example: 0.1,0.2,0.3,...,"memory text"
func (client *Client) InsertCSV(csvFilename string) error {
	file, err := os.Open(csvFilename)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	lineNum := 0

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading line %d: %v", lineNum, err)
		}
		lineNum++

		if len(record) < 2 {
			return fmt.Errorf("line %d: expected at least 2 fields (embedding + text), got %d", lineNum, len(record))
		}

		// Last field is the text, everything else is the embedding
		text := record[len(record)-1]
		embeddingStrs := record[:len(record)-1]

		if len(embeddingStrs) != client.Dimensions {
			return fmt.Errorf("line %d: dimension mismatch: expected %d, got %d", lineNum, client.Dimensions, len(embeddingStrs))
		}

		// Parse embedding
		embedding := make([]float32, client.Dimensions)
		for i, str := range embeddingStrs {
			val, err := strconv.ParseFloat(strings.TrimSpace(str), 32)
			if err != nil {
				return fmt.Errorf("line %d, dimension %d: invalid float: %v", lineNum, i, err)
			}
			embedding[i] = float32(val)
		}

		if err := client.Insert(embedding, text); err != nil {
			return fmt.Errorf("line %d: %v", lineNum, err)
		}
	}

	// Final flush
	return client.Flush()
}

// InsertJSON inserts from a JSON file containing an array of {embedding: [], text: ""}
func (client *Client) InsertJSON(jsonFilename string) error {
	file, err := os.Open(jsonFilename)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	var records []struct {
		Embedding []float32 `json:"embedding"`
		Text      string    `json:"text"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&records); err != nil {
		return fmt.Errorf("error parsing JSON: %v", err)
	}

	for i, record := range records {
		if err := client.Insert(record.Embedding, record.Text); err != nil {
			return fmt.Errorf("record %d: %v", i, err)
		}
	}

	return client.Flush()
}

// GetNodeCount returns the number of nodes in the database
func (client *Client) GetNodeCount() (int, error) {
	tree, err := client.getTree()
	if err != nil {
		return 0, err
	}
	return len(tree.Nodes), nil
}

// GetDimensions returns the dimensionality of vectors in the database
func (client *Client) GetDimensions() int {
	return client.Dimensions
}

// SearchWithFilter finds similar vectors that match the filter criteria
func (client *Client) SearchWithFilter(embedding []float32, epsilon float32, threshold float32, topK int, filter *hippotypes.Filter) ([]string, error) {
	if len(embedding) != client.Dimensions {
		return nil, fmt.Errorf("dimension mismatch: expected %d, got %d", client.Dimensions, len(embedding))
	}

	searchStart := time.Now()

	tree, err := client.getTree()
	if err != nil {
		return nil, fmt.Errorf("tree loading error: %w", err)
	}

	results, err := tree.SearchWithFilter(embedding, epsilon, threshold, topK, filter)
	if err != nil {
		return nil, fmt.Errorf("search error: %w", err)
	}

	searchDuration := time.Since(searchStart)

	values := make([]string, len(results))
	for i, node := range results {
		values[i] = node.Value
	}

	if client.verbose {
		fmt.Printf("\nFound %d results (top %d, threshold %.2f, with filters):\n", len(results), topK, threshold)
		for _, value := range values {
			fmt.Printf("  %s\n", value)
		}
		fmt.Printf("TIMING:SEARCH:%.6fms\n", searchDuration.Seconds()*1000)
	}

	return values, nil
}

// BatchInsert efficiently inserts multiple vectors at once
func (client *Client) BatchInsert(items []struct {
	Embedding []float32
	Text      string
	Metadata  hippotypes.Metadata
}) error {
	batchStart := time.Now()

	tree, err := client.getTree()
	if err != nil {
		return fmt.Errorf("tree loading error: %w", err)
	}

	// Ensure tree has correct dimensions
	if tree.Dimensions == 0 {
		tree.Dimensions = client.Dimensions
		tree.Index = make([][]int32, client.Dimensions)
	}

	// Convert to tree's batch format
	treeItems := make([]struct {
		Key      []float32
		Value    string
		Metadata hippotypes.Metadata
	}, len(items))

	for i, item := range items {
		if len(item.Embedding) != client.Dimensions {
			return fmt.Errorf("item %d: dimension mismatch: expected %d, got %d", i, client.Dimensions, len(item.Embedding))
		}
		treeItems[i].Key = item.Embedding
		treeItems[i].Value = item.Text
		treeItems[i].Metadata = item.Metadata
	}

	if err := tree.BatchInsert(treeItems); err != nil {
		return fmt.Errorf("batch insert error: %w", err)
	}
	client.dirty = true

	batchDuration := time.Since(batchStart)

	if client.verbose {
		fmt.Printf("Batch inserted %d items in %.3fms (%.1f items/ms)\n",
			len(items),
			batchDuration.Seconds()*1000,
			float64(len(items))/(batchDuration.Seconds()*1000))
	}

	return client.Flush()
}
