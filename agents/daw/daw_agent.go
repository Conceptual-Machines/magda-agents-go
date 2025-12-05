package daw

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/Conceptual-Machines/magda-agents-go/metrics"
	"github.com/Conceptual-Machines/magda-agents-go/models"
	"github.com/Conceptual-Machines/magda-agents-go/prompt"
	"github.com/getsentry/sentry-go"
	"github.com/openai/openai-go/responses"
)

const (
	streamEventCompleted = "completed"
	maxErrorPreviewChars = 500
)

// DawAgent handles DAW (Digital Audio Workstation) operations for MAGDA
// This is the main agent that translates natural language to REAPER actions
type DawAgent struct {
	provider      llm.Provider
	systemPrompt  string
	promptBuilder *prompt.MagdaPromptBuilder
	metrics       *metrics.SentryMetrics
	useDSL        bool // If true, use CFG/DSL mode; if false, use JSON Schema mode
}

func NewDawAgent(cfg *config.Config) *DawAgent {
	promptBuilder := prompt.NewMagdaPromptBuilder()
	systemPrompt, err := promptBuilder.BuildPrompt()
	if err != nil {
		log.Fatal("Failed to load MAGDA system prompt:", err)
	}

	// Use OpenAI provider (default for now)
	provider := llm.NewOpenAIProvider(cfg.OpenAIAPIKey)

	// Check environment variable to determine mode (default to DSL for now)
	useDSL := true // Can be made configurable via env var: os.Getenv("MAGDA_USE_DSL") != "false"

	agent := &DawAgent{
		provider:      provider,
		systemPrompt:  systemPrompt,
		promptBuilder: promptBuilder,
		metrics:       metrics.NewSentryMetrics(),
		useDSL:        useDSL,
	}

	log.Printf("🤖 DAW AGENT INITIALIZED:")
	log.Printf("   Provider: %s", provider.Name())
	log.Printf("   System prompt loaded: %d chars", len(systemPrompt))
	log.Printf("   Mode: %s", map[bool]string{true: "DSL (CFG)", false: "JSON Schema"}[useDSL])

	return agent
}

type DawResult struct {
	Actions []map[string]interface{} `json:"actions"`
	Usage   any                      `json:"usage"`
}

func (a *DawAgent) GenerateActions(
	ctx context.Context, question string, state map[string]interface{},
) (*DawResult, error) {
	startTime := time.Now()
	log.Printf("🤖 MAGDA REQUEST STARTED: question=%s", question)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "magda.generate_actions")
	defer transaction.Finish()

	transaction.SetTag("model", "gpt-5.1") // GPT-5.1 for MAGDA
	transaction.SetContext("magda", map[string]interface{}{
		"question_length": len(question),
		"has_state":       state != nil,
	})

	// Build input messages
	inputArray := a.buildInputMessages(question, state)

	// Build provider request - support both JSON Schema and CFG/DSL modes
	request := &llm.GenerationRequest{
		Model:         "gpt-5.1", // GPT-5.1 for MAGDA - best for complex reasoning and code-heavy tasks
		InputArray:    inputArray,
		ReasoningMode: "none", // GPT-5.1 defaults to "none" for faster, low-latency responses
		SystemPrompt:  a.systemPrompt,
	}

	// Choose between JSON Schema and CFG/DSL based on configuration
	if a.useDSL {
		// Use CFG grammar for DSL output
		request.CFGGrammar = &llm.CFGConfig{
			ToolName: "magda_dsl",
			Description: "Executes REAPER operations using the MAGDA DSL. " +
				"Generate functional script code like: track(instrument=\"Serum\").new_clip(bar=3, length_bars=4).add_midi(notes=[...]). " +
				"When user says 'create track with [instrument]' or 'track with [instrument]', ALWAYS generate track(instrument=\"[instrument]\") - never generate track() without the instrument parameter when an instrument is mentioned. " +
				"For existing tracks, use track(id=1).new_clip(bar=3) where id is 1-based (track 1 = first track). " +
				"For selection operations on multiple tracks (e.g., 'select all tracks named X'), use filter() with predicate and chain set_selected: filter(tracks, track.name == \"X\").set_selected(selected=true). " +
				"This efficiently filters the collection and applies the action to all matching tracks. " +
				"Use functional methods for collections when appropriate: filter(tracks, track.name == \"FX\"), map(@get_name, tracks), for_each(tracks, @add_reverb). " +
				"ALWAYS check the current REAPER state to see which tracks exist and use the correct track indices. " +
				"If no track is specified in a chain, it applies to the track created by track(). " +
				"YOU MUST REASON HEAVILY ABOUT THE OPERATIONS AND MAKE SURE THE CODE OBEYS THE GRAMMAR.",
			Grammar: GetMagdaDSLGrammarForFunctional(),
			Syntax:  "lark",
		}
		log.Printf("🔧 Using DSL mode (CFG grammar)")
	} else {
		// Use JSON Schema for structured output (legacy mode)
		request.OutputSchema = &llm.OutputSchema{
			Name:        "MagdaActions",
			Description: "REAPER actions to execute",
			Schema:      llm.GetMagdaActionsSchema(),
		}
		log.Printf("🔧 Using JSON Schema mode (legacy)")
	}

	// Call provider
	log.Printf("🚀 MAGDA PROVIDER REQUEST: %s", a.provider.Name())

	resp, err := a.provider.Generate(ctx, request)
	if err != nil {
		transaction.SetTag("success", "false")
		transaction.SetTag("error_type", "provider_error")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("provider request failed: %w", err)
	}

	// Parse actions from response
	// For MAGDA, we need to parse the raw JSON since the provider expects MusicalOutput format
	// We'll need to get the raw response text and parse it into MagdaActionsOutput
	actions, err := a.parseActionsFromResponse(resp, state)
	if err != nil {
		transaction.SetTag("success", "false")
		transaction.SetTag("error_type", "parse_error")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("failed to parse actions: %w", err)
	}

	result := &DawResult{
		Actions: actions,
		Usage:   resp.Usage,
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

	log.Printf("✅ MAGDA REQUEST COMPLETE: actions=%d, duration=%v", len(actions), duration)

	return result, nil
}

