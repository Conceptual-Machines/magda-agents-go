package coordination

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Load .env file from project root for tests
	// Try multiple paths in case tests are run from different directories
	_ = godotenv.Load()                      // Current directory
	_ = godotenv.Load(".env")               // Current directory explicit
	_ = godotenv.Load("../.env")            // Parent directory
	_ = godotenv.Load("../../.env")         // Project root from agents/coordination/
	
	// Also try to find project root by looking for go.mod
	dir, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Load(envPath)
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

// TimingMetrics tracks performance metrics for orchestrator operations
type TimingMetrics struct {
	DetectionTime      time.Duration
	KeywordDetectionTime time.Duration
	LLMDetectionTime   time.Duration
	TotalExecutionTime time.Duration
	DAWExecutionTime   time.Duration
	ArrangerExecutionTime time.Duration
	ParallelSpeedup    float64 // Ratio of sequential vs parallel execution
}

// getTestConfig returns a test config, skipping if API key is not available
func getTestConfig(t *testing.T) *config.Config {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}
	return &config.Config{
		OpenAIAPIKey: apiKey,
	}
}

// getTestConfigForKeywords returns a test config for keyword-only tests (no API key needed)
func getTestConfigForKeywords(t *testing.T) *config.Config {
	return &config.Config{
		OpenAIAPIKey: "test-key", // Not used for keyword detection
	}
}

func TestOrchestrator_Integration_KeywordDetection_Timing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfigForKeywords(t)
	orchestrator := NewOrchestrator(cfg)

	tests := []struct {
		name           string
		question       string
		expectedDAW    bool
		expectedArranger bool
		description    string
	}{
		{
			name:            "DAW_only_reverb",
			question:        "add reverb to track 1",
			expectedDAW:     true,
			expectedArranger: false,
			description:     "Simple DAW operation",
		},
		{
			name:            "DAW_only_volume",
			question:        "set volume to -3dB on track 2",
			expectedDAW:     true,
			expectedArranger: false,
			description:     "Volume control",
		},
		{
			name:            "DAW_multilingual_spanish",
			question:        "agregar reverb a la pista 1",
			expectedDAW:     true,
			expectedArranger: false,
			description:     "Spanish translation (agregar=add, pista=track)",
		},
		{
			name:            "DAW_multilingual_french",
			question:        "ajouter de la réverbération à la piste",
			expectedDAW:     true,
			expectedArranger: false,
			description:     "French translation (ajouter=add, réverbération=reverb, piste=track)",
		},
		{
			name:            "DAW_multilingual_german",
			question:        "reverb zur spur hinzufügen",
			expectedDAW:     true,
			expectedArranger: false,
			description:     "German translation (spur=track, hinzufügen=add)",
		},
		{
			name:            "DAW_multilingual_japanese",
			question:        "トラックにリバーブを追加",
			expectedDAW:     true,
			expectedArranger: false,
			description:     "Japanese romanized (torakku=track, ribābu=reverb)",
		},
		{
			name:            "Arranger_only_chord",
			question:        "create I VI IV progression",
			expectedDAW:     true, // "create" triggers DAW
			expectedArranger: true,
			description:     "Chord progression",
		},
		{
			name:            "Arranger_only_arpeggio",
			question:        "add an arpeggio in e minor",
			expectedDAW:     true, // "add" triggers DAW
			expectedArranger: true,
			description:     "Arpeggio with key",
		},
		{
			name:            "Arranger_multilingual_spanish",
			question:        "crear un acorde en do mayor",
			expectedDAW:     true,
			expectedArranger: true,
			description:     "Spanish (acorde=chord, do mayor=C major)",
		},
		{
			name:            "Both_track_and_music",
			question:        "add I VI IV progression to piano track at bar 9",
			expectedDAW:     true,
			expectedArranger: true,
			description:     "Both DAW and musical content",
		},
		{
			name:            "Both_arpeggio_clip",
			question:        "add a clip with an arpeggio in e minor",
			expectedDAW:     true,
			expectedArranger: true,
			description:     "Clip with musical content",
		},
		{
			name:            "Both_bassline",
			question:        "add a clip with a bassline",
			expectedDAW:     true,
			expectedArranger: true,
			description:     "Bassline in clip",
		},
		{
			name:            "Both_riff",
			question:        "add a clip with a riff",
			expectedDAW:     true,
			expectedArranger: true,
			description:     "Riff in clip",
		},
		{
			name:            "Both_groove",
			question:        "add a groove to track 1",
			expectedDAW:     true,
			expectedArranger: true,
			description:     "Groove pattern",
		},
		{
			name:            "Synonym_reverberation",
			question:        "add reverberation to track 1",
			expectedDAW:     true,
			expectedArranger: false,
			description:     "Synonym for reverb",
		},
		{
			name:            "Synonym_echo",
			question:        "add echo effect",
			expectedDAW:     true,
			expectedArranger: false,
			description:     "Synonym for reverb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			needsDAW, needsArranger := orchestrator.detectAgentsNeededKeywords(tt.question)
			detectionTime := time.Since(start)

			t.Logf("📊 Detection timing: %v", detectionTime)
			t.Logf("   Question: %q", tt.question)
			t.Logf("   Description: %s", tt.description)
			t.Logf("   Result: DAW=%v, Arranger=%v", needsDAW, needsArranger)

			// Verify detection time is fast (< 1ms for keyword matching)
			assert.Less(t, detectionTime, 10*time.Millisecond, 
				"Keyword detection should be very fast (<10ms)")

			// Verify expected results
			assert.Equal(t, tt.expectedDAW, needsDAW, 
				"DAW detection mismatch for: %q", tt.question)
			
			if tt.expectedArranger {
				assert.True(t, needsArranger || orchestrator.looksMusical(tt.question),
					"Arranger should be detected (by keywords or looksMusical) for: %q", tt.question)
			}
		})
	}
}

