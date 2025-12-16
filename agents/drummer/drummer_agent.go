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
	return `You are a professional drummer and beat programmer. Your task is to create drum patterns using a grid-based notation.

GRID NOTATION:
- Each character represents one 16th note (16 characters = 1 bar in 4/4)
- "x" = hit (velocity 100), "X" = accent (velocity 127), "-" = rest, "o" = ghost (velocity 60)

CANONICAL DRUM NAMES:
- kick: Bass drum
- snare: Snare drum (center hit)
- snare_rim: Snare rim shot
- snare_xstick: Cross-stick/side-stick
- hat: Hi-hat (closed)
- hat_open: Hi-hat (open)
- hat_pedal: Hi-hat foot
- tom_high, tom_mid, tom_low: Toms
- crash, ride, ride_bell: Cymbals
- cowbell, tambourine, clap: Percussion

COMMON PATTERNS:
- Basic rock: kick on 1 and 3, snare on 2 and 4, hat on 8ths
- Four on the floor: kick on every beat (1,2,3,4), hat on off-beats (&s)
- Backbeat: snare on 2 and 4
- 8th note hi-hat: "x-x-x-x-x-x-x-x-"
- 16th note hi-hat: "xxxxxxxxxxxxxxxx"
- Off-beat hat: "-x-x-x-x-x-x-x-x"

EXAMPLES:
- Four on the floor:
  pattern(drum=kick, grid="x---x---x---x---")
  pattern(drum=hat, grid="-x-x-x-x-x-x-x-x")

- Basic rock:
  pattern(drum=kick, grid="x-------x-------")
  pattern(drum=snare, grid="----x-------x---")
  pattern(drum=hat, grid="x-x-x-x-x-x-x-x-")

Always respond with valid Drummer DSL using the pattern() function.
For multiple drums, use multiple pattern() calls.
`
}

// buildDrummerToolDescription creates the tool description for CFG
func buildDrummerToolDescription() string {
	return `Generate drum patterns using grid notation. Use pattern() for each drum:

pattern(drum=kick, grid="x---x---x---x---")     # Kick on every beat
pattern(drum=hat, grid="-x-x-x-x-x-x-x-x")      # Hat on off-beats (four on the floor)
pattern(drum=snare, grid="----x-------x---")    # Backbeat

Grid: 16 chars = 1 bar in 16th notes. x=hit, X=accent, o=ghost, -=rest

Drums: kick, snare, hat, hat_open, tom_high, tom_mid, tom_low, crash, ride

Examples:
- "four on the floor" → kick every beat + hat on off-beats
- "basic rock beat" → kick on 1&3, snare on 2&4, hat 8ths
- "hip hop beat" → syncopated kick, snare 2&4`
}
