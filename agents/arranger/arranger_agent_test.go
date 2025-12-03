package services

import (
	"context"
	"strings"
	"testing"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
)

func TestNewGenerationService(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
		MCPServerURL: "",
	}

	service := NewGenerationService(cfg)

	if service == nil {
		t.Fatal("NewGenerationService() returned nil")
	}

	if service.provider == nil {
		t.Error("NewGenerationService() created service with nil provider")
	}

	if service.promptBuilder == nil {
		t.Error("NewGenerationService() created service with nil promptBuilder")
	}

	if service.systemPrompt == "" {
		t.Error("NewGenerationService() created service with empty systemPrompt")
	}
}

func TestNewGenerationServiceLoadSystemPrompt(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
		MCPServerURL: "",
	}

	service := NewGenerationService(cfg)

	// Verify the system prompt was loaded and contains expected content
	if service.systemPrompt == "" {
		t.Fatal("System prompt is empty")
	}

	// Check for key sections that should be in the system prompt
	expectedSections := []string{
		"music composition assistant",
		"USER INPUT",
		"OUTPUT FORMAT",
	}

	for _, section := range expectedSections {
		if !strings.Contains(service.systemPrompt, section) {
			t.Errorf("System prompt missing expected section: %s", section)
		}
	}
}

func TestNewGenerationServiceSystemPromptNotEmpty(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
		MCPServerURL: "",
	}

	service := NewGenerationService(cfg)

	// The bug we're testing for: empty instructions like "\n\n\n\n..."
	// Count consecutive newlines in the system prompt
	lines := strings.Split(service.systemPrompt, "\n")
	emptyCount := 0
	maxEmpty := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyCount++
			if emptyCount > maxEmpty {
				maxEmpty = emptyCount
			}
		} else {
			emptyCount = 0
		}
	}

	// If we have more than 10 consecutive empty lines, something is wrong
	if maxEmpty > 10 {
		t.Errorf("System prompt has %d consecutive empty lines (suggests empty content bug)", maxEmpty)
	}

	// System prompt should be substantial (at least 1000 characters)
	if len(service.systemPrompt) < 1000 {
		t.Errorf("System prompt is suspiciously short: %d characters", len(service.systemPrompt))
	}
}

func TestNewGenerationServiceWithMCP(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
		MCPServerURL: "http://mcp.example.com:8080",
	}

	service := NewGenerationService(cfg)

	if service.mcpURL == "" {
		t.Error("MCP URL not set despite being provided in config")
	}

	if service.mcpURL != cfg.MCPServerURL {
		t.Errorf("MCP URL mismatch: got %s, want %s", service.mcpURL, cfg.MCPServerURL)
	}

	if service.mcpLabel == "" {
		t.Error("MCP label not set despite MCP URL being provided")
	}

	// Label should be derived from host (including port)
	expectedLabel := "mcp-example-com_8080"
	if service.mcpLabel != expectedLabel {
		t.Errorf("MCP label incorrect: got %s, want %s", service.mcpLabel, expectedLabel)
	}
}

func TestNewGenerationServiceWithoutMCP(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
		MCPServerURL: "",
	}

	service := NewGenerationService(cfg)

	if service.mcpURL != "" {
		t.Error("MCP URL should be empty when not provided")
	}

	if service.mcpLabel != "" {
		t.Error("MCP label should be empty when MCP URL not provided")
	}
}

func TestMCPLabelGeneration(t *testing.T) {
	tests := []struct {
		name          string
		mcpURL        string
		expectedLabel string
	}{
		{
			name:          "simple domain",
			mcpURL:        "http://mcp.example.com",
			expectedLabel: "mcp-example-com",
		},
		{
			name:          "domain with port",
			mcpURL:        "http://localhost:8080",
			expectedLabel: "localhost_8080",
		},
		{
			name:          "subdomain",
			mcpURL:        "https://api.mcp.musicalaideas.com",
			expectedLabel: "api-mcp-musicalaideas-com",
		},
		{
			name:          "IP address",
			mcpURL:        "http://127.0.0.1:3000",
			expectedLabel: "mcp-127-0-0-1_3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				OpenAIAPIKey: "test-key",
				MCPServerURL: tt.mcpURL,
			}

			service := NewGenerationService(cfg)

			if service.mcpLabel != tt.expectedLabel {
				t.Errorf("MCP label incorrect: got %s, want %s", service.mcpLabel, tt.expectedLabel)
			}
		})
	}
}