// buildInputMessages constructs the input array for the LLM
func (a *DawAgent) buildInputMessages(question string, state map[string]interface{}) []map[string]interface{} {
	messages := []map[string]interface{}{}

	// Add user question
	userMessage := map[string]interface{}{
		"role":    "user",
		"content": question,
	}
	messages = append(messages, userMessage)

	// Add REAPER state if provided
	if len(state) > 0 {
		stateMessage := map[string]interface{}{
			"role":    "user",
			"content": fmt.Sprintf("Current REAPER state: %+v", state),
		}
		messages = append(messages, stateMessage)
	}

	return messages
}

// parseActionsFromResponse extracts actions from the LLM response
// For CFG/DSL mode: RawOutput contains DSL code (e.g., track().newClip().addMidi())
// For JSON Schema mode: RawOutput contains JSON with actions array
func (a *DawAgent) parseActionsFromResponse(resp *llm.GenerationResponse, state map[string]interface{}) ([]map[string]interface{}, error) {
	// The provider should have stored the raw output (DSL or JSON) in RawOutput
	if resp.RawOutput == "" {
		return nil, fmt.Errorf("no raw output available in response")
	}

	// If using DSL mode, try parsing as DSL first
	if a.useDSL {
		dslCode := strings.TrimSpace(resp.RawOutput)
		// Check if it's DSL (starts with "track" or similar function call)
		if strings.HasPrefix(dslCode, "track(") || strings.Contains(dslCode, ".new_clip(") || strings.Contains(dslCode, ".add_midi(") || strings.Contains(dslCode, ".filter(") || strings.Contains(dslCode, ".map(") || strings.Contains(dslCode, ".for_each(") {
			// This is DSL code - parse and translate to REAPER API actions
			log.Printf("✅ Found DSL code in response: %s", truncate(dslCode, MaxDSLPreviewLength))

			parser, err := NewFunctionalDSLParser()
			if err != nil {
				return nil, fmt.Errorf("failed to create functional DSL parser: %w", err)
			}
			parser.SetState(map[string]interface{}{"state": state}) // Pass state for track resolution
			actions, err := parser.ParseDSL(dslCode)
			if err != nil {
				return nil, fmt.Errorf("failed to parse DSL: %w", err)
			}

			log.Printf("✅ Translated DSL to %d REAPER API actions", len(actions))
			return actions, nil
		}
		// If DSL mode but output doesn't look like DSL, try parsing as JSON (fallback)
		log.Printf("⚠️  DSL mode enabled but output doesn't look like DSL, trying JSON parse")
	}

	// JSON Schema mode (legacy) or fallback: parse as JSON
	var magdaOutput models.MagdaActionsOutput
	if err := json.Unmarshal([]byte(resp.RawOutput), &magdaOutput); err != nil {
		log.Printf("❌ Failed to parse MAGDA output as JSON: %v", err)
		const maxLogLength = 500
		log.Printf("Raw output (first %d chars): %s", maxLogLength, truncate(resp.RawOutput, maxLogLength))
		return nil, fmt.Errorf("failed to parse MAGDA output: %w", err)
	}

	if len(magdaOutput.Actions) == 0 {
		return nil, fmt.Errorf("no actions found in MAGDA output")
	}

	log.Printf("✅ Parsed %d actions from MAGDA JSON output", len(magdaOutput.Actions))
	return magdaOutput.Actions, nil
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// StreamActionCallback is called for each action found in the stream
type StreamActionCallback func(action map[string]interface{}) error

// GenerateActionsStream generates actions using streaming (without structured output)
// It parses JSON incrementally from the text stream and calls callback for each action found
func (a *DawAgent) GenerateActionsStream(
	ctx context.Context,
	question string,
	state map[string]interface{},
	callback StreamActionCallback,
) (*DawResult, error) {
	startTime := time.Now()
	log.Printf("🤖 MAGDA STREAMING REQUEST STARTED: question=%s", question)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "magda.generate_actions_stream")
	defer transaction.Finish()

	transaction.SetTag("model", "gpt-5.1")
	transaction.SetTag("streaming", "true")
	transaction.SetContext("magda", map[string]interface{}{
		"question_length": len(question),
		"has_state":       state != nil,
	})

	// Build input messages
	inputArray := a.buildInputMessages(question, state)

	// Build provider request - support both JSON Schema and CFG/DSL modes
	request := &llm.GenerationRequest{
		Model:         "gpt-5.1",
		InputArray:    inputArray,
		ReasoningMode: "none",
		SystemPrompt:  a.systemPrompt,
	}

	// Choose between JSON Schema and CFG/DSL based on configuration
	if a.useDSL {
		// Use CFG grammar for DSL output
		request.CFGGrammar = &llm.CFGConfig{
			ToolName: "magda_dsl",
			Description: "Executes REAPER operations using the MAGDA DSL. " +
				"Generate functional script code like: track(instrument=\"Serum\").new_clip(bar=3, length_bars=4).add_midi(notes=[...]). " +
				"When user says 'create track with [instrument]' or 'track with [instrument]', ALWAYS generate track(instrument=\"[instrument]\") - never generate track() without the instrument parameter when an instrument is mentioned. " +
				"For existing tracks, use track(id=1).new_clip(bar=3) where id is 1-based (track 1 = first track). " +
				"For selection operations on multiple tracks (e.g., 'select all tracks named X'), use filter() with predicate and chain set_selected: filter(tracks, track.name == \"X\").set_selected(selected=true). " +
				"This efficiently filters the collection and applies the action to all matching tracks. " +
				"Use functional methods for collections when appropriate: filter(tracks, track.name == \"FX\"), map(@get_name, tracks), for_each(tracks, @add_reverb). " +
				"ALWAYS check the current REAPER state to see which tracks exist and use the correct track indices. " +
				"If no track is specified in a chain, it applies to the track created by track(). " +
				"YOU MUST REASON HEAVILY ABOUT THE OPERATIONS AND MAKE SURE THE CODE OBEYS THE GRAMMAR.",
			Grammar: GetMagdaDSLGrammarForFunctional(),
			Syntax:  "lark",
		}
		log.Printf("🔧 Using DSL mode (CFG grammar) for streaming")
	} else {
		// No OutputSchema - we'll parse raw JSON from text stream
		log.Printf("🔧 Using JSON Schema mode (legacy) for streaming")
	}

	// Track accumulated text and parsed actions
	var accumulatedText string
	var allActions []map[string]interface{}
	var usage any

	// Stream callback that processes text deltas and parses actions incrementally
	streamCallback := func(event llm.StreamEvent) error {
		return a.handleStreamEvent(event, &accumulatedText, &allActions, &usage, callback, state)
	}

	// Call streaming provider
	log.Printf("🚀 MAGDA STREAMING PROVIDER REQUEST: %s", a.provider.Name())
	resp, err := a.provider.GenerateStream(ctx, request, streamCallback)

	// If we already received actions, don't treat provider errors as fatal
	// (DSL mode generates tool calls, not text, so "no output" errors are expected)
	if err != nil {
		if len(allActions) > 0 {
			log.Printf("⚠️  MAGDA: Provider reported error but %d actions were already received: %v", len(allActions), err)
			// Continue processing - we have actions, so this is a success
		} else {
			transaction.SetTag("success", "false")
			transaction.SetTag("error_type", "provider_error")
			sentry.CaptureException(err)
			return nil, fmt.Errorf("provider stream failed: %w", err)
		}
	}

	// If we still have accumulated text, try final parse
	if accumulatedText != "" && len(allActions) == 0 {
		actions, err := a.parseActionsIncremental(accumulatedText, state)
		if err == nil {
			allActions = actions
			for _, action := range actions {
				_ = callback(action)
			}
		}
	}

	if len(allActions) == 0 {
		transaction.SetTag("success", "false")
		transaction.SetTag("error_type", "no_actions")
		return nil, fmt.Errorf("no actions found in stream")
	}

	result := &DawResult{
		Actions: allActions,
		Usage:   usage,
	}

	if resp != nil && resp.Usage != nil {
		result.Usage = resp.Usage
	}

	transaction.SetTag("success", "true")
	transaction.SetTag("actions_count", fmt.Sprintf("%d", len(allActions)))

	duration := time.Since(startTime)
	a.metrics.RecordGenerationDuration(ctx, duration, true)

	log.Printf("✅ MAGDA STREAMING REQUEST COMPLETE: actions=%d, duration=%v", len(allActions), duration)

	return result, nil
}

