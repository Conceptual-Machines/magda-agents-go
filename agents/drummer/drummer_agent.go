package drummer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/Conceptual-Machines/magda-agents-go/metrics"
	"github.com/Conceptual-Machines/magda-agents-go/models"
	"github.com/getsentry/sentry-go"
)

// DrummerAgent generates drum patterns using LLM + CFG grammar
type DrummerAgent struct {
	provider     llm.Provider
	systemPrompt string
	metrics      *metrics.SentryMetrics
}

// DrummerResult contains the generated drum beat
type DrummerResult struct {
	Beat     *DrumBeat           `json:"beat"`
	Notes    []models.NoteEvent  `json:"notes"`
	DSL      string              `json:"dsl,omitempty"`
	Usage    any                 `json:"usage"`
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

// Generate creates a drum pattern from natural language
func (a *DrummerAgent) Generate(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
	drumMapping map[string]int, // Optional custom drum mapping
) (*DrummerResult, error) {
	startTime := time.Now()
	log.Printf("🥁 DRUMMER REQUEST STARTED (Model: %s)", model)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "drummer.generate")
	defer transaction.Finish()

	transaction.SetTag("model", model)

	// Build provider request with CFG grammar
	request := &llm.GenerationRequest{
		Model:         model,
		InputArray:    inputArray,
		SystemPrompt:  a.systemPrompt,
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

	// Parse DSL to DrumBeat
	parser, err := NewDrummerDSLParser()
	if err != nil {
		transaction.SetTag("success", "false")
		return nil, fmt.Errorf("failed to create DSL parser: %w", err)
	}

	beat, err := parser.ParseDSL(dslCode)
	if err != nil {
		transaction.SetTag("success", "false")
		return nil, fmt.Errorf("failed to parse DSL: %w", err)
	}

	// Convert to MIDI notes
	notes, err := parser.ConvertToNoteEvents(beat, drumMapping)
	if err != nil {
		transaction.SetTag("success", "false")
		return nil, fmt.Errorf("failed to convert to notes: %w", err)
	}

	result := &DrummerResult{
		Beat:  beat,
		Notes: notes,
		DSL:   dslCode,
		Usage: resp.Usage,
	}

	// Record metrics
	transaction.SetTag("success", "true")
	transaction.SetTag("pattern_count", fmt.Sprintf("%d", len(beat.Patterns)))
	transaction.SetTag("note_count", fmt.Sprintf("%d", len(notes)))

	duration := time.Since(startTime)
	a.metrics.RecordGenerationDuration(ctx, duration, true)

	log.Printf("✅ DRUMMER COMPLETE: %d patterns, %d notes", len(beat.Patterns), len(notes))

	return result, nil
}

// buildDrummerSystemPrompt creates the system prompt for the drummer agent
func buildDrummerSystemPrompt() string {
	return `You are a professional drummer and beat programmer. Your task is to create drum patterns using a grid-based notation.

GRID NOTATION:
- Each character represents one 16th note (16 characters = 1 bar in 4/4)
- "x" = normal hit (velocity 100)
- "X" = accent (velocity 127)  
- "o" = ghost note (velocity 60)
- "-" = rest (no hit)

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
- Basic rock: kick on 1 and 3, snare on 2 and 4
- Four-on-floor: kick on every beat
- Backbeat: snare on 2 and 4
- 8th note hi-hat: "x-x-x-x-x-x-x-x-"
- 16th note hi-hat: "xxxxxxxxxxxxxxxx"

Always respond with valid Drummer DSL using the pattern() function.
For multiple drums, use multiple pattern() calls.
`
}

// buildDrummerToolDescription creates the tool description for CFG
func buildDrummerToolDescription() string {
	return `Generate drum patterns using grid notation. Use pattern() for each drum:

pattern(drum=kick, grid="x---x---x---x---")     # 4 on the floor
pattern(drum=snare, grid="----x-------x---")    # Backbeat
pattern(drum=hat, grid="x-x-x-x-x-x-x-x-")      # 8th notes

Grid: 16 chars = 1 bar. x=hit, X=accent, o=ghost, -=rest

Drums: kick, snare, hat, hat_open, tom_high, tom_mid, tom_low, crash, ride

Examples:
- "basic rock beat" → kick on 1&3, snare on 2&4, hat 8ths
- "four on the floor" → kick every beat, hat 16ths
- "hip hop beat" → syncopated kick, snare 2&4, hat with opens`
}

