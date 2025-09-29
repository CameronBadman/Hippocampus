package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type TitanEmbeddingRequest struct {
    InputText  string `json:"inputText"`
    Dimensions int    `json:"dimensions,omitempty"`
    Normalize  bool   `json:"normalize,omitempty"`
}

type TitanEmbeddingResponse struct {
    Embedding           []float32 `json:"embedding"`
    InputTextTokenCount int       `json:"inputTextTokenCount"`
}

func getEmbedding(ctx context.Context, client *bedrockruntime.Client, text string) ([]float32, error) {
    payload := TitanEmbeddingRequest{
        InputText:  text,
        Dimensions: 1024,
        Normalize:  true,
    }
    
    body, err := json.Marshal(payload)
    if err != nil {
        return nil, err
    }
    
    output, err := client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
        ModelId:     aws.String("amazon.titan-embed-text-v2:0"),
        ContentType: aws.String("application/json"),
        Body:        body,
    })
    if err != nil {
        return nil, err
    }
    
    var response TitanEmbeddingResponse
    err = json.Unmarshal(output.Body, &response)
    return response.Embedding, err
}