// parseActionsIncremental tries to parse actions from accumulated text (DSL or JSON)
// It looks for complete DSL code or JSON objects in the text and extracts them
//
//nolint:gocyclo // Complex parsing logic is necessary for handling both DSL and JSON formats
func (a *DawAgent) parseActionsIncremental(text string, state map[string]interface{}) ([]map[string]interface{}, error) {
	text = strings.TrimSpace(text)

	// If using DSL mode, try parsing as DSL first
	if a.useDSL {
		// Check if it's DSL (starts with "track" or similar function call)
		if strings.HasPrefix(text, "track(") || strings.Contains(text, ".new_clip(") || strings.Contains(text, ".add_midi(") || strings.Contains(text, ".filter(") || strings.Contains(text, ".map(") || strings.Contains(text, ".for_each(") {
			// This is DSL code - parse and translate to REAPER API actions
			log.Printf("✅ Found DSL code in stream: %s", truncate(text, MaxDSLPreviewLength))
			log.Printf("📋 FULL DSL CODE (all %d chars, NO TRUNCATION):\n%s", len(text), text)

			parser, err := NewFunctionalDSLParser()
			if err != nil {
				log.Printf("⚠️  Failed to create functional DSL parser: %v, trying JSON parse", err)
			} else {
				parser.SetState(map[string]interface{}{"state": state}) // Pass state for track resolution
				actions, err := parser.ParseDSL(text)
				if err == nil && len(actions) > 0 {
					log.Printf("✅ Translated DSL to %d REAPER API actions", len(actions))
					return actions, nil
				}
				// If DSL parsing failed, fall through to JSON parsing
				log.Printf("⚠️  DSL parsing failed, trying JSON parse: %v", err)
			}
		}
	}

	// First, try to parse as complete MagdaActionsOutput
	var magdaOutput models.MagdaActionsOutput
	if err := json.Unmarshal([]byte(text), &magdaOutput); err == nil {
		if len(magdaOutput.Actions) > 0 {
			return magdaOutput.Actions, nil
		}
	}

	// Try to find and extract the actions array directly
	// Look for "actions": [ ... ]
	actionsStart := strings.Index(text, `"actions"`)
	if actionsStart == -1 {
		// Maybe it's just an array of actions?
		if strings.HasPrefix(text, "[") {
			var actions []map[string]interface{}
			if err := json.Unmarshal([]byte(text), &actions); err == nil {
				return actions, nil
			}
		}
		return nil, fmt.Errorf("no actions array found")
	}

	// Find the opening bracket after "actions"
	arrayStart := strings.Index(text[actionsStart:], "[")
	if arrayStart == -1 {
		return nil, fmt.Errorf("actions array not found")
	}
	arrayStart += actionsStart

	// Find matching closing bracket
	bracketCount := 0
	arrayEnd := -1
	for i := arrayStart; i < len(text); i++ {
		if text[i] == '[' {
			bracketCount++
		} else if text[i] == ']' {
			bracketCount--
			if bracketCount == 0 {
				arrayEnd = i + 1
				break
			}
		}
	}

	if arrayEnd == -1 {
		// Array not complete yet
		return nil, fmt.Errorf("actions array incomplete")
	}

	// Extract and parse the actions array
	actionsJSON := text[arrayStart:arrayEnd]
	var actions []map[string]interface{}
	if err := json.Unmarshal([]byte(actionsJSON), &actions); err != nil {
		return nil, fmt.Errorf("failed to parse actions array: %w", err)
	}

	return actions, nil
}

