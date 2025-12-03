package prompt

import (
	"strings"
	"testing"
)

func TestNewPromptBuilder(t *testing.T) {
	builder := NewPromptBuilder()
	if builder == nil {
		t.Fatal("NewPromptBuilder() returned nil")
		return
	}
	if builder.loader == nil {
		t.Fatal("NewPromptBuilder() created builder with nil loader")
	}
}

func TestBuildPrompt(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	if prompt == "" {
		t.Fatal("BuildPrompt() returned empty string")
	}

	// Verify minimum expected length (combined prompts should be substantial)
	if len(prompt) < 1000 {
		t.Errorf("BuildPrompt() returned suspiciously short prompt: %d characters", len(prompt))
	}
}

func TestBuildPromptContainsSystemPrompt(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	// Should contain system prompt content
	if !strings.Contains(prompt, "music composition assistant") {
		t.Error("BuildPrompt() does not contain system prompt content")
	}
}

func TestBuildPromptContainsUserContextInstructions(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	// Should contain user context instructions
	if !strings.Contains(prompt, "USER INPUT") {
		t.Error("BuildPrompt() does not contain user context instructions")
	}
}

func TestBuildPromptContainsMCPInstructions(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	// Should contain MCP server instructions
	if !strings.Contains(prompt, "MCP") {
		t.Error("BuildPrompt() does not contain MCP server instructions")
	}
}

func TestBuildPromptContainsChordProgressions(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	// Should contain chord progressions reference
	if !strings.Contains(prompt, "Chord Progressions") {
		t.Error("BuildPrompt() does not contain chord progressions reference")
	}
}

func TestBuildPromptContainsKeyEmotionalQualities(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	// Should contain key emotional qualities
	if !strings.Contains(prompt, "Key Emotional Qualities") {
		t.Error("BuildPrompt() does not contain key emotional qualities")
	}
}

func TestBuildPromptContainsScalesDescriptions(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	// Should contain scales descriptions
	if !strings.Contains(prompt, "Musical Scales") {
		t.Error("BuildPrompt() does not contain musical scales descriptions")
	}
}

func TestBuildPromptContainsOutputFormatInstructions(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	// Should contain output format instructions
	if !strings.Contains(prompt, "OUTPUT FORMAT") {
		t.Error("BuildPrompt() does not contain output format instructions")
	}
}

func TestBuildPromptNoExcessiveWhitespace(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	// Check for excessive consecutive newlines (more than 5)
	if strings.Contains(prompt, "\n\n\n\n\n\n") {
		t.Error("BuildPrompt() contains excessive consecutive newlines (6+)")
	}

	// Check for sections with only newlines (the original bug)
	// This should catch if the bug returns (empty instructions like "\n\n\n\n\n...")
	lines := strings.Split(prompt, "\n")
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

	// Allow up to 10 empty lines (section separators), but catch if it's more
	if maxEmpty > 10 {
		t.Errorf("BuildPrompt() has %d consecutive empty lines (max allowed: 10)", maxEmpty)
	}
}

func TestBuildPromptSectionOrder(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	// Verify section order by checking positions
	systemPromptPos := strings.Index(prompt, "music composition assistant")
	userContextPos := strings.Index(prompt, "USER INPUT")
	outputFormatPos := strings.Index(prompt, "OUTPUT FORMAT")

	// System prompt should come first
	if systemPromptPos == -1 {
		t.Error("System prompt not found in built prompt")
	}

	// User context should come after system prompt (if present)
	if userContextPos != -1 && systemPromptPos != -1 && userContextPos < systemPromptPos {
		t.Error("User context instructions appear before system prompt")
	}

	// Output format should be near the end (in the second half of the prompt)
	if outputFormatPos != -1 && outputFormatPos < len(prompt)/2 {
		t.Error("Output format instructions appear too early in prompt")
	}

	// Just verify all expected sections are present in some order
	if systemPromptPos == -1 {
		t.Error("System prompt section missing")
	}
	if userContextPos == -1 {
		t.Error("User context section missing")
	}
	if outputFormatPos == -1 {
		t.Error("Output format section missing")
	}
}

func TestGetChordProgressionsReference(t *testing.T) {
	builder := NewPromptBuilder()
	reference, err := builder.getChordProgressionsReference()

	if err != nil {
		t.Fatalf("getChordProgressionsReference() returned error: %v", err)
	}

	if reference == "" {
		t.Error("getChordProgressionsReference() returned empty string")
	}

	// Should contain chord progressions header
	if !strings.Contains(reference, "Chord Progressions") {
		t.Error("getChordProgressionsReference() does not contain expected header")
	}
}

