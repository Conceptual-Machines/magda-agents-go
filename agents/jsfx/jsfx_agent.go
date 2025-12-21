package jsfx

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/Conceptual-Machines/magda-agents-go/metrics"
	"github.com/getsentry/sentry-go"
)

// JSFXAgent generates JSFX audio effects using LLM with direct EEL2 output
// Based on REAPER JSFX: https://www.reaper.fm/sdk/js/js.php
type JSFXAgent struct {
	provider     llm.Provider
	systemPrompt string
	metrics      *metrics.SentryMetrics
}

// JSFXResult contains the generated JSFX effect
type JSFXResult struct {
	JSFXCode     string `json:"jsfx_code"`               // Complete JSFX file content (direct from LLM)
	Description  string `json:"description,omitempty"`   // Description extracted from code comments
	CompileError string `json:"compile_error,omitempty"` // EEL2 compile error if validation enabled
	Usage        any    `json:"usage"`
}

// parseDescriptionFromCode extracts a description from JSFX code
// It looks for:
// 1. A // DESCRIPTION: ... // END_DESCRIPTION block
// 2. Falls back to extracting from desc: line
// Returns (description, codeWithoutDescriptionBlock)
func parseDescriptionFromCode(code string) (string, string) {
	// First try: look for explicit DESCRIPTION block
	const startMarker = "// DESCRIPTION:"
	const endMarker = "// END_DESCRIPTION"

	startIdx := strings.Index(code, startMarker)
	if startIdx != -1 {
		endIdx := strings.Index(code, endMarker)
		if endIdx != -1 {
			// Extract description block
			descBlock := code[startIdx+len(startMarker) : endIdx]

			// Clean up the description - remove leading "//" from each line
			var descLines []string
			for _, line := range strings.Split(descBlock, "\n") {
				line = strings.TrimSpace(line)
				line = strings.TrimPrefix(line, "//")
				line = strings.TrimSpace(line)
				if line != "" {
					descLines = append(descLines, line)
				}
			}
			description := strings.Join(descLines, " ")

			// Remove the description block from code
			cleanCode := strings.TrimSpace(code[endIdx+len(endMarker):])
			return description, cleanCode
		}
	}

	// Fallback: extract description from desc: line
	// Look for desc: line and use it as description
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "desc:") {
			// Extract the effect name from desc: line
			effectName := strings.TrimPrefix(trimmed, "desc:")
			effectName = strings.TrimSpace(effectName)
			if effectName != "" {
				// Create a simple description from the effect name
				return "Generated JSFX effect: " + effectName, code
			}
			break
		}
	}

	return "", code
}

// NewJSFXAgent creates a new JSFX agent
func NewJSFXAgent(cfg *config.Config) *JSFXAgent {
	return NewJSFXAgentWithProvider(cfg, nil)
}

// NewJSFXAgentWithProvider creates a JSFX agent with a specific LLM provider
func NewJSFXAgentWithProvider(cfg *config.Config, provider llm.Provider) *JSFXAgent {
	// Use provided provider or create OpenAI provider (default)
	if provider == nil {
		provider = llm.NewOpenAIProvider(cfg.OpenAIAPIKey)
	}

	systemPrompt := llm.GetJSFXDirectSystemPrompt()

	agent := &JSFXAgent{
		provider:     provider,
		systemPrompt: systemPrompt,
		metrics:      metrics.NewSentryMetrics(),
	}

	log.Printf("🔧 JSFX AGENT INITIALIZED (Direct EEL2 mode):")
	log.Printf("   Provider: %s", provider.Name())

	return agent
}

