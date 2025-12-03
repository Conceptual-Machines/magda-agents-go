package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/Conceptual-Machines/magda-agents-go/models"
	"google.golang.org/genai"
)

const (
	providerNameGemini = "gemini"
	mimeTypeJSON       = "application/json"
	maxLogEventCount   = 5
	geminiUserRole     = "user"
)

// GeminiProvider implements the Provider interface using Google's Gemini API
type GeminiProvider struct {
	client *genai.Client
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(ctx context.Context, apiKey string) (*GeminiProvider, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiProvider{
		client: client,
	}, nil
}

// Name returns the provider name
func (p *GeminiProvider) Name() string {
	return providerNameGemini
}

// Generate implements non-streaming generation using Gemini's API
func (p *GeminiProvider) Generate(ctx context.Context, request *GenerationRequest) (*GenerationResponse, error) {
	startTime := time.Now()
	log.Printf("🎵 GEMINI GENERATION REQUEST STARTED (Model: %s)", request.Model)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "gemini.generate")
	defer transaction.Finish()

	transaction.SetTag("model", request.Model)
	transaction.SetTag("provider", "gemini")

	// Build Gemini-specific request
	contents, err := p.buildGeminiContents(request.InputArray)
	if err != nil {
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("failed to build Gemini contents: %w", err)
	}

	// Configure generation with structured output
	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: request.SystemPrompt}},
		},
	}

	// Add JSON schema for structured output if provided
	if request.OutputSchema != nil {
		config.ResponseMIMEType = mimeTypeJSON
		config.ResponseSchema = p.convertSchemaToGemini(request.OutputSchema.Schema)
	}

	log.Printf("🚨 GEMINI: About to call Gemini API with model='%s'", request.Model)

	// Call Gemini API
	span := transaction.StartChild("gemini.api_call")
	apiStartTime := time.Now()
	result, err := p.client.Models.GenerateContent(ctx, request.Model, contents, config)
	apiDuration := time.Since(apiStartTime)
	span.Finish()

	if err != nil {
		log.Printf("❌ GEMINI REQUEST FAILED after %v: %v", apiDuration, err)
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}

	log.Printf("⏱️  GEMINI API CALL COMPLETED in %v", apiDuration)

	// Process response
	response, err := p.processGeminiResponse(result, startTime, transaction)
	if err != nil {
		return nil, err
	}

	transaction.SetTag("success", "true")
	return response, nil
}

// GenerateStream implements streaming generation for Gemini
func (p *GeminiProvider) GenerateStream(
	ctx context.Context, request *GenerationRequest, callback StreamCallback,
) (*GenerationResponse, error) {
	startTime := time.Now()
	log.Printf("🎵 GEMINI STREAMING GENERATION REQUEST STARTED (Model: %s)", request.Model)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "gemini.generate_stream")
	defer transaction.Finish()

	transaction.SetTag("model", request.Model)
	transaction.SetTag("provider", "gemini")
	transaction.SetTag("streaming", "true")

	// Build Gemini-specific request
	contents, err := p.buildGeminiContents(request.InputArray)
	if err != nil {
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("failed to build Gemini contents: %w", err)
	}

	// Configure generation with structured output
	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: request.SystemPrompt}},
		},
	}

	// Add JSON schema for structured output
	if request.OutputSchema != nil {
		config.ResponseMIMEType = mimeTypeJSON
		config.ResponseSchema = p.convertSchemaToGemini(request.OutputSchema.Schema)
	}

	log.Printf("🚨 GEMINI STREAMING: About to call Gemini streaming API with model='%s'", request.Model)

	// Call Gemini streaming API
	iter := p.client.Models.GenerateContentStream(ctx, request.Model, contents, config)

	// Process stream
	response, err := p.processGeminiStream(iter, callback, transaction, startTime)
	if err != nil {
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, err
	}

	transaction.SetTag("success", "true")
	log.Printf("✅ GEMINI STREAMING GENERATION COMPLETED in %v", time.Since(startTime))

	return response, nil
}

// buildGeminiContents converts our input array to Gemini Content format
func (p *GeminiProvider) buildGeminiContents(inputArray []map[string]any) ([]*genai.Content, error) {
	var contents []*genai.Content

	for _, item := range inputArray {
		role, hasRole := item["role"].(string)
		content, hasContent := item["content"].(string)

		if !hasRole || !hasContent {
			log.Printf("⚠️  Skipping invalid input item (missing role or content): %v", item)
			continue
		}

		// Convert role to Gemini format
		geminiRole := geminiUserRole // Gemini uses "user" and "model"
		if role == "developer" || role == "system" {
			geminiRole = geminiUserRole // System messages go as user in Gemini
		}

		contents = append(contents, &genai.Content{
			Role:  geminiRole,
			Parts: []*genai.Part{{Text: content}},
		})
	}

	return contents, nil
}

