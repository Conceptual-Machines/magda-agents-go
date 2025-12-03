package prompt

import (
	"strings"
	"testing"
)

func TestNewPromptLoader(t *testing.T) {
	loader := NewPromptLoader()
	if loader == nil {
		t.Fatal("NewPromptLoader() returned nil")
	}
}

func TestGetSystemPrompt(t *testing.T) {
	loader := NewPromptLoader()
	content, err := loader.GetSystemPrompt()

	if err != nil {
		t.Fatalf("GetSystemPrompt() returned error: %v", err)
	}

	if content == "" {
		t.Error("GetSystemPrompt() returned empty string")
	}

	// Check for expected content
	if !strings.Contains(content, "music composition assistant") {
		t.Error("GetSystemPrompt() does not contain expected content")
	}

	// Ensure no excessive whitespace
	if strings.HasPrefix(content, "\n\n\n") {
		t.Error("GetSystemPrompt() has excessive leading newlines")
	}
}

func TestGetOutputFormatInstructions(t *testing.T) {
	loader := NewPromptLoader()
	content, err := loader.GetOutputFormatInstructions()

	if err != nil {
		t.Fatalf("GetOutputFormatInstructions() returned error: %v", err)
	}

	if content == "" {
		t.Error("GetOutputFormatInstructions() returned empty string")
	}

	// Check for expected format content
	if !strings.Contains(content, "OUTPUT FORMAT") || !strings.Contains(content, "JSON") {
		t.Error("GetOutputFormatInstructions() does not contain expected content")
	}
}

func TestGetUserContextInstructions(t *testing.T) {
	loader := NewPromptLoader()
	content, err := loader.GetUserContextInstructions()

	if err != nil {
		t.Fatalf("GetUserContextInstructions() returned error: %v", err)
	}

	if content == "" {
		t.Error("GetUserContextInstructions() returned empty string")
	}

	// Check for expected context content
	if !strings.Contains(content, "USER INPUT") {
		t.Error("GetUserContextInstructions() does not contain expected content")
	}
}

func TestGetMCPServerInstructions(t *testing.T) {
	loader := NewPromptLoader()
	content, err := loader.GetMCPServerInstructions()

	if err != nil {
		t.Fatalf("GetMCPServerInstructions() returned error: %v", err)
	}

	if content == "" {
		t.Error("GetMCPServerInstructions() returned empty string")
	}

	// Check for MCP-specific content
	if !strings.Contains(content, "MCP") {
		t.Error("GetMCPServerInstructions() does not contain expected content")
	}
}

func TestLoaderGetTheoryBooksChapters(t *testing.T) {
	loader := NewPromptLoader()
	content, err := loader.GetTheoryBooksChapters()

	if err != nil {
		t.Fatalf("GetTheoryBooksChapters() returned error: %v", err)
	}

	if content == "" {
		t.Error("GetTheoryBooksChapters() returned empty string")
	}

	// Should contain chapter information
	if !strings.Contains(content, "Chapter") {
		t.Error("GetTheoryBooksChapters() does not contain expected content")
	}
}

func TestGetMusicalScalesHeuristics(t *testing.T) {
	loader := NewPromptLoader()
	content, err := loader.GetMusicalScalesHeuristics()

	if err != nil {
		t.Fatalf("GetMusicalScalesHeuristics() returned error: %v", err)
	}

	if content == "" {
		t.Error("GetMusicalScalesHeuristics() returned empty string")
	}

	// CSV should have comma-separated values
	if !strings.Contains(content, ",") {
		t.Error("GetMusicalScalesHeuristics() does not appear to be valid CSV")
	}
}

func TestGetKeyEmotionalQualities(t *testing.T) {
	loader := NewPromptLoader()
	content, err := loader.GetKeyEmotionalQualities()

	if err != nil {
		t.Fatalf("GetKeyEmotionalQualities() returned error: %v", err)
	}

	if content == "" {
		t.Error("GetKeyEmotionalQualities() returned empty string")
	}

	// CSV should have comma-separated values
	if !strings.Contains(content, ",") {
		t.Error("GetKeyEmotionalQualities() does not appear to be valid CSV")
	}
}

func TestGetProgressions(t *testing.T) {
	loader := NewPromptLoader()
	content, err := loader.GetProgressions()

	if err != nil {
		t.Fatalf("GetProgressions() returned error: %v", err)
	}

	if content == "" {
		t.Error("GetProgressions() returned empty string")
	}

	// JSON should have braces
	if !strings.Contains(content, "{") || !strings.Contains(content, "}") {
		t.Error("GetProgressions() does not appear to be valid JSON")
	}
}