// handleStreamEvent processes a single stream event to reduce cyclomatic complexity
func (a *DawAgent) handleStreamEvent(
	event llm.StreamEvent,
	accumulatedText *string,
	allActions *[]map[string]interface{},
	usage *any,
	callback StreamActionCallback,
	state map[string]interface{},
) error {
	switch event.Type {
	case "output_text.delta":
		return a.handleTextDelta(event, accumulatedText, allActions, callback, state)
	case "output_progress", "output_started":
		// Just acknowledge these events
		return nil
	case streamEventCompleted:
		return a.handleStreamCompleted(event, accumulatedText, allActions, usage, callback, state)
	}
	return nil
}

// handleTextDelta processes text delta events
func (a *DawAgent) handleTextDelta(
	event llm.StreamEvent,
	accumulatedText *string,
	allActions *[]map[string]interface{},
	callback StreamActionCallback,
	state map[string]interface{},
) error {
	text, ok := event.Data["text"].(string)
	if !ok || text == "" {
		return nil
	}

	*accumulatedText += text
	log.Printf("📝 MAGDA: Accumulated %d chars (delta: %d)", len(*accumulatedText), len(text))

	// Log full accumulated text every 100 chars to see DSL as it builds
	if len(*accumulatedText)%100 < len(text) || len(*accumulatedText) < 500 {
		log.Printf("📋 MAGDA: Full accumulated text so far (%d chars): %s", len(*accumulatedText), *accumulatedText)
	}

	// Try to parse actions from accumulated text after each delta
	actions, err := a.parseActionsIncremental(*accumulatedText, state)
	if err == nil && len(actions) > len(*allActions) {
		// New actions found - call callback for each new one
		for i := len(*allActions); i < len(actions); i++ {
			log.Printf("✅ MAGDA: Parsed action %d: %s", i+1, actions[i]["action"])
			if callbackErr := callback(actions[i]); callbackErr != nil {
				return callbackErr
			}
			*allActions = append(*allActions, actions[i])
		}
	} else if err != nil {
		// Log parse errors but continue (might be incomplete JSON)
		log.Printf("⚠️  MAGDA: Parse attempt failed (might be incomplete): %v", err)
	}
	return nil
}

