package main

import (
	"context"
	"fmt"
	"log"
	"main/embedding"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)



func main1() {
	text := os.Args[1]

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)

	embedding, err := embedding.GetEmbedding(ctx, client, text)
	if err != nil {
		log.Fatalf("Failed to get embedding: %v", err)
	}

	fmt.Printf("\nEmbedding\n", embedding)
}
