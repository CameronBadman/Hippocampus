package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"Hippocampus/src/lambda/storage"

	"github.com/aws/aws-lambda-go/events"
)

type Handler struct {
	storage *storage.Manager
}

func New(storageManager *storage.Manager, _ interface{}) *Handler {
	return &Handler{
		storage: storageManager,
	}
}

func (h *Handler) Route(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod != "POST" {
		return errorResponse(400, "only POST method is supported")
	}

	switch request.Path {
	case "/insert":
		return h.handleInsert(request)
	case "/search":
		return h.handleSearch(request)
	case "/insert-csv":
		return h.handleInsertCSV(request)
	default:
		return errorResponse(404, "unknown endpoint")
	}
}

func (h *Handler) handleInsert(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var req InsertRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return errorResponse(400, fmt.Sprintf("invalid request body: %v", err))
	}

	if req.AgentID == "" || req.Key == "" || req.Text == "" {
		return errorResponse(400, "agent_id, key, and text are required")
	}

	if err := h.storage.Insert(req.AgentID, req.Key, req.Text); err != nil {
		return errorResponse(500, fmt.Sprintf("insert failed: %v", err))
	}

	return successResponse("insert successful", nil)
}

func (h *Handler) handleSearch(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var req SearchRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return errorResponse(400, fmt.Sprintf("invalid request body: %v", err))
	}
	
	if req.AgentID == "" || req.Text == "" {
		return errorResponse(400, "agent_id and text are required")
	}
	
	if req.Epsilon == 0 {
		req.Epsilon = 0.3
	}
	if req.Threshold == 0 {
		req.Threshold = 0.5
	}
	if req.TopK == 0 {
		req.TopK = 5
	}
	
	results, err := h.storage.Search(req.AgentID, req.Text, req.Epsilon, req.Threshold, req.TopK)
	if err != nil {
		return errorResponse(500, fmt.Sprintf("search failed: %v", err))
	}
	
	return successResponse("search successful", results)
}

func (h *Handler) handleInsertCSV(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var req InsertCSVRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return errorResponse(400, fmt.Sprintf("invalid request body: %v", err))
	}

	if req.AgentID == "" || req.CSVFile == "" {
		return errorResponse(400, "agent_id and csv_file are required")
	}

	if err := h.storage.InsertCSV(req.AgentID, req.CSVFile); err != nil {
		return errorResponse(500, fmt.Sprintf("insert-csv failed: %v", err))
	}

	return successResponse("csv insert successful", nil)
}

func successResponse(message string, data interface{}) (events.APIGatewayProxyResponse, error) {
	resp := Response{
		Message: message,
		Data:    data,
	}
	body, _ := json.Marshal(resp)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

func errorResponse(statusCode int, errMsg string) (events.APIGatewayProxyResponse, error) {
	resp := Response{
		Error: errMsg,
	}
	body, _ := json.Marshal(resp)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       string(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}