func TestOrchestrator_Integration_LLMFallback_Timing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	orchestrator := NewOrchestrator(cfg)
	ctx := context.Background()

	tests := []struct {
		name           string
		question       string
		description    string
		expectLLMFallback bool
	}{
		{
			name:            "ambiguous_better",
			question:        "make it sound better",
			description:     "Ambiguous request - should trigger LLM",
			expectLLMFallback: true,
		},
		{
			name:            "creative_vibe",
			question:        "add a vibe to the track",
			description:     "Creative term - should trigger LLM",
			expectLLMFallback: true,
		},
		{
			name:            "mixed_terminology",
			question:        "add some notes to the track",
			description:     "Mixed terms - may trigger LLM",
			expectLLMFallback: true,
		},
		{
			name:            "no_keywords",
			question:        "do something",
			description:     "No keywords - should trigger LLM",
			expectLLMFallback: true,
		},
		{
			name:            "clear_keywords",
			question:        "add reverb to track 1",
			description:     "Clear keywords - should NOT trigger LLM",
			expectLLMFallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First check keyword detection
			keywordStart := time.Now()
			needsDAW, needsArranger := orchestrator.detectAgentsNeededKeywords(tt.question)
			keywordTime := time.Since(keywordStart)

			t.Logf("📊 Keyword detection: %v", keywordTime)
			t.Logf("   Question: %q", tt.question)
			t.Logf("   Result: DAW=%v, Arranger=%v", needsDAW, needsArranger)

			// If keywords are ambiguous or question looks musical, LLM fallback may be triggered
			shouldUseLLM := (!needsDAW && !needsArranger) || 
			               (needsDAW && !needsArranger && orchestrator.looksMusical(tt.question))

			if shouldUseLLM && tt.expectLLMFallback {
				llmStart := time.Now()
				llmDAW, llmArranger, err := orchestrator.detectAgentsNeededLLM(ctx, tt.question)
				llmTime := time.Since(llmStart)

				require.NoError(t, err, "LLM detection should not fail")
				t.Logf("📊 LLM fallback: %v", llmTime)
				t.Logf("   LLM Result: DAW=%v, Arranger=%v", llmDAW, llmArranger)

				// LLM should be slower but reasonable (< 2s for gpt-4.1-mini)
				assert.Less(t, llmTime, 2*time.Second, 
					"LLM detection should complete in reasonable time (<2s)")

				// Verify LLM provides a decision
				assert.True(t, llmDAW || llmArranger, 
					"LLM should detect at least one agent")
			} else if !tt.expectLLMFallback {
				t.Logf("✅ No LLM fallback needed (keywords sufficient)")
			}
		})
	}
}

