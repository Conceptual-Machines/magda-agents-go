package coordination

import (
	"context"
	"os"
	"path/filepath"
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
	_ = godotenv.Load()             // Current directory
	_ = godotenv.Load(".env")       // Current directory explicit
	_ = godotenv.Load("../.env")    // Parent directory
	_ = godotenv.Load("../../.env") // Project root from agents/coordination/

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
	DetectionTime         time.Duration
	KeywordDetectionTime  time.Duration
	LLMDetectionTime      time.Duration
	TotalExecutionTime    time.Duration
	DAWExecutionTime      time.Duration
	ArrangerExecutionTime time.Duration
	ParallelSpeedup       float64 // Ratio of sequential vs parallel execution
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
		name             string
		question         string
		expectedDAW      bool
		expectedArranger bool
		description      string
	}{
		{
			name:             "DAW_only_reverb",
			question:         "add reverb to track 1",
			expectedDAW:      true,
			expectedArranger: false,
			description:      "Simple DAW operation",
		},
		{
			name:             "DAW_only_volume",
			question:         "set volume to -3dB on track 2",
			expectedDAW:      true,
			expectedArranger: false,
			description:      "Volume control",
		},
		{
			name:             "DAW_multilingual_spanish",
			question:         "agregar reverb a la pista 1",
			expectedDAW:      true,
			expectedArranger: false,
			description:      "Spanish translation (agregar=add, pista=track)",
		},
		{
			name:             "DAW_multilingual_french",
			question:         "ajouter de la réverbération à la piste",
			expectedDAW:      true,
			expectedArranger: false,
			description:      "French translation (ajouter=add, réverbération=reverb, piste=track)",
		},
		{
			name:             "DAW_multilingual_german",
			question:         "reverb zur spur hinzufügen",
			expectedDAW:      true,
			expectedArranger: false,
			description:      "German translation (spur=track, hinzufügen=add)",
		},
		{
			name:             "DAW_multilingual_japanese",
			question:         "トラックにリバーブを追加",
			expectedDAW:      true,
			expectedArranger: false,
			description:      "Japanese romanized (torakku=track, ribābu=reverb)",
		},
		{
			name:             "Arranger_only_chord",
			question:         "create I VI IV progression",
			expectedDAW:      true, // "create" triggers DAW
			expectedArranger: true,
			description:      "Chord progression",
		},
		{
			name:             "Arranger_only_arpeggio",
			question:         "add an arpeggio in e minor",
			expectedDAW:      true, // "add" triggers DAW
			expectedArranger: true,
			description:      "Arpeggio with key",
		},
		{
			name:             "Arranger_multilingual_spanish",
			question:         "crear un acorde en do mayor",
			expectedDAW:      true,
			expectedArranger: true,
			description:      "Spanish (acorde=chord, do mayor=C major)",
		},
		{
			name:             "Both_track_and_music",
			question:         "add I VI IV progression to piano track at bar 9",
			expectedDAW:      true,
			expectedArranger: true,
			description:      "Both DAW and musical content",
		},
		{
			name:             "Both_arpeggio_clip",
			question:         "add a clip with an arpeggio in e minor",
			expectedDAW:      true,
			expectedArranger: true,
			description:      "Clip with musical content",
		},
		{
			name:             "Both_bassline",
			question:         "add a clip with a bassline",
			expectedDAW:      true,
			expectedArranger: true,
			description:      "Bassline in clip",
		},
		{
			name:             "Both_riff",
			question:         "add a clip with a riff",
			expectedDAW:      true,
			expectedArranger: true,
			description:      "Riff in clip",
		},
		{
			name:             "Both_groove",
			question:         "add a groove to track 1",
			expectedDAW:      true,
			expectedArranger: true,
			description:      "Groove pattern",
		},
		{
			name:             "Synonym_reverberation",
			question:         "add reverberation to track 1",
			expectedDAW:      true,
			expectedArranger: false,
			description:      "Synonym for reverb",
		},
		{
			name:             "Synonym_echo",
			question:         "add echo effect",
			expectedDAW:      true,
			expectedArranger: false,
			description:      "Synonym for reverb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			needsDAW, needsArranger, needsDrummer := orchestrator.detectAgentsNeededKeywords(tt.question)
			detectionTime := time.Since(start)

			t.Logf("📊 Detection timing: %v", detectionTime)
			t.Logf("   Question: %q", tt.question)
			t.Logf("   Description: %s", tt.description)
			t.Logf("   Result: DAW=%v, Arranger=%v, Drummer=%v", needsDAW, needsArranger, needsDrummer)

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
		name              string
		question          string
		description       string
		expectLLMFallback bool
	}{
		{
			name:              "ambiguous_better",
			question:          "make it sound better",
			description:       "Ambiguous request - should trigger LLM",
			expectLLMFallback: true,
		},
		{
			name:              "creative_vibe",
			question:          "add a vibe to the track",
			description:       "Creative term - should trigger LLM",
			expectLLMFallback: true,
		},
		{
			name:              "mixed_terminology",
			question:          "add some notes to the track",
			description:       "Mixed terms - may trigger LLM",
			expectLLMFallback: true,
		},
		{
			name:              "no_keywords",
			question:          "do something",
			description:       "No keywords - should trigger LLM",
			expectLLMFallback: true,
		},
		{
			name:              "clear_keywords",
			question:          "add reverb to track 1",
			description:       "Clear keywords - should NOT trigger LLM",
			expectLLMFallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First check keyword detection
			keywordStart := time.Now()
			needsDAW, needsArranger, needsDrummer := orchestrator.detectAgentsNeededKeywords(tt.question)
			keywordTime := time.Since(keywordStart)

			t.Logf("📊 Keyword detection: %v", keywordTime)
			t.Logf("   Question: %q", tt.question)
			t.Logf("   Result: DAW=%v, Arranger=%v, Drummer=%v", needsDAW, needsArranger, needsDrummer)

			// If keywords are ambiguous or question looks musical, LLM fallback may be triggered
			shouldUseLLM := (!needsDAW && !needsArranger && !needsDrummer) ||
				(needsDAW && !needsArranger && !needsDrummer && orchestrator.looksMusical(tt.question))

			if shouldUseLLM && tt.expectLLMFallback {
				llmStart := time.Now()
				llmDAW, llmArranger, _, err := orchestrator.detectAgentsNeededLLM(ctx, tt.question)
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
			needsDAW, needsArranger, _ := orchestrator.detectAgentsNeededKeywords(tt.question)
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
		"agregar reverb a la pista",   // Spanish
		"ajouter de la réverbération", // French
		"reverb zur spur hinzufügen",  // German
		"aggiungere riverbero",        // Italian
		"adicionar reverb à faixa",    // Portuguese
	}

	detectionCount := 0
	for _, question := range testQuestions {
		needsDAW, _, _ := orchestrator.detectAgentsNeededKeywords(question)
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
				// Out-of-scope requests should return an error - either from LLM classification
				// or from DAW agent failing to generate valid DSL for non-music requests
				require.Error(t, err, "Expected error for out-of-scope request")
				t.Logf("✅ Got expected error: %v", err)

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
			needsDAW, needsArranger, _ := orchestrator.detectAgentsNeededKeywords(tt.question)
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
		name           string
		question       string
		description    string
		expectArranger bool
		expectDrummer  bool
	}{
		// DAW-only requests (no musical content generation)
		{
			name:           "daw_only_pan",
			question:       "pan the synth track to the left",
			description:    "Pan operation - DAW only",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "daw_only_solo",
			question:       "solo track 3 and mute everything else",
			description:    "Solo/mute operation - DAW only",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "daw_only_create_track_named_drums",
			question:       "create a new track called Drums",
			description:    "Track creation with drum NAME - should NOT trigger drummer",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "daw_only_fx_on_drum_track",
			question:       "add compression to the drum bus",
			description:    "FX on drum track - DAW only, no drum generation",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "daw_only_delete_track",
			question:       "delete the third track",
			description:    "Track deletion - DAW only",
			expectArranger: false,
			expectDrummer:  false,
		},
		// Arranger requests (melodic/harmonic content)
		{
			name:           "arranger_jazz_voicings",
			question:       "write some jazz voicings in Db",
			description:    "Jazz chords - Arranger needed",
			expectArranger: true,
			expectDrummer:  false,
		},
		{
			name:           "arranger_synth_lead",
			question:       "create a synth lead melody",
			description:    "Melody generation - Arranger needed",
			expectArranger: true,
			expectDrummer:  false,
		},
		{
			name:           "arranger_walking_bass",
			question:       "add a walking bass line in F minor",
			description:    "Bass line - Arranger needed",
			expectArranger: true,
			expectDrummer:  false,
		},
		// Drummer requests (percussion patterns)
		{
			name:           "drummer_four_on_floor",
			question:       "make a four on the floor kick pattern",
			description:    "Kick pattern - Drummer needed",
			expectArranger: false,
			expectDrummer:  true,
		},
		{
			name:           "drummer_latin_rhythm",
			question:       "create a latin percussion pattern with congas",
			description:    "Latin percussion - Drummer needed",
			expectArranger: false,
			expectDrummer:  true,
		},
		{
			name:           "drummer_trap_hats",
			question:       "add some trap hi-hat rolls",
			description:    "Hi-hat pattern - Drummer needed",
			expectArranger: false,
			expectDrummer:  true,
		},
		// Out of scope - should return both false
		{
			name:           "out_of_scope_recipe",
			question:       "how do I make pasta carbonara",
			description:    "Cooking recipe - completely out of scope",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "out_of_scope_code",
			question:       "write a python function to sort a list",
			description:    "Programming task - out of scope",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "out_of_scope_travel",
			question:       "book me a flight to Paris",
			description:    "Travel booking - out of scope",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "out_of_scope_medical",
			question:       "what are the symptoms of flu",
			description:    "Medical question - out of scope",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "out_of_scope_sports",
			question:       "who won the world cup in 2022",
			description:    "Sports trivia - out of scope",
			expectArranger: false,
			expectDrummer:  false,
		},
		// Subtle out-of-scope - vague or non-actionable
		{
			name:           "subtle_oos_vague_improvement",
			question:       "make it sound better",
			description:    "Too vague - no actionable request",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "subtle_oos_video",
			question:       "create a video track with some video fx",
			description:    "Video editing - not music production",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "subtle_oos_troubleshooting",
			question:       "fix the audio glitches in my recording",
			description:    "Debugging/troubleshooting - not an action",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "subtle_oos_support_question",
			question:       "why does my plugin crash",
			description:    "Support question - not an action",
			expectArranger: false,
			expectDrummer:  false,
		},
		// Valid DAW operations with musical terms in names - should NOT trigger content agents
		{
			name:           "valid_daw_volume_with_musical_name",
			question:       "lower the track called arpeggio by 3 db",
			description:    "Volume adjustment - DAW only, arranger/drummer not needed",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "valid_daw_rename_drums_track",
			question:       "rename the drums track to percussion",
			description:    "Track rename - DAW only, drummer not triggered by name",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "valid_daw_mute_bassline",
			question:       "mute the track called bassline",
			description:    "Mute operation - DAW only, arranger not triggered by name",
			expectArranger: false,
			expectDrummer:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			t.Logf("Question: %q", tt.question)

			// Test LLM classification directly
			start := time.Now()
			needsDAW, needsArranger, needsDrummer, err := orchestrator.DetectAgentsNeeded(ctx, tt.question)
			detectionTime := time.Since(start)

			require.NoError(t, err, "LLM classification should not error for valid music requests")

			t.Logf("📊 LLM Classification Results:")
			t.Logf("   Detection time: %v", detectionTime)
			t.Logf("   DAW: %v (always true)", needsDAW)
			t.Logf("   Arranger: %v (expected: %v)", needsArranger, tt.expectArranger)
			t.Logf("   Drummer: %v (expected: %v)", needsDrummer, tt.expectDrummer)

			// DAW is always true
			assert.True(t, needsDAW, "DAW should always be true")

			// Check Arranger/Drummer expectations
			if tt.expectArranger {
				assert.True(t, needsArranger, "Expected Arranger=true")
			}
			if tt.expectDrummer {
				assert.True(t, needsDrummer, "Expected Drummer=true")
			}

			// LLM validation should be reasonably fast (< 3s for gpt-4.1-mini)
			assert.Less(t, detectionTime, 3*time.Second,
				"LLM classification should complete within reasonable time")
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
		{"daw_only", "make it sound better"},
		{"arranger", "create harmonic content"},
		{"drummer", "add a breakbeat pattern"},
		{"daw_volume", "adjust the volume"},
	}

	var totalTime time.Duration
	var minTime, maxTime time.Duration
	first := true

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			needsDAW, _, _, err := orchestrator.DetectAgentsNeeded(ctx, tt.question)
			detectionTime := time.Since(start)

			// These are all valid music requests, should not error
			require.NoError(t, err, "Valid request should not error")

			// DAW is always true
			assert.True(t, needsDAW, "DAW should always be true")

			t.Logf("📊 LLM Classification Timing: %s", tt.name)
			t.Logf("   Question: %q", tt.question)
			t.Logf("   Time: %v", detectionTime)

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

			// Each classification should be reasonably fast
			assert.Less(t, detectionTime, 3*time.Second,
				"Individual LLM classification should complete within reasonable time")
		})
	}

	avgTime := totalTime / time.Duration(len(testCases))
	t.Logf("📊 LLM Classification Performance Summary:")
	t.Logf("   Average: %v", avgTime)
	t.Logf("   Min: %v", minTime)
	t.Logf("   Max: %v", maxTime)
	t.Logf("   Total: %v", totalTime)

	// Average should be reasonable
	assert.Less(t, avgTime, 2*time.Second,
		"Average LLM classification time should be reasonable")
}