// Generate creates JSFX effect code from natural language
func (a *JSFXAgent) Generate(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
) (*JSFXResult, error) {
	startTime := time.Now()
	log.Printf("🔧 JSFX REQUEST STARTED (Model: %s)", model)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "jsfx.generate")
	defer transaction.Finish()

	transaction.SetTag("model", model)

	// Build provider request with CFG grammar for structure validation
	request := &llm.GenerationRequest{
		Model:        model,
		InputArray:   inputArray,
		SystemPrompt: a.systemPrompt,
		CFGGrammar: &llm.CFGConfig{
			ToolName:    "jsfx_generator",
			Description: buildJSFXToolDescription(),
			Grammar:     llm.GetJSFXGrammar(),
			Syntax:      "lark",
		},
	}

	// Call provider
	log.Printf("🚀 JSFX REQUEST: %s model=%s, input_messages=%d",
		a.provider.Name(), model, len(inputArray))

	resp, err := a.provider.Generate(ctx, request)
	if err != nil {
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("provider request failed: %w", err)
	}

	// Extract JSFX code directly from response
	jsfxCode := resp.RawOutput
	if jsfxCode == "" {
		transaction.SetTag("success", "false")
		return nil, fmt.Errorf("no JSFX output in response")
	}

	// Clean up the output (remove any markdown code fences if present)
	jsfxCode = cleanJSFXOutput(jsfxCode)

	log.Printf("🔧 JSFX Output (%d bytes):\n%s", len(jsfxCode), truncateForLog(jsfxCode, 500))

	// Extract description from code comments
	description, cleanCode := parseDescriptionFromCode(jsfxCode)
	if description != "" {
		log.Printf("📝 Extracted description: %s", truncateForLog(description, 200))
	}

	// TODO: Add EEL2 compilation validation here
	// compileErr := validateEEL2(jsfxCode)

	result := &JSFXResult{
		JSFXCode:    cleanCode,
		Description: description,
		Usage:       resp.Usage,
	}

	// Record metrics
	transaction.SetTag("success", "true")

	duration := time.Since(startTime)
	a.metrics.RecordGenerationDuration(ctx, duration, true)

	log.Printf("✅ JSFX COMPLETE: %d bytes of JSFX code", len(jsfxCode))

	return result, nil
}

// cleanJSFXOutput removes markdown code fences, garbage text, and validates output
func cleanJSFXOutput(code string) string {
	code = strings.TrimSpace(code)

	// Remove markdown code fences if present
	if strings.HasPrefix(code, "```") {
		lines := strings.Split(code, "\n")
		if len(lines) > 2 {
			// Remove first line (```jsfx or ```)
			lines = lines[1:]
			// Remove last line if it's just ```
			if strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
			code = strings.Join(lines, "\n")
		}
	}

	// Validate and clean each line
	lines := strings.Split(code, "\n")
	var cleanLines []string

	for _, line := range lines {
		// Check for non-ASCII characters (Korean, Chinese, etc.)
		if containsNonASCII(line) {
			log.Printf("⚠️ JSFX: Removing line with non-ASCII: %s", truncateForLog(line, 50))
			continue
		}

		// Check for garbage patterns (LLM commentary leaking through)
		if isGarbageLine(line) {
			log.Printf("⚠️ JSFX: Removing garbage line: %s", truncateForLog(line, 50))
			continue
		}

		cleanLines = append(cleanLines, line)
	}

	return strings.TrimSpace(strings.Join(cleanLines, "\n"))
}

// containsNonASCII checks if a string contains non-ASCII characters
func containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

// isGarbageLine detects LLM commentary/garbage that leaked into JSFX output
func isGarbageLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Empty lines are fine
	if trimmed == "" {
		return false
	}

	// Comments are fine
	if strings.HasPrefix(trimmed, "//") {
		return false
	}

	// Valid JSFX directives
	validPrefixes := []string{
		"desc:", "tags:", "in_pin:", "out_pin:", "slider", "import", "options:", "filename:",
		"@init", "@slider", "@block", "@sample", "@serialize", "@gfx",
	}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return false
		}
	}

	// Lines that are clearly code (contain operators, assignments, function calls)
	// EEL2 code patterns
	codePatterns := []string{
		"=", ";", "(", ")", "[", "]", "+", "-", "*", "/", "%", "^", "|", "&",
		"?", ":", "<", ">", "!", "~",
	}
	for _, pattern := range codePatterns {
		if strings.Contains(trimmed, pattern) {
			// But check for obvious English sentences
			if looksLikeSentence(trimmed) {
				return true
			}
			return false
		}
	}

	// Single words that could be variable names or EEL2 code
	if !strings.Contains(trimmed, " ") && len(trimmed) < 50 {
		return false
	}

	// If it looks like an English sentence, it's garbage
	if looksLikeSentence(trimmed) {
		return true
	}

	return false
}

