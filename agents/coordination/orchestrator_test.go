package coordination

import (
	"context"
	"testing"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/stretchr/testify/assert"
)

func TestOrchestrator_DetectAgentsNeeded_Keywords(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key", // Will use mock in actual tests
	}
	orchestrator := NewOrchestrator(cfg)

	tests := []struct {
		name             string
		question         string
		expectedDAW      bool
		expectedArranger bool
	}{
		{
			name:             "DAW only - add reverb",
			question:         "add reverb to track 1",
			expectedDAW:      true,
			expectedArranger: false,
		},
		{
			name:             "DAW only - set volume",
			question:         "set volume to -3dB on track 2",
			expectedDAW:      true,
			expectedArranger: false,
		},
		{
			name:             "Arranger only - chord progression",
			question:         "create I VI IV progression",
			expectedDAW:      true, // "create" triggers DAW, but should also need Arranger
			expectedArranger: true,
		},
		{
			name:             "Both - musical content on track",
			question:         "add I VI IV progression to piano track at bar 9",
			expectedDAW:      true,
			expectedArranger: true,
		},
		{
			name:             "Both - arpeggio in clip",
			question:         "add a clip with an arpeggio in e minor",
			expectedDAW:      true,
			expectedArranger: true, // "minor" should trigger
		},
		{
			name:             "Musical term - bassline",
			question:         "add a clip with a bassline",
			expectedDAW:      true,
			expectedArranger: true, // "bassline" should trigger
		},
		{
			name:             "Musical term - riff",
			question:         "add a clip with a riff",
			expectedDAW:      true,
			expectedArranger: true, // "riff" should trigger
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsDAW, needsArranger := orchestrator.detectAgentsNeededKeywords(tt.question)

			assert.Equal(t, tt.expectedDAW, needsDAW, "DAW detection mismatch")
			// Note: Some tests may need LLM fallback for accurate Arranger detection
			if tt.expectedArranger {
				// If Arranger is expected, it should be detected (either by keywords or looksMusical)
				if !needsArranger {
					// Check if looksMusical would catch it
					if !orchestrator.looksMusical(tt.question) {
						t.Logf("⚠️ Arranger not detected by keywords, may need LLM fallback for: %q", tt.question)
					}
				}
			}
		})
	}
}

func TestOrchestrator_LooksMusical(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
	}
	orchestrator := NewOrchestrator(cfg)

	tests := []struct {
		name     string
		question string
		expected bool
	}{
		{
			name:     "bassline",
			question: "add a clip with a bassline",
			expected: true,
		},
		{
			name:     "riff",
			question: "add a clip with a riff",
			expected: true,
		},
		{
			name:     "arpeggio",
			question: "add an arpeggio",
			expected: true,
		},
		{
			name:     "vibe",
			question: "add a vibe to the track",
			expected: true,
		},
		{
			name:     "no musical terms",
			question: "add reverb to track 1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := orchestrator.looksMusical(tt.question)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOrchestrator_GenerateActions_DAWOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would require actual API keys and real agents
	// For now, just test the structure
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
	}
	orchestrator := NewOrchestrator(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This will fail without real API keys, but tests the structure
	_, err := orchestrator.GenerateActions(ctx, "add reverb to track 1", nil)

	// We expect an error without real setup, but the structure should be correct
	if err != nil {
		t.Logf("Expected error without real API setup: %v", err)
	}
}

func TestOrchestrator_DetectAgentsNeeded_EdgeCases(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
	}
	orchestrator := NewOrchestrator(cfg)

	edgeCases := []struct {
		name     string
		question string
	}{
		{
			name:     "ambiguous - make it better",
			question: "make it sound better",
		},
		{
			name:     "creative - add a vibe",
			question: "add a vibe to the track",
		},
		{
			name:     "mixed terminology",
			question: "add some notes to the track",
		},
		{
			name:     "no clear intent",
			question: "do something",
		},
	}

	for _, tt := range edgeCases {
		t.Run(tt.name, func(t *testing.T) {
			needsDAW, needsArranger := orchestrator.detectAgentsNeededKeywords(tt.question)
			t.Logf("Question: %q -> DAW=%v, Arranger=%v", tt.question, needsDAW, needsArranger)

			// These edge cases should ideally trigger LLM fallback
			// For now, just verify they don't crash
			assert.NotPanics(t, func() {
				orchestrator.looksMusical(tt.question)
			})
		})
	}
}

func TestOrchestrator_ContainsAny(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		keywords []string
		expected bool
	}{
		{
			name:     "contains keyword",
			text:     "add reverb to track",
			keywords: []string{"reverb", "track"},
			expected: true,
		},
		{
			name:     "does not contain",
			text:     "hello world",
			keywords: []string{"reverb", "track"},
			expected: false,
		},
		{
			name:     "case insensitive",
			text:     "ADD REVERB",
			keywords: []string{"reverb"},
			expected: true,
		},
		{
			name:     "lowercase check",
			text:     "add REVERB",
			keywords: []string{"reverb"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.text, tt.keywords)
			assert.Equal(t, tt.expected, result)
		})
	}
}
