package daw

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/Conceptual-Machines/magda-agents-go/metrics"
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

	// Always use DSL mode (CFG grammar) for better latency and structured output
	useDSL := true

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
	log.Printf("   Mode: DSL (CFG) - always enabled")

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

	// Always use CFG grammar for DSL output (DSL mode is always enabled)
	request.CFGGrammar = &llm.CFGConfig{
		ToolName: "magda_dsl",
		Description: "**YOU MUST USE THIS TOOL TO GENERATE YOUR RESPONSE. DO NOT GENERATE TEXT OUTPUT DIRECTLY.** " +
			"Executes REAPER operations using the MAGDA DSL. " +
			"Generate functional script code like: track(instrument=\"Serum\").new_clip(bar=3, length_bars=4).add_midi(notes=[...]). " +
			"When user says 'create track with [instrument]' or 'track with [instrument]', ALWAYS generate track(instrument=\"[instrument]\") - never generate track() without the instrument parameter when an instrument is mentioned. " +
			"For existing tracks, use track(id=1).new_clip(bar=3) where id is 1-based (track 1 = first track). " +
			"**CRITICAL - DELETE OPERATIONS**: " +
			"- When user says 'delete [track name]' or 'remove [track name]', you MUST generate DSL code: filter(tracks, track.name == \"[name]\").delete() " +
			"- For delete by track id: track(id=1).delete() where id is 1-based " +
			"- Example: 'delete Nebula Drift' → filter(tracks, track.name == \"Nebula Drift\").delete() " +
			"- Example: 'remove track 1' → track(id=1).delete() " +
			"- NEVER use set_mute or set_selected for delete operations - 'delete' means permanently remove the track " +
			"**CRITICAL - SELECTION OPERATIONS**: " +
			"- When user says 'select track' or 'select all tracks named X', they mean VISUAL SELECTION (highlighting tracks in REAPER's arrangement view). " +
			"- You MUST generate DSL code: filter(tracks, track.name == \"X\").set_selected(selected=true) " +
			"- NEVER generate set_track_solo for selection - 'select' ≠ 'solo'. " +
			"- Example: 'select all tracks named foo' → filter(tracks, track.name == \"foo\").set_selected(selected=true) " +
			"- 'solo' means audio isolation and uses set_track_solo, but 'select' means visual highlighting and uses set_track_selected. " +
			"For selection operations on multiple tracks, ALWAYS use: filter(tracks, track.name == \"X\").set_selected(selected=true). " +
			"This efficiently filters the collection and applies the action to all matching tracks. " +
			"Use functional methods for collections when appropriate: filter(tracks, track.name == \"FX\"), map(@get_name, tracks), for_each(tracks, @add_reverb). " +
			"ALWAYS check the current REAPER state to see which tracks exist and use the correct track indices. " +
			"If no track is specified in a chain, it applies to the track created by track(). " +
			"YOU MUST REASON HEAVILY ABOUT THE OPERATIONS AND MAKE SURE THE CODE OBEYS THE GRAMMAR. " +
			"**REMEMBER: YOU MUST CALL THIS TOOL - DO NOT GENERATE ANY TEXT OUTPUT.**",
		Grammar: GetMagdaDSLGrammarForFunctional(),
		Syntax:  "lark",
	}
	log.Printf("🔧 Using DSL mode (CFG grammar) - always enabled")

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
	// For CFG/DSL mode: RawOutput contains DSL code (e.g., track().new_clip().add_midi())