// looksLikeSentence checks if a line looks like English prose rather than code
func looksLikeSentence(line string) bool {
	lower := strings.ToLower(line)

	// Common English words that shouldn't appear in JSFX code
	sentencePatterns := []string{
		"the ", " the ", " is ", " are ", " was ", " were ",
		" to ", " for ", " with ", " that ", " this ",
		" you ", " your ", " make ", " ensure ", " please ",
		" not ", " don't ", " doesn't ", " can't ", " won't ",
		"commentary", "comment", "algorithm", "functionality",
		"include", "optional", "necessary", "needed",
	}

	for _, pattern := range sentencePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	// Starts with capital letter and has multiple spaces = likely prose
	if len(line) > 20 && line[0] >= 'A' && line[0] <= 'Z' {
		spaceCount := strings.Count(line, " ")
		if spaceCount > 3 {
			return true
		}
	}

	return false
}

// truncateForLog truncates a string for logging
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// JSFXStreamCallback is called for each chunk of generated JSFX code as it arrives
type JSFXStreamCallback func(chunk string) error

// GenerateStream creates JSFX effect code with true real-time streaming from the LLM
// Each chunk of JSFX code is streamed to the callback as it's generated
func (a *JSFXAgent) GenerateStream(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
	callback JSFXStreamCallback,
) (*JSFXResult, error) {
	startTime := time.Now()
	log.Printf("🔧 JSFX STREAMING REQUEST STARTED (Model: %s)", model)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "jsfx.generate_stream")
	defer transaction.Finish()

	transaction.SetTag("model", model)
	transaction.SetTag("streaming", "true")

	// Check if provider supports streaming
	streamingProvider, ok := a.provider.(llm.StreamingProvider)
	if !ok {
		// Fall back to non-streaming with simulated streaming output
		log.Printf("⚠️ Provider %s does not support streaming, falling back to non-streaming", a.provider.Name())
		return a.generateStreamFallback(ctx, model, inputArray, callback)
	}

	// Build provider request with CFG grammar for structure validation
	request := &llm.GenerationRequest{
		Model:        model,
		InputArray:   inputArray,
		SystemPrompt: a.systemPrompt,
		CFGGrammar: &llm.CFGConfig{
			ToolName:    "jsfx_generator",
			Description: buildJSFXToolDescription(),
			Grammar:     llm.GetJSFXGrammar(),
			Syntax:      "lark",
		},
	}

	// Stream callback adapter - converts LLM stream events to JSFX chunks
	var accumulatedCode string
	streamCallback := func(event llm.StreamEvent) error {
		switch event.Type {
		case "text_delta", "chunk":
			// Stream the text delta directly to the JSFX callback
			// OpenAI provider sends "chunk", others may send "text_delta"
			chunk := event.Message
			if chunk != "" && callback != nil {
				accumulatedCode += chunk
				if err := callback(chunk); err != nil {
					log.Printf("⚠️ JSFX Stream callback error: %v", err)
				}
			}
		case "started":
			log.Printf("🚀 JSFX streaming started")
		case "completed":
			log.Printf("✅ JSFX streaming completed: %d chars", len(accumulatedCode))
		case "heartbeat":
			// Could forward heartbeat to client if needed
		}
		return nil
	}

	// Call streaming provider
	log.Printf("🚀 JSFX STREAMING REQUEST: %s model=%s, input_messages=%d",
		a.provider.Name(), model, len(inputArray))

	resp, err := streamingProvider.GenerateStream(ctx, request, streamCallback)
	if err != nil {
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("streaming provider request failed: %w", err)
	}

	// Extract JSFX code from response (already accumulated during streaming)
	jsfxCode := resp.RawOutput
	if jsfxCode == "" {
		jsfxCode = accumulatedCode
	}

	if jsfxCode == "" {
		transaction.SetTag("success", "false")
		return nil, fmt.Errorf("no JSFX output in streaming response")
	}

	// Clean up the output (remove any markdown code fences if present)
	jsfxCode = cleanJSFXOutput(jsfxCode)

	log.Printf("🔧 JSFX Streaming Output (%d bytes):\n%s", len(jsfxCode), truncateForLog(jsfxCode, 500))

	// Extract description from code comments
	description, cleanCode := parseDescriptionFromCode(jsfxCode)
	if description != "" {
		log.Printf("📝 Extracted description: %s", truncateForLog(description, 200))
	}

	result := &JSFXResult{
		JSFXCode:    cleanCode,
		Description: description,
		Usage:       resp.Usage,
	}

	// Record metrics
	transaction.SetTag("success", "true")

	duration := time.Since(startTime)
	a.metrics.RecordGenerationDuration(ctx, duration, true)

	log.Printf("✅ JSFX STREAMING COMPLETE: %d bytes of JSFX code in %v", len(jsfxCode), duration)

	return result, nil
}

