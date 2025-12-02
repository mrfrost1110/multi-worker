package ai

import (
	"context"
	"fmt"

	"github.com/multi-worker/internal/config"
)

// Provider represents an AI provider interface
type Provider interface {
	Name() string
	Complete(ctx context.Context, prompt string, systemPrompt string) (string, error)
	CompleteWithJSON(ctx context.Context, prompt string, systemPrompt string) (string, error)
}

// ProviderRegistry manages all AI providers
type ProviderRegistry struct {
	providers       map[string]Provider
	defaultProvider string
}

// NewProviderRegistry creates a new provider registry with all configured providers
func NewProviderRegistry(cfg *config.AIConfig) *ProviderRegistry {
	registry := &ProviderRegistry{
		providers:       make(map[string]Provider),
		defaultProvider: cfg.DefaultProvider,
	}

	// Register OpenAI
	if cfg.OpenAI.APIKey != "" {
		registry.providers["openai"] = NewOpenAIProvider(cfg.OpenAI)
	}

	// Register Anthropic
	if cfg.Anthropic.APIKey != "" {
		registry.providers["anthropic"] = NewAnthropicProvider(cfg.Anthropic)
	}

	// Register Google
	if cfg.Google.APIKey != "" {
		registry.providers["google"] = NewGoogleProvider(cfg.Google)
	}

	// Register OpenRouter
	if cfg.OpenRouter.APIKey != "" {
		registry.providers["openrouter"] = NewOpenRouterProvider(cfg.OpenRouter)
	}

	// Register DeepSeek
	if cfg.DeepSeek.APIKey != "" {
		registry.providers["deepseek"] = NewDeepSeekProvider(cfg.DeepSeek)
	}

	return registry
}

// Get returns a provider by name
func (r *ProviderRegistry) Get(name string) (Provider, error) {
	if name == "" {
		name = r.defaultProvider
	}

	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("AI provider '%s' not found or not configured", name)
	}

	return provider, nil
}

// GetDefault returns the default provider
func (r *ProviderRegistry) GetDefault() (Provider, error) {
	return r.Get(r.defaultProvider)
}

// Available returns list of available provider names
func (r *ProviderRegistry) Available() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// HasProvider checks if a provider is available
func (r *ProviderRegistry) HasProvider(name string) bool {
	_, ok := r.providers[name]
	return ok
}