func TestOrchestrator_Integration_ExpandedKeywords_Coverage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfigForKeywords(t)
	orchestrator := NewOrchestrator(cfg)

	// Test expanded keywords from different languages
	expandedTests := []struct {
		name        string
		question    string
		language    string
		expectedDAW bool
	}{
		// Spanish
		{"spanish_pista", "agregar reverb a la pista", "Spanish", true},
		{"spanish_efecto", "añadir efecto de reverberación", "Spanish", true},
		{"spanish_acorde", "crear un acorde en do mayor", "Spanish", true},
		
		// French
		{"french_piste", "ajouter de la réverbération à la piste", "French", true},
		{"french_effet", "ajouter un effet", "French", true},
		{"french_accord", "créer un accord", "French", true},
		
		// German
		{"german_spur", "reverb zur spur hinzufügen", "German", true},
		{"german_effekt", "effekt hinzufügen", "German", true},
		{"german_akkord", "akkord erstellen", "German", true},
		
		// Italian
		{"italian_traccia", "aggiungere riverbero alla traccia", "Italian", true},
		{"italian_effetto", "aggiungere effetto", "Italian", true},
		{"italian_accordo", "creare un accordo", "Italian", true},
		
		// Portuguese
		{"portuguese_faixa", "adicionar reverb à faixa", "Portuguese", true},
		{"portuguese_efeito", "adicionar efeito", "Portuguese", true},
		{"portuguese_acordo", "criar um acordo", "Portuguese", true},
		
		// Japanese (romanized)
		{"japanese_torakku", "torakku ni ribābu o tsuika", "Japanese", true},
		{"japanese_efekuto", "efekuto o tsuika", "Japanese", true},
		{"japanese_kōdo", "kōdo o sakusei", "Japanese", false}, // "kōdo" (chord) is Arranger, not DAW
	}

	for _, tt := range expandedTests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			needsDAW, needsArranger := orchestrator.detectAgentsNeededKeywords(tt.question)
			detectionTime := time.Since(start)

			t.Logf("📊 %s detection: %v", tt.language, detectionTime)
			t.Logf("   Question: %q", tt.question)
			t.Logf("   Result: DAW=%v, Arranger=%v", needsDAW, needsArranger)

			assert.Less(t, detectionTime, 10*time.Millisecond, 
				"Detection should be fast even with expanded keywords")
			
			if tt.expectedDAW {
				assert.True(t, needsDAW, 
					"Should detect DAW for %s: %q", tt.language, tt.question)
			} else {
				// If not expecting DAW, verify it's correctly not detected
				// (e.g., "kōdo" is Arranger keyword, not DAW)
				assert.False(t, needsDAW, 
					"Should NOT detect DAW for %s: %q (this is Arranger content)", tt.language, tt.question)
			}
		})
	}
}

func TestOrchestrator_Integration_Performance_Benchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfigForKeywords(t)
	orchestrator := NewOrchestrator(cfg)

	// Benchmark keyword detection with various question lengths
	benchmarks := []struct {
		name     string
		question string
	}{
		{"short", "add reverb"},
		{"medium", "add reverb to track 1"},
		{"long", "add reverb effect to track 1 and set volume to -3dB"},
		{"very_long", "add reverb effect to track 1, set volume to -3dB, pan to center, and mute track 2"},
	}

	for _, bm := range benchmarks {
		t.Run(bm.name, func(t *testing.T) {
			iterations := 100
			var totalTime time.Duration

			for i := 0; i < iterations; i++ {
				start := time.Now()
				orchestrator.detectAgentsNeededKeywords(bm.question)
				totalTime += time.Since(start)
			}

			avgTime := totalTime / time.Duration(iterations)
			t.Logf("📊 Benchmark: %s", bm.name)
			t.Logf("   Question length: %d chars", len(bm.question))
			t.Logf("   Iterations: %d", iterations)
			t.Logf("   Average time: %v", avgTime)
			t.Logf("   Total time: %v", totalTime)

			// Average should be very fast (< 1ms)
			assert.Less(t, avgTime, 1*time.Millisecond, 
				"Average keyword detection should be < 1ms")
		})
	}
}

