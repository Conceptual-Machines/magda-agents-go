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
	Actions []map[string]any `json:"actions"`
	Usage   any              `json:"usage"`
}

// getCFGGrammarConfig returns the CFG grammar configuration for the DAW agent
// This is shared between GenerateActions and GenerateActionsStream to avoid duplication
func (a *DawAgent) getCFGGrammarConfig() *llm.CFGConfig {
	return &llm.CFGConfig{
		ToolName: "magda_dsl",
		Description: "**YOU MUST USE THIS TOOL TO GENERATE YOUR RESPONSE. DO NOT GENERATE TEXT OUTPUT DIRECTLY.** " +
			"Executes REAPER operations using the MAGDA DSL. " +
			"Generate functional script code like: track(instrument=\"Serum\").new_clip(bar=3, length_bars=4). " +
			"Your job is to create tracks, clips, and set track properties. " +
			"**IMPORTANT**: Musical content (notes, chords, arpeggios, progressions) is handled by the ARRANGER agent, NOT you. " +
			"When user requests musical content like 'add E1 note', 'sustained note', 'chord', 'arpeggio', just create the track/clip structure - the arranger will add the notes. " +
			"When user says 'create track with [instrument]' or 'track with [instrument]', ALWAYS generate track(instrument=\"[instrument]\") - never generate track() without the instrument parameter when an instrument is mentioned. " +
			"**TRACK CREATION**: To create a new track, use track() or track(name=\"Track Name\") - DO NOT chain .set_track() after track() unless you explicitly need to set a property. For simple track creation, track() or track(name=\"...\") is sufficient. " +
			"**MULTIPLE TRACK CREATION**: When user requests multiple tracks (e.g., 'create 5 tracks'), generate separate track() calls: track(); track(); track(); track(); track(). For named tracks: track(name=\"Track 1\"); track(name=\"Track 2\"); etc. Each track() call creates ONE track - do NOT chain .set_track() unless explicitly needed. " +
			"**RANDOM VALUES**: When user requests 'random' (names, positions, values, etc.), generate varied, diverse values instead of sequential or predictable ones. For random names: use creative, varied names (e.g., 'Aurora', 'Nebula', 'Phoenix', 'Echo', 'Vortex') not sequential like 'Track 1', 'Track 2'. For random positions: use varied bar positions (e.g., bar=3, bar=7, bar=12) not sequential. Make each value truly different and varied. " +
			"For existing tracks, use track(id=1).new_clip(bar=3) where id is 1-based (track 1 = first track). " +
			"**CRITICAL - DELETE OPERATIONS**: " +
			"- When user says 'delete [track name]' or 'remove [track name]', you MUST generate DSL code: filter(tracks, track.name == \"[name]\").delete() " +
			"- For delete by track id: track(id=1).delete() where id is 1-based " +
			"- Example: 'delete Nebula Drift' → filter(tracks, track.name == \"Nebula Drift\").delete() " +
			"- Example: 'remove track 1' → track(id=1).delete() " +
			"- NEVER use set_track(mute=true) or set_track(selected=true) for delete operations - 'delete' means permanently remove the track " +
			"**CRITICAL - SELECTION OPERATIONS**: " +
			"- When user says 'select track' or 'select all tracks named X', they mean VISUAL SELECTION (highlighting tracks in REAPER's arrangement view). " +
			"- You MUST generate DSL code: filter(tracks, track.name == \"X\").set_track(selected=true) " +
			"- NEVER generate set_track(solo=true) for selection - 'select' ≠ 'solo'. " +
			"- Example: 'select all tracks named foo' → filter(tracks, track.name == \"foo\").set_track(selected=true) " +
			"- 'solo' means audio isolation and uses set_track(solo=true), but 'select' means visual highlighting and uses set_track(selected=true). " +
			"For selection operations on multiple tracks, ALWAYS use: filter(tracks, track.name == \"X\").set_track(selected=true). " +
			"This efficiently filters the collection and applies the action to all matching tracks. " +
			"Use functional methods for collections when appropriate: filter(tracks, track.name == \"FX\"), map(@get_name, tracks), for_each(tracks, @add_reverb). " +
			"ALWAYS check the current REAPER state to see which tracks exist and use the correct track indices. " +
			"If no track is specified in a chain, it applies to the track created by track(). " +
			"YOU MUST REASON HEAVILY ABOUT THE OPERATIONS AND MAKE SURE THE CODE OBEYS THE GRAMMAR. " +
			"**REMEMBER: YOU MUST CALL THIS TOOL - DO NOT GENERATE ANY TEXT OUTPUT.**",
		Grammar: GetMagdaDSLGrammarForFunctional(),
		Syntax:  "lark",
	}
}

