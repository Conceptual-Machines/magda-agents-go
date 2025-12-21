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
	CompileError string `json:"compile_error,omitempty"` // EEL2 compile error if validation enabled
	Usage        any    `json:"usage"`
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

	// TODO: Add EEL2 compilation validation here
	// compileErr := validateEEL2(jsfxCode)

	result := &JSFXResult{
		JSFXCode: jsfxCode,
		Usage:    resp.Usage,
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

// JSFXStreamCallback is called for each line of generated JSFX code
type JSFXStreamCallback func(line string) error

// GenerateStream creates JSFX effect code with progressive line-by-line streaming
// The LLM call is non-streaming, but the response is streamed back line by line
func (a *JSFXAgent) GenerateStream(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
	callback JSFXStreamCallback,
) (*JSFXResult, error) {
	// First, generate the full response using the normal method
	result, err := a.Generate(ctx, model, inputArray)
	if err != nil {
		return nil, err
	}

	// Now stream the response back line by line
	if callback != nil && result.JSFXCode != "" {
		lines := strings.Split(result.JSFXCode, "\n")
		for _, line := range lines {
			if err := callback(line); err != nil {
				log.Printf("⚠️ JSFX Stream callback error: %v", err)
				// Continue anyway - don't fail the whole generation
			}
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
