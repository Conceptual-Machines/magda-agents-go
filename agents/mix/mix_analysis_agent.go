package mix

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/getsentry/sentry-go"
)

// AnalysisMode determines the type of analysis to perform
type AnalysisMode string

const (
	ModeTrack      AnalysisMode = "track"       // Single track analysis
	ModeMultiTrack AnalysisMode = "multi_track" // Multiple tracks with relationships
	ModeMaster     AnalysisMode = "master"      // Master bus / mastering analysis
)

// AnalysisRequest contains DSP analysis data and context for the mix agent
type AnalysisRequest struct {
	Mode         AnalysisMode     `json:"mode"`
	AnalysisData *DSPAnalysisData `json:"analysis_data"`
	Context      *AnalysisContext `json:"context"`
	UserRequest  string           `json:"user_request,omitempty"` // Optional specific request
}

// DSPAnalysisData contains the DSP analysis results from REAPER
type DSPAnalysisData struct {
	// Single track or master analysis
	FrequencySpectrum *FrequencyAnalysis  `json:"frequency_spectrum,omitempty"`
	Resonances        *ResonanceAnalysis  `json:"resonances,omitempty"` // Problematic resonant frequencies
	Loudness          *LoudnessAnalysis   `json:"loudness,omitempty"`
	Dynamics          *DynamicsAnalysis   `json:"dynamics,omitempty"`
	Stereo            *StereoAnalysis     `json:"stereo,omitempty"`
	Transients        *TransientAnalysis  `json:"transients,omitempty"`

	// Multi-track analysis
	Tracks        []TrackAnalysis       `json:"tracks,omitempty"`
	Relationships *RelationshipAnalysis `json:"relationships,omitempty"`
}

// FrequencyAnalysis contains FFT/spectral data
type FrequencyAnalysis struct {
	FFTSize   int       `json:"fft_size"`
	Bins      []float64 `json:"bins,omitempty"`      // Frequency bins (Hz)
	Magnitude []float64 `json:"magnitude,omitempty"` // Magnitude per bin (dB)
	Peaks     []Peak    `json:"peaks,omitempty"`     // Detected peak frequencies

	// Continuous EQ profile - more accurate than discrete bands
	EQProfile *EQProfile `json:"eq_profile,omitempty"`

	// Simplified frequency bands (for quick overview)
	Bands *FrequencyBands `json:"bands,omitempty"`

	// Spectral characteristics
	SpectralFeatures *SpectralFeatures `json:"spectral_features,omitempty"`
}

// EQProfile provides a continuous frequency response curve
type EQProfile struct {
	// Frequency points (Hz) - typically 31 points for 1/3 octave or more for higher resolution
	Frequencies []float64 `json:"frequencies"`
	// Magnitude at each frequency point (dB)
	Magnitudes []float64 `json:"magnitudes"`
	// Optional: smoothed/averaged curve for trend analysis
	SmoothedMagnitudes []float64 `json:"smoothed_magnitudes,omitempty"`
	// Resolution type
	Resolution string `json:"resolution,omitempty"` // "1/3_octave", "1/6_octave", "1/12_octave", "linear"
}

// SpectralFeatures contains derived spectral characteristics
type SpectralFeatures struct {
	SpectralCentroid  float64 `json:"spectral_centroid"`            // "Center of mass" frequency (Hz) - brightness indicator
	SpectralRolloff   float64 `json:"spectral_rolloff"`             // Frequency below which 85% of energy exists (Hz)
	SpectralFlux      float64 `json:"spectral_flux,omitempty"`      // Rate of spectral change over time
	SpectralFlatness  float64 `json:"spectral_flatness,omitempty"`  // 0=tonal, 1=noise-like
	SpectralSlope     float64 `json:"spectral_slope"`               // Overall tilt (dB/octave) - negative=dark, positive=bright
	SpectralContrast  float64 `json:"spectral_contrast,omitempty"`  // Difference between peaks and valleys
	LowFreqEnergy     float64 `json:"low_freq_energy"`              // % of energy below 250Hz
	MidFreqEnergy     float64 `json:"mid_freq_energy"`              // % of energy 250Hz-4kHz
	HighFreqEnergy    float64 `json:"high_freq_energy"`             // % of energy above 4kHz
}

