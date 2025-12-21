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

// JSFXAgent generates JSFX audio effects using LLM + CFG grammar
// Based on REAPER JSFX: https://www.reaper.fm/sdk/js/js.php
type JSFXAgent struct {
	provider     llm.Provider
	systemPrompt string
	metrics      *metrics.SentryMetrics
}

// JSFXResult contains the generated JSFX effect
type JSFXResult struct {
	DSL      string           `json:"dsl"`       // Raw DSL code from LLM
	Actions  []map[string]any `json:"actions"`   // Parsed actions from Grammar School
	JSFXCode string           `json:"jsfx_code"` // Complete JSFX file content
	Usage    any              `json:"usage"`
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

	// Parse DSL using Grammar School to get actions
	parser, err := NewJSFXDSLParser()
	if err != nil {
		transaction.SetTag("success", "false")
		return nil, fmt.Errorf("failed to create DSL parser: %w", err)
	}

	actions, err := parser.ParseDSL(dslCode)
	if err != nil {
		transaction.SetTag("success", "false")
		return nil, fmt.Errorf("failed to parse DSL: %w", err)
	}

	// Generate complete JSFX code from actions
	jsfxCode := generateJSFXCode(actions)

	result := &JSFXResult{
		DSL:      dslCode,
		Actions:  actions,
		JSFXCode: jsfxCode,
		Usage:    resp.Usage,
	}

	// Record metrics
	transaction.SetTag("success", "true")
	transaction.SetTag("action_count", fmt.Sprintf("%d", len(actions)))

	duration := time.Since(startTime)
	a.metrics.RecordGenerationDuration(ctx, duration, true)

	log.Printf("✅ JSFX COMPLETE: %d actions, %d bytes of code", len(actions), len(jsfxCode))

	return result, nil
}

// generateJSFXCode converts parsed actions into a complete JSFX file
func generateJSFXCode(actions []map[string]any) string {
	var sb strings.Builder

	effectName := "AI Generated Effect"
	effectTags := "utility"
	var sliders []map[string]any
	initCode := ""
	sliderCode := ""
	sampleCode := ""
	blockCode := ""
	gfxCode := ""
	gfxWidth := 0
	gfxHeight := 0

	// Process actions
	for _, action := range actions {
		switch action["action"] {
		case "effect":
			if name, ok := action["name"].(string); ok {
				effectName = name
			}
			if tags, ok := action["tags"].(string); ok {
				effectTags = tags
			}
		case "slider":
			sliders = append(sliders, action)
		case "init_code":
			if code, ok := action["code"].(string); ok {
				initCode = code
			}
		case "slider_code":
			if code, ok := action["code"].(string); ok {
				sliderCode = code
			}
		case "sample_code":
			if code, ok := action["code"].(string); ok {
				sampleCode = code
			}
		case "block_code":
			if code, ok := action["code"].(string); ok {
				blockCode = code
			}
		case "gfx_code":
			if code, ok := action["code"].(string); ok {
				gfxCode = code
			}
			if w, ok := action["width"].(float64); ok {
				gfxWidth = int(w)
			}
			if h, ok := action["height"].(float64); ok {
				gfxHeight = int(h)
			}
		}
	}

	// Build JSFX file
	sb.WriteString(fmt.Sprintf("desc:%s\n", effectName))
	sb.WriteString(fmt.Sprintf("tags:%s\n", effectTags))
	sb.WriteString("\n")

	// Sliders
	for _, slider := range sliders {
		id := int(slider["id"].(float64))
		defaultVal := slider["default"].(float64)
		minVal := slider["min"].(float64)
		maxVal := slider["max"].(float64)
		step := 1.0
		if s, ok := slider["step"].(float64); ok {
			step = s
		}
		name := "Parameter"
		if n, ok := slider["name"].(string); ok {
			name = n
		}
		hidden := ""
		if h, ok := slider["hidden"].(bool); ok && h {
			hidden = "-"
		}
		varName := ""
		if v, ok := slider["var"].(string); ok {
			varName = v + "="
		}

		sb.WriteString(fmt.Sprintf("slider%d:%s%.2f<%.2f,%.2f,%.4f>%s%s\n",
			id, varName, defaultVal, minVal, maxVal, step, hidden, name))
	}

	// @init section
	if initCode != "" {
		sb.WriteString("\n@init\n")
		sb.WriteString(unescapeCode(initCode))
		sb.WriteString("\n")
	}

	// @slider section
	if sliderCode != "" {
		sb.WriteString("\n@slider\n")
		sb.WriteString(unescapeCode(sliderCode))
		sb.WriteString("\n")
	}

	// @block section
	if blockCode != "" {
		sb.WriteString("\n@block\n")
		sb.WriteString(unescapeCode(blockCode))
		sb.WriteString("\n")
	}

	// @sample section
	if sampleCode != "" {
		sb.WriteString("\n@sample\n")
		sb.WriteString(unescapeCode(sampleCode))
		sb.WriteString("\n")
	}

	// @gfx section
	if gfxCode != "" {
		if gfxWidth > 0 && gfxHeight > 0 {
			sb.WriteString(fmt.Sprintf("\n@gfx %d %d\n", gfxWidth, gfxHeight))
		} else {
			sb.WriteString("\n@gfx\n")
		}
		sb.WriteString(unescapeCode(gfxCode))
		sb.WriteString("\n")
	}

	return sb.String()
}

// unescapeCode converts escaped strings back to normal code
func unescapeCode(code string) string {
	// Remove surrounding quotes if present
	code = strings.Trim(code, "\"")
	// Unescape common escapes
	code = strings.ReplaceAll(code, "\\n", "\n")
	code = strings.ReplaceAll(code, "\\t", "\t")
	code = strings.ReplaceAll(code, "\\\"", "\"")
	return code
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
