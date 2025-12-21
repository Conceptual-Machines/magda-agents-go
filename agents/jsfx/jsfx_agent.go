package jsfx

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

// JSFXAgent generates JSFX audio effects using LLM + CFG grammar
// Based on REAPER JSFX: https://www.reaper.fm/sdk/js/js.php
type JSFXAgent struct {
	provider     llm.Provider
	systemPrompt string
	metrics      *metrics.SentryMetrics
}

// JSFXResult contains the generated JSFX effect
type JSFXResult struct {
	DSL        string `json:"dsl"`                  // Raw DSL code from LLM
	JSFXCode   string `json:"jsfx_code"`            // Complete JSFX file content
	ParseError string `json:"parse_error,omitempty"` // Parser error if any (for human-in-the-loop)
	Usage      any    `json:"usage"`
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

	systemPrompt := llm.GetJSFXSystemPrompt()

	agent := &JSFXAgent{
		provider:     provider,
		systemPrompt: systemPrompt,
		metrics:      metrics.NewSentryMetrics(),
	}

	log.Printf("🔧 JSFX AGENT INITIALIZED:")
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

	// Build provider request with CFG grammar
	request := &llm.GenerationRequest{
		Model:        model,
		InputArray:   inputArray,
		SystemPrompt: a.systemPrompt,
		CFGGrammar: &llm.CFGConfig{
			ToolName:    "jsfx_dsl",
			Description: buildJSFXToolDescription(),
			Grammar:     llm.GetJSFXDSLGrammar(),
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

	// Extract DSL from response
	dslCode := resp.RawOutput
	if dslCode == "" {
		transaction.SetTag("success", "false")
		return nil, fmt.Errorf("no DSL output in response")
	}

	log.Printf("🔧 DSL Output: %s", dslCode)

	// Parse DSL and generate JSFX code directly (using Grammar School for validation)
	// On parse errors, return the DSL + error for human-in-the-loop feedback
	parser, err := NewJSFXDSLParser()
	if err != nil {
		transaction.SetTag("success", "false")
		log.Printf("⚠️ JSFX Parser creation failed: %v", err)
		return &JSFXResult{
			DSL:        dslCode,
			JSFXCode:   "",
			ParseError: fmt.Sprintf("Parser initialization failed: %v", err),
			Usage:      resp.Usage,
		}, nil // Return result with error, not a fatal error
	}

	jsfxCode, err := parser.ParseDSL(dslCode)
	if err != nil {
		transaction.SetTag("success", "partial")
		log.Printf("⚠️ JSFX DSL parse failed: %v", err)
		// Return DSL + error so user can provide feedback
		return &JSFXResult{
			DSL:        dslCode,
			JSFXCode:   "",
			ParseError: fmt.Sprintf("DSL parsing failed: %v", err),
			Usage:      resp.Usage,
		}, nil // Return result with error, not a fatal error
	}

	result := &JSFXResult{
		DSL:      dslCode,
		JSFXCode: jsfxCode,
		Usage:    resp.Usage,
	}

	// Record metrics
	transaction.SetTag("success", "true")

	duration := time.Since(startTime)
	a.metrics.RecordGenerationDuration(ctx, duration, true)

	log.Printf("✅ JSFX COMPLETE: %d bytes of DSL, %d bytes of JSFX code", len(dslCode), len(jsfxCode))

	return result, nil
}

// buildJSFXToolDescription creates the tool description for CFG
func buildJSFXToolDescription() string {
	return `Generate JSFX audio effects for REAPER. Define the effect structure using DSL:

effect(name="Effect Name", tags="category");
slider(id=1, default=0, min=-60, max=0, step=0.1, name="Gain (dB)");
init_code(code="gain = 1;");
slider_code(code="gain = 10^(slider1/20);");
sample_code(code="spl0 *= gain; spl1 *= gain;")

Effect types: compressor, limiter, eq, filter, distortion, delay, reverb, chorus, flanger, phaser, gate
Audio vars: spl0-spl63, srate, samplesblock
Math: sin, cos, log, exp, pow, sqrt, abs, min, max`
}