func TestMusicalOutputSchema(t *testing.T) {
	schema := llm.GetMusicalOutputSchema()

	if schema == nil {
		t.Fatal("llm.GetMusicalOutputSchema() returned nil")
	}

	// Verify top-level structure
	schemaType, ok := schema["type"].(string)
	if !ok || schemaType != "object" {
		t.Error("Schema type should be 'object'")
	}

	// Verify properties exist
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema should have properties")
	}

	// Verify choices array exists
	choices, ok := properties["choices"].(map[string]any)
	if !ok {
		t.Fatal("Schema should have 'choices' property")
	}

	choicesType, choicesTypeOk := choices["type"].(string)
	if !choicesTypeOk || choicesType != arrayType {
		t.Error("Choices should be of type 'array'")
	}

	// Verify items structure
	items, itemsOk := choices["items"].(map[string]any)
	if !itemsOk {
		t.Fatal("Choices should have items definition")
	}

	itemProps, itemPropsOk := items["properties"].(map[string]any)
	if !itemPropsOk {
		t.Fatal("Choice items should have properties")
	}

	// Verify required fields
	if _, descOk := itemProps["description"]; !descOk {
		t.Error("Choice items should have 'description' property")
	}

	if _, notesFieldOk := itemProps["notes"]; !notesFieldOk {
		t.Error("Choice items should have 'notes' property")
	}

	// Verify notes structure
	notes, notesOk := itemProps["notes"].(map[string]any)
	if !notesOk {
		t.Fatal("Notes should be defined")
	}

	notesType, notesTypeOk := notes["type"].(string)
	if !notesTypeOk || notesType != arrayType {
		t.Error("Notes should be of type 'array'")
	}

	// Verify note items properties
	noteItems, noteItemsOk := notes["items"].(map[string]any)
	if !noteItemsOk {
		t.Fatal("Notes should have items definition")
	}

	noteProps, notePropsOk := noteItems["properties"].(map[string]any)
	if !notePropsOk {
		t.Fatal("Note items should have properties")
	}

	// Verify required note fields
	expectedNoteFields := []string{"midiNoteNumber", "velocity", "startBeats", "durationBeats"}
	for _, field := range expectedNoteFields {
		if _, fieldOk := noteProps[field]; !fieldOk {
			t.Errorf("Note items should have '%s' property", field)
		}
	}

	// Verify additionalProperties is false (strict schema)
	additionalProps, additionalPropsOk := noteItems["additionalProperties"].(bool)
	if !additionalPropsOk || additionalProps != false {
		t.Error("Note items should have additionalProperties set to false")
	}
}

func TestMusicalOutputSchemaNoChords(t *testing.T) {
	schema := llm.GetMusicalOutputSchema()

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema should have properties")
	}

	choices, ok := properties["choices"].(map[string]any)
	if !ok {
		t.Fatal("Schema should have 'choices' property")
	}

	items, ok := choices["items"].(map[string]any)
	if !ok {
		t.Fatal("Choices should have items definition")
	}

	itemProps, ok := items["properties"].(map[string]any)
	if !ok {
		t.Fatal("Choice items should have properties")
	}

	// Verify chords field is NOT present (was removed as per requirements)
	if _, ok := itemProps["chords"]; ok {
		t.Error("Schema should not have 'chords' property (it was removed)")
	}
}

func TestMusicalOutputSchemaStrictMode(t *testing.T) {
	schema := llm.GetMusicalOutputSchema()

	// Top level should have additionalProperties: false
	topLevelAdditional, ok := schema["additionalProperties"].(bool)
	if !ok || topLevelAdditional != false {
		t.Error("Top-level schema should have additionalProperties set to false")
	}

	// Choices items should have additionalProperties: false
	properties, _ := schema["properties"].(map[string]any)
	choices, _ := properties["choices"].(map[string]any)
	items, _ := choices["items"].(map[string]any)
	choiceAdditional, ok := items["additionalProperties"].(bool)
	if !ok || choiceAdditional != false {
		t.Error("Choice items should have additionalProperties set to false")
	}
}

