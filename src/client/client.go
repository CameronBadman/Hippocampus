package client

import (
	"Hippocampus/src/embedding"
	"Hippocampus/src/storage"
	hippotypes "Hippocampus/src/types"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

type Client struct {
	Storage storage.FileStorage
	Region string
	AWS aws.Config
	Bedrock *bedrockruntime.Client

	// In-memory cache
	cachedTree *hippotypes.Tree
	dirty      bool
	verbose    bool
}


func New(binaryPath, region string) (c *Client, err error) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		log.Fatalf("aws config error: %v", err)
		return nil, err
	}

	return &Client{
		Storage: *storage.New(binaryPath),
		Region: region,
		AWS: cfg,
		Bedrock: bedrockruntime.NewFromConfig(cfg),
		cachedTree: nil,
		dirty: false,
		verbose: true, // Can be set to false for benchmarks
	}, nil
}


// getTree returns the in-memory tree, loading from disk if needed
func (client *Client) getTree() (*hippotypes.Tree, error) {
	if client.cachedTree == nil {
		tree, err := client.Storage.Load()
		if err != nil {
			return nil, err
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

func (client *Client) Insert(key, text string) error {
	ctx := context.Background()

	// Time embedding generation
	embedStart := time.Now()
	embeddingSlice, err := embedding.GetEmbedding(ctx, client.Bedrock, text)
	embedDuration := time.Since(embedStart)
	if err != nil {
		return fmt.Errorf("embedding error: %w", err)
	}

	var embeddingArray [512]float32
	copy(embeddingArray[:], embeddingSlice)

	// Time tree loading
	loadStart := time.Now()
	tree, err := client.getTree()
	loadDuration := time.Since(loadStart)
	if err != nil {
		return fmt.Errorf("tree loading error: %w", err)
	}

	// Time pure insert operation
	insertStart := time.Now()
	tree.Insert(embeddingArray, text)
	insertDuration := time.Since(insertStart)
	client.dirty = true

	// Time file flush (if needed)
	var flushDuration time.Duration
	if len(tree.Nodes) % 100 == 0 {
		flushStart := time.Now()
		if err := client.Flush(); err != nil {
			return fmt.Errorf("flush error: %w", err)
		}
		flushDuration = time.Since(flushStart)
	}

	if client.verbose {
		fmt.Printf("Successfully inserted %s (total nodes: %d)\n", key, len(tree.Nodes))
		fmt.Printf("TIMING:EMBED:%.3f:LOAD:%.3f:INSERT:%.3f:FLUSH:%.3f\n",
			embedDuration.Seconds()*1000,
			loadDuration.Seconds()*1000,
			insertDuration.Seconds()*1000,
			flushDuration.Seconds()*1000)
	}
	return nil
}



func (client *Client) Search(text string, epsilon float32, threshold float32, topK int) ([]string, error) {
	ctx := context.Background()

	// Time embedding generation
	embedStart := time.Now()
	embeddingSlice, err := embedding.GetEmbedding(ctx, client.Bedrock, text)
	embedDuration := time.Since(embedStart)
	if err != nil {
		return nil, fmt.Errorf("embedding error: %w", err)
	}

	var embeddingArray [512]float32
	copy(embeddingArray[:], embeddingSlice)

	// Time tree loading
	loadStart := time.Now()
	tree, err := client.getTree()
	loadDuration := time.Since(loadStart)
	if err != nil {
		return nil, fmt.Errorf("tree loading error: %w", err)
	}

	// Time pure search operation
	searchStart := time.Now()
	results := tree.Search(embeddingArray, epsilon, threshold, topK)
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
		fmt.Printf("TIMING:EMBED:%.3f:LOAD:%.6f:SEARCH:%.6f\n",
			embedDuration.Seconds()*1000,
			loadDuration.Seconds()*1000,
			searchDuration.Seconds()*1000)
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

		if err := client.Insert(record[0], record[1]); err != nil {
			return err
		}
	}

	// Flush after bulk insert
	return client.Flush()
}

// CurationResult represents a single memory extracted by the curation agent
type CurationResult struct {
	Key       string `json:"key"`
	Text      string `json:"text"`
	Reasoning string `json:"reasoning"`
}

// AgentCurate uses an internal AI agent to analyze text and extract discrete memories
func (client *Client) AgentCurate(text, importance, modelID, bedrockRegion string, timeoutMs int) ([]CurationResult, error) {
	ctx := context.Background()

	// Set defaults
	if modelID == "" {
		modelID = "us.amazon.nova-lite-v1:0"
	}
	if bedrockRegion == "" {
		bedrockRegion = "us-east-1"
	}
	if importance == "" {
		importance = "medium"
	}
	if timeoutMs == 0 {
		timeoutMs = 50
	}

	systemPrompt := fmt.Sprintf(`You are a memory curation agent. Your task is to analyze text and extract discrete facts as structured memories.

Importance Level: %s
- high: Extract every possible detail, even minor facts
- medium: Extract key facts and important details
- low: Extract only critical information

Guidelines:
- Create separate memories for separate facts
- Use descriptive, searchable keys: category_subcategory_detail
- Each memory should be atomic and self-contained
- Provide brief reasoning for each key choice

Return ONLY valid JSON array, no markdown:
[
  {"key": "category_detail", "text": "the fact", "reasoning": "why this key"},
  ...
]`, importance)

	userPrompt := fmt.Sprintf("Analyze and extract memories from:\n\n%s", text)

	// Load AWS config for the specified Bedrock region
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(bedrockRegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	bedrock := bedrockruntime.NewFromConfig(cfg)

	input := &bedrockruntime.ConverseInput{
		ModelId: aws.String(modelID),
		Messages: []types.Message{
			{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{
						Value: userPrompt,
					},
				},
			},
		},
		System: []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{
				Value: systemPrompt,
			},
		},
	}

	response, err := bedrock.Converse(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("bedrock converse failed: %w", err)
	}

	var responseText string
	for _, block := range response.Output.(*types.ConverseOutputMemberMessage).Value.Content {
		if textBlock, ok := block.(*types.ContentBlockMemberText); ok {
			responseText = textBlock.Value
			break
		}
	}

	var results []CurationResult
	if err := json.Unmarshal([]byte(responseText), &results); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	// Insert each memory with configured delay
	for i, result := range results {
		if err := client.Insert(result.Key, result.Text); err != nil {
			return nil, fmt.Errorf("failed to insert memory %d: %w", i, err)
		}

		// Add delay between insertions (except after the last one)
		if i < len(results)-1 && timeoutMs > 0 {
			time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
		}
	}

	fmt.Printf("\nAgent curation complete: %d memories created\n", len(results))
	for i, result := range results {
		fmt.Printf("  %d. %s: %s\n", i+1, result.Key, result.Text)
		if result.Reasoning != "" {
			fmt.Printf("     → %s\n", result.Reasoning)
		}
	}

	return results, nil
}






