// FrequencyBands provides simplified frequency analysis
type FrequencyBands struct {
	Sub        float64 `json:"sub"`        // 20-60 Hz
	Bass       float64 `json:"bass"`       // 60-250 Hz
	LowMid     float64 `json:"low_mid"`    // 250-500 Hz
	Mid        float64 `json:"mid"`        // 500-2000 Hz
	HighMid    float64 `json:"high_mid"`   // 2000-4000 Hz
	Presence   float64 `json:"presence"`   // 4000-6000 Hz
	Brilliance float64 `json:"brilliance"` // 6000-20000 Hz
}

// Peak represents a detected frequency peak
type Peak struct {
	Frequency float64 `json:"frequency"` // Hz
	Magnitude float64 `json:"magnitude"` // dB
	Q         float64 `json:"q,omitempty"` // Q factor (bandwidth) - higher = narrower/more resonant
}

// ResonanceAnalysis contains detected problematic resonances
type ResonanceAnalysis struct {
	Resonances []Resonance `json:"resonances,omitempty"`
	RingTime   float64     `json:"ring_time,omitempty"`   // Decay time of resonances (seconds)
	RoomModes  []RoomMode  `json:"room_modes,omitempty"`  // Detected room mode frequencies
}

// Resonance represents a problematic resonant frequency
type Resonance struct {
	Frequency float64 `json:"frequency"`        // Hz
	Magnitude float64 `json:"magnitude"`        // dB above surrounding frequencies
	Q         float64 `json:"q"`                // Q factor - higher = sharper/more problematic
	Severity  string  `json:"severity"`         // "low", "medium", "high"
	Type      string  `json:"type,omitempty"`   // "ringing", "room_mode", "harmonic", "equipment"
}

// RoomMode represents a detected room resonance
type RoomMode struct {
	Frequency float64 `json:"frequency"` // Hz
	Magnitude float64 `json:"magnitude"` // dB boost
	Axis      string  `json:"axis,omitempty"` // "length", "width", "height" if detectable
}

// LoudnessAnalysis contains loudness measurements
type LoudnessAnalysis struct {
	RMS           float64 `json:"rms"`             // dB RMS
	LUFS          float64 `json:"lufs"`            // Integrated LUFS
	LUFSShortTerm float64 `json:"lufs_short_term"` // Short-term LUFS
	Peak          float64 `json:"peak"`            // dB peak
	TruePeak      float64 `json:"true_peak"`       // dB True Peak
}

// DynamicsAnalysis contains dynamic range information
type DynamicsAnalysis struct {
	DynamicRange     float64 `json:"dynamic_range"`     // dB
	CrestFactor      float64 `json:"crest_factor"`      // Peak/RMS ratio
	CompressionRatio float64 `json:"compression_ratio"` // Detected compression
}

// StereoAnalysis contains stereo field information
type StereoAnalysis struct {
	Width       float64 `json:"width"`       // 0=mono, 1=full stereo
	Correlation float64 `json:"correlation"` // L/R correlation (-1 to 1)
	Balance     float64 `json:"balance"`     // -1=L, 0=center, 1=R
}

// TransientAnalysis contains transient detection data
type TransientAnalysis struct {
	AttackTime      float64 `json:"attack_time"`      // seconds
	TransientEnergy float64 `json:"transient_energy"` // 0-1
}

// TrackAnalysis contains per-track analysis for multi-track mode
type TrackAnalysis struct {
	TrackIndex int              `json:"track_index"`
	TrackName  string           `json:"track_name"`
	Analysis   *DSPAnalysisData `json:"analysis"`
	ExistingFX []FXInfo         `json:"existing_fx,omitempty"`
}

