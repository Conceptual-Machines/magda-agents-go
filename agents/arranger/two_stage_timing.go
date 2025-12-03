package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/Conceptual-Machines/magda-agents-go/pkg/embedded"
)

// TimingSlot represents a timing position in the skeleton
type TimingSlot struct {
	SlotID        int     `json:"slot_id"`
	StartBeats    float64 `json:"startBeats"`
	DurationBeats float64 `json:"durationBeats"`
	Source        string  `json:"source"` // "original" or "continuation"
}

// TimingSkeleton represents the timing structure
type TimingSkeleton struct {
	PatternDescription string       `json:"pattern_description"`
	Slots              []TimingSlot `json:"slots"`
}

// GenerateTwoStageTiming generates music using timing-first two-stage approach (non-streaming)
func (s *GenerationService) GenerateTwoStageTiming(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
	variations int,
) (*GenerationResult, error) {
	return s.GenerateTwoStageTimingStream(ctx, model, inputArray, variations, nil, reasoningEffortMedium, reasoningEffortMedium)
}

// GenerateTwoStageTimingStream generates music using timing-first two-stage approach with streaming
func (s *GenerationService) GenerateTwoStageTimingStream(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
	variations int,
	callback StreamCallback,
	harmonicReasoning string,
	rhythmicReasoning string,
) (*GenerationResult, error) {
	if harmonicReasoning == "" {
		harmonicReasoning = reasoningEffortMedium
	}
	if rhythmicReasoning == "" {
		rhythmicReasoning = reasoningEffortMedium
	}

	log.Printf("🎵 TWO-STAGE TIMING GENERATION STARTED (Model: %s, Stage1 Reasoning: %s, Stage2 Reasoning: %s)",
		model, harmonicReasoning, rhythmicReasoning)

	// Stage 1: Fill in harmony (higher reasoning, with MCP)
	stage1Start := time.Now()
	if callback != nil {
		_ = callback(StreamEvent{Type: "progress", Message: "Stage 1: Musical placement..."})
	}

	// Stage 1: Harmonic enrichment (doesn't need to return structured data)
	err := s.enrichWithHarmonyStage1(ctx, model, inputArray, variations, callback, harmonicReasoning)
	stage1Duration := time.Since(stage1Start)
	if err != nil {
		return nil, fmt.Errorf("stage 1 (harmonic enrichment) failed: %w", err)
	}

	log.Printf("✅ Stage 1 complete: Musical placement (took %v)", stage1Duration)
	if callback != nil {
		stage1Rounded := stage1Duration.Round(time.Second)
		_ = callback(StreamEvent{Type: "progress", Message: fmt.Sprintf("✅ Stage 1 complete: Musical placement (took %v)", stage1Rounded)})
	}

	// Stage 2: Create timing/rhythmical placement (returns structured MusicalOutput schema)
	stage2Start := time.Now()
	if callback != nil {
		_ = callback(StreamEvent{Type: "progress", Message: "Stage 2: Rhythmical placement..."})
	}

	timingResult, err := s.createTimingSkeleton(ctx, model, inputArray, callback, rhythmicReasoning)
	stage2Duration := time.Since(stage2Start)
	if err != nil {
		return nil, fmt.Errorf("stage 2 (timing skeleton) failed: %w", err)
	}

	log.Printf("✅ Stage 2 complete: Rhythmical placement - generated %d variations (took %v)",
		len(timingResult.OutputParsed.Choices), stage2Duration)
	if callback != nil {
		stage2Rounded := stage2Duration.Round(time.Second)
		totalDuration := time.Since(stage1Start) // Total time including both stages
		_ = callback(StreamEvent{
			Type: "progress",
			Message: fmt.Sprintf("✅ Stage 2 complete: Rhythmical placement (took %v, total: %v)",
				stage2Rounded, totalDuration.Round(time.Second)),
		})

		// Result event was already sent in createTimingSkeleton after stream completes
		// No need to send it again here - it's already in the stream
		log.Printf("ℹ️  Result event was sent during Stage 2 stream completion")
	}

	return timingResult, nil
}

