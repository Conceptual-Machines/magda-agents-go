package services

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/Conceptual-Machines/magda-agents-go/metrics"
	"github.com/Conceptual-Machines/magda-agents-go/models"
	"github.com/Conceptual-Machines/magda-agents-go/prompt"
	"github.com/getsentry/sentry-go"
	"github.com/openai/openai-go/responses"
)

const (
	mcpCallType             = "mcp_call"
	arrayType               = "array"
	maxOutputTruncateLength = 200
)

type GenerationService struct {
	provider      llm.Provider
	mcpURL        string
	mcpLabel      string
	systemPrompt  string
	promptBuilder *prompt.Builder
	metrics       *metrics.SentryMetrics
}

// MetricsRecorder interface for recording metrics
type MetricsRecorder interface {
	RecordTokenUsage(ctx context.Context, model string, totalTokens, inputTokens, outputTokens, reasoningTokens int)
	RecordMCPUsage(used bool, callCount int)
	RecordGenerationDuration(ctx context.Context, duration time.Duration, success bool)
}

func NewGenerationService(cfg *config.Config) *GenerationService {
	return NewGenerationServiceWithProvider(cfg, nil)
}

// NewGenerationServiceWithProvider creates a service with a specific provider
// If provider is nil, OpenAI is used as default
func NewGenerationServiceWithProvider(cfg *config.Config, provider llm.Provider) *GenerationService {
	promptBuilder := prompt.NewPromptBuilder()
	systemPrompt, err := promptBuilder.BuildPrompt()
	if err != nil {
		log.Fatal("Failed to load system prompt:", err)
	}

	// Use provided provider or create OpenAI provider (default)
	if provider == nil {
		provider = llm.NewOpenAIProvider(cfg.OpenAIAPIKey)
	}

	var mcpLabel string
	var mcpURL string

	if cfg.MCPServerURL != "" && strings.TrimSpace(cfg.MCPServerURL) != "" {
		mcpURL = cfg.MCPServerURL
		if parsed, err := url.Parse(cfg.MCPServerURL); err == nil {
			host := strings.TrimSpace(parsed.Host)
			if host != "" {
				// Convert host to valid MCP label format (letters, digits, dashes, underscores only)
				mcpLabel = strings.ReplaceAll(host, ".", "-")
				mcpLabel = strings.ReplaceAll(mcpLabel, ":", "_")
				// Ensure it starts with a letter
				if len(mcpLabel) > 0 && !unicode.IsLetter(rune(mcpLabel[0])) {
					mcpLabel = "mcp-" + mcpLabel
				}
			}
		}
		if mcpLabel == "" {
			mcpLabel = "mcp-server"
		}
	}

	service := &GenerationService{
		provider:      provider,
		mcpURL:        mcpURL,
		mcpLabel:      mcpLabel,
		systemPrompt:  systemPrompt,
		promptBuilder: promptBuilder,
		metrics:       metrics.NewSentryMetrics(),
	}

	// Log MCP configuration
	log.Printf("🎵 GENERATION SERVICE INITIALIZED:")
	log.Printf("   Provider: %s", provider.Name())
	if mcpURL != "" {
		log.Printf("   MCP URL: %s", mcpURL)
		log.Printf("   MCP Label: %s", mcpLabel)
		log.Printf("   MCP Status: ✅ ENABLED")
	} else {
		log.Printf("   MCP Status: ❌ DISABLED (no MCP_SERVER_URL)")
	}

	return service
}

type GenerationResult struct {
	OutputParsed struct {
		Choices []models.MusicalChoice `json:"choices"`
	} `json:"output_parsed"`
	Usage    any      `json:"usage"`
	MCPUsed  bool     `json:"mcpUsed,omitempty"`
	MCPCalls int      `json:"mcpCalls,omitempty"`
	MCPTools []string `json:"mcpTools,omitempty"`
}

func (s *GenerationService) Generate(
	ctx context.Context, model string, inputArray []map[string]any, reasoningMode string,
) (*GenerationResult, error) {
	startTime := time.Now()
	log.Printf("🎵 GENERATION REQUEST STARTED (Model: %s)", model)

	// Start Sentry transaction for performance monitoring
	transaction := sentry.StartTransaction(ctx, "generation.generate")
	defer transaction.Finish()

	// Add tags for better dashboard filtering
	transaction.SetTag("model", model)
	transaction.SetTag("mcp_enabled", fmt.Sprintf("%t", s.mcpURL != ""))
	transaction.SetTag("input_count", fmt.Sprintf("%d", len(inputArray)))

	// Set transaction context
	transaction.SetContext("generation", map[string]interface{}{
		"model":       model,
		"mcp_url":     s.mcpURL,
		"mcp_label":   s.mcpLabel,
		"input_count": len(inputArray),
	})

	// Build provider request
	request := &llm.GenerationRequest{
		Model:         model,
		InputArray:    inputArray,
		ReasoningMode: reasoningMode,
		SystemPrompt:  s.systemPrompt,
		OutputSchema: &llm.OutputSchema{
			Name:        "MusicalOutput",
			Description: "Musical composition with multiple choices",
			Schema:      llm.GetMusicalOutputSchema(),
		},
	}

	// Add MCP config if enabled
	if s.mcpURL != "" {
		request.MCPConfig = &llm.MCPConfig{
			URL:   s.mcpURL,
			Label: s.mcpLabel,
		}
	}

	// Call provider
	log.Printf("🚀 PROVIDER REQUEST: %s model=%s, mcp_enabled=%t, input_messages=%d",
		s.provider.Name(), model, s.mcpURL != "", len(inputArray))

	resp, err := s.provider.Generate(ctx, request)
	if err != nil {
		transaction.SetTag("success", "false")
		transaction.SetTag("error_type", "provider_error")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("provider request failed: %w", err)
	}

	// Convert to GenerationResult
	result := &GenerationResult{
		Usage:    resp.Usage,
		MCPUsed:  resp.MCPUsed,
		MCPCalls: resp.MCPCalls,
		MCPTools: resp.MCPTools,
	}
	result.OutputParsed.Choices = resp.OutputParsed.Choices

	// Mark transaction as successful
	transaction.SetTag("success", "true")
	transaction.SetTag("mcp_used", fmt.Sprintf("%t", result.MCPUsed))
	transaction.SetTag("mcp_calls", fmt.Sprintf("%d", result.MCPCalls))

	// Record metrics
	duration := time.Since(startTime)
	s.metrics.RecordGenerationDuration(ctx, duration, true)
	s.metrics.RecordMCPUsage(result.MCPUsed, result.MCPCalls)

	// Record token usage if available
	if result.Usage != nil {
		// Type assert to get the actual usage structure
		if usage, ok := result.Usage.(responses.ResponseUsage); ok {
			reasoningTokens := int(usage.OutputTokensDetails.ReasoningTokens)
			fmt.Printf("DEBUG: Token usage - Total: %d, Input: %d, Output: %d, Reasoning: %d\n",
				int(usage.TotalTokens), int(usage.InputTokens), int(usage.OutputTokens), reasoningTokens)
			s.metrics.RecordTokenUsage(ctx, model,
				int(usage.TotalTokens),
				int(usage.InputTokens),
				int(usage.OutputTokens),
				reasoningTokens)
		}
	}

	return result, nil
}