func (a *DawAgent) GenerateActions(
	ctx context.Context, question string, state map[string]any,
) (*DawResult, error) {
	startTime := time.Now()
	log.Printf("🤖 MAGDA REQUEST STARTED: question=%s", question)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "magda.generate_actions")
	defer transaction.Finish()

	transaction.SetTag("model", "gpt-5.1") // GPT-5.1 for MAGDA
	transaction.SetContext("magda", map[string]any{
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
	request.CFGGrammar = a.getCFGGrammarConfig()
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
func (a *DawAgent) buildInputMessages(question string, state map[string]any) []map[string]any {
	messages := []map[string]any{}

	// Add user question
	userMessage := map[string]any{
		"role":    "user",
		"content": question,
	}
	messages = append(messages, userMessage)

	// Add REAPER state if provided
	if len(state) > 0 {
		stateMessage := map[string]any{
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
func (a *DawAgent) parseActionsFromResponse(resp *llm.GenerationResponse, state map[string]any) ([]map[string]any, error) {
	// The provider should have stored the raw output (DSL or JSON) in RawOutput
	if resp.RawOutput == "" {
		return nil, fmt.Errorf("no raw output available in response")
	}

	// Parse as DSL only - no fallback to JSON
	dslCode := strings.TrimSpace(resp.RawOutput)

	// Check for out-of-scope error comments
	if strings.HasPrefix(dslCode, "// ERROR:") {
		errorMsg := strings.TrimPrefix(dslCode, "// ERROR:")
		errorMsg = strings.TrimSpace(errorMsg)
		return nil, fmt.Errorf("request is out of scope: %s", errorMsg)
	}

	// Check if it's DSL (starts with "track" or similar function call)
	// NOTE: We only support snake_case methods (new_clip, delete_clip) - NOT camelCase
	// NOTE: add_midi is NOT generated by DAW agent - arranger agent handles MIDI notes
	hasTrackPrefix := strings.HasPrefix(dslCode, "track(")
	hasFilter := strings.HasPrefix(dslCode, "filter(") || strings.Contains(dslCode, ".filter(")
	hasMap := strings.HasPrefix(dslCode, "map(") || strings.Contains(dslCode, ".map(")
	hasForEach := strings.HasPrefix(dslCode, "for_each(") || strings.Contains(dslCode, ".for_each(")
	hasNewClip := strings.Contains(dslCode, ".new_clip(")
	hasDelete := strings.Contains(dslCode, ".delete(")
	hasDeleteClip := strings.Contains(dslCode, ".delete_clip(")
	hasSetTrack := strings.Contains(dslCode, ".set_track(")
	hasSetClip := strings.Contains(dslCode, ".set_clip(")
	hasAddFx := strings.Contains(dslCode, ".add_fx(")

	isDSL := hasTrackPrefix || hasNewClip || hasFilter || hasMap || hasForEach || hasDelete || hasDeleteClip ||
		hasSetTrack || hasSetClip || hasAddFx

	if !isDSL {
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
	// Pass state directly - SetState handles both {"state": {...}} and {...} formats
	parser.SetState(state)
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
type StreamActionCallback func(action map[string]any) error

// GenerateActionsStream generates actions using streaming (without structured output)
// It parses JSON incrementally from the text stream and calls callback for each action found
func (a *DawAgent) GenerateActionsStream(
	ctx context.Context,
	question string,
	state map[string]any,
	callback StreamActionCallback,
) (*DawResult, error) {
	startTime := time.Now()
	log.Printf("🤖 MAGDA STREAMING REQUEST STARTED: question=%s", question)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "magda.generate_actions_stream")
	defer transaction.Finish()

	transaction.SetTag("model", "gpt-5.1")
	transaction.SetTag("streaming", "false")
	transaction.SetContext("magda", map[string]any{
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
	request.CFGGrammar = a.getCFGGrammarConfig()
	log.Printf("🔧 Using DSL mode (CFG grammar) - always enabled")

	// Call non-streaming provider
	log.Printf("🚀 MAGDA PROVIDER REQUEST: %s", a.provider.Name())
	resp, err := a.provider.Generate(ctx, request)

	if err != nil {
		transaction.SetTag("success", "false")
		transaction.SetTag("error_type", "provider_error")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("provider failed: %w", err)
	}

	// Extract DSL code from response
	if resp == nil || resp.RawOutput == "" {
		transaction.SetTag("success", "false")
		transaction.SetTag("error_type", "no_output")
		return nil, fmt.Errorf("no DSL output from provider")
	}

	// Parse DSL code into actions
	allActions, err := a.parseActionsIncremental(resp.RawOutput, state)
	if err != nil {
		transaction.SetTag("success", "false")
		transaction.SetTag("error_type", "parse_error")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("failed to parse DSL: %w", err)
	}

	// Call callback for each action
	for _, action := range allActions {
		_ = callback(action)
	}

	if len(allActions) == 0 {
		transaction.SetTag("success", "false")
		transaction.SetTag("error_type", "no_actions")
		return nil, fmt.Errorf("no actions found in DSL output")
	}

	result := &DawResult{
		Actions: allActions,
		Usage:   nil,
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
func (a *DawAgent) parseActionsIncremental(text string, state map[string]any) ([]map[string]any, error) {
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
	// NOTE: We only support snake_case methods (new_clip, delete_clip) - NOT camelCase
	// NOTE: add_midi is NOT generated by DAW agent - arranger agent handles MIDI notes
	hasTrackPrefix := strings.HasPrefix(text, "track(")
	hasFilter := strings.Contains(text, ".filter(") || strings.Contains(text, "filter(")
	hasNewClip := strings.Contains(text, ".new_clip(")
	hasMap := strings.Contains(text, ".map(")
	hasForEach := strings.Contains(text, ".for_each(")
	hasDelete := strings.Contains(text, ".delete(")
	hasDeleteClip := strings.Contains(text, ".delete_clip(")
	hasSetTrack := strings.Contains(text, ".set_track(")
	hasSetClip := strings.Contains(text, ".set_clip(")
	hasAddFx := strings.Contains(text, ".add_fx(")

	isDSL := hasTrackPrefix || hasNewClip || hasFilter || hasMap || hasForEach || hasDelete || hasDeleteClip ||
		hasSetTrack || hasSetClip || hasAddFx

	log.Printf("🔍 DSL detection: hasTrackPrefix=%v, hasFilter=%v, hasNewClip=%v, hasMap=%v, hasForEach=%v, hasSetTrack=%v, hasSetClip=%v, hasAddFx=%v, isDSL=%v",
		hasTrackPrefix, hasFilter, hasNewClip, hasMap, hasForEach, hasSetTrack, hasSetClip, hasAddFx, isDSL)

	// Check for out-of-scope error comments
	if strings.HasPrefix(text, "// ERROR:") {
		errorMsg := strings.TrimPrefix(text, "// ERROR:")
		errorMsg = strings.TrimSpace(errorMsg)
		return nil, fmt.Errorf("request is out of scope: %s", errorMsg)
	}

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
	// Pass state directly - SetState handles both {"state": {...}} and {...} formats
	parser.SetState(state)
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
