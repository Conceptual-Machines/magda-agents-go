package mix

import (
	"math"
	"math/rand"
)

// SyntheticDataGenerator creates realistic mock DSP analysis data for testing
type SyntheticDataGenerator struct {
	rng *rand.Rand
}

// NewSyntheticDataGenerator creates a new generator with optional seed
func NewSyntheticDataGenerator(seed int64) *SyntheticDataGenerator {
	return &SyntheticDataGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// GenerateThirdOctaveFrequencies returns standard 1/3 octave center frequencies (31 bands)
func GenerateThirdOctaveFrequencies() []float64 {
	return []float64{
		20, 25, 31.5, 40, 50, 63, 80, 100, 125, 160,
		200, 250, 315, 400, 500, 630, 800, 1000, 1250, 1600,
		2000, 2500, 3150, 4000, 5000, 6300, 8000, 10000, 12500, 16000,
		20000,
	}
}

// TrackPreset defines common track archetypes for generating realistic data
type TrackPreset string

const (
	PresetKick        TrackPreset = "kick"
	PresetBass        TrackPreset = "bass"
	PresetBassGuitar  TrackPreset = "bass_guitar"
	PresetDrums       TrackPreset = "drums"
	PresetSnare       TrackPreset = "snare"
	PresetHiHat       TrackPreset = "hihat"
	PresetVocals      TrackPreset = "vocals"
	PresetAcousticGtr TrackPreset = "acoustic_guitar"
	PresetElectricGtr TrackPreset = "electric_guitar"
	PresetPiano       TrackPreset = "piano"
	PresetSynth       TrackPreset = "synth"
	PresetStrings     TrackPreset = "strings"
	PresetMaster      TrackPreset = "master"
)

// IssueType defines what problems to inject into the data
type IssueType string

const (
	IssueMuddy        IssueType = "muddy"    // Too much 200-400Hz
	IssueHarsh        IssueType = "harsh"    // Too much 2-4kHz
	IssueBoxy         IssueType = "boxy"     // Resonance around 300-500Hz
	IssueThin         IssueType = "thin"     // Lacking low end
	IssueDull         IssueType = "dull"     // Lacking high end
	IssueBoomy        IssueType = "boomy"    // Excessive sub/bass
	IssueSibilant     IssueType = "sibilant" // Too much 6-8kHz
	IssueResonant     IssueType = "resonant" // Sharp resonant peak
	IssueOverComp     IssueType = "overcompressed"
	IssueTooDynamic   IssueType = "too_dynamic"
	IssuePhaseIssue   IssueType = "phase_issue"
	IssueStereoWide   IssueType = "too_wide"
	IssueStereoNarrow IssueType = "too_narrow"
)

// GenerateTrackAnalysis creates synthetic analysis data for a specific track type
func (g *SyntheticDataGenerator) GenerateTrackAnalysis(preset TrackPreset, issues []IssueType) *DSPAnalysisData {
	freqs := GenerateThirdOctaveFrequencies()
	mags := g.generateBaseMagnitudes(preset, freqs)

	// Apply issues
	for _, issue := range issues {
		mags = g.applyIssue(issue, freqs, mags)
	}

	// Generate frequency bands from the curve
	bands := g.calculateBands(freqs, mags)

	// Generate spectral features
	spectral := g.calculateSpectralFeatures(freqs, mags)

	// Generate resonances if applicable
	var resonances *ResonanceAnalysis
	for _, issue := range issues {
		if issue == IssueResonant || issue == IssueBoxy {
			resonances = g.generateResonances(preset, issues)
			break
		}
	}

	// Generate dynamics based on preset and issues
	dynamics := g.generateDynamics(preset, issues)

	// Generate loudness
	loudness := g.generateLoudness(preset, issues)

	// Generate stereo info
	stereo := g.generateStereo(preset, issues)

	// Generate transients
	transients := g.generateTransients(preset)

	return &DSPAnalysisData{
		FrequencySpectrum: &FrequencyAnalysis{
			FFTSize: 4096,
			EQProfile: &EQProfile{
				Frequencies: freqs,
				Magnitudes:  mags,
				Resolution:  "1/3_octave",
			},
			Bands:            bands,
			SpectralFeatures: spectral,
			Peaks:            g.findPeaks(freqs, mags),
		},
		Resonances: resonances,
		Loudness:   loudness,
		Dynamics:   dynamics,
		Stereo:     stereo,
		Transients: transients,
	}
}

func (g *SyntheticDataGenerator) generateBaseMagnitudes(preset TrackPreset, freqs []float64) []float64 {
	mags := make([]float64, len(freqs))

	// Base spectral shapes for different instruments
	switch preset {
	case PresetKick:
		// Strong sub and bass, quick rolloff
		for i, f := range freqs {
			if f < 60 {
				mags[i] = -8 + g.noise(2)
			} else if f < 150 {
				mags[i] = -10 + g.noise(2)
			} else if f < 500 {
				mags[i] = -18 + g.noise(3)
			} else {
				mags[i] = -25 - math.Log10(f/500)*10 + g.noise(3)
			}
		}
	case PresetBass, PresetBassGuitar:
		// Strong bass, some mids for definition
		for i, f := range freqs {
			if f < 40 {
				mags[i] = -15 + g.noise(2)
			} else if f < 200 {
				mags[i] = -8 + g.noise(2)
			} else if f < 800 {
				mags[i] = -14 + g.noise(3)
			} else if f < 3000 {
				mags[i] = -20 + g.noise(3)
			} else {
				mags[i] = -30 - math.Log10(f/3000)*8 + g.noise(3)
			}
		}
	case PresetVocals:
		// Fundamental 100-300Hz, presence 2-5kHz, air 8-12kHz
		for i, f := range freqs {
			if f < 100 {
				mags[i] = -30 + g.noise(3)
			} else if f < 400 {
				mags[i] = -12 + g.noise(2)
			} else if f < 1000 {
				mags[i] = -10 + g.noise(2)
			} else if f < 4000 {
				mags[i] = -8 + g.noise(2)
			} else if f < 10000 {
				mags[i] = -14 + g.noise(3)
			} else {
				mags[i] = -22 + g.noise(4)
			}
		}
	case PresetDrums:
		// Full range with emphasis on transients
		for i, f := range freqs {
			if f < 100 {
				mags[i] = -12 + g.noise(3)
			} else if f < 500 {
				mags[i] = -10 + g.noise(3)
			} else if f < 5000 {
				mags[i] = -12 + g.noise(4)
			} else {
				mags[i] = -16 + g.noise(4)
			}
		}
	case PresetElectricGtr:
		// Mid-focused with presence
		for i, f := range freqs {
			if f < 100 {
				mags[i] = -25 + g.noise(3)
			} else if f < 300 {
				mags[i] = -15 + g.noise(3)
			} else if f < 3000 {
				mags[i] = -10 + g.noise(3)
			} else if f < 6000 {
				mags[i] = -14 + g.noise(3)
			} else {
				mags[i] = -22 + g.noise(4)
			}
		}
	case PresetPiano:
		// Wide range, relatively flat
		for i, f := range freqs {
			if f < 50 {
				mags[i] = -20 + g.noise(2)
			} else if f < 4000 {
				mags[i] = -12 + g.noise(3)
			} else {
				mags[i] = -16 - math.Log10(f/4000)*5 + g.noise(3)
			}
		}
	case PresetMaster:
		// Balanced full mix
		for i, f := range freqs {
			// Typical music spectrum slope of about -3dB/octave
			mags[i] = -10 - math.Log2(f/1000)*3 + g.noise(2)
		}
	default:
		// Generic mid-range instrument
		for i, f := range freqs {
			mags[i] = -15 - math.Abs(math.Log10(f/1000))*5 + g.noise(3)
		}
	}

	return mags
}

func (g *SyntheticDataGenerator) applyIssue(issue IssueType, freqs, mags []float64) []float64 {
	result := make([]float64, len(mags))
	copy(result, mags)

	switch issue {
	case IssueMuddy:
		// Boost 200-400Hz
		for i, f := range freqs {
			if f >= 200 && f <= 400 {
				result[i] += 6 + g.noise(2)
			}
		}
	case IssueHarsh:
		// Boost 2-4kHz
		for i, f := range freqs {
			if f >= 2000 && f <= 4000 {
				result[i] += 5 + g.noise(2)
			}
		}
	case IssueBoxy:
		// Sharp boost around 300-500Hz
		for i, f := range freqs {
			if f >= 300 && f <= 500 {
				result[i] += 8 + g.noise(2)
			}
		}
	case IssueThin:
		// Cut low end
		for i, f := range freqs {
			if f < 200 {
				result[i] -= 8 + g.noise(2)
			}
		}
	case IssueDull:
		// Cut high end
		for i, f := range freqs {
			if f > 4000 {
				result[i] -= 6 - math.Log10(f/4000)*3 + g.noise(2)
			}
		}
	case IssueBoomy:
		// Excessive sub/bass
		for i, f := range freqs {
			if f < 150 {
				result[i] += 8 + g.noise(2)
			}
		}
	case IssueSibilant:
		// Boost 6-8kHz
		for i, f := range freqs {
			if f >= 6000 && f <= 8000 {
				result[i] += 6 + g.noise(2)
			}
		}
	case IssueResonant:
		// Add a sharp resonant peak (will be detected separately)
		peakFreq := 800 + g.rng.Float64()*2000 // Random between 800-2800Hz
		for i, f := range freqs {
			// Narrow Q boost
			distance := math.Abs(math.Log2(f / peakFreq))
			if distance < 0.2 {
				result[i] += 10 * (1 - distance/0.2)
			}
		}
	}

	return result
}

func (g *SyntheticDataGenerator) calculateBands(freqs, mags []float64) *FrequencyBands {
	bands := &FrequencyBands{}

	// Average magnitudes in each band
	for i, f := range freqs {
		switch {
		case f <= 60:
			bands.Sub = avgAccumulate(bands.Sub, mags[i])
		case f <= 250:
			bands.Bass = avgAccumulate(bands.Bass, mags[i])
		case f <= 500:
			bands.LowMid = avgAccumulate(bands.LowMid, mags[i])
		case f <= 2000:
			bands.Mid = avgAccumulate(bands.Mid, mags[i])
		case f <= 4000:
			bands.HighMid = avgAccumulate(bands.HighMid, mags[i])
		case f <= 6000:
			bands.Presence = avgAccumulate(bands.Presence, mags[i])
		default:
			bands.Brilliance = avgAccumulate(bands.Brilliance, mags[i])
		}
	}

	return bands
}

func avgAccumulate(current, value float64) float64 {
	if current == 0 {
		return value
	}
	return (current + value) / 2
}

func (g *SyntheticDataGenerator) calculateSpectralFeatures(freqs, mags []float64) *SpectralFeatures {
	// Convert dB to linear for calculations
	linear := make([]float64, len(mags))
	totalEnergy := 0.0
	weightedSum := 0.0

	for i, m := range mags {
		linear[i] = math.Pow(10, m/20)
		totalEnergy += linear[i] * linear[i]
		weightedSum += freqs[i] * linear[i] * linear[i]
	}

	centroid := weightedSum / totalEnergy

	// Calculate energy distribution
	var lowE, midE, highE float64
	for i, f := range freqs {
		e := linear[i] * linear[i]
		if f < 250 {
			lowE += e
		} else if f < 4000 {
			midE += e
		} else {
			highE += e
		}
	}

	// Calculate spectral slope (dB/octave)
	// Simple linear regression on log-frequency vs magnitude
	n := float64(len(freqs))
	var sumX, sumY, sumXY, sumX2 float64
	for i, f := range freqs {
		x := math.Log2(f)
		y := mags[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Find rolloff (frequency containing 85% of energy)
	cumEnergy := 0.0
	rolloff := freqs[len(freqs)-1]
	for i := range freqs {
		cumEnergy += linear[i] * linear[i]
		if cumEnergy >= 0.85*totalEnergy {
			rolloff = freqs[i]
			break
		}
	}

	return &SpectralFeatures{
		SpectralCentroid: centroid,
		SpectralRolloff:  rolloff,
		SpectralSlope:    slope,
		LowFreqEnergy:    lowE / totalEnergy * 100,
		MidFreqEnergy:    midE / totalEnergy * 100,
		HighFreqEnergy:   highE / totalEnergy * 100,
	}
}

func (g *SyntheticDataGenerator) findPeaks(freqs, mags []float64) []Peak {
	var peaks []Peak

	for i := 1; i < len(mags)-1; i++ {
		// Local maximum that stands out
		if mags[i] > mags[i-1] && mags[i] > mags[i+1] {
			prominence := mags[i] - (mags[i-1]+mags[i+1])/2
			if prominence > 3 { // At least 3dB above neighbors
				// Estimate Q from prominence
				q := 2 + prominence*2
				peaks = append(peaks, Peak{
					Frequency: freqs[i],
					Magnitude: mags[i],
					Q:         q,
				})
			}
		}
	}

	return peaks
}

func (g *SyntheticDataGenerator) generateResonances(preset TrackPreset, issues []IssueType) *ResonanceAnalysis {
	var resonances []Resonance

	for _, issue := range issues {
		switch issue {
		case IssueResonant:
			resonances = append(resonances, Resonance{
				Frequency: 800 + g.rng.Float64()*2000,
				Magnitude: 8 + g.rng.Float64()*4,
				Q:         15 + g.rng.Float64()*10,
				Severity:  "high",
				Type:      "equipment",
			})
		case IssueBoxy:
			resonances = append(resonances, Resonance{
				Frequency: 300 + g.rng.Float64()*200,
				Magnitude: 6 + g.rng.Float64()*3,
				Q:         8 + g.rng.Float64()*6,
				Severity:  "medium",
				Type:      "room_mode",
			})
		}
	}

	if len(resonances) == 0 {
		return nil
	}

	return &ResonanceAnalysis{
		Resonances: resonances,
		RingTime:   0.1 + g.rng.Float64()*0.2,
	}
}

func (g *SyntheticDataGenerator) generateDynamics(preset TrackPreset, issues []IssueType) *DynamicsAnalysis {
	// Base values by preset
	var dr, crest float64

	switch preset {
	case PresetKick, PresetSnare:
		dr, crest = 10, 12
	case PresetBass:
		dr, crest = 8, 6
	case PresetVocals:
		dr, crest = 12, 10
	case PresetDrums:
		dr, crest = 14, 14
	case PresetMaster:
		dr, crest = 10, 8
	default:
		dr, crest = 12, 10
	}

	// Apply issues
	for _, issue := range issues {
		switch issue {
		case IssueOverComp:
			dr -= 5
			crest -= 4
		case IssueTooDynamic:
			dr += 6
			crest += 5
		}
	}

	return &DynamicsAnalysis{
		DynamicRange: dr + g.noise(1),
		CrestFactor:  crest + g.noise(1),
	}
}

func (g *SyntheticDataGenerator) generateLoudness(preset TrackPreset, issues []IssueType) *LoudnessAnalysis {
	// Base LUFS by preset
	var lufs float64

	switch preset {
	case PresetKick:
		lufs = -18
	case PresetBass:
		lufs = -16
	case PresetVocals:
		lufs = -18
	case PresetDrums:
		lufs = -14
	case PresetMaster:
		lufs = -14
	default:
		lufs = -18
	}

	// Peak is typically 6-12dB above LUFS
	peak := lufs + 8 + g.noise(2)

	return &LoudnessAnalysis{
		RMS:      lufs + 2 + g.noise(1),
		LUFS:     lufs + g.noise(1),
		Peak:     peak,
		TruePeak: peak + 0.5 + g.noise(0.3),
	}
}

func (g *SyntheticDataGenerator) generateStereo(preset TrackPreset, issues []IssueType) *StereoAnalysis {
	// Base values by preset
	var width, corr, balance float64

	switch preset {
	case PresetKick, PresetBass:
		width, corr = 0.1, 0.98 // Mono-ish
	case PresetVocals:
		width, corr = 0.3, 0.95
	case PresetDrums:
		width, corr = 0.6, 0.85
	case PresetMaster:
		width, corr = 0.7, 0.80
	default:
		width, corr = 0.5, 0.90
	}
	balance = g.noise(0.1) // Slight random imbalance

	// Apply issues
	for _, issue := range issues {
		switch issue {
		case IssueStereoWide:
			width = 0.95
			corr = 0.5
		case IssueStereoNarrow:
			width = 0.15
			corr = 0.98
		case IssuePhaseIssue:
			corr = 0.3 + g.rng.Float64()*0.3
		}
	}

	return &StereoAnalysis{
		Width:       width,
		Correlation: corr,
		Balance:     balance,
	}
}

func (g *SyntheticDataGenerator) generateTransients(preset TrackPreset) *TransientAnalysis {
	var attack, energy float64

	switch preset {
	case PresetKick, PresetSnare:
		attack, energy = 0.002, 0.9
	case PresetBass:
		attack, energy = 0.02, 0.4
	case PresetVocals:
		attack, energy = 0.05, 0.3
	case PresetDrums:
		attack, energy = 0.005, 0.85
	case PresetPiano:
		attack, energy = 0.01, 0.7
	default:
		attack, energy = 0.02, 0.5
	}

	return &TransientAnalysis{
		AttackTime:      attack + g.noise(attack*0.2),
		TransientEnergy: energy + g.noise(0.1),
	}
}

func (g *SyntheticDataGenerator) noise(scale float64) float64 {
	return (g.rng.Float64() - 0.5) * 2 * scale
}

// Preset scenarios for common mixing problems
func (g *SyntheticDataGenerator) GenerateMuddyBass() *AnalysisRequest {
	return &AnalysisRequest{
		Mode:         ModeTrack,
		AnalysisData: g.GenerateTrackAnalysis(PresetBass, []IssueType{IssueMuddy, IssueDull}),
		Context: &AnalysisContext{
			TrackIndex: 1,
			TrackName:  "Bass",
			TrackType:  "bass",
			Genre:      "rock",
		},
		UserRequest: "The bass sounds muddy and doesn't cut through the mix",
	}
}

func (g *SyntheticDataGenerator) GenerateHarshVocals() *AnalysisRequest {
	return &AnalysisRequest{
		Mode:         ModeTrack,
		AnalysisData: g.GenerateTrackAnalysis(PresetVocals, []IssueType{IssueHarsh, IssueTooDynamic, IssueSibilant}),
		Context: &AnalysisContext{
			TrackIndex: 2,
			TrackName:  "Lead Vocal",
			TrackType:  "vocals",
			Genre:      "pop",
		},
		UserRequest: "Vocals sound harsh and inconsistent",
	}
}

func (g *SyntheticDataGenerator) GenerateBoxyDrums() *AnalysisRequest {
	return &AnalysisRequest{
		Mode:         ModeTrack,
		AnalysisData: g.GenerateTrackAnalysis(PresetDrums, []IssueType{IssueBoxy, IssueResonant}),
		Context: &AnalysisContext{
			TrackIndex: 0,
			TrackName:  "Drum Bus",
			TrackType:  "drums",
			Genre:      "rock",
		},
		UserRequest: "Drums sound boxy and have a weird ring",
	}
}

func (g *SyntheticDataGenerator) GenerateMasterForStreaming() *AnalysisRequest {
	return &AnalysisRequest{
		Mode:         ModeMaster,
		AnalysisData: g.GenerateTrackAnalysis(PresetMaster, []IssueType{}),
		Context: &AnalysisContext{
			Target: "streaming",
			Genre:  "electronic",
		},
		UserRequest: "Prepare for Spotify/Apple Music release",
	}
}

func (g *SyntheticDataGenerator) GenerateOvercompressedMaster() *AnalysisRequest {
	return &AnalysisRequest{
		Mode:         ModeMaster,
		AnalysisData: g.GenerateTrackAnalysis(PresetMaster, []IssueType{IssueOverComp}),
		Context: &AnalysisContext{
			Target: "streaming",
			Genre:  "pop",
		},
		UserRequest: "Check if the master is too squashed",
	}
}