func TestGetAdvancedHarmonicTheory(t *testing.T) {
	loader := NewPromptLoader()
	content, err := loader.GetAdvancedHarmonicTheory()

	if err != nil {
		t.Fatalf("GetAdvancedHarmonicTheory() returned error: %v", err)
	}

	if content == "" {
		t.Error("GetAdvancedHarmonicTheory() returned empty string")
	}

	// Should contain music theory terms
	if !strings.Contains(content, "harmonic") && !strings.Contains(content, "chord") {
		t.Error("GetAdvancedHarmonicTheory() does not contain expected music theory content")
	}
}

func TestGetAdvancedRhythmPhrasing(t *testing.T) {
	loader := NewPromptLoader()
	content, err := loader.GetAdvancedRhythmPhrasing()

	if err != nil {
		t.Fatalf("GetAdvancedRhythmPhrasing() returned error: %v", err)
	}

	if content == "" {
		t.Error("GetAdvancedRhythmPhrasing() returned empty string")
	}

	// Should contain rhythm-related terms
	if !strings.Contains(content, "rhythm") && !strings.Contains(content, "phrasing") {
		t.Error("GetAdvancedRhythmPhrasing() does not contain expected rhythm content")
	}
}

func TestGetAntiChromaticHeuristics(t *testing.T) {
	loader := NewPromptLoader()
	content, err := loader.GetAntiChromaticHeuristics()

	if err != nil {
		t.Fatalf("GetAntiChromaticHeuristics() returned error: %v", err)
	}

	if content == "" {
		t.Error("GetAntiChromaticHeuristics() returned empty string")
	}

	// Should contain chromatic-related terms
	if !strings.Contains(content, "chromatic") {
		t.Error("GetAntiChromaticHeuristics() does not contain expected content")
	}
}

func TestAllLoadersReturnNonEmptyContent(t *testing.T) {
	loader := NewPromptLoader()

	tests := []struct {
		name string
		fn   func() (string, error)
	}{
		{"SystemPrompt", loader.GetSystemPrompt},
		{"OutputFormatInstructions", loader.GetOutputFormatInstructions},
		{"UserContextInstructions", loader.GetUserContextInstructions},
		{"MCPServerInstructions", loader.GetMCPServerInstructions},
		{"TheoryBooksChapters", loader.GetTheoryBooksChapters},
		{"MusicalScalesHeuristics", loader.GetMusicalScalesHeuristics},
		{"KeyEmotionalQualities", loader.GetKeyEmotionalQualities},
		{"Progressions", loader.GetProgressions},
		{"AdvancedHarmonicTheory", loader.GetAdvancedHarmonicTheory},
		{"AdvancedRhythmPhrasing", loader.GetAdvancedRhythmPhrasing},
		{"AntiChromaticHeuristics", loader.GetAntiChromaticHeuristics},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := tt.fn()
			if err != nil {
				t.Errorf("%s returned error: %v", tt.name, err)
			}
			if content == "" {
				t.Errorf("%s returned empty string", tt.name)
			}
			if len(content) < 10 {
				t.Errorf("%s returned suspiciously short content: %d characters", tt.name, len(content))
			}
		})
	}
}

func TestNoExcessiveWhitespace(t *testing.T) {
	loader := NewPromptLoader()

	tests := []struct {
		name string
		fn   func() (string, error)
	}{
		{"SystemPrompt", loader.GetSystemPrompt},
		{"OutputFormatInstructions", loader.GetOutputFormatInstructions},
		{"UserContextInstructions", loader.GetUserContextInstructions},
		{"MCPServerInstructions", loader.GetMCPServerInstructions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := tt.fn()
			if err != nil {
				t.Fatalf("%s returned error: %v", tt.name, err)
			}

			// Check for excessive leading/trailing newlines (more than 2)
			if strings.HasPrefix(content, "\n\n\n") {
				t.Errorf("%s has excessive leading newlines", tt.name)
			}
			if strings.HasSuffix(content, "\n\n\n") {
				t.Errorf("%s has excessive trailing newlines", tt.name)
			}

			// Check for sections with only newlines (original bug)
			lines := strings.Split(content, "\n")
			emptyCount := 0
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					emptyCount++
				} else {
					emptyCount = 0
				}
				if emptyCount > 5 {
					t.Errorf("%s has more than 5 consecutive empty lines", tt.name)
					break
				}
			}
		})
	}
}
