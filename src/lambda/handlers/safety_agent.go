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

type SafetyAgentRequest struct {
	AgentID      string                   `json:"agent_id"`
	Message      string                   `json:"message"`
	Conversation []map[string]interface{} `json:"conversation,omitempty"`
	ModelID      string                   `json:"model_id"`
	BedrockRegion string                  `json:"bedrock_region"`
	Timeout      int                      `json:"timeout_ms"`
}

type SafetyAgentResponse struct {
	Response     string                   `json:"response"`
	Conversation []map[string]interface{} `json:"conversation,omitempty"`
	AgentID      string                   `json:"agent_id"`
	ToolsUsed    []map[string]interface{} `json:"tools_used,omitempty"`
}

// HandleSafetyAgent is the Lambda handler
func (h *Handler) HandleSafetyAgent(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var req SafetyAgentRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return errorResponse(400, fmt.Sprintf("invalid request body: %v", err))
	}

	if req.AgentID == "" || req.Message == "" {
		return errorResponse(400, "agent_id and message are required")
	}

	if req.ModelID == "" {
		req.ModelID = "us.amazon.nova-lite-v1:0"
	}
	if req.BedrockRegion == "" {
		req.BedrockRegion = "us-east-1"
	}
	if req.Timeout == 0 {
		req.Timeout = 1000
	}

	assistantResp, toolsUsed, err := h.processSafetyMessage(req)
	if err != nil {
		return errorResponse(500, fmt.Sprintf("safety agent failed: %v", err))
	}

	resp := SafetyAgentResponse{
		Response:     assistantResp,
		Conversation: req.Conversation,
		AgentID:      req.AgentID,
		ToolsUsed:    toolsUsed,
	}

	respBody, _ := json.Marshal(resp)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(respBody),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

// processSafetyMessage calls Bedrock LLM and integrates Hippocampus memory
func (h *Handler) processSafetyMessage(req SafetyAgentRequest) (string, []map[string]interface{}, error) {
	ctx := context.Background()

	systemPrompt := `You are a safety-critical assistant. Always check stored memories before giving advice, especially:
- Allergies or dietary restrictions
- Health/medical safety
- Children's safety

Use high-threshold searches for safety-critical queries.`

	userPrompt := fmt.Sprintf("User says: %s\nCheck memory and respond safely.", req.Message)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(req.BedrockRegion))
	if err != nil {
		return "", nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	bedrock := bedrockruntime.NewFromConfig(cfg)

	input := &bedrockruntime.ConverseInput{
		ModelId: aws.String(req.ModelID),
		Messages: []types.Message{
			{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: userPrompt},
				},
			},
		},
		System: []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{Value: systemPrompt},
		},
	}

	response, err := bedrock.Converse(ctx, input)
	if err != nil {
		return "", nil, fmt.Errorf("bedrock converse failed: %w", err)
	}

	// extract response text
	var assistantText string
	for _, block := range response.Output.(*types.ConverseOutputMemberMessage).Value.Content {
		if textBlock, ok := block.(*types.ContentBlockMemberText); ok {
			assistantText = textBlock.Value
			break
		}
	}

	// Simulate memory tool calls
	toolsUsed := []map[string]interface{}{}

	// Example: search memory first
	searchResult, _ := h.searchMemory(req.AgentID, req.Message)
	toolsUsed = append(toolsUsed, map[string]interface{}{"tool": "search_memory", "result": searchResult})

	// Example: insert message into memory
	insertResult, _ := h.insertMemory(req.AgentID, "last_message", req.Message)
	toolsUsed = append(toolsUsed, map[string]interface{}{"tool": "insert_memory", "result": insertResult})

	if req.Timeout > 0 {
		time.Sleep(time.Duration(req.Timeout) * time.Millisecond)
	}

	return assistantText, toolsUsed, nil
}

// Dummy methods — replace with Hippocampus integration
func (h *Handler) searchMemory(agentID, query string) (map[string]interface{}, error) {
	// call Hippocampus /search here
	return map[string]interface{}{"found": true, "memories": []string{"example memory"}}, nil
}

func (h *Handler) insertMemory(agentID, key, text string) (map[string]interface{}, error) {
	// call Hippocampus /insert here
	return map[string]interface{}{"success": true}, nil
}
