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

type GoogleProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

type googleRequest struct {
	Contents         []googleContent        `json:"contents"`
	SystemInstruction *googleContent        `json:"systemInstruction,omitempty"`
	GenerationConfig googleGenerationConfig `json:"generationConfig,omitempty"`
}

type googleContent struct {
	Parts []googlePart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type googlePart struct {
	Text string `json:"text"`
}

type googleGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type googleResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

func NewGoogleProvider(cfg config.GoogleConfig) *GoogleProvider {
	return &GoogleProvider{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (p *GoogleProvider) Name() string {
	return "google"
}

func (p *GoogleProvider) Complete(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	reqBody := googleRequest{
		Contents: []googleContent{
			{
				Parts: []googlePart{{Text: prompt}},
				Role:  "user",
			},
		},
		GenerationConfig: googleGenerationConfig{
			Temperature:     0.7,
			MaxOutputTokens: 4096,
		},
	}

	if systemPrompt != "" {
		reqBody.SystemInstruction = &googleContent{
			Parts: []googlePart{{Text: systemPrompt}},
		}
	}

	return p.doRequest(ctx, reqBody)
}

func (p *GoogleProvider) CompleteWithJSON(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	return p.Complete(ctx, prompt, systemPrompt+" Respond only with valid JSON.")
}

func (p *GoogleProvider) doRequest(ctx context.Context, reqBody googleRequest) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
		var result googleResponse
		if err := json.Unmarshal(body, &result); err == nil && result.Error != nil {
			return "", fmt.Errorf("Google API error (HTTP %d): %s", resp.StatusCode, result.Error.Message)
		}
		return "", fmt.Errorf("Google API error: HTTP %d - %s", resp.StatusCode, string(body))
	}

	var result googleResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("Google API error: %s", result.Error.Message)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Google")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}
