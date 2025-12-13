package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/getsentry/sentry-go"
)

// StreamEvent represents a server-sent event for streaming generation
type StreamEvent struct {
	Type    string `json:"type"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// StreamCallback is called for each event during streaming
type StreamCallback func(event StreamEvent) error

// GenerateStream generates music with streaming updates
// TODO: Implement proper streaming using provider.GenerateStream
// For now, this uses non-streaming Generate and simulates basic events
func (s *GenerationService) GenerateStream(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
	reasoningMode string,
	callback StreamCallback,
) (*GenerationResult, error) {
	startTime := time.Now()

	// Use existing transaction from HTTP middleware
	transaction := sentry.TransactionFromContext(ctx)
	if transaction == nil {
		transaction = sentry.StartTransaction(ctx, "generation.generate_stream")
		defer transaction.Finish()
	}

	// Send initial events
	if err := callback(StreamEvent{Type: "start", Message: "Starting generation..."}); err != nil {
		return nil, err
	}

	if s.mcpURL != "" {
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "mcp",
			Message:  "MCP server enabled for streaming",
			Level:    sentry.LevelInfo,
			Data: map[string]interface{}{
				"mcp_url":   s.mcpURL,
				"mcp_label": s.mcpLabel,
			},
		})

		if err := callback(StreamEvent{
			Type:    "mcp_enabled",
			Message: fmt.Sprintf("MCP server: %s", s.mcpURL),
		}); err != nil {
			return nil, err
		}
	}

	if err := callback(StreamEvent{Type: "processing", Message: "Generating..."}); err != nil {
		return nil, err
	}

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

	// Use provider non-streaming
	log.Printf("🚀 PROVIDER REQUEST: %s model=%s, mcp_enabled=%t",
		s.provider.Name(), model, s.mcpURL != "")

	resp, err := s.provider.Generate(ctx, request)
	if err != nil {
		return nil, err
	}

	// Convert to GenerationResult
	result := &GenerationResult{
		Usage:    resp.Usage,
		MCPUsed:  resp.MCPUsed,
		MCPCalls: resp.MCPCalls,
		MCPTools: resp.MCPTools,
	}
	result.OutputParsed.Choices = resp.OutputParsed.Choices

	transaction.SetTag("success", "true")
	transaction.SetTag("mcp_used", fmt.Sprintf("%t", result.MCPUsed))
	transaction.SetTag("model", model)
	transaction.SetTag("streaming", "true")
	transaction.SetTag("provider", s.provider.Name())

	// Record metrics
	duration := time.Since(startTime)
	s.metrics.RecordGenerationDuration(ctx, duration, true)
	s.metrics.RecordMCPUsage(result.MCPUsed, result.MCPCalls)

	log.Printf("⏱️  STREAMING GENERATION (fallback) COMPLETED in %v", duration)

	return result, nil
}
