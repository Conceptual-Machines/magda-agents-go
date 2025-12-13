package mix

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Load .env file for tests
	_ = godotenv.Load("../../.env")
}

func getTestConfig(t *testing.T) *config.Config {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	return &config.Config{OpenAIAPIKey: apiKey}
}

func TestMixAnalysisAgent_AnalyzeTrack_Bass(t *testing.T) {
	cfg := getTestConfig(t)
	agent := NewMixAnalysisAgent(cfg)

	// Use synthetic data generator for realistic test data
	gen := NewSyntheticDataGenerator(42)
	request := gen.GenerateMuddyBass()

	ctx := context.Background()
	result, err := agent.Analyze(ctx, request)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Analysis)

	// Check that we got analysis
	assert.NotEmpty(t, result.Analysis.Summary)
	t.Logf("📊 Analysis Summary: %s", result.Analysis.Summary)

	// Should identify issues related to muddiness/frequency
	foundRelevantIssue := false
	for _, issue := range result.Analysis.Issues {
		t.Logf("   Issue [%s]: %s - %s", issue.Severity, issue.Type, issue.Description)
		// Look for any high/medium severity issue mentioning frequency-related terms
		if issue.Severity == "high" || issue.Severity == "medium" {
			foundRelevantIssue = true
		}
	}
	assert.True(t, foundRelevantIssue, "Should identify issues")

	// Check recommendations
	assert.NotEmpty(t, result.Recommendations)
	t.Logf("\n📋 Recommendations:")
	for _, rec := range result.Recommendations {
		t.Logf("   [%s] %s", rec.Priority, rec.Description)
		t.Logf("      Why: %s", rec.Explanation)
		t.Logf("      Action Type: %s, FX: %s", rec.Action["action_type"], rec.Action["fx_name"])
	}

	// Should recommend ReaEQ or ReaComp
	foundReaPluginRec := false
	for _, rec := range result.Recommendations {
		desc := strings.ToLower(rec.Description)
		if strings.Contains(desc, "reaeq") || strings.Contains(desc, "reacomp") {
			foundReaPluginRec = true
			break
		}
	}
	assert.True(t, foundReaPluginRec, "Should recommend REAPER stock plugin (ReaEQ or ReaComp)")
}

func TestMixAnalysisAgent_AnalyzeTrack_Vocals(t *testing.T) {
	cfg := getTestConfig(t)
	agent := NewMixAnalysisAgent(cfg)

	// Use synthetic data generator for realistic test data
	gen := NewSyntheticDataGenerator(43)
	request := gen.GenerateHarshVocals()

	ctx := context.Background()
	result, err := agent.Analyze(ctx, request)

	require.NoError(t, err)
	require.NotNil(t, result)

	t.Logf("📊 Vocal Analysis Summary: %s", result.Analysis.Summary)

	for _, issue := range result.Analysis.Issues {
		t.Logf("   Issue [%s]: %s", issue.Severity, issue.Description)
	}

	t.Logf("\n📋 Recommendations:")
	for _, rec := range result.Recommendations {
		t.Logf("   [%s] %s", rec.Priority, rec.Description)
	}

	// Should identify harshness and dynamics issues
	assert.GreaterOrEqual(t, len(result.Analysis.Issues), 1)
	assert.GreaterOrEqual(t, len(result.Recommendations), 1)
}

func TestMixAnalysisAgent_AnalyzeMaster(t *testing.T) {
	cfg := getTestConfig(t)
	agent := NewMixAnalysisAgent(cfg)

	// Use synthetic data generator for realistic test data
	gen := NewSyntheticDataGenerator(44)
	request := gen.GenerateMasterForStreaming()

	ctx := context.Background()
	result, err := agent.Analyze(ctx, request)

	require.NoError(t, err)
	require.NotNil(t, result)

	t.Logf("📊 Mastering Analysis: %s", result.Analysis.Summary)

	for _, rec := range result.Recommendations {
		t.Logf("   [%s] %s", rec.Priority, rec.Description)
		t.Logf("      Action: %v", rec.Action)
	}

	// Should recommend loudness increase for streaming
	assert.NotEmpty(t, result.Recommendations)
}

