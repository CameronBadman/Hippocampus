package main

import (
	"context"
	"fmt"
	"log"
	"main/embedding"
	"main/memory"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

func main(){
	if len(os.Args) < 3 {
		fmt.Println("Usage: ")
		fmt.Println("  go run main.go store \"Some text to remember\"")
        fmt.Println("  go run main.go search \"query text\"")
		return
	}

	mode := os.Args[1]
	text := os.Args[2]

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("ap-southeast-2"))
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)
	vector, err := embedding.GetEmbedding(ctx, client, text)
	if err != nil {
		log.Fatalf("Failed to get embedding: %v", err)
	}

	switch mode {
	case "store":
		m := memory.Memory{
			ID: fmt.Sprintf("mem-%d", time.Now().UnixNano()),
			Text: text,
			Embedding: vector,
		}

		if err := memory.SaveMemory(m); err != nil {
			log.Fatalf("Failed to save memory: %v", err)
		}
		fmt.Println("Memory stored successfully")

	case "search":
		memories, err := memory.LoadAllMemories()
		if err != nil {
			log.Fatalf("Failed to load memories: %v", err)
		}

		if len(memories) == 0{
			fmt.Println("No memories stored yet")
			return
		}

		results := memory.SemanticSearch(memories, vector, 3)
		fmt.Println("Top matches: ")
		for i, r := range results{
			fmt.Printf("%d. %s (similarity=%.4f)\n", i+1, r.Text, r.Similarity)
		}

	default:
		fmt.Println("Unknown mode: ", mode);
	}


}