// createTimingSkeleton creates the timing skeleton with structured MusicalOutput (Stage 2)
// Returns structured data that the plugin can parse
func (s *GenerationService) createTimingSkeleton(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
	callback StreamCallback,
	reasoningMode string,
) (*GenerationResult, error) {
	// Build timing skeleton prompt - combine base system prompt with stage-specific instructions
	timingPrompt := string(embedded.RhythmicPlacementPromptTxt)

	// Use the full system prompt that includes all musical knowledge, then append stage-specific instructions
	systemPrompt := s.systemPrompt + "\n\n## STAGE 2: RHYTHMIC PATTERN ANALYSIS\n\n" + timingPrompt

	// Use reasoning mode from request (default to "medium" if empty)
	if reasoningMode == "" {
		reasoningMode = reasoningEffortMedium
	}

	// Pass the input array directly to the provider (same as one-shot mode)
	request := &llm.GenerationRequest{
		Model:         model,
		InputArray:    inputArray,
		ReasoningMode: reasoningMode,
		SystemPrompt:  systemPrompt,
		OutputSchema: &llm.OutputSchema{
			Name:        "MusicalOutput",
			Description: "Musical composition output",
			Schema:      llm.GetMusicalOutputSchema(),
		},
	}

	log.Printf("🎯 Stage 2 (Timing): Calling provider with %s reasoning", reasoningMode)

	// Use streaming for Stage 2 - forward all events including "completed"
	streamCallback := s.buildStage2StreamCallback(callback)
	resp, err := s.provider.GenerateStream(ctx, request, streamCallback)
	if err != nil {
		// Check if context was cancelled (client disconnected)
		if ctx.Err() != nil {
			log.Printf("⚠️  Stage 2 context cancelled (client may have disconnected): %v", ctx.Err())
		}
		return nil, fmt.Errorf("timing skeleton generation failed: %w", err)
	}

	// Log detailed response info for debugging
	log.Printf("🔍 Stage 2 response details: resp=%v, choices=%d, usage=%v", resp != nil, len(resp.OutputParsed.Choices), resp.Usage != nil)
	if resp.Usage != nil {
		log.Printf("📊 Stage 2 usage: %+v", resp.Usage)
	}

	// Parse the timing skeleton - the AI should encode it in the first choice's description
	if len(resp.OutputParsed.Choices) == 0 {
		log.Printf("❌ Stage 2 failed: no choices in response (resp.OutputParsed.Choices is empty)")
		log.Printf("🔍 Response structure: resp=%+v", resp)
		if resp.OutputParsed.Choices == nil {
			log.Printf("⚠️  resp.OutputParsed.Choices is nil")
		} else {
			log.Printf("⚠️  resp.OutputParsed.Choices is empty slice (length=0)")
		}
		return nil, fmt.Errorf("no output from timing skeleton generation")
	}

	// Stage 2 returns structured MusicalOutput data that the plugin can parse
	// Convert to GenerationResult
	result := &GenerationResult{
		Usage:    resp.Usage,
		MCPUsed:  resp.MCPUsed,
		MCPCalls: resp.MCPCalls,
		MCPTools: resp.MCPTools,
	}
	result.OutputParsed.Choices = resp.OutputParsed.Choices

	log.Printf("✅ Stage 2 generated %d structured choices for timing", len(result.OutputParsed.Choices))

	// ALWAYS send result event after Stage 2 completes - even if choices are empty
	// This ensures the client knows the generation is complete
	if callback != nil {
		log.Printf("📤 Sending result event after Stage 2 completion with %d choices", len(result.OutputParsed.Choices))
		resultErr := callback(StreamEvent{
			Type:    "result",
			Message: "Generation complete",
			Data: map[string]interface{}{
				"output_parsed": map[string]interface{}{
					"choices": result.OutputParsed.Choices,
				},
				"mcp_used": result.MCPUsed,
			},
		})
		if resultErr != nil {
			log.Printf("⚠️  Error sending result event: %v", resultErr)
		} else {
			log.Printf("✅ Result event sent successfully")
		}
	} else {
		log.Printf("⚠️  Cannot send result event: callback is nil")
	}

	return result, nil
}

// buildStage2StreamCallback creates the streaming callback for Stage 2 to reduce complexity
func (s *GenerationService) buildStage2StreamCallback(callback StreamCallback) func(llm.StreamEvent) error {
	return func(event llm.StreamEvent) error {
		// Forward LLM streaming events to the main callback
		if callback != nil {
			forwardErr := callback(StreamEvent{
				Type:    event.Type,
				Message: event.Message,
				Data:    event.Data,
			})
			if forwardErr != nil {
				return forwardErr
			}

			// Log when we receive "completed" event - this is critical for debugging
			if event.Type == "completed" {
				choicesCount := s.extractChoicesCountFromEvent(event)
				log.Printf("✅ Stage 2 COMPLETED event received from provider (choices: %d) - forwarding to client", choicesCount)
			}
		}
		return nil
	}
}