func TestMixAnalysisAgent_MultiTrack(t *testing.T) {
	cfg := getTestConfig(t)
	agent := NewMixAnalysisAgent(cfg)

	// Simulate multi-track analysis with frequency masking
	request := &AnalysisRequest{
		Mode: ModeMultiTrack,
		AnalysisData: &DSPAnalysisData{
			Tracks: []TrackAnalysis{
				{
					TrackIndex: 0,
					TrackName:  "Kick",
					Analysis: &DSPAnalysisData{
						FrequencySpectrum: &FrequencyAnalysis{
							Bands: &FrequencyBands{
								Sub:    -10.0,
								Bass:   -8.0,
								LowMid: -12.0,
							},
						},
					},
				},
				{
					TrackIndex: 1,
					TrackName:  "Bass",
					Analysis: &DSPAnalysisData{
						FrequencySpectrum: &FrequencyAnalysis{
							Bands: &FrequencyBands{
								Sub:    -8.0,  // Competing with kick
								Bass:   -6.0,  // Competing with kick
								LowMid: -10.0,
							},
						},
					},
				},
			},
			Relationships: &RelationshipAnalysis{
				FrequencyMasking: []MaskingIssue{
					{
						Track1:      0,
						Track2:      1,
						FrequencyHz: 80,
						Severity:    0.8, // High masking
					},
				},
			},
		},
		Context: &AnalysisContext{
			Genre: "electronic",
		},
		UserRequest: "Kick and bass are fighting for space",
	}

	ctx := context.Background()
	result, err := agent.Analyze(ctx, request)

	require.NoError(t, err)
	require.NotNil(t, result)

	t.Logf("📊 Multi-Track Analysis: %s", result.Analysis.Summary)

	// Should identify masking issue
	for _, issue := range result.Analysis.Issues {
		t.Logf("   Issue: %s", issue.Description)
	}

	t.Logf("\n📋 Recommendations:")
	for _, rec := range result.Recommendations {
		t.Logf("   [%s] %s", rec.Priority, rec.Description)
	}

	// Check for relationship issues
	if len(result.RelationshipIssues) > 0 {
		t.Logf("\n🔗 Relationship Issues:")
		for _, ri := range result.RelationshipIssues {
			t.Logf("   Tracks %v: %s", ri.Tracks, ri.Issue)
			t.Logf("   Recommendation: %s", ri.Recommendation)
		}
	}
}

func TestMixAnalysisAgent_AnalyzeTrack_BoxyDrums(t *testing.T) {
	cfg := getTestConfig(t)
	agent := NewMixAnalysisAgent(cfg)

	gen := NewSyntheticDataGenerator(45)
	request := gen.GenerateBoxyDrums()

	ctx := context.Background()
	result, err := agent.Analyze(ctx, request)

	require.NoError(t, err)
	require.NotNil(t, result)

	t.Logf("📊 Drums Analysis Summary: %s", result.Analysis.Summary)

	for _, issue := range result.Analysis.Issues {
		t.Logf("   Issue [%s]: %s", issue.Severity, issue.Description)
	}

	t.Logf("\n📋 Recommendations:")
	for _, rec := range result.Recommendations {
		t.Logf("   [%s] %s", rec.Priority, rec.Description)
	}

	// Should identify boxiness/resonance
	assert.GreaterOrEqual(t, len(result.Recommendations), 1)
}

func TestSyntheticDataGenerator_GeneratesValidData(t *testing.T) {
	gen := NewSyntheticDataGenerator(123)

	testCases := []struct {
		name    string
		preset  TrackPreset
		issues  []IssueType
	}{
		{"clean_kick", PresetKick, nil},
		{"muddy_bass", PresetBass, []IssueType{IssueMuddy}},
		{"harsh_vocals", PresetVocals, []IssueType{IssueHarsh, IssueSibilant}},
		{"boxy_drums", PresetDrums, []IssueType{IssueBoxy, IssueResonant}},
		{"overcompressed_master", PresetMaster, []IssueType{IssueOverComp}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := gen.GenerateTrackAnalysis(tc.preset, tc.issues)

			// Validate structure
			require.NotNil(t, data.FrequencySpectrum)
			require.NotNil(t, data.FrequencySpectrum.EQProfile)
			require.NotNil(t, data.FrequencySpectrum.Bands)
			require.NotNil(t, data.FrequencySpectrum.SpectralFeatures)
			require.NotNil(t, data.Loudness)
			require.NotNil(t, data.Dynamics)
			require.NotNil(t, data.Stereo)
			require.NotNil(t, data.Transients)

			// Validate EQ profile
			eq := data.FrequencySpectrum.EQProfile
			assert.Equal(t, 31, len(eq.Frequencies), "Should have 31 frequency bands")
			assert.Equal(t, 31, len(eq.Magnitudes), "Should have 31 magnitudes")
			assert.Equal(t, "1/3_octave", eq.Resolution)

			// Validate frequency range
			assert.Equal(t, 20.0, eq.Frequencies[0], "Should start at 20Hz")
			assert.Equal(t, 20000.0, eq.Frequencies[30], "Should end at 20kHz")

			// Validate magnitudes are in reasonable range
			for i, m := range eq.Magnitudes {
				assert.Greater(t, m, -60.0, "Magnitude at %dHz should be > -60dB", int(eq.Frequencies[i]))
				assert.Less(t, m, 10.0, "Magnitude at %dHz should be < 10dB", int(eq.Frequencies[i]))
			}

			// Validate spectral features
			sf := data.FrequencySpectrum.SpectralFeatures
			assert.Greater(t, sf.SpectralCentroid, 0.0, "Centroid should be positive")
			assert.Greater(t, sf.SpectralRolloff, 0.0, "Rolloff should be positive")
			energySum := sf.LowFreqEnergy + sf.MidFreqEnergy + sf.HighFreqEnergy
			assert.InDelta(t, 100, energySum, 1, "Energy should sum to ~100%%")

			// Validate loudness
			assert.Less(t, data.Loudness.LUFS, 0.0, "LUFS should be negative")
			assert.Less(t, data.Loudness.Peak, 0.0, "Peak should be negative (no clipping)")

			// Validate stereo
			assert.GreaterOrEqual(t, data.Stereo.Width, 0.0)
			assert.LessOrEqual(t, data.Stereo.Width, 1.0)
			assert.GreaterOrEqual(t, data.Stereo.Correlation, -1.0)
			assert.LessOrEqual(t, data.Stereo.Correlation, 1.0)

			t.Logf("✅ %s: centroid=%.0fHz, slope=%.1fdB/oct, LUFS=%.1f, DR=%.1fdB",
				tc.name, sf.SpectralCentroid, sf.SpectralSlope, data.Loudness.LUFS, data.Dynamics.DynamicRange)
		})
	}
}