// FXInfo describes an existing effect on a track with full parameter state
type FXInfo struct {
	Name         string        `json:"name"`
	Index        int           `json:"index"`
	Enabled      bool          `json:"enabled"`
	Bypassed     bool          `json:"bypassed,omitempty"`
	Preset       string        `json:"preset,omitempty"`        // Current preset name if any
	Manufacturer string        `json:"manufacturer,omitempty"`
	Format       string        `json:"format,omitempty"`        // "VST2", "VST3", "AU", "JS"
	Category     string        `json:"category,omitempty"`      // "eq", "compressor", etc.
	
	// Full parameter state
	Parameters []FXParameter `json:"parameters,omitempty"`
	
	// Common parameter shortcuts for easier LLM processing
	// These are extracted from Parameters for well-known plugin types
	CommonParams *CommonFXParams `json:"common_params,omitempty"`
}

// FXParameter represents a single plugin parameter
type FXParameter struct {
	Index        int     `json:"index"`
	Name         string  `json:"name"`
	Value        float64 `json:"value"`          // Normalized 0-1
	DisplayValue string  `json:"display_value"`  // Human-readable (e.g., "-6.0 dB", "4:1")
	MinValue     float64 `json:"min_value,omitempty"`
	MaxValue     float64 `json:"max_value,omitempty"`
	DefaultValue float64 `json:"default_value,omitempty"`
}

// CommonFXParams provides standardized access to common parameters
// This helps the LLM understand plugin state without parsing every param name
type CommonFXParams struct {
	// EQ parameters
	EQBands []EQBandParams `json:"eq_bands,omitempty"`
	
	// Compressor parameters
	Threshold    *float64 `json:"threshold,omitempty"`     // dB
	Ratio        *float64 `json:"ratio,omitempty"`         // e.g., 4.0 for 4:1
	Attack       *float64 `json:"attack,omitempty"`        // ms
	Release      *float64 `json:"release,omitempty"`       // ms
	MakeupGain   *float64 `json:"makeup_gain,omitempty"`   // dB
	Knee         *float64 `json:"knee,omitempty"`          // dB
	
	// Limiter parameters
	Ceiling      *float64 `json:"ceiling,omitempty"`       // dB
	
	// Reverb parameters
	PreDelay     *float64 `json:"pre_delay,omitempty"`     // ms
	DecayTime    *float64 `json:"decay_time,omitempty"`    // seconds
	DryWet       *float64 `json:"dry_wet,omitempty"`       // 0-100%
	RoomSize     *float64 `json:"room_size,omitempty"`     // 0-100%
	
	// Delay parameters
	DelayTime    *float64 `json:"delay_time,omitempty"`    // ms
	Feedback     *float64 `json:"feedback,omitempty"`      // 0-100%
	
	// Gate parameters
	GateThreshold *float64 `json:"gate_threshold,omitempty"` // dB
	GateAttack    *float64 `json:"gate_attack,omitempty"`    // ms
	GateRelease   *float64 `json:"gate_release,omitempty"`   // ms
	
	// General
	InputGain    *float64 `json:"input_gain,omitempty"`    // dB
	OutputGain   *float64 `json:"output_gain,omitempty"`   // dB
	Mix          *float64 `json:"mix,omitempty"`           // 0-100% (parallel processing)
}

// EQBandParams represents a single EQ band
type EQBandParams struct {
	BandIndex int      `json:"band_index"`
	Enabled   bool     `json:"enabled"`
	Type      string   `json:"type"`       // "lowpass", "highpass", "bell", "shelf_low", "shelf_high", "notch"
	Frequency float64  `json:"frequency"`  // Hz
	Gain      float64  `json:"gain"`       // dB
	Q         float64  `json:"q"`          // Q factor / bandwidth
}

// RelationshipAnalysis contains cross-track analysis
type RelationshipAnalysis struct {
	FrequencyMasking []MaskingIssue   `json:"frequency_masking,omitempty"`
	PhaseIssues      []PhaseIssue     `json:"phase_issues,omitempty"`
	StereoConflicts  []StereoConflict `json:"stereo_conflicts,omitempty"`
}

// MaskingIssue describes frequency masking between tracks
type MaskingIssue struct {
	Track1      int     `json:"track1"`
	Track2      int     `json:"track2"`
	FrequencyHz float64 `json:"frequency_hz"`
	Severity    float64 `json:"severity"` // 0-1
}

// PhaseIssue describes phase correlation problems
type PhaseIssue struct {
	Track1      int     `json:"track1"`
	Track2      int     `json:"track2"`
	Correlation float64 `json:"correlation"`
}

