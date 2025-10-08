package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"encoding/csv"
	"io"

	"Hippocampus/embedding"
	"Hippocampus/storage"
	"Hippocampus/types"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

func main() {
	insertCmd := flag.NewFlagSet("insert", flag.ExitOnError)
	insertFile := insertCmd.String("file", "tree.bin", "database file")
	insertKey := insertCmd.String("key", "", "key/identifier for the text")
	insertText := insertCmd.String("text", "", "text to embed and store")
	
	csvCmd := flag.NewFlagSet("insert-csv", flag.ExitOnError)
	csvFile := csvCmd.String("csv", "csvFile.csv", "csv file")
	csvBinary := csvCmd.String("file", "tree.bin", "database file")

	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	searchFile := searchCmd.String("file", "tree.bin", "database file")
	searchText := searchCmd.String("text", "", "text to search for")
	searchEpsilon := searchCmd.Float64("epsilon", 0.1, "search tolerance")

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  hippocampus insert -file tree.bin -key <id> -text <text>")
		fmt.Println("  hippocampus search -file tree.bin -text <text> -epsilon 0.1")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "insert":
		insertCmd.Parse(os.Args[2:])
		if *insertKey == "" || *insertText == "" {
			log.Fatal("both -key and -text are required")
		}
		handleInsert(*insertFile, *insertKey, *insertText)

	case "search":
		searchCmd.Parse(os.Args[2:])
		if *searchText == "" {
			log.Fatal("-text is required")
		}
		handleSearch(*searchFile, *searchText, float32(*searchEpsilon))
	
	case "insert-csv":
		csvCmd.Parse(os.Args[2:])
		if *csvFile == "" {
			log.Fatalf("-file is required")
		}
		parseCSV(*csvFile, *csvBinary)

	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}

func handleInsert(filename, key, text string) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("aws config error: %v", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)

	fmt.Println("Getting embedding from Titan...")
	embeddingSlice, err := embedding.GetEmbedding(ctx, client, text)
	if err != nil {
		log.Fatalf("embedding error: %v", err)
	}

	var embeddingArray [512]float32
	copy(embeddingArray[:], embeddingSlice)

	fs := storage.New(filename)
	tree, err := fs.Load()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Creating new database...")
			tree = types.NewTree()
		} else {
			log.Fatalf("load error: %v", err)
		}
	}

	fmt.Printf("Inserting key=%s into tree...\n", key)
	tree.Insert(embeddingArray, key)

	fmt.Println("Saving tree...")
	if err := fs.Save(tree); err != nil {
		log.Fatalf("save error: %v", err)
	}

	fmt.Printf("Successfully inserted %s (total nodes: %d)\n", key, len(tree.Nodes))
}

func handleSearch(filename, text string, epsilon float32) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("aws config error: %v", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)

	fmt.Println("Getting embedding from Titan...")
	embeddingSlice, err := embedding.GetEmbedding(ctx, client, text)
	if err != nil {
		log.Fatalf("embedding error: %v", err)
	}

	var embeddingArray [512]float32
	copy(embeddingArray[:], embeddingSlice)

	fs := storage.New(filename)
	tree, err := fs.Load()
	if err != nil {
		log.Fatalf("load error: %v", err)
	}

	fmt.Printf("Searching with epsilon=%.3f...\n", epsilon)
	results := tree.Search(embeddingArray, epsilon)

	fmt.Printf("\nFound %d results:\n", len(results))
	for _, node := range results {
		fmt.Printf("  %s\n", node.Value)
	}
}

func parseCSV(csvFilename, binaryFilename string){
	file, err := os.Open(csvFilename)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	for {
		record, err := reader.Read()
		
		if err != nil {
			if err == io.EOF{
				break
			}
			log.Fatalf("Error in reading line: %v", err)
		}
		
		fmt.Printf("Record: %v\n", record)
		go handleInsert(binaryFilename, record[0], record[1])
	}
}