// For JSON Schema mode: RawOutput contains JSON with actions array
func (a *DawAgent) parseActionsFromResponse(resp *llm.GenerationResponse, state map[string]interface{}) ([]map[string]interface{}, error) {
	// The provider should have stored the raw output (DSL or JSON) in RawOutput
	if resp.RawOutput == "" {
		return nil, fmt.Errorf("no raw output available in response")
	}

	// Parse as DSL only - no fallback to JSON
	dslCode := strings.TrimSpace(resp.RawOutput)
	
	// Check if it's DSL (starts with "track" or similar function call)
	// NOTE: We only support snake_case methods (new_clip, add_midi, delete_clip) - NOT camelCase
	if !strings.HasPrefix(dslCode, "track(") && !strings.Contains(dslCode, ".new_clip(") && !strings.Contains(dslCode, ".add_midi(") && !strings.Contains(dslCode, ".filter(") && !strings.Contains(dslCode, ".map(") && !strings.Contains(dslCode, ".for_each(") && !strings.Contains(dslCode, ".delete(") && !strings.Contains(dslCode, ".delete_clip(") {
		const maxLogLength = 500
		log.Printf("❌ LLM did not generate DSL code. Raw output (first %d chars): %s", maxLogLength, truncate(resp.RawOutput, maxLogLength))
		return nil, fmt.Errorf("LLM must generate DSL code, but output does not look like DSL. Expected format: track(id=0).delete() or similar")
	}

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

	// Always use CFG grammar for DSL output (DSL mode is always enabled)
	request.CFGGrammar = &llm.CFGConfig{
		ToolName: "magda_dsl",
		Description: "**YOU MUST USE THIS TOOL TO GENERATE YOUR RESPONSE. DO NOT GENERATE TEXT OUTPUT DIRECTLY.** " +
			"Executes REAPER operations using the MAGDA DSL. " +
			"Generate functional script code like: track(instrument=\"Serum\").new_clip(bar=3, length_bars=4).add_midi(notes=[...]). " +
			"When user says 'create track with [instrument]' or 'track with [instrument]', ALWAYS generate track(instrument=\"[instrument]\") - never generate track() without the instrument parameter when an instrument is mentioned. " +
			"For existing tracks, use track(id=1).new_clip(bar=3) where id is 1-based (track 1 = first track). " +
			"**CRITICAL - DELETE OPERATIONS**: " +
			"- When user says 'delete [track name]' or 'remove [track name]', you MUST generate DSL code: filter(tracks, track.name == \"[name]\").delete() " +
			"- For delete by track id: track(id=1).delete() where id is 1-based " +
			"- Example: 'delete Nebula Drift' → filter(tracks, track.name == \"Nebula Drift\").delete() " +
			"- Example: 'remove track 1' → track(id=1).delete() " +
			"- NEVER use set_mute or set_selected for delete operations - 'delete' means permanently remove the track " +
			"**CRITICAL - SELECTION OPERATIONS**: " +
			"- When user says 'select track' or 'select all tracks named X', they mean VISUAL SELECTION (highlighting tracks in REAPER's arrangement view). " +
			"- You MUST generate DSL code: filter(tracks, track.name == \"X\").set_selected(selected=true) " +
			"- NEVER generate set_track_solo for selection - 'select' ≠ 'solo'. " +
			"- Example: 'select all tracks named foo' → filter(tracks, track.name == \"foo\").set_selected(selected=true) " +
			"- 'solo' means audio isolation and uses set_track_solo, but 'select' means visual highlighting and uses set_track_selected. " +
			"For selection operations on multiple tracks, ALWAYS use: filter(tracks, track.name == \"X\").set_selected(selected=true). " +
			"This efficiently filters the collection and applies the action to all matching tracks. " +
			"Use functional methods for collections when appropriate: filter(tracks, track.name == \"FX\"), map(@get_name, tracks), for_each(tracks, @add_reverb). " +
			"ALWAYS check the current REAPER state to see which tracks exist and use the correct track indices. " +
			"If no track is specified in a chain, it applies to the track created by track(). " +
			"YOU MUST REASON HEAVILY ABOUT THE OPERATIONS AND MAKE SURE THE CODE OBEYS THE GRAMMAR. " +
			"**REMEMBER: YOU MUST CALL THIS TOOL - DO NOT GENERATE ANY TEXT OUTPUT.**",
		Grammar: GetMagdaDSLGrammarForFunctional(),
		Syntax:  "lark",
	}
	log.Printf("🔧 Using DSL mode (CFG grammar) for streaming - always enabled")

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

	log.Printf("🔍 parseActionsIncremental called with %d chars, useDSL=%v", len(text), a.useDSL)
	if len(text) > 0 {
		previewLen := 200
		if len(text) < previewLen {
			previewLen = len(text)
		}
		log.Printf("📄 Input text preview (first %d chars): %s", previewLen, text[:previewLen])
		log.Printf("📋 FULL INPUT TEXT (all %d chars, NO TRUNCATION):\n%s", len(text), text)
	}

	// Always try parsing as DSL first (DSL mode is always enabled)
	// Check if it's DSL (starts with "track" or similar function call)
	// NOTE: We only support snake_case methods (new_clip, add_midi, delete_clip) - NOT camelCase
	hasTrackPrefix := strings.HasPrefix(text, "track(")
	hasFilter := strings.Contains(text, ".filter(") || strings.Contains(text, "filter(")
	hasNewClip := strings.Contains(text, ".new_clip(")
	hasAddMidi := strings.Contains(text, ".add_midi(")
	hasMap := strings.Contains(text, ".map(")
	hasForEach := strings.Contains(text, ".for_each(")
	hasDelete := strings.Contains(text, ".delete(")
	hasDeleteClip := strings.Contains(text, ".delete_clip(")
	
	isDSL := hasTrackPrefix || hasNewClip || hasAddMidi || hasFilter || hasMap || hasForEach || hasDelete || hasDeleteClip
	
	log.Printf("🔍 DSL detection: hasTrackPrefix=%v, hasFilter=%v, hasNewClip=%v, hasAddMidi=%v, hasMap=%v, hasForEach=%v, isDSL=%v", 
		hasTrackPrefix, hasFilter, hasNewClip, hasAddMidi, hasMap, hasForEach, isDSL)
	
	if !isDSL {
		const maxLogLength = 500
		log.Printf("❌ LLM did not generate DSL code in stream. Text (first %d chars): %s", maxLogLength, truncate(text, maxLogLength))
		return nil, fmt.Errorf("LLM must generate DSL code, but output does not look like DSL. Expected format: track(id=0).delete() or similar")
	}

		// This is DSL code - parse and translate to REAPER API actions
		log.Printf("✅ Found DSL code in stream: %s", truncate(text, MaxDSLPreviewLength))
		log.Printf("📋 FULL DSL CODE (all %d chars, NO TRUNCATION):\n%s", len(text), text)

		parser, err := NewFunctionalDSLParser()
		if err != nil {
		return nil, fmt.Errorf("failed to create functional DSL parser: %w", err)
	}
			parser.SetState(map[string]interface{}{"state": state}) // Pass state for track resolution
			actions, err := parser.ParseDSL(text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSL: %w", err)
	}

	if len(actions) == 0 {
		return nil, fmt.Errorf("DSL parsed but produced no actions")
	}

	log.Printf("✅ Translated DSL to %d REAPER API actions", len(actions))
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