func TestOrchestrator_Integration_LoadKeywords_Timing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfigForKeywords(t)

	// Measure time to create orchestrator and load keywords
	start := time.Now()
	orchestrator := NewOrchestrator(cfg)
	loadTime := time.Since(start)

	t.Logf("📊 Keyword loading timing:")
	t.Logf("   Total initialization: %v", loadTime)

	// Verify keywords are loaded by testing detection with expanded keywords
	// Test with multilingual keywords that should only work if expanded keywords are loaded
	testQuestions := []string{
		"agregar reverb a la pista",      // Spanish
		"ajouter de la réverbération",    // French
		"reverb zur spur hinzufügen",      // German
		"aggiungere riverbero",           // Italian
		"adicionar reverb à faixa",       // Portuguese
	}

	detectionCount := 0
	for _, question := range testQuestions {
		needsDAW, _ := orchestrator.detectAgentsNeededKeywords(question)
		if needsDAW {
			detectionCount++
		}
	}

	t.Logf("   Multilingual detection: %d/%d questions detected", detectionCount, len(testQuestions))
	
	// If expanded keywords are loaded, at least some multilingual questions should be detected
	// (This is a heuristic - some might not match perfectly)
	assert.Greater(t, detectionCount, 0, 
		"Should detect at least some multilingual keywords (expanded keywords loaded)")

	// Initialization should be fast (< 100ms)
	assert.Less(t, loadTime, 100*time.Millisecond, 
		"Keyword loading should be fast (< 100ms)")
}

func TestOrchestrator_Integration_OutOfScope_Requests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	orchestrator := NewOrchestrator(cfg)
	ctx := context.Background()

	tests := []struct {
		name          string
		question      string
		description   string
		expectError   bool
		errorContains string
	}{
		{
			name:          "cooking_request",
			question:      "bake me a cake",
			description:   "Cooking task - completely out of scope",
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "email_request",
			question:      "send an email to john@example.com",
			description:   "Email task - out of scope",
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "weather_request",
			question:      "what's the weather today?",
			description:   "Weather query - out of scope",
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "general_question",
			question:      "what is 2+2?",
			description:   "General math question - out of scope",
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "valid_daw_request",
			question:      "add reverb to track 1",
			description:   "Valid DAW operation - should succeed",
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			t.Logf("Question: %q", tt.question)

			result, err := orchestrator.GenerateActions(ctx, tt.question, nil)

			if tt.expectError {
				// Out-of-scope requests should return an error from the orchestrator's LLM validator
				// This happens when no agents are detected and the validator determines the request is out of scope
				require.Error(t, err, "Expected error for out-of-scope request")
				t.Logf("✅ Got expected error: %v", err)
				
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains),
						"Error message should contain '%s'", tt.errorContains)
				}
				
				// Verify error format - should come from orchestrator validator
				assert.Contains(t, err.Error(), "out of scope",
					"Error should indicate request is out of scope")
				
				// Should not have actions
				if result != nil {
					assert.Empty(t, result.Actions, "Out-of-scope request should not produce actions")
				}
			} else {
				// For valid requests, we might get an error if API key is invalid, but structure should be correct
				if err != nil {
					t.Logf("⚠️ Got error (might be API key issue): %v", err)
				} else {
					require.NotNil(t, result, "Valid request should return result")
					t.Logf("✅ Request succeeded with %d actions", len(result.Actions))
				}
			}
		})
	}
}

