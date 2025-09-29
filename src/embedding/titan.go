package embedding

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)


type TitanRequest struct {
	InputText  string `json:"inputText"`
	Dimensions int    `json:"dimensions,omitempty"`
	Normalize  bool   `json:"normalize,omitempty"`
}

// notice the json tags, they allow us to easily marshal json into structs to work with
type TitanResponse struct {
	Embedding           []float32 `json:"embedding"`
	InputTextTokenCount int       `json:"inputTextTokenCount"`
}

func GetEmbedding(ctx context.Context, client *bedrockruntime.Client, text string) ([]float32, error) {
	payload := TitanRequest{
		InputText:  text,
		Dimensions: 512,
		Normalize:  true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	output, err := client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String("amazon.titan-embed-text-v2:0"),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return nil, fmt.Errorf("invoke error: %w", err)
	}
	// this is the important part!!! ISSAC and VAL, we can take type structs with meta data and use the struct
	var response TitanResponse
	if err := json.Unmarshal(output.Body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	fmt.Printf("Token count: %d\n", response.InputTextTokenCount)
	fmt.Printf("Embedding dimensions: %d\n", len(response.Embedding))

	return response.Embedding, nil
}