func TestBuilderGetKeyEmotionalQualities(t *testing.T) {
	builder := NewPromptBuilder()
	qualities, err := builder.getKeyEmotionalQualities()

	if err != nil {
		t.Fatalf("getKeyEmotionalQualities() returned error: %v", err)
	}

	if qualities == "" {
		t.Error("getKeyEmotionalQualities() returned empty string")
	}

	// Should contain key emotional qualities header
	if !strings.Contains(qualities, "Key Emotional Qualities") {
		t.Error("getKeyEmotionalQualities() does not contain expected header")
	}
}

func TestGetScalesDescriptions(t *testing.T) {
	builder := NewPromptBuilder()
	descriptions, err := builder.getScalesDescriptions()

	if err != nil {
		t.Fatalf("getScalesDescriptions() returned error: %v", err)
	}

	if descriptions == "" {
		t.Error("getScalesDescriptions() returned empty string")
	}

	// Should contain scales descriptions header
	if !strings.Contains(descriptions, "Musical Scales") {
		t.Error("getScalesDescriptions() does not contain expected header")
	}
}

func TestGetTheoryBooksChapters(t *testing.T) {
	builder := NewPromptBuilder()
	chapters, err := builder.getTheoryBooksChapters()

	if err != nil {
		t.Fatalf("getTheoryBooksChapters() returned error: %v", err)
	}

	if chapters == "" {
		t.Error("getTheoryBooksChapters() returned empty string")
	}

	// Should contain theory books header
	if !strings.Contains(chapters, "Music Theory Books") {
		t.Error("getTheoryBooksChapters() does not contain expected header")
	}
}

func TestGetMusicalGuidelines(t *testing.T) {
	builder := NewPromptBuilder()
	guidelines, err := builder.getMusicalGuidelines()

	if err != nil {
		t.Fatalf("getMusicalGuidelines() returned error: %v", err)
	}

	if guidelines == "" {
		t.Error("getMusicalGuidelines() returned empty string")
	}

	// Should contain musical guidelines content
	if !strings.Contains(guidelines, "harmonic") && !strings.Contains(guidelines, "rhythm") {
		t.Error("getMusicalGuidelines() does not contain expected musical content")
	}
}

func TestBuildPromptConsistency(t *testing.T) {
	builder := NewPromptBuilder()

	// Build prompt multiple times and ensure consistency
	prompt1, err1 := builder.BuildPrompt()
	if err1 != nil {
		t.Fatalf("First BuildPrompt() returned error: %v", err1)
	}

	prompt2, err2 := builder.BuildPrompt()
	if err2 != nil {
		t.Fatalf("Second BuildPrompt() returned error: %v", err2)
	}

	if prompt1 != prompt2 {
		t.Error("BuildPrompt() returns inconsistent results")
	}
}

func TestBuildPromptAllSectionsPresent(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	expectedSections := []string{
		"music composition assistant", // System prompt
		"USER INPUT",                  // User context
		"MCP",                         // MCP instructions
		"Chord Progressions",          // Chord progressions
		"Key Emotional Qualities",     // Emotional qualities
		"Musical Scales",              // Scales
		"OUTPUT FORMAT",               // Output format
	}

	for _, section := range expectedSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("BuildPrompt() missing expected section: %s", section)
		}
	}
}

func TestBuildPromptNoPlaceholders(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	// Check for common placeholder patterns that might indicate missing content
	placeholders := []string{
		"TODO",
		"FIXME",
		"{{",
		"}}",
		"[placeholder]",
		"<insert",
	}

	for _, placeholder := range placeholders {
		if strings.Contains(strings.ToUpper(prompt), strings.ToUpper(placeholder)) {
			t.Errorf("BuildPrompt() contains placeholder: %s", placeholder)
		}
	}
}

func TestBuildPromptValidUTF8(t *testing.T) {
	builder := NewPromptBuilder()
	prompt, err := builder.BuildPrompt()

	if err != nil {
		t.Fatalf("BuildPrompt() returned error: %v", err)
	}

	// Verify the prompt is valid UTF-8 by checking it can be converted
	// If it contains invalid UTF-8, this will replace invalid sequences
	cleaned := strings.ToValidUTF8(prompt, "")
	if len(cleaned) != len(prompt) {
		t.Error("BuildPrompt() returned string with invalid UTF-8 sequences")
	}
}
