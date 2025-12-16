package drummer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/Conceptual-Machines/magda-agents-go/metrics"
	"github.com/getsentry/sentry-go"
)

// DrummerAgent generates drum patterns using LLM + CFG grammar
type DrummerAgent struct {
	provider     llm.Provider
	systemPrompt string
	metrics      *metrics.SentryMetrics
}

// DrummerResult contains the DSL output
// Note: Conversion to MIDI happens in the Reaper extension, NOT here
type DrummerResult struct {
	DSL     string           `json:"dsl"`     // Raw DSL code from LLM
	Actions []map[string]any `json:"actions"` // Parsed actions from Grammar School
	Usage   any              `json:"usage"`
}

// NewDrummerAgent creates a new drummer agent
func NewDrummerAgent(cfg *config.Config) *DrummerAgent {
	return NewDrummerAgentWithProvider(cfg, nil)
}

// NewDrummerAgentWithProvider creates a drummer agent with a specific LLM provider
func NewDrummerAgentWithProvider(cfg *config.Config, provider llm.Provider) *DrummerAgent {
	// Use provided provider or create OpenAI provider (default)
	if provider == nil {
		provider = llm.NewOpenAIProvider(cfg.OpenAIAPIKey)
	}

	systemPrompt := buildDrummerSystemPrompt()

	agent := &DrummerAgent{
		provider:     provider,
		systemPrompt: systemPrompt,
		metrics:      metrics.NewSentryMetrics(),
	}

	log.Printf("🥁 DRUMMER AGENT INITIALIZED:")
	log.Printf("   Provider: %s", provider.Name())

	return agent
}

// Generate creates drum pattern DSL from natural language
func (a *DrummerAgent) Generate(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
) (*DrummerResult, error) {
	startTime := time.Now()
	log.Printf("🥁 DRUMMER REQUEST STARTED (Model: %s)", model)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "drummer.generate")
	defer transaction.Finish()

	transaction.SetTag("model", model)

	// Build provider request with CFG grammar
	request := &llm.GenerationRequest{
		Model:        model,
		InputArray:   inputArray,
		SystemPrompt: a.systemPrompt,
		CFGGrammar: &llm.CFGConfig{
			ToolName:    "drummer_dsl",
			Description: buildDrummerToolDescription(),
			Grammar:     llm.GetDrummerDSLGrammar(),
			Syntax:      "lark",
		},
	}

	// Call provider
	log.Printf("🚀 DRUMMER REQUEST: %s model=%s, input_messages=%d",
		a.provider.Name(), model, len(inputArray))

	resp, err := a.provider.Generate(ctx, request)
	if err != nil {
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("provider request failed: %w", err)
	}

	// Extract DSL from response
	dslCode := resp.RawOutput
	if dslCode == "" {
		transaction.SetTag("success", "false")
		return nil, fmt.Errorf("no DSL output in response")
	}

	log.Printf("🥁 DSL Output: %s", dslCode)

	// Parse DSL using Grammar School to get actions
	parser, err := NewDrummerDSLParser()
	if err != nil {
		transaction.SetTag("success", "false")
		return nil, fmt.Errorf("failed to create DSL parser: %w", err)
	}

	actions, err := parser.ParseDSL(dslCode)
	if err != nil {
		transaction.SetTag("success", "false")
		return nil, fmt.Errorf("failed to parse DSL: %w", err)
	}

	result := &DrummerResult{
		DSL:     dslCode,
		Actions: actions,
		Usage:   resp.Usage,
	}

	// Record metrics
	transaction.SetTag("success", "true")
	transaction.SetTag("action_count", fmt.Sprintf("%d", len(actions)))

	duration := time.Since(startTime)
	a.metrics.RecordGenerationDuration(ctx, duration, true)

	log.Printf("✅ DRUMMER COMPLETE: %d actions", len(actions))

	return result, nil
}

// buildDrummerSystemPrompt creates the system prompt for the drummer agent
func buildDrummerSystemPrompt() string {
	return `You are a professional drummer. Create drum patterns using pattern() calls.

SCOPE: You ONLY handle requests about drums, beats, percussion, and rhythm patterns.
If a request is not about drums/percussion (e.g., chords, melodies, mixing, or non-music topics),
return NOTHING - do not generate any output. Only generate patterns for drum-related requests.

SYNTAX: pattern(drum=NAME, grid="GRID")

GRID: 16 chars = 1 bar. "x"=hit, "X"=accent, "o"=ghost, "-"=rest

DRUMS: kick, snare, hat, hat_open, tom_high, tom_mid, tom_low, crash, ride

EXAMPLES:
- Four on the floor:
pattern(drum=kick, grid="x---x---x---x---"); pattern(drum=hat, grid="-x-x-x-x-x-x-x-x")

- Basic rock:
pattern(drum=kick, grid="x-------x-------"); pattern(drum=snare, grid="----x-------x---"); pattern(drum=hat, grid="x-x-x-x-x-x-x-x-")

- Breakbeat:
pattern(drum=kick, grid="x--x--x---x-x---"); pattern(drum=snare, grid="----x--o----x-o-"); pattern(drum=hat, grid="x-x-x-x-x-x-x-x-")

Use semicolons to separate multiple patterns. Always output valid DSL.
`
}

// buildDrummerToolDescription creates the tool description for CFG
func buildDrummerToolDescription() string {
	return `Generate drum patterns. Use pattern() for each drum, separated by semicolons:

pattern(drum=kick, grid="x---x---x---x---"); pattern(drum=snare, grid="----x-------x---")

Grid: 16 chars = 1 bar. x=hit, X=accent, o=ghost, -=rest
Drums: kick, snare, hat, hat_open, tom_high, tom_mid, tom_low, crash, ride`
}
