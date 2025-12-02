package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/multi-worker/internal/config"
)

// DeepSeekProvider uses OpenAI-compatible API
type DeepSeekProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewDeepSeekProvider(cfg config.DeepSeekConfig) *DeepSeekProvider {
	return &DeepSeekProvider{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (p *DeepSeekProvider) Name() string {
	return "deepseek"
}

func (p *DeepSeekProvider) Complete(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	messages := []openAIMessage{}

	if systemPrompt != "" {
		messages = append(messages, openAIMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	messages = append(messages, openAIMessage{
		Role:    "user",
		Content: prompt,
	})

	reqBody := openAIRequest{
		Model:       p.model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	return p.doRequest(ctx, reqBody)
}

func (p *DeepSeekProvider) CompleteWithJSON(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	return p.Complete(ctx, prompt, systemPrompt+" Respond only with valid JSON.")
}

func (p *DeepSeekProvider) doRequest(ctx context.Context, reqBody openAIRequest) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		var result openAIResponse
		if err := json.Unmarshal(body, &result); err == nil && result.Error != nil {
			return "", fmt.Errorf("DeepSeek API error (HTTP %d): %s", resp.StatusCode, result.Error.Message)
		}
		return "", fmt.Errorf("DeepSeek API error: HTTP %d - %s", resp.StatusCode, string(body))
	}

	var result openAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("DeepSeek API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from DeepSeek")
	}

	return result.Choices[0].Message.Content, nil
}