func TestSyntheticDataGenerator_IssuesAffectData(t *testing.T) {
	gen := NewSyntheticDataGenerator(456)

	// Generate clean bass
	clean := gen.GenerateTrackAnalysis(PresetBass, nil)
	// Generate muddy bass
	muddy := gen.GenerateTrackAnalysis(PresetBass, []IssueType{IssueMuddy})

	// Muddy bass should have more energy in 200-400Hz range
	cleanLowMid := clean.FrequencySpectrum.Bands.LowMid
	muddyLowMid := muddy.FrequencySpectrum.Bands.LowMid

	assert.Greater(t, muddyLowMid, cleanLowMid, "Muddy bass should have more low-mid energy")
	t.Logf("Clean low-mid: %.1fdB, Muddy low-mid: %.1fdB (diff: %.1fdB)",
		cleanLowMid, muddyLowMid, muddyLowMid-cleanLowMid)
}

func TestMixAnalysisAgent_AccuracyComparison(t *testing.T) {
	cfg := getTestConfig(t)
	agent := NewMixAnalysisAgent(cfg)
	gen := NewSyntheticDataGenerator(999)

	// Test different accuracy levels
	testCases := []struct {
		name     string
		accuracy AccuracyLevel
	}{
		{"fast", AccuracyFast},
		{"balanced", AccuracyBalanced},
		{"deep", AccuracyDeep},
	}

	ctx := context.Background()
	baseRequest := gen.GenerateMuddyBass()

	t.Logf("\n📊 ACCURACY COMPARISON TEST")
	t.Logf("═══════════════════════════════════════════════════════════════")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := &AnalysisRequest{
				Mode:         baseRequest.Mode,
				AnalysisData: baseRequest.AnalysisData,
				Context:      baseRequest.Context,
				UserRequest:  baseRequest.UserRequest,
				Accuracy:     tc.accuracy,
			}

			start := time.Now()
			result, err := agent.Analyze(ctx, request)
			elapsed := time.Since(start)

			require.NoError(t, err)
			require.NotNil(t, result)

			t.Logf("\n🎯 Accuracy: %s", tc.accuracy)
			t.Logf("   ⏱️  Time: %v", elapsed.Round(time.Millisecond))
			t.Logf("   📋 Issues: %d", len(result.Analysis.Issues))
			t.Logf("   💡 Recommendations: %d", len(result.Recommendations))
			t.Logf("   📝 Summary length: %d chars", len(result.Analysis.Summary))
		})
	}
}

func TestAnalysisRequest_Serialization(t *testing.T) {
	// Test using synthetic data
	gen := NewSyntheticDataGenerator(789)
	request := gen.GenerateMuddyBass()

	agent := &MixAnalysisAgent{}
	prompt, err := agent.buildAnalysisPrompt(request)

	require.NoError(t, err)
	assert.Contains(t, prompt, "track")
	assert.Contains(t, prompt, "Bass")
	assert.Contains(t, prompt, "eq_profile")
	assert.Contains(t, prompt, "frequencies")
	assert.Contains(t, prompt, "spectral_features")

	t.Logf("Generated prompt length: %d chars", len(prompt))
	// Log first 500 chars
	if len(prompt) > 500 {
		t.Logf("Prompt preview:\n%s...", prompt[:500])
	} else {
		t.Logf("Generated prompt:\n%s", prompt)
	}
}