// extractChoicesCountFromEvent extracts choices count from completed event data
func (s *GenerationService) extractChoicesCountFromEvent(event llm.StreamEvent) int {
	if event.Data == nil {
		return 0
	}
	if count, ok := event.Data["choices_count"]; ok {
		if countInt, ok := count.(int); ok {
			return countInt
		}
	}
	return 0
}

// enrichWithHarmonyStage1 generates harmony without timing skeleton (Stage 1)
// This doesn't need to return structured data - just processes harmonically
func (s *GenerationService) enrichWithHarmonyStage1(
	ctx context.Context,
	model string,
	inputArray []map[string]any,
	_ int, // variations - not used in Stage 1, kept for API consistency
	callback StreamCallback,
	reasoningMode string,
) error {
	// Build harmonic enrichment prompt - combine base system prompt with stage-specific instructions
	harmonicPrompt := string(embedded.HarmonicPlannerPromptTxt)

	// Use the full system prompt that includes all musical knowledge, then append stage-specific instructions
	fullPrompt := s.systemPrompt + "\n\n## STAGE 1: HARMONIC ENRICHMENT\n\n" + harmonicPrompt

	// Note: Stage 1 generates harmony without timing constraints
	// Timing skeleton will be created in Stage 2

	// Build request for harmonic enrichment (higher reasoning, with MCP)
	if reasoningMode == "" {
		reasoningMode = reasoningEffortMedium
	}

	request := &llm.GenerationRequest{
		Model:         model,
		InputArray:    inputArray,
		ReasoningMode: reasoningMode,
		SystemPrompt:  fullPrompt,
		OutputSchema: &llm.OutputSchema{
			Name:        "MusicalOutput",
			Description: "Musical composition with multiple choices",
			Schema:      llm.GetMusicalOutputSchema(),
		},
	}

	// Add MCP config for harmonic analysis
	if s.mcpURL != "" {
		request.MCPConfig = &llm.MCPConfig{
			URL:   s.mcpURL,
			Label: s.mcpLabel,
		}
	}

	log.Printf("🎯 Stage 1 (Harmony): Calling provider with %s reasoning and MCP enabled", reasoningMode)

	// Use streaming for Stage 1 to avoid timeouts
	// Stage 1 doesn't need structured data back, just processes harmonically
	// We wrap the callback to catch parse errors and continue (Stage 1 doesn't need valid JSON)
	_, err := s.provider.GenerateStream(ctx, request, func(event llm.StreamEvent) error {
		// Forward LLM streaming events to the main callback
		if callback != nil {
			// If we get a parse error, log it but don't fail Stage 1
			// Stage 1 is just for harmonic processing and doesn't need structured output
			if event.Type == "error" && event.Message != "" {
				log.Printf("⚠️  Stage 1 parse error (non-fatal): %s", event.Message)
				// Still forward the error event, but don't fail the stage
				// The error might be about JSON parsing, which is OK for Stage 1
			}
			return callback(StreamEvent{
				Type:    event.Type,
				Message: event.Message,
				Data:    event.Data,
			})
		}
		return nil
	})
	if err != nil {
		// Check if the error is a parse error - Stage 1 can continue even if parsing fails
		// since it doesn't need structured output
		errStr := err.Error()
		isParseError := errStr != "" &&
			(strings.Contains(errStr, "failed to parse output") ||
				strings.Contains(errStr, "no output received") ||
				strings.Contains(errStr, "Parse error") ||
				strings.Contains(errStr, "invalid character"))
		if isParseError {
			log.Printf("⚠️  Stage 1 parse error (non-fatal): %v - continuing anyway since Stage 1 doesn't need structured output", err)
			// Stage 1 is just for harmonic processing - parse errors are OK
			return nil
		}
		return fmt.Errorf("harmonic enrichment failed: %w", err)
	}

	// Stage 1 just processes harmonically - doesn't need to return structured data
	log.Printf("✅ Stage 1 harmonic processing complete")
	return nil
}