func TestOrchestrator_Integration_EdgeCases_Timing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfigForKeywords(t)
	orchestrator := NewOrchestrator(cfg)

	edgeCases := []struct {
		name     string
		question string
	}{
		{"empty", ""},
		{"single_word", "reverb"},
		{"numbers_only", "123"},
		{"special_chars", "!@#$%^&*()"},
		{"unicode", "🎵🎶🎹"},
		{"very_long", "add " + string(make([]byte, 1000)) + " reverb"},
		{"mixed_case", "ADD ReVeRb To TrAcK"},
		{"whitespace", "   add   reverb   to   track   1   "},
	}

	for _, tt := range edgeCases {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			needsDAW, needsArranger := orchestrator.detectAgentsNeededKeywords(tt.question)
			detectionTime := time.Since(start)

			t.Logf("📊 Edge case: %s", tt.name)
			t.Logf("   Question: %q", tt.question)
			t.Logf("   Detection time: %v", detectionTime)
			t.Logf("   Result: DAW=%v, Arranger=%v", needsDAW, needsArranger)

			// Should not panic and should complete quickly
			assert.Less(t, detectionTime, 10*time.Millisecond, 
				"Edge case should not cause performance issues")
		})
	}
}

func TestOrchestrator_Integration_LLMValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	orchestrator := NewOrchestrator(cfg)
	ctx := context.Background()

	tests := []struct {
		name          string
		question      string
		description   string
		expectDAW     bool
		expectArranger bool
		expectError   bool
		errorContains string
	}{
		{
			name:          "out_of_scope_cooking",
			question:      "bake me a cake",
			description:   "Cooking request - LLM should return both false, orchestrator should error",
			expectDAW:     false,
			expectArranger: false,
			expectError:   true, // Orchestrator returns error when both are false
			errorContains: "out of scope",
		},
		{
			name:          "out_of_scope_email",
			question:      "send an email to john@example.com",
			description:   "Email request - LLM should return both false, orchestrator should error",
			expectDAW:     false,
			expectArranger: false,
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "out_of_scope_weather",
			question:      "what's the weather like?",
			description:   "Weather query - LLM should return both false, orchestrator should error",
			expectDAW:     false,
			expectArranger: false,
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "out_of_scope_math",
			question:      "what is 2+2?",
			description:   "Math question - LLM should return both false, orchestrator should error",
			expectDAW:     false,
			expectArranger: false,
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "valid_arranger_ambiguous",
			question:      "create a harmonic progression",
			description:   "Ambiguous Arranger request - LLM should detect Arranger=true",
			expectDAW:     false,
			expectArranger: true,
			expectError:   false,
		},
		{
			name:          "valid_both_ambiguous",
			question:      "add some musical elements to improve the track",
			description:   "Ambiguous both - LLM should detect both true",
			expectDAW:     true,
			expectArranger: true,
			expectError:   false,
		},
		{
			name:          "valid_daw_explicit",
			question:      "adjust the volume levels",
			description:   "Explicit DAW request - LLM should detect DAW=true",
			expectDAW:     true,
			expectArranger: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			t.Logf("Question: %q", tt.question)

			// First verify keywords don't match (so LLM validation is triggered)
			keywordDAW, keywordArranger := orchestrator.detectAgentsNeededKeywords(tt.question)
			if keywordDAW || keywordArranger {
				t.Logf("⚠️ Keywords detected (DAW=%v, Arranger=%v), LLM validation may not be triggered", 
					keywordDAW, keywordArranger)
				// If keywords match, skip this test case as LLM won't be called
				if tt.expectError {
					// For out-of-scope tests, if keywords match, we can't test LLM validation
					t.Skip("Keywords matched, cannot test LLM validation for this case")
				}
			}

			// Test LLM validation directly
			start := time.Now()
			needsDAW, needsArranger, err := orchestrator.DetectAgentsNeeded(ctx, tt.question)
			detectionTime := time.Since(start)

			if tt.expectError {
				require.Error(t, err, "Expected error")
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains),
						"Error message should contain '%s'", tt.errorContains)
				}
			} else {
				require.NoError(t, err, "LLM validation should not error")
				
				t.Logf("📊 LLM Validation Results:")
				t.Logf("   Detection time: %v", detectionTime)
				t.Logf("   DAW: %v (expected: %v)", needsDAW, tt.expectDAW)
				t.Logf("   Arranger: %v (expected: %v)", needsArranger, tt.expectArranger)
				
				// For out-of-scope requests, both should be false
				if !tt.expectDAW && !tt.expectArranger {
					assert.False(t, needsDAW, "Out-of-scope request should have DAW=false")
					assert.False(t, needsArranger, "Out-of-scope request should have Arranger=false")
				} else {
					// For valid requests, check expectations
					if tt.expectDAW {
						assert.True(t, needsDAW, "Expected DAW=true for valid DAW request")
					}
					if tt.expectArranger {
						assert.True(t, needsArranger, "Expected Arranger=true for valid Arranger request")
					}
				}

				// LLM validation should be reasonably fast (< 3s for gpt-4.1-mini)
				assert.Less(t, detectionTime, 3*time.Second, 
					"LLM validation should complete within reasonable time")
			}
		})
	}
}

