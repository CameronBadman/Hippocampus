package main

import (
	"Hippocampus/src/client"
	"Hippocampus/src/embedding"
	hippotypes "Hippocampus/src/types"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "insert":
		handleInsert()
	case "search":
		handleSearch()
	case "insert-csv":
		handleInsertCSV()
	case "insert-json":
		handleInsertJSON()
	case "info":
		handleInfo()
	default:
		log.Fatalf("unknown command: %s\nRun 'hippocampus' with no arguments for usage", command)
	}
}

func printUsage() {
	fmt.Println("Hippocampus CLI - Fast Local Vector Database")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  hippocampus insert -db tree.bin -vector <json_array> -text <text>")
	fmt.Println("  hippocampus insert -db tree.bin -text <text> -ollama")
	fmt.Println("  hippocampus search -db tree.bin -vector <json_array>")
	fmt.Println("  hippocampus search -db tree.bin -text <text> -ollama")
	fmt.Println("  hippocampus insert-csv -db tree.bin -csv <file.csv>")
	fmt.Println("  hippocampus insert-json -db tree.bin -json <file.json>")
	fmt.Println("  hippocampus info -db tree.bin")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  insert       Insert a vector with text")
	fmt.Println("  search       Search for similar vectors")
	fmt.Println("  insert-csv   Bulk insert from CSV (vectors + text)")
	fmt.Println("  insert-json  Bulk insert from JSON")
	fmt.Println("  info         Show database info (node count, dimensions)")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -db          Database file path (default: tree.bin)")
	fmt.Println("  -dims        Vector dimensions (default: 512, auto-detected from file)")
	fmt.Println("  -vector      Vector as JSON array: [0.1, 0.2, ...]")
	fmt.Println("  -text        Text to store/search")
	fmt.Println("  -epsilon     Search epsilon (default: 0.3)")
	fmt.Println("  -threshold   Search threshold (default: 0.5)")
	fmt.Println("  -top-k       Max results (default: 5)")
	fmt.Println()
	fmt.Println("Local Embedding Options:")
	fmt.Println("  -ollama      Use Ollama for embeddings")
	fmt.Println("  -ollama-url  Ollama server URL (default: http://localhost:11434)")
	fmt.Println("  -ollama-model Ollama model (default: nomic-embed-text)")
	fmt.Println("  -llama-cpp   Use llama.cpp server")
	fmt.Println("  -llama-url   llama.cpp URL (default: http://localhost:8080)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Insert with explicit vector")
	fmt.Println("  hippocampus insert -db memory.bin -vector '[0.1,0.2,...]' -text 'Hello'")
	fmt.Println()
	fmt.Println("  # Insert with Ollama (local, fast!)")
	fmt.Println("  hippocampus insert -db memory.bin -text 'Hello' -ollama")
	fmt.Println()
	fmt.Println("  # Search with Ollama")
	fmt.Println("  hippocampus search -db memory.bin -text 'greeting' -ollama")
	fmt.Println()
	fmt.Println("  # Show database info")
	fmt.Println("  hippocampus info -db memory.bin")
}

func parseVectorFlag(vectorStr string) ([]float32, error) {
	vectorStr = strings.TrimSpace(vectorStr)
	if !strings.HasPrefix(vectorStr, "[") || !strings.HasSuffix(vectorStr, "]") {
		return nil, fmt.Errorf("vector must be a JSON array: [0.1, 0.2, ...]")
	}

	var floats []float64
	if err := json.Unmarshal([]byte(vectorStr), &floats); err != nil {
		return nil, fmt.Errorf("invalid JSON array: %w", err)
	}

	vector := make([]float32, len(floats))
	for i, v := range floats {
		vector[i] = float32(v)
	}

	return vector, nil
}

func getEmbeddingProvider(ollama bool, ollamaURL, ollamaModel string, llamaCpp bool, llamaURL string) embedding.EmbeddingProvider {
	if ollama {
		return embedding.NewOllamaClient(ollamaURL, ollamaModel)
	}
	if llamaCpp {
		return embedding.NewLlamaCppClient(llamaURL)
	}
	return nil
}

func handleInsert() {
	insertCmd := flag.NewFlagSet("insert", flag.ExitOnError)
	dbPath := insertCmd.String("db", "tree.bin", "database file")
	dims := insertCmd.Int("dims", 512, "vector dimensions")
	vectorStr := insertCmd.String("vector", "", "vector as JSON array")
	text := insertCmd.String("text", "", "text to store")

	// Embedding provider flags
	useOllama := insertCmd.Bool("ollama", false, "use Ollama for embeddings")
	ollamaURL := insertCmd.String("ollama-url", "http://localhost:11434", "Ollama server URL")
	ollamaModel := insertCmd.String("ollama-model", "nomic-embed-text", "Ollama model")
	useLlamaCpp := insertCmd.Bool("llama-cpp", false, "use llama.cpp server")
	llamaURL := insertCmd.String("llama-url", "http://localhost:8080", "llama.cpp server URL")

	insertCmd.Parse(os.Args[2:])

	if *text == "" {
		log.Fatal("-text is required")
	}

	c, err := client.New(*dbPath, *dims)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	var vector []float32

	// Get embedding
	if *vectorStr != "" {
		// User provided explicit vector
		vector, err = parseVectorFlag(*vectorStr)
		if err != nil {
			log.Fatalf("Invalid vector: %v", err)
		}
	} else if provider := getEmbeddingProvider(*useOllama, *ollamaURL, *ollamaModel, *useLlamaCpp, *llamaURL); provider != nil {
		// Use local embedding provider
		fmt.Println("Generating embedding...")
		vector, err = provider.GetEmbedding(*text)
		if err != nil {
			log.Fatalf("Failed to generate embedding: %v", err)
		}
		fmt.Printf("Generated %d-dimensional embedding\n", len(vector))
	} else {
		log.Fatal("Either provide -vector or use -ollama/-llama-cpp for local embeddings")
	}

	if err := c.Insert(vector, *text); err != nil {
		log.Fatalf("Insert failed: %v", err)
	}

	if err := c.Flush(); err != nil {
		log.Fatalf("Flush failed: %v", err)
	}
}