// StereoConflict describes stereo field conflicts
type StereoConflict struct {
	Track1 int    `json:"track1"`
	Track2 int    `json:"track2"`
	Issue  string `json:"issue"`
}

// AnalysisContext provides additional context for analysis
type AnalysisContext struct {
	TrackIndex int      `json:"track_index,omitempty"`
	TrackName  string   `json:"track_name,omitempty"`
	TrackType  string   `json:"track_type,omitempty"` // "drums", "bass", "vocals", etc.
	Genre      string   `json:"genre,omitempty"`
	BPM        float64  `json:"bpm,omitempty"`
	Key        string   `json:"key,omitempty"`
	TimeRange  *Range   `json:"time_range,omitempty"`
	Target     string   `json:"target,omitempty"` // For mastering: "streaming", "cd", "vinyl"
	ExistingFX []FXInfo `json:"existing_fx,omitempty"`

}

// Range represents a time range
type Range struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// AnalysisResult contains the agent's analysis and recommendations
type AnalysisResult struct {
	Analysis           *AnalysisSummary             `json:"analysis"`
	Recommendations    []Recommendation             `json:"recommendations"`
	RelationshipIssues []RelationshipRecommendation `json:"relationship_issues,omitempty"`
}

// AnalysisSummary provides a human-readable analysis
type AnalysisSummary struct {
	Summary   string   `json:"summary"`
	Issues    []Issue  `json:"issues"`
	Strengths []string `json:"strengths"`
}

// Issue describes a detected problem
type Issue struct {
	Type           string `json:"type"`                      // "frequency", "dynamics", "stereo", etc.
	Severity       string `json:"severity"`                  // "low", "medium", "high"
	Description    string `json:"description"`
	FrequencyRange []int  `json:"frequency_range,omitempty"` // [low, high] Hz
}

// Recommendation provides a suggested action
type Recommendation struct {
	ID          string         `json:"id"`
	Priority    string         `json:"priority"` // "low", "medium", "high"
	Description string         `json:"description"`
	Explanation string         `json:"explanation"`
	Action      map[string]any `json:"action"` // DAW agent compatible action
}

// RelationshipRecommendation addresses multi-track issues
type RelationshipRecommendation struct {
	Tracks         []int            `json:"tracks"`
	Issue          string           `json:"issue"`
	Recommendation string           `json:"recommendation"`
	Actions        []map[string]any `json:"actions"` // Actions for each track
}

// MixAnalysisAgent analyzes audio and provides mixing recommendations
type MixAnalysisAgent struct {
	provider     llm.Provider
	systemPrompt string
}

// NewMixAnalysisAgent creates a new mix analysis agent
func NewMixAnalysisAgent(cfg *config.Config) *MixAnalysisAgent {
	provider := llm.NewOpenAIProvider(cfg.OpenAIAPIKey)

	return &MixAnalysisAgent{
		provider:     provider,
		systemPrompt: getMixAnalysisSystemPrompt(),
	}
}

