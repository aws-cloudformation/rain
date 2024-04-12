package bedrock

import (
	"context"
	"encoding/json"
	"fmt"

	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type Source struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type Content struct {
	Type    string    `json:"type"`
	Sources []*Source `json:"source,omitempty"`
	Text    string    `json:"text,omitempty"`
}

type Message struct {
	Role     string     `json:"role"`
	Contents []*Content `json:"content"`
}

type MessageUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type MessageResponse struct {
	Id           string        `json:"id"`
	Type         string        `json:"type"`
	Role         string        `json:"role"`
	Contents     []*Content    `json:"content"`
	Model        string        `json:"model"`
	StopReason   string        `json:"stop_reason"`
	StopSequence string        `json:"stop_sequence"`
	Usage        *MessageUsage `json:"usage"`
}

type Request struct {
	Prompt            string     `json:"prompt,omitempty"`
	System            string     `json:"system,omitempty"`
	Messages          []*Message `json:"messages,omitempty"`
	MaxTokensToSample int        `json:"max_tokens_to_sample,omitempty"`
	MaxTokens         int        `json:"max_tokens,omitempty"`
	Temperature       float64    `json:"temperature,omitempty"`
	TopP              float64    `json:"top_p,omitempty"`
	TopK              int        `json:"top_k,omitempty"`
	StopSequences     []string   `json:"stop_sequences,omitempty"`
	AnthropicVersion  string     `json:"anthropic_version,omitempty"`
}

type Response struct {
	Completion string `json:"completion"`
}

func getClient() *bedrockruntime.Client {
	return bedrockruntime.NewFromConfig(rainaws.Config())
}

const (
	claudePromptFormat = "\n\nHuman:%s\n\nAssistant:"
	claudeV2ModelID    = "anthropic.claude-v2:1"
)

// Invoke invokes the Claude V2 model with the provided prompt.
func Invoke(p string) (string, error) {

	// Create the Claude 2 payload
	payload := Request{
		Prompt:            fmt.Sprintf(claudePromptFormat, p),
		MaxTokensToSample: 2048,
		Temperature:       0.5,
		TopK:              250,
		TopP:              1,
	}

	// Convert the request to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling payload: %w", err)
	}

	// Make the SDK call to the API
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

// InvokeClaude3 invokes the Claude V3 model with the provided prompt.
func InvokeClaude3(p string, model string, system string) (string, error) {

	maxTokens := 2048
	anthropicVersion := "bedrock-2023-05-31"
	contents := make([]*Content, 0)
	contents = append(contents, &Content{
		Type: "text",
		Text: p,
	})

	message := Message{
		Role:     "user",
		Contents: contents,
	}

	messages := []*Message{&message}

	// Create the Claude3 payload
	payload := Request{
		System:           system,
		Messages:         messages,
		MaxTokens:        maxTokens,
		AnthropicVersion: anthropicVersion,
		Temperature:      0.5,
		TopK:             250,
		TopP:             1,
	}

	// Convert the request to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling payload: %w", err)
	}

	config.Debugf("About to invoke bedrock with Body: %s", string(payloadBytes))

	// Make the SDK call to the API
	output, err := getClient().InvokeModel(context.Background(),
		&bedrockruntime.InvokeModelInput{
			Body:        payloadBytes,
			ModelId:     aws.String(model),
			ContentType: aws.String("application/json"),
		})

	if err != nil {
		return "", fmt.Errorf("error invoking model: %w", err)
	}

	config.Debugf("Got output from bedrock: %+v", output)

	var resp MessageResponse

	err = json.Unmarshal(output.Body, &resp)

	config.Debugf("Got response from bedrock: %s\nJSON:\n %+v", string(output.Body), resp)

	if err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}

	var t string
	if len(resp.Contents) > 0 {
		t = resp.Contents[0].Text
	}

	return t, nil
}