func handleSearch() {
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	dbPath := searchCmd.String("db", "tree.bin", "database file")
	dims := searchCmd.Int("dims", 0, "vector dimensions (0 = auto-detect)")
	vectorStr := searchCmd.String("vector", "", "query vector as JSON array")
	text := searchCmd.String("text", "", "text to search (requires embedding provider)")
	epsilon := searchCmd.Float64("epsilon", 0.3, "search radius")
	threshold := searchCmd.Float64("threshold", 0.5, "similarity threshold")
	topK := searchCmd.Int("top-k", 5, "max results")
	radiusWord := searchCmd.String("radius", "", "semantic radius: exact|precise|similar|related|broad|fuzzy (overrides -epsilon)")

	// Embedding provider flags
	useOllama := searchCmd.Bool("ollama", false, "use Ollama for embeddings")
	ollamaURL := searchCmd.String("ollama-url", "http://localhost:11434", "Ollama server URL")
	ollamaModel := searchCmd.String("ollama-model", "nomic-embed-text", "Ollama model")
	useLlamaCpp := searchCmd.Bool("llama-cpp", false, "use llama.cpp server")
	llamaURL := searchCmd.String("llama-url", "http://localhost:8080", "llama.cpp server URL")

	searchCmd.Parse(os.Args[2:])

	c, err := client.New(*dbPath, *dims)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Auto-detect dimensions from existing database
	if *dims == 0 {
		c.SetVerbose(false) // Don't print dimension mismatch warning
		tree, err := c.Storage.Load()
		if err != nil {
			log.Fatalf("Failed to load database: %v", err)
		}
		*dims = tree.Dimensions
		c, err = client.New(*dbPath, *dims)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
	}

	var vector []float32

	// Get query embedding
	if *vectorStr != "" {
		vector, err = parseVectorFlag(*vectorStr)
		if err != nil {
			log.Fatalf("Invalid vector: %v", err)
		}
	} else if *text != "" {
		provider := getEmbeddingProvider(*useOllama, *ollamaURL, *ollamaModel, *useLlamaCpp, *llamaURL)
		if provider == nil {
			log.Fatal("Text search requires -ollama or -llama-cpp flag")
		}
		fmt.Println("Generating query embedding...")
		vector, err = provider.GetEmbedding(*text)
		if err != nil {
			log.Fatalf("Failed to generate embedding: %v", err)
		}
	} else {
		log.Fatal("Either -vector or -text (with embedding provider) is required")
	}

	// Apply semantic radius if specified (overrides epsilon)
	finalEpsilon := float32(*epsilon)
	if *radiusWord != "" {
		finalEpsilon = hippotypes.GetRadiusValue(*radiusWord, finalEpsilon)
		fmt.Printf("Using semantic radius '%s' → epsilon %.2f\n", *radiusWord, finalEpsilon)
	}

	_, err = c.Search(vector, finalEpsilon, float32(*threshold), *topK)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}
}

func handleInsertCSV() {
	csvCmd := flag.NewFlagSet("insert-csv", flag.ExitOnError)
	dbPath := csvCmd.String("db", "tree.bin", "database file")
	dims := csvCmd.Int("dims", 512, "vector dimensions")
	csvFile := csvCmd.String("csv", "", "CSV file path")
	csvCmd.Parse(os.Args[2:])

	if *csvFile == "" {
		log.Fatal("-csv is required")
	}

	c, err := client.New(*dbPath, *dims)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if err := c.InsertCSV(*csvFile); err != nil {
		log.Fatalf("CSV insert failed: %v", err)
	}

	fmt.Println("CSV import complete")
}

func handleInsertJSON() {
	jsonCmd := flag.NewFlagSet("insert-json", flag.ExitOnError)
	dbPath := jsonCmd.String("db", "tree.bin", "database file")
	dims := jsonCmd.Int("dims", 512, "vector dimensions")
	jsonFile := jsonCmd.String("json", "", "JSON file path")
	jsonCmd.Parse(os.Args[2:])

	if *jsonFile == "" {
		log.Fatal("-json is required")
	}

	c, err := client.New(*dbPath, *dims)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if err := c.InsertJSON(*jsonFile); err != nil {
		log.Fatalf("JSON insert failed: %v", err)
	}

	fmt.Println("JSON import complete")
}

func handleInfo() {
	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)
	dbPath := infoCmd.String("db", "tree.bin", "database file")
	infoCmd.Parse(os.Args[2:])

	c, err := client.New(*dbPath, 0) // 0 = auto-detect dimensions
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	count, err := c.GetNodeCount()
	if err != nil {
		log.Fatalf("Failed to get node count: %v", err)
	}

	dims := c.GetDimensions()

	fmt.Println("Database Info:")
	fmt.Printf("  File: %s\n", *dbPath)
	fmt.Printf("  Nodes: %s\n", formatNumber(count))
	fmt.Printf("  Dimensions: %d\n", dims)
	fmt.Printf("  Estimated size: %s\n", formatBytes(estimateSize(count, dims)))
}

func formatNumber(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	// Add thousand separators
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func estimateSize(nodes, dimensions int) int64 {
	// Rough estimate: 4 bytes per float + some overhead
	bytesPerNode := int64(dimensions*4 + 100) // +100 for text and metadata
	return int64(nodes) * bytesPerNode
}