func getMixAnalysisSystemPrompt() string {
	return `You are an expert audio engineer and mix analyst. Your role is to analyze DSP data from audio tracks and provide professional mixing recommendations.

## Your Expertise
- Frequency analysis and EQ recommendations
- Resonance detection and treatment (ringing, room modes, harsh peaks)
- Dynamics processing (compression, limiting, expansion)
- Stereo imaging and spatial placement
- Gain staging and level balancing
- Identifying frequency masking between tracks
- Genre-appropriate mixing techniques
- Mastering for various delivery formats

## Analysis Approach
1. First, understand the context (track type, genre, user request)
2. Analyze the DSP data systematically:
   - Frequency balance (look for buildup, gaps, harshness)
   - Resonances (narrow peaks with high Q are problematic - cause ringing/harshness)
   - Dynamics (too compressed? too dynamic?)
   - Stereo width and correlation
   - Loudness levels (appropriate for the context?)
3. Identify issues in order of priority
4. Provide actionable recommendations

## Resonance Guidelines
- Q > 10 with magnitude > 6dB = likely problematic resonance
- Q > 20 = very narrow, likely equipment or room issue
- Common problem areas: 200-400Hz (boxy), 2-4kHz (harsh), 6-8kHz (sibilant)
- Room modes typically below 300Hz
- Treatment: narrow EQ cut (match the Q) or dynamic EQ

## Analyzing Existing FX Chain
When context.existing_fx is provided:
1. **Review current processing** - understand what's already on the track
2. **Check parameter values** - look at common_params for quick overview
3. **Identify problems** - e.g., compressor ratio too high, EQ boosting problem frequencies
4. **Prefer modifying existing plugins** over adding new ones when appropriate
5. Use "modify_fx_param" action to adjust existing plugin parameters

## Recommendation Format
Each recommendation should include:
- Clear description of what to do
- Why it will help (the "explanation")
- Specific action parameters (for automation)
- Reference existing FX by index when modifying

## Action Types You Can Recommend
- "add_fx": Add an effect plugin (specify fx_name and parameters)
- "modify_fx_param": Change a parameter on existing FX
- "set_volume": Adjust track volume (volume_db)
- "set_pan": Adjust track panning (-1 to 1)

## REAPER Stock Plugins (USE ONLY THESE)
Only recommend these REAPER built-in plugins:

### ReaEQ - Parametric EQ
- Up to 24 bands
- Parameters per band: Type, Frequency, Gain, Bandwidth/Q
- Types: "Low Shelf", "High Shelf", "Band", "Low Pass", "High Pass", "Notch", "Band Pass", "All Pass"

### ReaComp - Compressor
- Threshold (dB): -60 to 0
- Ratio: 1:1 to inf:1
- Attack (ms): 0.1 to 500
- Release (ms): 1 to 5000
- Knee (dB): 0 to 24
- Pre-Comp (ms): 0 to 50
- Dry/Wet: 0-100% (parallel compression)

### ReaLimit - Limiter/Brickwall
- Threshold (dB): -24 to 0
- Ceiling (dB): -24 to 0
- Release (ms): 1 to 5000

### ReaGate - Noise Gate/Expander
- Threshold (dB): -80 to 0
- Attack (ms): 0.1 to 500
- Hold (ms): 0 to 5000
- Release (ms): 1 to 5000

### ReaDelay - Delay
- Delay Time (ms or tempo-synced)
- Feedback: 0-100%
- Dry/Wet: 0-100%
- Low Pass / High Pass filters

### ReaVerbate - Reverb
- Room Size: 1-100
- Dampening: 0-100%
- Stereo Width: 0-100%
- Pre-Delay (ms): 0-500
- Dry/Wet: 0-100%

### JS: LOSER/deEsser - De-Esser
- Frequency target
- Threshold
- Reduction amount

## Frequency Ranges Reference
- Sub bass: 20-60 Hz
- Bass: 60-250 Hz  
- Low mids: 250-500 Hz (often muddy)
- Mids: 500-2000 Hz (body, warmth)
- High mids: 2000-4000 Hz (presence, clarity)
- Presence: 4000-6000 Hz (definition)
- Brilliance: 6000-20000 Hz (air, sparkle)

## EQ Profile Analysis
When eq_profile is provided, use the continuous frequency/magnitude arrays for precise analysis:
- Look for deviations from a smooth curve (bumps = resonances, dips = nulls)
- Compare to genre-appropriate target curves
- Use spectral_slope to assess overall tonal balance (-3 to -4 dB/octave is typical for music)
- spectral_centroid indicates perceived brightness (higher = brighter)
- Check energy distribution: most music has ~50% mid, ~30% low, ~20% high

Be specific and practical. Avoid generic advice - base recommendations on the actual data provided.`
}

