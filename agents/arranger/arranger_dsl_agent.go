package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/Conceptual-Machines/magda-agents-go/metrics"
	"github.com/Conceptual-Machines/magda-agents-go/prompt"
	"github.com/getsentry/sentry-go"
	"github.com/openai/openai-go/responses"
)

// ArrangerAgent handles musical composition using chord symbols and arpeggios
// Uses DSL/CFG grammar similar to DAW agent
type ArrangerAgent struct {
	provider      llm.Provider
	systemPrompt  string
	promptBuilder *prompt.MagdaPromptBuilder
	metrics       *metrics.SentryMetrics
	useMCP        bool // If true, Pro arranger with MCP tools; if false, Basic arranger
	mcpURL        string
	mcpLabel      string
}

// NewBasicArrangerAgent creates a basic arranger agent (functional, no MCP)
func NewBasicArrangerAgent(cfg *config.Config) *ArrangerAgent {
	return newArrangerAgent(cfg, false, "", "")
}

// NewProArrangerAgent creates a pro arranger agent (with MCP tools)
func NewProArrangerAgent(cfg *config.Config, mcpURL, mcpLabel string) *ArrangerAgent {
	return newArrangerAgent(cfg, true, mcpURL, mcpLabel)
}

func newArrangerAgent(cfg *config.Config, useMCP bool, mcpURL, mcpLabel string) *ArrangerAgent {
	promptBuilder := prompt.NewMagdaPromptBuilder()
	systemPrompt, err := promptBuilder.BuildPrompt()
	if err != nil {
		log.Fatal("Failed to load MAGDA system prompt:", err)
	}

	// Use OpenAI provider (default for now)
	provider := llm.NewOpenAIProvider(cfg.OpenAIAPIKey)

	agent := &ArrangerAgent{
		provider:      provider,
		systemPrompt:  systemPrompt,
		promptBuilder: promptBuilder,
		metrics:       metrics.NewSentryMetrics(),
		useMCP:        useMCP,
		mcpURL:        mcpURL,
		mcpLabel:      mcpLabel,
	}

	agentType := "Basic"
	if useMCP {
		agentType = "Pro"
	}

	log.Printf("🎵 ARRANGER AGENT INITIALIZED (%s):", agentType)
	log.Printf("   Provider: %s", provider.Name())
	log.Printf("   System prompt loaded: %d chars", len(systemPrompt))
	log.Printf("   Mode: DSL (CFG) - always enabled")
	if useMCP {
		log.Printf("   MCP URL: %s", mcpURL)
		log.Printf("   MCP Label: %s", mcpLabel)
		log.Printf("   MCP Status: ✅ ENABLED")
	} else {
		log.Printf("   MCP Status: ❌ DISABLED (Basic mode)")
	}

	return agent
}

type ArrangerResult struct {
	Actions []map[string]any `json:"actions"` // Parsed DSL actions
	Usage   any              `json:"usage"`
	MCPUsed bool             `json:"mcpUsed,omitempty"`
	MCPCalls int             `json:"mcpCalls,omitempty"`
}