// generateStreamFallback is used when the provider doesn't support streaming
// It generates the full response then streams it back line by line (simulated streaming)
func (a *JSFXAgent) generateStreamFallback(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
	callback JSFXStreamCallback,
) (*JSFXResult, error) {
	// Generate the full response
	result, err := a.Generate(ctx, model, inputArray)
	if err != nil {
		return nil, err
	}

	// Stream the response back line by line (simulated)
	if callback != nil && result.JSFXCode != "" {
		lines := strings.Split(result.JSFXCode, "\n")
		for _, line := range lines {
			if err := callback(line + "\n"); err != nil {
				log.Printf("⚠️ JSFX Stream callback error: %v", err)
			}
		}
	}

	return result, nil
}

// DescribeJSFX generates a natural language description of JSFX code
// This is a separate call that can be optionally made after generation
// based on user preference. Uses plain text output (no schema) for speed/cost.
func (a *JSFXAgent) DescribeJSFX(
	ctx context.Context,
	model string,
	jsfxCode string,
) (string, error) {
	log.Printf("📝 JSFX DESCRIBE REQUEST (Model: %s)", model)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "jsfx.describe")
	defer transaction.Finish()

	// Build simple plain text request - no schema needed for descriptions
	request := &llm.GenerationRequest{
		Model: model,
		InputArray: []map[string]any{
			{
				"role": "user",
				"content": fmt.Sprintf(`Describe this JSFX audio effect in 2-3 sentences.
Explain what it does, its main controls, and typical use cases.
Be concise and practical. Output only the description, no code or formatting.

JSFX Code:
%s`, jsfxCode),
			},
		},
		SystemPrompt: "You are a helpful audio engineering assistant. Provide brief, clear descriptions of audio effects. Output only plain text descriptions.",
	}

	resp, err := a.provider.Generate(ctx, request)
	if err != nil {
		log.Printf("❌ JSFX Describe error: %v", err)
		return "", fmt.Errorf("failed to generate description: %w", err)
	}

	description := strings.TrimSpace(resp.RawOutput)
	log.Printf("✅ JSFX Description: %s", truncateForLog(description, 200))

	return description, nil
}

// GenerateWithDescription generates JSFX code and then describes it
// This is a convenience method that combines Generate + DescribeJSFX
func (a *JSFXAgent) GenerateWithDescription(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
) (*JSFXResult, error) {
	// First generate the code
	result, err := a.Generate(ctx, model, inputArray)
	if err != nil {
		return nil, err
	}

	// Then generate the description
	if result.JSFXCode != "" {
		description, descErr := a.DescribeJSFX(ctx, model, result.JSFXCode)
		if descErr != nil {
			log.Printf("⚠️ Failed to generate description: %v", descErr)
			// Don't fail the whole request if description fails
		} else {
			result.Description = description
		}
	}

	return result, nil
}

// buildJSFXToolDescription creates the tool description for CFG
func buildJSFXToolDescription() string {
	return `Generate complete JSFX audio effects for REAPER.
Output raw JSFX/EEL2 code that can be saved directly as a .jsfx file.

Structure:
desc:Effect Name
tags:category
in_pin:Left / in_pin:Right
out_pin:Left / out_pin:Right
slider1:var=default<min,max,step>Label
@init (initialization)
@slider (parameter changes)
@sample (per-sample processing)
@gfx (optional graphics)

Effect types: filter, compressor, limiter, eq, distortion, delay, reverb, chorus, modulation, utility
Audio vars: spl0-spl63, srate, samplesblock
Math: sin, cos, log, exp, pow, sqrt, abs, min, max, $pi`
}