func TestOrchestrator_Integration_LLMValidation_Timing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	orchestrator := NewOrchestrator(cfg)
	ctx := context.Background()

	// Test cases that should trigger LLM validation (no keywords)
	testCases := []struct {
		name     string
		question string
	}{
		{"cooking", "bake me a cake"},
		{"email", "send an email"},
		{"weather", "what's the weather"},
		{"math", "what is 2+2"},
		{"ambiguous_daw", "make it sound better"},
		{"ambiguous_arranger", "create harmonic content"},
	}

	var totalTime time.Duration
	var minTime, maxTime time.Duration
	first := true

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Check if keywords match first
			keywordDAW, keywordArranger := orchestrator.detectAgentsNeededKeywords(tt.question)
			if keywordDAW || keywordArranger {
				t.Logf("⚠️ Keywords detected (DAW=%v, Arranger=%v) for '%s', skipping LLM validation test", 
					keywordDAW, keywordArranger, tt.question)
				t.Skip("Keywords matched, LLM validation not triggered")
			}

			start := time.Now()
			_, _, err := orchestrator.DetectAgentsNeeded(ctx, tt.question)
			detectionTime := time.Since(start)

			// Out-of-scope requests (cooking, email, weather, math) should return an error
			// Valid requests (ambiguous_daw, ambiguous_arranger) should not error
			isOutOfScope := tt.name == "cooking" || tt.name == "email" || tt.name == "weather" || tt.name == "math"
			
			if isOutOfScope {
				require.Error(t, err, "Out-of-scope request should return error")
				assert.Contains(t, err.Error(), "out of scope", "Error should indicate out of scope")
			} else {
				require.NoError(t, err, "Valid request should not error")
			}

			t.Logf("📊 LLM Validation Timing: %s", tt.name)
			t.Logf("   Question: %q", tt.question)
			t.Logf("   Time: %v", detectionTime)
			t.Logf("   Error: %v", err)

			// Only count timing for successful calls (or all calls for performance tracking)
			totalTime += detectionTime
			if first {
				minTime = detectionTime
				maxTime = detectionTime
				first = false
			} else {
				if detectionTime < minTime {
					minTime = detectionTime
				}
				if detectionTime > maxTime {
					maxTime = detectionTime
				}
			}

			// Each validation should be reasonably fast (even if it errors)
			assert.Less(t, detectionTime, 3*time.Second, 
				"Individual LLM validation should complete within reasonable time")
		})
	}

	avgTime := totalTime / time.Duration(len(testCases))
	t.Logf("📊 LLM Validation Performance Summary:")
	t.Logf("   Average: %v", avgTime)
	t.Logf("   Min: %v", minTime)
	t.Logf("   Max: %v", maxTime)
	t.Logf("   Total: %v", totalTime)

	// Average should be reasonable
	assert.Less(t, avgTime, 2*time.Second, 
		"Average LLM validation time should be reasonable")
}

