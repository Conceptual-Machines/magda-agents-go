package llm

import (
	"context"
	"fmt"
	"strings"
)

// ProviderFactory creates providers based on model name or explicit provider choice
type ProviderFactory struct {
	openaiAPIKey string
	geminiAPIKey string
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(openaiAPIKey, geminiAPIKey string) *ProviderFactory {
	return &ProviderFactory{
		openaiAPIKey: openaiAPIKey,
		geminiAPIKey: geminiAPIKey,
	}
}

// GetProvider returns the appropriate provider for the given model/provider name
func (f *ProviderFactory) GetProvider(ctx context.Context, model, providerName string) (Provider, error) {
	// If provider is explicitly specified, use that
	if providerName != "" {
		return f.getProviderByName(ctx, providerName)
	}

	// Otherwise, infer from model name
	return f.getProviderByModel(ctx, model)
}

// getProviderByName creates a provider by explicit name
func (f *ProviderFactory) getProviderByName(ctx context.Context, providerName string) (Provider, error) {
	switch strings.ToLower(providerName) {
	case "openai":
		if f.openaiAPIKey == "" {
			return nil, fmt.Errorf("openai API key not configured")
		}
		return NewOpenAIProvider(f.openaiAPIKey), nil

	case "gemini":
		if f.geminiAPIKey == "" {
			return nil, fmt.Errorf("gemini API key not configured")
		}
		return NewGeminiProvider(ctx, f.geminiAPIKey)

	default:
		return nil, fmt.Errorf("unknown provider: %s (allowed: openai, gemini)", providerName)
	}
}

// getProviderByModel infers provider from model name
func (f *ProviderFactory) getProviderByModel(ctx context.Context, model string) (Provider, error) {
	modelLower := strings.ToLower(model)

	// GPT models use OpenAI
	if strings.HasPrefix(modelLower, "gpt-") {
		if f.openaiAPIKey == "" {
			return nil, fmt.Errorf("openai API key not configured")
		}
		return NewOpenAIProvider(f.openaiAPIKey), nil
	}

	// Gemini models use Gemini
	if strings.HasPrefix(modelLower, "gemini-") {
		if f.geminiAPIKey == "" {
			return nil, fmt.Errorf("gemini API key not configured")
		}
		return NewGeminiProvider(ctx, f.geminiAPIKey)
	}

	// Default to OpenAI for unknown models
	if f.openaiAPIKey == "" {
		return nil, fmt.Errorf("openai API key not configured (default provider)")
	}
	return NewOpenAIProvider(f.openaiAPIKey), nil
}