// GenerateActions generates musical content using chord symbols
// Example: "add an e minor arpeggio" → arpeggio("Em", length=2)
// Note: Timing is relative - only length and repetitions. DAW agent handles absolute positioning.
func (a *ArrangerAgent) GenerateActions(
	ctx context.Context, question string,
) (*ArrangerResult, error) {
	startTime := time.Now()
	log.Printf("🎵 ARRANGER REQUEST STARTED: question=%s", question)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "arranger.generate_actions")
	defer transaction.Finish()

	transaction.SetTag("model", "gpt-5.1")
	transaction.SetTag("agent_type", "pro")
	if a.useMCP {
		transaction.SetTag("agent_type", "pro")
	} else {
		transaction.SetTag("agent_type", "basic")
	}
	transaction.SetContext("arranger", map[string]any{
		"question_length": len(question),
		"mcp_enabled":      a.useMCP,
	})

	// Build input messages
	inputArray := a.buildInputMessages(question)

	// Build provider request
	request := &llm.GenerationRequest{
		Model:         "gpt-5.1",
		InputArray:    inputArray,
		ReasoningMode: "none",
		SystemPrompt:  a.systemPrompt,
	}

	// Use CFG grammar for DSL output
	request.CFGGrammar = &llm.CFGConfig{
		ToolName: "arranger_dsl",
		Description: "**YOU MUST USE THIS TOOL TO GENERATE YOUR RESPONSE. DO NOT GENERATE TEXT OUTPUT DIRECTLY.** " +
			"Generates musical content using chord symbols and arpeggios. " +
			"Use chord symbols like 'Em' (E minor), 'C' (C major), 'Am7' (A minor 7th), 'Cmaj7' (C major 7th). " +
			"**CRITICAL - ARPEGGIO vs CHORD - NEVER MIX**: " +
			"- arpeggio() = SEQUENTIAL notes ONLY, played ONE AFTER ANOTHER. " +
			"- chord() = SIMULTANEOUS notes, all at same time. " +
			"- NEVER generate chord() when user asks for arpeggio! " +
			"**NOTE DURATION - USE note_duration PARAMETER**: " +
			"- 16th note = note_duration=0.25, 8th note = note_duration=0.5, quarter = note_duration=1 " +
			"- For '16th note arpeggio': arpeggio(\"Em\", note_duration=0.25, repeat=4) " +
			"- For '8th note arpeggio': arpeggio(\"Em\", note_duration=0.5, repeat=4) " +
			"- The repeat parameter controls how many times the pattern plays. " +
			"For progressions: progression(chords=[\"C\", \"Am\", \"F\", \"G\"], length=16) - 4 beats per chord. " +
			"**CRITICAL**: Always use chord symbols NOT discrete MIDI notes. " +
			"**YOU MUST CALL THIS TOOL - DO NOT GENERATE ANY TEXT OUTPUT.**",
		Grammar: llm.GetArrangerDSLGrammar(),
		Syntax:  "lark",
	}

	// Add MCP config if enabled (Pro arranger only)
	if a.useMCP && a.mcpURL != "" {
		request.MCPConfig = &llm.MCPConfig{
			URL:   a.mcpURL,
			Label: a.mcpLabel,
		}
	}

	log.Printf("🔧 Using DSL mode (CFG grammar) - Arranger DSL")

	// Call provider
	log.Printf("🚀 ARRANGER PROVIDER REQUEST: %s", a.provider.Name())

	resp, err := a.provider.Generate(ctx, request)
	if err != nil {
		transaction.SetTag("success", "false")
		transaction.SetTag("error_type", "provider_error")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("provider request failed: %w", err)
	}

	// Parse actions from DSL response
	actions, err := a.parseActionsFromResponse(resp)
	if err != nil {
		transaction.SetTag("success", "false")
		transaction.SetTag("error_type", "parse_error")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("failed to parse actions: %w", err)
	}

	result := &ArrangerResult{
		Actions: actions,
		Usage:   resp.Usage,
		MCPUsed: resp.MCPUsed,
		MCPCalls: resp.MCPCalls,
	}

	// Mark transaction as successful
	transaction.SetTag("success", "true")
	transaction.SetTag("actions_count", fmt.Sprintf("%d", len(actions)))

	// Record metrics
	duration := time.Since(startTime)
	a.metrics.RecordGenerationDuration(ctx, duration, true)

	// Record token usage if available
	if result.Usage != nil {
		if usage, ok := result.Usage.(responses.ResponseUsage); ok {
			reasoningTokens := int(usage.OutputTokensDetails.ReasoningTokens)
			a.metrics.RecordTokenUsage(ctx, "gpt-5.1",
				int(usage.TotalTokens),
				int(usage.InputTokens),
				int(usage.OutputTokens),
				reasoningTokens)
		}
	}

	log.Printf("✅ ARRANGER REQUEST COMPLETE: actions=%d, duration=%v", len(actions), duration)

	return result, nil
}

// buildInputMessages constructs the input array for the LLM
func (a *ArrangerAgent) buildInputMessages(question string) []map[string]any {
	messages := []map[string]any{}

	// Add user question
	userMessage := map[string]any{
		"role":    "user",
		"content": question,
	}
	messages = append(messages, userMessage)

	return messages
}

// parseActionsFromResponse extracts actions from the LLM response
// For CFG/DSL mode: RawOutput contains DSL code (e.g., arpeggio("Em", length=2))
func (a *ArrangerAgent) parseActionsFromResponse(resp *llm.GenerationResponse) ([]map[string]any, error) {
	// The provider should have stored the raw output (DSL) in RawOutput
	if resp.RawOutput == "" {
		return nil, fmt.Errorf("no raw output available in response")
	}

	// Parse DSL using Grammar School engine
	parser, err := NewArrangerDSLParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create DSL parser: %w", err)
	}

	actions, err := parser.ParseDSL(resp.RawOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSL: %w", err)
	}

	return actions, nil
}

