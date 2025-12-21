package jsfx

import (
	"context"
	"os"
	"strings"
	"testing"

	magdaconfig "github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestConfig returns a config with API key from environment
func getTestConfig() *magdaconfig.Config {
	return &magdaconfig.Config{
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
	}
}

// skipIfNoAPIKey skips the test if no API key is available
func skipIfNoAPIKey(t *testing.T, cfg *magdaconfig.Config) {
	if cfg.OpenAIAPIKey == "" {
		t.Skip("Skipping test: OPENAI_API_KEY not set")
	}
}

// TestJSFXAgent_SimpleGain tests generating a simple gain effect
func TestJSFXAgent_SimpleGain(t *testing.T) {
	cfg := getTestConfig()
	skipIfNoAPIKey(t, cfg)

	agent := NewJSFXAgent(cfg)
	ctx := context.Background()

	inputArray := []map[string]any{
		{
			"role":    "user",
			"content": "Create a simple stereo gain plugin with a gain slider from -60 to +12 dB",
		},
	}

	result, err := agent.Generate(ctx, "gpt-5.2", inputArray)
	require.NoError(t, err, "Failed to generate JSFX")
	require.NotNil(t, result)

	t.Logf("📄 Generated JSFX Code:\n%s", result.JSFXCode)

	// Verify JSFX contains expected elements
	assert.NotEmpty(t, result.JSFXCode, "JSFX code should not be empty")
	assert.True(t, strings.Contains(result.JSFXCode, "desc:"), "Should have desc:")
	assert.True(t, strings.Contains(result.JSFXCode, "slider"), "Should have slider")
	assert.True(t, strings.Contains(result.JSFXCode, "@sample") || strings.Contains(result.JSFXCode, "spl0"), "Should have sample processing")
}

// TestJSFXAgent_Compressor tests generating a compressor effect
func TestJSFXAgent_Compressor(t *testing.T) {
	cfg := getTestConfig()
	skipIfNoAPIKey(t, cfg)

	agent := NewJSFXAgent(cfg)
	ctx := context.Background()

	inputArray := []map[string]any{
		{
			"role":    "user",
			"content": "Create a basic compressor with threshold, ratio, attack, and release controls",
		},
	}

	result, err := agent.Generate(ctx, "gpt-5.2", inputArray)
	require.NoError(t, err, "Failed to generate JSFX")
	require.NotNil(t, result)

	t.Logf("📄 Generated JSFX Code:\n%s", result.JSFXCode)

	// Verify structure
	assert.NotEmpty(t, result.JSFXCode, "JSFX code should not be empty")

	// Verify compressor-specific elements
	jsfxLower := strings.ToLower(result.JSFXCode)
	assert.True(t, strings.Contains(jsfxLower, "threshold") || strings.Contains(jsfxLower, "thresh"),
		"Should have threshold parameter")
	assert.True(t, strings.Contains(jsfxLower, "ratio"),
		"Should have ratio parameter")
}

// TestJSFXAgent_LowpassFilter tests generating a lowpass filter
func TestJSFXAgent_LowpassFilter(t *testing.T) {
	cfg := getTestConfig()
	skipIfNoAPIKey(t, cfg)

	agent := NewJSFXAgent(cfg)
	ctx := context.Background()

	inputArray := []map[string]any{
		{
			"role":    "user",
			"content": "Create a biquad lowpass filter with cutoff frequency and resonance (Q) controls",
		},
	}

	result, err := agent.Generate(ctx, "gpt-5.2", inputArray)
	require.NoError(t, err, "Failed to generate JSFX")
	require.NotNil(t, result)

	t.Logf("📄 Generated JSFX Code:\n%s", result.JSFXCode)

	// Verify structure
	assert.NotEmpty(t, result.JSFXCode, "JSFX code should not be empty")

	// Verify filter-specific elements
	jsfxLower := strings.ToLower(result.JSFXCode)
	assert.True(t, strings.Contains(jsfxLower, "cutoff") || strings.Contains(jsfxLower, "freq"),
		"Should have frequency parameter")
}

// TestJSFXAgent_Delay tests generating a delay effect
func TestJSFXAgent_Delay(t *testing.T) {
	cfg := getTestConfig()
	skipIfNoAPIKey(t, cfg)

	agent := NewJSFXAgent(cfg)
	ctx := context.Background()

	inputArray := []map[string]any{
		{
			"role":    "user",
			"content": "Create a simple delay effect with delay time in milliseconds, feedback, and wet/dry mix",
		},
	}

	result, err := agent.Generate(ctx, "gpt-5.2", inputArray)
	require.NoError(t, err, "Failed to generate JSFX")
	require.NotNil(t, result)

	t.Logf("📄 Generated JSFX Code:\n%s", result.JSFXCode)

	// Verify structure
	assert.NotEmpty(t, result.JSFXCode, "JSFX code should not be empty")

	// Verify delay-specific elements
	jsfxLower := strings.ToLower(result.JSFXCode)
	assert.True(t, strings.Contains(jsfxLower, "delay") || strings.Contains(jsfxLower, "time"),
		"Should have delay time parameter")
	assert.True(t, strings.Contains(jsfxLower, "feedback") || strings.Contains(jsfxLower, "fb"),
		"Should have feedback parameter")
}

// TestJSFXAgent_MIDITranspose tests generating a MIDI effect
func TestJSFXAgent_MIDITranspose(t *testing.T) {
	cfg := getTestConfig()
	skipIfNoAPIKey(t, cfg)

	agent := NewJSFXAgent(cfg)
	ctx := context.Background()

	inputArray := []map[string]any{
		{
			"role":    "user",
			"content": "Create a MIDI transpose effect that shifts all incoming notes by a configurable number of semitones (-24 to +24)",
		},
	}

	result, err := agent.Generate(ctx, "gpt-5.2", inputArray)
	require.NoError(t, err, "Failed to generate JSFX")
	require.NotNil(t, result)

	t.Logf("📄 Generated JSFX Code:\n%s", result.JSFXCode)

	// Verify structure
	assert.NotEmpty(t, result.JSFXCode, "JSFX code should not be empty")

	// Verify MIDI-specific elements
	jsfxLower := strings.ToLower(result.JSFXCode)
	assert.True(t, strings.Contains(jsfxLower, "midi") || strings.Contains(jsfxLower, "@block"),
		"Should have MIDI handling code")
}

// TestJSFXAgent_Saturator tests generating a saturation effect
func TestJSFXAgent_Saturator(t *testing.T) {
	cfg := getTestConfig()
	skipIfNoAPIKey(t, cfg)

	agent := NewJSFXAgent(cfg)
	ctx := context.Background()

	inputArray := []map[string]any{
		{
			"role":    "user",
			"content": "Create a soft saturation/distortion effect with drive control and output level",
		},
	}

	result, err := agent.Generate(ctx, "gpt-5.2", inputArray)
	require.NoError(t, err, "Failed to generate JSFX")
	require.NotNil(t, result)

	t.Logf("📄 Generated JSFX Code:\n%s", result.JSFXCode)

	// Verify structure
	assert.NotEmpty(t, result.JSFXCode, "JSFX code should not be empty")

	// Verify saturation-specific elements
	jsfxLower := strings.ToLower(result.JSFXCode)
	assert.True(t, strings.Contains(jsfxLower, "drive") || strings.Contains(jsfxLower, "saturation") || strings.Contains(jsfxLower, "gain"),
		"Should have drive/gain parameter")
}

// TestJSFXAgent_ConversationalFlow tests multi-turn conversation
func TestJSFXAgent_ConversationalFlow(t *testing.T) {
	cfg := getTestConfig()
	skipIfNoAPIKey(t, cfg)

	agent := NewJSFXAgent(cfg)
	ctx := context.Background()

	// First turn: create basic effect
	inputArray := []map[string]any{
		{
			"role":    "user",
			"content": "Create a simple gain plugin",
		},
	}

	result1, err := agent.Generate(ctx, "gpt-5.2", inputArray)
	require.NoError(t, err, "Failed to generate first JSFX")

	t.Logf("📄 Turn 1 - Basic Gain:\n%s", result1.JSFXCode)

	// Second turn: add feature (simulated by adding to conversation)
	inputArray2 := []map[string]any{
		{
			"role":    "user",
			"content": "Create a simple gain plugin",
		},
		{
			"role":    "assistant",
			"content": result1.JSFXCode,
		},
		{
			"role":    "user",
			"content": "Now add a soft clipper to prevent the signal from going above 0dB",
		},
	}

	result2, err := agent.Generate(ctx, "gpt-5.2", inputArray2)
	require.NoError(t, err, "Failed to generate second JSFX")

	t.Logf("📄 Turn 2 - Gain + Soft Clipper:\n%s", result2.JSFXCode)

	// Verify the second result has additional processing
	assert.NotEqual(t, result1.JSFXCode, result2.JSFXCode, "Second result should be different")
}

// TestJSFXAgent_OutputValidity tests that generated JSFX has valid syntax structure
func TestJSFXAgent_OutputValidity(t *testing.T) {
	cfg := getTestConfig()
	skipIfNoAPIKey(t, cfg)

	agent := NewJSFXAgent(cfg)
	ctx := context.Background()

	tests := []struct {
		name   string
		prompt string
	}{
		{"gain", "simple gain plugin"},
		{"eq", "simple 3-band equalizer"},
		{"chorus", "basic chorus effect"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputArray := []map[string]any{
				{"role": "user", "content": "Create a " + tt.prompt},
			}

			result, err := agent.Generate(ctx, "gpt-5.2", inputArray)
			require.NoError(t, err)
			require.NotNil(t, result)

			code := result.JSFXCode

			// Check basic JSFX structure
			assert.True(t, strings.Contains(code, "desc:"), "Should have desc: line")

			// Should have at least one code section
			hasCodeSection := strings.Contains(code, "@init") ||
				strings.Contains(code, "@slider") ||
				strings.Contains(code, "@sample") ||
				strings.Contains(code, "@block")
			assert.True(t, hasCodeSection, "Should have at least one code section")

			t.Logf("✅ %s: Generated %d bytes of JSFX code", tt.name, len(code))
		})
	}
}