// handleStreamCompleted processes the stream completion event
func (a *DawAgent) handleStreamCompleted(
	event llm.StreamEvent,
	accumulatedText *string,
	allActions *[]map[string]interface{},
	usage *any,
	callback StreamActionCallback,
	state map[string]interface{},
) error {
	log.Printf("📦 MAGDA: Stream completed, final parse of %d chars", len(*accumulatedText))
	log.Printf("📋 MAGDA: FULL accumulated text at completion (%d chars, NO TRUNCATION):\n%s", len(*accumulatedText), *accumulatedText)
	if *accumulatedText != "" {
		actions, err := a.parseActionsIncremental(*accumulatedText, state)
		if err == nil {
			// Call callback for any remaining actions
			for i := len(*allActions); i < len(actions); i++ {
				log.Printf("✅ MAGDA: Final parse - action %d: %s", i+1, actions[i]["action"])
				if callbackErr := callback(actions[i]); callbackErr != nil {
					return callbackErr
				}
				*allActions = append(*allActions, actions[i])
			}
		} else {
			log.Printf("❌ MAGDA: Final parse failed: %v", err)
			log.Printf("❌ MAGDA: Accumulated text (first %d chars): %s", maxErrorPreviewChars, truncate(*accumulatedText, maxErrorPreviewChars))
			log.Printf("📋 MAGDA: FULL accumulated text on error (%d chars, NO TRUNCATION):\n%s", len(*accumulatedText), *accumulatedText)
		}
	}
	if usageData, ok := event.Data["usage"]; ok {
		*usage = usageData
	}
	return nil
}