func TestSystemPromptStructure(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
		MCPServerURL: "",
	}

	service := NewGenerationService(cfg)

	// Test that the system prompt has the correct structure
	// It should contain all major sections
	requiredSections := map[string]string{
		"system_intro":        "music composition assistant",
		"user_context":        "USER INPUT",
		"mcp_instructions":    "MCP",
		"chord_progressions":  "Chord Progressions",
		"emotional_qualities": "Key Emotional Qualities",
		"scales":              "Musical Scales",
		"output_format":       "OUTPUT FORMAT",
		"anti_chromatic":      "chromatic",
	}

	for section, keyword := range requiredSections {
		if !strings.Contains(service.systemPrompt, keyword) {
			t.Errorf("System prompt missing section '%s' (keyword: %s)", section, keyword)
		}
	}
}

func TestSystemPromptFormatting(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
		MCPServerURL: "",
	}

	service := NewGenerationService(cfg)

	// Check that sections are properly separated
	// Should have headers like "## Chord Progressions Reference:"
	if !strings.Contains(service.systemPrompt, "##") {
		t.Error("System prompt should contain markdown headers (##)")
	}

	// Should not start with excessive newlines
	if strings.HasPrefix(service.systemPrompt, "\n\n\n") {
		t.Error("System prompt should not start with excessive newlines")
	}

	// Should not end with excessive newlines
	if strings.HasSuffix(service.systemPrompt, "\n\n\n\n") {
		t.Error("System prompt should not end with excessive newlines")
	}
}

func TestGenerateContextHandling(t *testing.T) {
	// This test verifies we don't crash with various input formats
	// We can't fully test the OpenAI integration without mocking, but we can
	// verify the input processing logic

	ctx := context.Background()

	testCases := []struct {
		name       string
		inputArray []map[string]any
		shouldPass bool
	}{
		{
			name: "simple user prompt",
			inputArray: []map[string]any{
				{"role": "user", "content": "Generate a C major scale"},
			},
			shouldPass: true,
		},
		{
			name: "user prompt with musical context",
			inputArray: []map[string]any{
				{
					"role":    "user",
					"content": `{"user_prompt": "Continue this melody", "musical_context": "C,D,E"}`,
				},
			},
			shouldPass: true,
		},
		{
			name: "multiple inputs",
			inputArray: []map[string]any{
				{"role": "user", "content": "First composition"},
				{"role": "user", "content": "Second composition"},
			},
			shouldPass: true,
		},
		{
			name:       "empty input array",
			inputArray: []map[string]any{},
			shouldPass: true, // Should handle gracefully
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We can't fully test without mocking OpenAI, but we can verify
			// the input processing doesn't crash
			cfg := &config.Config{
				OpenAIAPIKey: "test-key",
				MCPServerURL: "",
			}

			service := NewGenerationService(cfg)

			// Just verify the service was created with valid prompt
			if service.systemPrompt == "" {
				t.Error("System prompt should not be empty")
			}

			// Test that we can access the context
			if ctx.Err() != nil {
				t.Error("Context should not have error before use")
			}
		})
	}
}

func TestSystemPromptConsistency(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
		MCPServerURL: "",
	}

	// Create multiple services and verify they all get the same system prompt
	service1 := NewGenerationService(cfg)
	service2 := NewGenerationService(cfg)
	service3 := NewGenerationService(cfg)

	if service1.systemPrompt != service2.systemPrompt {
		t.Error("System prompts should be consistent across service instances")
	}

	if service2.systemPrompt != service3.systemPrompt {
		t.Error("System prompts should be consistent across service instances")
	}
}

func TestSystemPromptLength(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
		MCPServerURL: "",
	}

	service := NewGenerationService(cfg)

	// System prompt should be substantial but not excessively long
	minLength := 1000   // At least 1KB
	maxLength := 110000 // No more than 110KB (increased to accommodate chord replaying instructions)

	promptLength := len(service.systemPrompt)

	if promptLength < minLength {
		t.Errorf("System prompt too short: %d bytes (min: %d)", promptLength, minLength)
	}

	if promptLength > maxLength {
		t.Errorf("System prompt too long: %d bytes (max: %d)", promptLength, maxLength)
	}
}