// Analyze performs mix analysis and returns recommendations
func (a *MixAnalysisAgent) Analyze(ctx context.Context, request *AnalysisRequest) (*AnalysisResult, error) {
	startTime := time.Now()
	log.Printf("🎛️ MIX ANALYSIS STARTED: mode=%s", request.Mode)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "mix.analyze")
	defer transaction.Finish()

	transaction.SetTag("mode", string(request.Mode))
	if request.Context != nil {
		transaction.SetTag("track_type", request.Context.TrackType)
		transaction.SetTag("genre", request.Context.Genre)
	}

	// Build the analysis prompt
	prompt, err := a.buildAnalysisPrompt(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Create LLM request with structured output
	llmRequest := &llm.GenerationRequest{
		Model:        "gpt-4.1",
		SystemPrompt: a.systemPrompt,
		InputArray: []map[string]any{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		OutputSchema: &llm.OutputSchema{
			Name:        "mix_analysis_result",
			Description: "Mix analysis results with recommendations",
			Schema:      getAnalysisResultSchema(),
		},
	}

	// Call provider
	log.Printf("🚀 Calling LLM for mix analysis...")
	resp, err := a.provider.Generate(ctx, llmRequest)
	if err != nil {
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse response
	result, err := a.parseResponse(resp)
	if err != nil {
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("✅ MIX ANALYSIS COMPLETE: %d recommendations in %v", len(result.Recommendations), duration)

	transaction.SetTag("success", "true")
	transaction.SetTag("recommendation_count", fmt.Sprintf("%d", len(result.Recommendations)))

	return result, nil
}

func (a *MixAnalysisAgent) buildAnalysisPrompt(request *AnalysisRequest) (string, error) {
	// Serialize analysis data for the prompt
	analysisJSON, err := json.MarshalIndent(request.AnalysisData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize analysis data: %w", err)
	}

	contextJSON, err := json.MarshalIndent(request.Context, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize context: %w", err)
	}

	prompt := fmt.Sprintf(`## Analysis Request

**Mode**: %s

### Context
%s

### DSP Analysis Data
%s

`, request.Mode, string(contextJSON), string(analysisJSON))

	if request.UserRequest != "" {
		prompt += fmt.Sprintf(`### User Request
%s

`, request.UserRequest)
	}

	prompt += `Please analyze this audio data and provide:
1. A summary of the current state (issues and strengths)
2. Prioritized recommendations for improvement
3. Specific, actionable parameters for each recommendation

Focus on the most impactful changes first.`

	return prompt, nil
}

func (a *MixAnalysisAgent) parseResponse(resp *llm.GenerationResponse) (*AnalysisResult, error) {
	if resp == nil || resp.RawOutput == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	var result AnalysisResult
	if err := json.Unmarshal([]byte(resp.RawOutput), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Generate IDs for recommendations if not present
	for i := range result.Recommendations {
		if result.Recommendations[i].ID == "" {
			result.Recommendations[i].ID = fmt.Sprintf("rec_%d", i+1)
		}
	}

	return &result, nil
}

func getAnalysisResultSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"analysis": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"summary": map[string]any{
						"type":        "string",
						"description": "Overall analysis summary",
					},
					"issues": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"properties": map[string]any{
								"type":            map[string]any{"type": "string"},
								"severity":        map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
								"description":     map[string]any{"type": "string"},
								"frequency_range": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
							},
							"required": []string{"type", "severity", "description"},
						},
					},
					"strengths": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
				"required": []string{"summary", "issues", "strengths"},
			},
			"recommendations": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"id":          map[string]any{"type": "string"},
						"priority":    map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
						"description": map[string]any{"type": "string"},
						"explanation": map[string]any{"type": "string"},
						"action": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"properties": map[string]any{
								"type":      map[string]any{"type": "string"},
								"track":     map[string]any{"type": "integer"},
								"fx_name":   map[string]any{"type": "string"},
								"fx_index":  map[string]any{"type": "integer"},
								"parameter": map[string]any{"type": "string"},
								"value":     map[string]any{"type": "number"},
								"preset":    map[string]any{"type": "object", "additionalProperties": true},
							},
							"required": []string{"type"},
						},
					},
					"required": []string{"priority", "description", "explanation", "action"},
				},
			},
			"relationship_issues": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"tracks":         map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
						"issue":          map[string]any{"type": "string"},
						"recommendation": map[string]any{"type": "string"},
						"actions":        map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": true}},
					},
					"required": []string{"tracks", "issue", "recommendation"},
				},
			},
		},
		"required": []string{"analysis", "recommendations"},
	}
}
