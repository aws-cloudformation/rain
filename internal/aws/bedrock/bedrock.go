package bedrock

import (
	"context"
	"encoding/json"
	"fmt"

	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type Request struct {
	Prompt            string   `json:"prompt"`
	MaxTokensToSample int      `json:"max_tokens_to_sample"`
	Temperature       float64  `json:"temperature,omitempty"`
	TopP              float64  `json:"top_p,omitempty"`
	TopK              int      `json:"top_k,omitempty"`
	StopSequences     []string `json:"stop_sequences,omitempty"`
}

type Response struct {
	Completion string `json:"completion"`
}

func getClient() *bedrockruntime.Client {
	return bedrockruntime.NewFromConfig(rainaws.Config())
}

const (
	claudePromptFormat = "\n\nHuman:%s\n\nAssistant:"
	claudeV2ModelID    = "anthropic.claude-v2"
)

func Invoke(p string) (string, error) {
	payload := Request{
		Prompt:            fmt.Sprintf(claudePromptFormat, p),
		MaxTokensToSample: 2048,
		Temperature:       0.5,
		TopK:              250,
		TopP:              1,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling payload: %w", err)
	}

	output, err := getClient().InvokeModel(context.Background(),
		&bedrockruntime.InvokeModelInput{
			Body:        payloadBytes,
			ModelId:     aws.String(claudeV2ModelID),
			ContentType: aws.String("application/json"),
		})

	if err != nil {
		return "", fmt.Errorf("error invoking model: %w", err)
	}

	var resp Response

	err = json.Unmarshal(output.Body, &resp)

	if err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}

	return resp.Completion, nil
}
