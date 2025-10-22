package main

import (
	"Hippocampus/src/client"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Hippocampus CLI - AI Agent Memory Database")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  hippocampus insert -binary tree.bin -key <id> -text <text>")
		fmt.Println("  hippocampus search -binary tree.bin -text <text> -epsilon 0.3 -threshold 0.5 -top-k 5")
		fmt.Println("  hippocampus insert-csv -binary tree.bin -csv <file.csv>")
		fmt.Println("  hippocampus agent-curate -binary tree.bin -text <text> -importance high")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  insert        Store a single memory with a key")
		fmt.Println("  search        Search for similar memories")
		fmt.Println("  insert-csv    Bulk insert from CSV file")
		fmt.Println("  agent-curate  Use AI agent to decompose text into discrete memories")
		fmt.Println()
		fmt.Println("Global Flags:")
		fmt.Println("  -binary       Database file path (default: tree.bin)")
		fmt.Println("  -region       AWS region (default: us-east-1)")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "insert":
		insertCmd := flag.NewFlagSet("insert", flag.ExitOnError)
		binary := insertCmd.String("binary", "tree.bin", "database file")
		region := insertCmd.String("region", "us-east-1", "AWS region")
		key := insertCmd.String("key", "", "key/identifier for the text")
		text := insertCmd.String("text", "", "text to embed and store")
		insertCmd.Parse(os.Args[2:])

		if *key == "" || *text == "" {
			log.Fatal("both -key and -text are required")
		}

		client, err := client.New(*binary, *region)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}

		if err := client.Insert(*key, *text); err != nil {
			log.Fatalf("Insert failed: %v", err)
		}

	case "search":
		searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
		binary := searchCmd.String("binary", "tree.bin", "database file")
		region := searchCmd.String("region", "us-east-1", "AWS region")
		text := searchCmd.String("text", "", "text to search for")
		epsilon := searchCmd.Float64("epsilon", 0.3, "search radius (per-dimension bounding box)")
		threshold := searchCmd.Float64("threshold", 0.5, "similarity threshold (0.0-1.0, higher = stricter)")
		topK := searchCmd.Int("top-k", 5, "maximum number of results to return")
		searchCmd.Parse(os.Args[2:])

		if *text == "" {
			log.Fatal("-text is required")
		}

		client, err := client.New(*binary, *region)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}

		_, err = client.Search(*text, float32(*epsilon), float32(*threshold), *topK)
		if err != nil {
			log.Fatalf("Search failed: %v", err)
		}

	case "insert-csv":
		csvCmd := flag.NewFlagSet("insert-csv", flag.ExitOnError)
		binary := csvCmd.String("binary", "tree.bin", "database file")
		region := csvCmd.String("region", "us-east-1", "AWS region")
		csvFile := csvCmd.String("csv", "", "csv file path")
		csvCmd.Parse(os.Args[2:])

		if *csvFile == "" {
			log.Fatalf("-csv is required")
		}

		client, err := client.New(*binary, *region)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}

		if err := client.InsertCSV(*csvFile); err != nil {
			log.Fatalf("CSV insert failed: %v", err)
		}

	case "agent-curate":
		curateCmd := flag.NewFlagSet("agent-curate", flag.ExitOnError)
		binary := curateCmd.String("binary", "tree.bin", "database file")
		region := curateCmd.String("region", "us-east-1", "AWS region")
		text := curateCmd.String("text", "", "text to analyze and decompose into memories")
		importance := curateCmd.String("importance", "medium", "extraction level: high, medium, or low")
		modelID := curateCmd.String("model", "us.amazon.nova-lite-v1:0", "Bedrock model ID for curation")
		bedrockRegion := curateCmd.String("bedrock-region", "us-east-1", "AWS region for Bedrock curation agent")
		timeout := curateCmd.Int("timeout-ms", 50, "milliseconds between memory insertions")
		curateCmd.Parse(os.Args[2:])

		if *text == "" {
			log.Fatal("-text is required")
		}

		client, err := client.New(*binary, *region)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}

		_, err = client.AgentCurate(*text, *importance, *modelID, *bedrockRegion, *timeout)
		if err != nil {
			log.Fatalf("Agent curation failed: %v", err)
		}

	default:
		log.Fatalf("unknown command: %s\nRun 'hippocampus' with no arguments for usage", command)
	}
}
