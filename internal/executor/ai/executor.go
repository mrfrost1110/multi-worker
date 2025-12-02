package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/multi-worker/internal/model"
)

// Executor handles AI processing in pipelines
type Executor struct {
	registry *ProviderRegistry
}

// NewExecutor creates a new AI executor
func NewExecutor(registry *ProviderRegistry) *Executor {
	return &Executor{registry: registry}
}

func (e *Executor) Type() string {
	return "ai_processor"
}

func (e *Executor) Validate(config map[string]interface{}) error {
	if _, ok := config["prompt"]; !ok {
		return fmt.Errorf("ai_processor requires 'prompt' in config")
	}
	return nil
}

func (e *Executor) Execute(ctx context.Context, input *model.ExecutorResult, config map[string]interface{}) (*model.ExecutorResult, error) {
	// Get provider
	providerName, _ := config["provider"].(string)
	provider, err := e.registry.Get(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI provider: %w", err)
	}

	// Get prompt configuration
	promptTemplate, _ := config["prompt"].(string)
	systemPrompt, _ := config["system_prompt"].(string)

	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant that processes and summarizes information. Be concise and informative."
	}

	// Convert input data to string for processing
	var inputStr string
	if input != nil && input.Data != nil {
		inputBytes, err := json.MarshalIndent(input.Data, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input: %w", err)
		}
		inputStr = string(inputBytes)
	}

	// Build the full prompt
	fullPrompt := promptTemplate
	if inputStr != "" {
		fullPrompt = fmt.Sprintf("%s\n\nData to process:\n%s", promptTemplate, inputStr)
	}

	// Call AI provider
	response, err := provider.Complete(ctx, fullPrompt, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("AI processing failed: %w", err)
	}

	// Try to parse response as JSON, otherwise return as string
	var responseData interface{}
	if err := json.Unmarshal([]byte(response), &responseData); err != nil {
		responseData = response
	}

	// Build metadata with nil-safe input access
	metadata := map[string]interface{}{
		"provider":    provider.Name(),
		"prompt_used": promptTemplate,
	}
	if input != nil {
		metadata["input_items"] = input.ItemCount
	}

	// Calculate item count for result
	itemCount := 1
	if input != nil && input.ItemCount > 0 {
		itemCount = input.ItemCount
	}

	return &model.ExecutorResult{
		Data:      responseData,
		ItemCount: itemCount,
		Metadata:  metadata,
	}, nil
}

// ProcessItems processes a list of items through AI
func (e *Executor) ProcessItems(ctx context.Context, items []model.ScrapedItem, config map[string]interface{}) (string, error) {
	providerName, _ := config["provider"].(string)
	provider, err := e.registry.Get(providerName)
	if err != nil {
		return "", err
	}

	promptTemplate, _ := config["prompt"].(string)
	systemPrompt, _ := config["system_prompt"].(string)

	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant that summarizes job listings and news. Be concise, highlight key information like salary, requirements, and benefits."
	}

	itemsJSON, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return "", err
	}

	fullPrompt := fmt.Sprintf("%s\n\nItems:\n%s", promptTemplate, string(itemsJSON))

	return provider.Complete(ctx, fullPrompt, systemPrompt)
}

// Summarize creates a summary of the input data
func (e *Executor) Summarize(ctx context.Context, data interface{}, providerName string) (string, error) {
	provider, err := e.registry.Get(providerName)
	if err != nil {
		return "", err
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf("Summarize the following data in a clear, concise format suitable for a Discord message:\n\n%s", string(dataJSON))
	systemPrompt := "You are a helpful assistant. Create concise, well-formatted summaries. Use bullet points where appropriate. Keep responses under 2000 characters for Discord compatibility."

	return provider.Complete(ctx, prompt, systemPrompt)
}