// convertSchemaToGemini converts our JSON schema to Gemini's schema format
func (p *GeminiProvider) convertSchemaToGemini(_ map[string]any) *genai.Schema {
	// Gemini uses a specific Schema type
	// For now, create a basic schema - we'll need to map our schema properly
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"choices": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"description": {Type: genai.TypeString},
						"notes": {
							Type: genai.TypeArray,
							Items: &genai.Schema{
								Type: genai.TypeObject,
								Properties: map[string]*genai.Schema{
									"midiNoteNumber": {Type: genai.TypeInteger},
									"velocity":       {Type: genai.TypeInteger},
									"startBeats":     {Type: genai.TypeNumber},
									"durationBeats":  {Type: genai.TypeNumber},
								},
								Required: []string{"midiNoteNumber", "velocity", "startBeats", "durationBeats"},
							},
						},
					},
					Required: []string{"description", "notes"},
				},
			},
		},
		Required: []string{"choices"},
	}
}

// processGeminiResponse converts Gemini response to our GenerationResponse
func (p *GeminiProvider) processGeminiResponse(
	result *genai.GenerateContentResponse,
	startTime time.Time,
	transaction *sentry.Span,
) (*GenerationResponse, error) {
	span := transaction.StartChild("process_response")
	defer span.Finish()

	// Extract text from Gemini response
	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in Gemini response")
	}

	candidate := result.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no parts in Gemini response")
	}

	textOutput := candidate.Content.Parts[0].Text
	log.Printf("📥 GEMINI RESPONSE: output_length=%d", len(textOutput))

	if textOutput == "" {
		return nil, fmt.Errorf("gemini response did not include any output text")
	}

	// Log usage stats if available
	if result.UsageMetadata != nil {
		log.Printf("📊 GEMINI USAGE: input=%d, output=%d, total=%d",
			result.UsageMetadata.PromptTokenCount,
			result.UsageMetadata.CandidatesTokenCount,
			result.UsageMetadata.TotalTokenCount)
	}

	// Parse JSON output
	var output models.MusicalOutput
	if err := json.Unmarshal([]byte(textOutput), &output); err != nil {
		log.Printf("❌ Failed to parse output JSON: %v", err)
		log.Printf("Raw output (first %d chars): %s", maxOutputTrunc, truncate(textOutput, maxOutputTrunc))
		return nil, fmt.Errorf("failed to parse model output: %w", err)
	}

	totalDuration := time.Since(startTime)
	log.Printf("✅ GEMINI GENERATION COMPLETED in %v (choices: %d)", totalDuration, len(output.Choices))

	// Build result
	response := &GenerationResponse{
		Usage:    result.UsageMetadata,
		MCPUsed:  false, // Gemini doesn't support MCP (yet)
		MCPCalls: 0,
		MCPTools: []string{},
	}
	response.OutputParsed.Choices = output.Choices

	return response, nil
}

// processGeminiStream processes the Gemini streaming response
func (p *GeminiProvider) processGeminiStream(
	iter func(yield func(*genai.GenerateContentResponse, error) bool),
	callback StreamCallback,
	_ *sentry.Span,
	startTime time.Time,
) (*GenerationResponse, error) {
	var accumulatedText string
	var finalUsage *genai.GenerateContentResponseUsageMetadata
	eventCount := 0

	// Send initial event
	_ = callback(StreamEvent{Type: "output_started", Message: "Generating output..."})

	// Iterate over stream using Go 1.23+ iterator pattern
	for chunk, err := range iter {
		if err != nil {
			log.Printf("❌ GEMINI STREAMING ERROR: %v", err)
			return nil, fmt.Errorf("gemini stream error: %w", err)
		}

		eventCount++

		// Send heartbeat periodically
		if eventCount%10 == 0 {
			elapsed := time.Since(startTime)
			_ = callback(StreamEvent{
				Type:    "heartbeat",
				Message: "Processing...",
				Data: map[string]any{
					"events_received": eventCount,
					"elapsed_seconds": int(elapsed.Seconds()),
				},
			})
		}

		// Accumulate text from chunks
		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			text := chunk.Candidates[0].Content.Parts[0].Text
			accumulatedText += text
			if eventCount <= maxLogEventCount {
				log.Printf("✅ Gemini chunk #%d: +%d chars (total: %d)", eventCount, len(text), len(accumulatedText))
			}
		}

		// Save usage metadata
		if chunk.UsageMetadata != nil {
			finalUsage = chunk.UsageMetadata
		}
	}

	log.Printf("📦 Gemini stream complete - accumulated text: %d chars", len(accumulatedText))

	// Parse final accumulated text
	var output models.MusicalOutput
	if err := json.Unmarshal([]byte(accumulatedText), &output); err != nil {
		log.Printf("❌ Parse error: %v", err)
		sentry.CaptureException(err)
		return nil, fmt.Errorf("failed to parse gemini output: %w", err)
	}

	log.Printf("✅ Successfully parsed Gemini output: %d choices", len(output.Choices))

	// Send completion event
	_ = callback(StreamEvent{
		Type:    "completed",
		Message: "Generation complete",
		Data: map[string]any{
			"choices_count": len(output.Choices),
		},
	})

	// Build result
	response := &GenerationResponse{
		Usage:    finalUsage,
		MCPUsed:  false, // Gemini doesn't support MCP
		MCPCalls: 0,
		MCPTools: []string{},
	}
	response.OutputParsed.Choices = output.Choices

	totalDuration := time.Since(startTime)
	log.Printf("⏱️  GEMINI STREAMING TIME: %v (choices: %d)", totalDuration, len(output.Choices))

	return response, nil
}
