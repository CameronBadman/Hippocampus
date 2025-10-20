package client

import (
	"Hippocampus/src/embedding"
	"Hippocampus/src/storage"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type Client struct {
	Storage storage.FileStorage
	Region string
	AWS aws.Config
	Bedrock *bedrockruntime.Client
}


func New(binaryPath, region string) (c *Client, err error) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("aws config error: %v", err)
		return nil, err
	}

	return &Client{
		Storage: *storage.New(binaryPath),
		Region: region,
		AWS: cfg,
		Bedrock: bedrockruntime.NewFromConfig(cfg),
	}, nil
}


func (client *Client) Insert(key, text string) error {
	ctx := context.Background()

	embeddingSlice, err := embedding.GetEmbedding(ctx, client.Bedrock, text)
	if err != nil {
		log.Fatalf("embedding error: %v", err)
		return err
	}

	var embeddingArray [512]float32
	copy(embeddingArray[:], embeddingSlice)

	tree, err := client.Storage.Load()
	if err != nil {
		log.Fatalf("tree loading error: %v", err)
		return err
	}
	
	tree.Insert(embeddingArray, text)

	if err := client.Storage.Save(tree); err != nil {
		log.Fatalf("save error: %v", err)
	}

	fmt.Printf("Successfully inserted %s (total nodes: %d)\n", key, len(tree.Nodes))
	return nil
}



func (client *Client) Search(text string, epsilon float32, threshold float32, topK int) ([]string, error) {
	ctx := context.Background()
	embeddingSlice, err := embedding.GetEmbedding(ctx, client.Bedrock, text)
	if err != nil {
		return nil, fmt.Errorf("embedding error: %v", err)
	}
	
	var embeddingArray [512]float32
	copy(embeddingArray[:], embeddingSlice)
	
	tree, err := client.Storage.Load()
	if err != nil {
		return nil, fmt.Errorf("tree loading error: %v", err)
	}
	
	results := tree.Search(embeddingArray, epsilon, threshold, topK)
	
	values := make([]string, len(results))
	for i, node := range results {
		values[i] = node.Value
	}
	
	fmt.Printf("\nFound %d results (top %d, threshold %.2f):\n", len(results), topK, threshold)
	for _, value := range values {
		fmt.Printf("  %s\n", value)
	}
	
	return values, nil
}



func (client *Client) InsertCSV(csvFilename string) error {
	file, err := os.Open(csvFilename)
	if err != nil {
		return fmt.Errorf("Error opening file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF{
				break
			}
			return fmt.Errorf("Error in reading line: %v", err)
		}
	
		client.Insert(record[0], record[1])
	}

	return nil
}






















