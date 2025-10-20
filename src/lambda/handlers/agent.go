package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

type AgentCurateRequest struct {
	AgentID       string `json:"agent_id"`
	Text          string `json:"text"`
	ModelID       string `json:"model_id"`
	BedrockRegion string `json:"bedrock_region"`
	Importance    string `json:"importance"`
	Timeout       int    `json:"timeout_ms"`
}

type CurationResult struct {
	Key   string `json:"key"`
	Text  string `json:"text"`
	Reasoning string `json:"reasoning"`
}

func (h *Handler) handleAgentCurate(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var req AgentCurateRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return errorResponse(400, fmt.Sprintf("invalid request body: %v", err))
	}

	if req.AgentID == "" || req.Text == "" {
		return errorResponse(400, "agent_id and text are required")
	}

	if req.ModelID == "" {
		req.ModelID = "us.amazon.nova-lite-v1:0"
	}

	if req.BedrockRegion == "" {
		req.BedrockRegion = "us-east-1"
	}

	if req.Importance == "" {
		req.Importance = "medium"
	}

	if req.Timeout == 0 {
		req.Timeout = 1000
	}

	memories, err := h.curateWithAgent(req)
	if err != nil {
		return errorResponse(500, fmt.Sprintf("curation failed: %v", err))
	}

	return successResponse("agent curation successful", map[string]interface{}{
		"memories_created": len(memories),
		"memories":         memories,
	})
}

func (h *Handler) curateWithAgent(req AgentCurateRequest) ([]CurationResult, error) {
	ctx := context.Background()

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
]`, req.Importance)

	userPrompt := fmt.Sprintf("Analyze and extract memories from:\n\n%s", req.Text)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(req.BedrockRegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	bedrock := bedrockruntime.NewFromConfig(cfg)

	input := &bedrockruntime.ConverseInput{
		ModelId: aws.String(req.ModelID),
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

	for i, result := range results {
		if err := h.storage.Insert(req.AgentID, result.Key, result.Text); err != nil {
			return nil, fmt.Errorf("failed to insert memory %d: %w", i, err)
		}

		if i < len(results)-1 && req.Timeout > 0 {
			time.Sleep(time.Duration(req.Timeout) * time.Millisecond)
		}
	}

	return results, nil
}
