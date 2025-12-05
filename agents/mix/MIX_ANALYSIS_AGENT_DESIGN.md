# Mix/Analysis Agent Design

## Overview

The **Mix/Analysis Agent** is a unified agent that combines:
- **Analysis**: DSP analysis of audio content
- **Mixing**: Provides mixing advice based on analysis
- **Mastering**: Provides mastering advice based on master bus analysis

Instead of separate agents, we use one intelligent agent that understands audio analysis and provides context-appropriate recommendations.

## Core Workflow

### Standard Mode: Per-Track Analysis

```
User Request: "Make the bass sit better in the mix"
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 1: Track Selection & Bounce                            │
│ - User specifies track(s) or selects in REAPER             │
│ - REAPER bounces selected track(s) to audio                │
│ - Optional: Bounce specific time range (e.g., "the chorus")│
│ - Output: Audio file(s) ready for analysis                 │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 2: DSP Analysis (Inside REAPER)                        │
│ - Load audio into JSFX analysis plugin                      │
│ - Perform real-time DSP analysis:                          │
│   • Frequency spectrum (FFT)                                │
│   • RMS/LUFS loudness                                       │
│   • Peak levels                                             │
│   • Dynamic range                                           │
│   • Stereo width                                            │
│   • Transient analysis                                      │
│   • Harmonic content                                        │
│ - Extract analysis data as JSON                            │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 3: Send to Mix/Analysis Agent                          │
│ - POST /api/v1/magda/mix/analyze                           │
│ - Payload:                                                  │
│   {                                                         │
│     "analysis_data": {                                      │
│       "frequency_spectrum": [...],                          │
│       "loudness": {...},                                    │
│       "dynamics": {...},                                    │
│       ...                                                   │
│     },                                                      │
│     "context": {                                            │
│       "track_index": 1,                                     │
│       "track_name": "Bass",                                 │
│       "existing_fx": [...],                                 │
│       "user_request": "Make the bass sit better"           │
│     }                                                       │
│   }                                                         │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 4: Mix Agent Analysis & Recommendations                │
│ - LLM analyzes DSP data                                     │
│ - Identifies issues (muddiness, masking, frequency gaps)    │
│ - Generates recommendations:                                │
│   {                                                         │
│     "analysis": "Bass has excessive low-mid buildup...",   │
│     "recommendations": [                                    │
│       {                                                     │
│         "action": "add_fx",                                 │
│         "fx_name": "ReaEQ",                                 │
│         "track": 1,                                         │
│         "preset": "High-pass at 40Hz, cut 3dB at 250Hz"   │
│       },                                                    │
│       {                                                     │
│         "action": "modify_fx_param",                        │
│         "fx_index": 0,                                      │
│         "parameter": "ratio",                               │
│         "value": 4.0,                                       │
│         "reason": "Add subtle compression for consistency" │
│       }                                                     │
│     ]                                                       │
│   }                                                         │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 5: Review & Accept Workflow                            │
│ - REAPER shows recommendations in UI                        │
│ - User reviews each recommendation:                         │
│   • Description of issue                                    │
│   • Recommended change                                      │
│   • [Accept] [Reject] [Modify] buttons                     │
│ - User can accept all, accept selected, or modify          │
│ - Accepted changes generate REAPER actions                  │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 6: Apply Changes                                       │
│ - Accepted recommendations → REAPER actions                 │
│ - Actions applied to track(s)                               │
│ - User can undo if needed                                   │
└─────────────────────────────────────────────────────────────┘
```

## Advanced Mode: Multi-Track Relationship Analysis

### Full Mix Analysis

```
User Request: "Analyze the whole mix and optimize it"
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 1: Bounce All Tracks                                   │
│ - Bounce each track to separate audio files                 │
│ - Bounce master bus                                         │
│ - Optional: Bounce specific section (e.g., chorus)         │
│ - Maintain track relationships (timing, grouping)          │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 2: Multi-Track DSP Analysis                            │
│ - Analyze each track individually                           │
│ - Analyze master bus                                        │
│ - Perform relationship analysis:                            │
│   • Frequency masking between tracks                        │
│   • Phase relationships                                     │
│   • Stereo field distribution                               │
│   • Dynamic interaction                                     │
│   • Clarity/masking issues                                  │
│ - Generate comprehensive analysis JSON                      │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 3: Mix Agent - Relationship Analysis                   │
│ - LLM analyzes:                                             │
│   • Individual track issues                                 │
│   • Cross-track interactions                                │
│   • Frequency conflicts                                     │
│   • Stereo field balance                                    │
│   • Overall mix balance                                     │
│ - Generates prioritized recommendations:                    │
│   {                                                         │
│     "overall_analysis": "...",                              │
│     "track_issues": [                                       │
│       {"track": 1, "issue": "...", "priority": "high"},    │
│       ...                                                   │
│     ],                                                      │
│     "relationship_issues": [                                │
│       {                                                     │
│         "tracks": [1, 3],                                   │
│         "issue": "Frequency masking at 2kHz",              │
│         "recommendation": "EQ track 1: cut 2kHz, boost track 3"│
│       }                                                     │
│     ],                                                      │
│     "recommendations": [...]                                │
│   }                                                         │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 4: Review & Accept (Multi-Track)                       │
│ - UI shows:                                                 │
│   • Overall mix analysis                                    │
│   • Individual track recommendations                        │
│   • Cross-track relationship issues                         │
│   • Priority ordering                                       │
│ - User can accept/reject/modify individually                │
│ - Batch accept option for related changes                   │
└─────────────────────────────────────────────────────────────┘
```

## DSP Analysis Implementation (JSFX)

### Analysis Parameters

```javascript
// JSFX Analysis Plugin Output
{
  "frequency_spectrum": {
    "fft_size": 2048,
    "bins": [/* frequency bins */],
    "magnitude": [/* magnitude per bin */],
    "phase": [/* phase per bin */],
    "peaks": [/* peak frequencies */]
  },
  "loudness": {
    "rms": -18.5,           // dB
    "lufs": -16.2,          // LUFS
    "peak": -1.2,           // dB peak
    "true_peak": -0.8       // dB TP
  },
  "dynamics": {
    "dynamic_range": 12.3,  // dB
    "crest_factor": 8.2,    // peak/rms ratio
    "compression_ratio": 1.2
  },
  "stereo": {
    "width": 0.85,          // 0=mono, 1=full stereo
    "correlation": 0.92,    // L/R correlation
    "balance": 0.02         // -1=L, 0=center, 1=R
  },
  "transients": {
    "attack_time": 0.003,   // seconds
    "transient_energy": 0.65
  },
  "harmonics": {
    "fundamental": 82.4,    // Hz
    "harmonic_ratio": [/* harmonic content */]
  },
  "time_domain": {
    "duration": 120.5,      // seconds
    "segments": [/* time-segmented analysis */]
  }
}
```

### JSFX Plugin Structure

```javascript
// magda_analyzer.jsfx
desc: MAGDA DSP Analyzer

@slider
analyze = 0;  // Trigger analysis

@sample
// Capture audio, perform FFT, calculate metrics

@gfx
// Display analysis results (optional)

// Export function to output JSON
function export_analysis() {
  // Serialize analysis data to JSON string
  // Output via file or clipboard
}
```

### REAPER Integration

```cpp
// C++ code in magda-reaper
class MagdaAnalyzer {
    // Load JSFX analyzer on track
    // Configure analysis parameters
    // Trigger analysis
    // Capture output JSON
    // Clean up
};
```

## Mix/Analysis Agent API

### Request Format

```json
{
  "mode": "track" | "multi_track" | "master",
  "analysis_data": {
    // DSP analysis results
  },
  "context": {
    "track_index": 1,
    "track_name": "Bass",
    "time_range": {
      "start": 20.0,
      "end": 36.0
    },
    "existing_fx": [
      {
        "name": "ReaEQ",
        "index": 0,
        "parameters": {...}
      }
    ],
    "project_context": {
      "bpm": 120,
      "time_signature": "4/4",
      "key": "C major"
    },
    "user_request": "Make the bass sit better in the mix"
  }
}
```

### Response Format

```json
{
  "analysis": {
    "summary": "Bass track has excessive low-mid buildup at 250Hz causing muddiness...",
    "issues": [
      {
        "type": "frequency",
        "severity": "high",
        "description": "Excessive energy at 250Hz",
        "frequency_range": [200, 300]
      }
    ],
    "strengths": ["Good low-end foundation", "Clear transient attack"]
  },
  "recommendations": [
    {
      "id": "rec_1",
      "priority": "high",
      "description": "Reduce muddiness by cutting 250Hz",
      "action": {
        "type": "add_fx",
        "fx_name": "ReaEQ",
        "track": 1,
        "preset": {
          "bands": [
            {"type": "highpass", "freq": 40, "q": 1.0},
            {"type": "band", "freq": 250, "gain": -3.0, "q": 2.0}
          ]
        }
      },
      "explanation": "High-pass at 40Hz removes subsonic content. Cut at 250Hz reduces muddiness without affecting punch."
    },
    {
      "id": "rec_2",
      "priority": "medium",
      "description": "Add subtle compression for consistency",
      "action": {
        "type": "modify_fx_param",
        "track": 1,
        "fx_index": 1,
        "fx_name": "ReaComp",
        "parameter": "ratio",
        "value": 4.0,
        "preset": {
          "threshold": -12.0,
          "ratio": 4.0,
          "attack": 10.0,
          "release": 100.0
        }
      },
      "explanation": "Subtle compression will even out dynamics and help bass sit consistently in the mix."
    }
  ],
  "relationship_issues": [
    {
      "tracks": [1, 3],
      "issue": "Frequency masking at 2kHz between bass and guitar",
      "recommendation": {
        "track_1_action": {"type": "cut", "freq": 2000, "gain": -2.0},
        "track_3_action": {"type": "boost", "freq": 2000, "gain": 1.5}
      }
    }
  ]
}
```

## Review & Accept Workflow

### UI Components

#### Recommendation Card
```
┌─────────────────────────────────────────┐
│ 🔴 High Priority                        │
│                                         │
│ Issue: Excessive energy at 250Hz        │
│ Recommendation: Add ReaEQ, cut 250Hz    │
│ Explanation: Reduces muddiness...       │
│                                         │
│ [Accept] [Reject] [Modify] [Preview]   │
└─────────────────────────────────────────┘
```

#### Batch Actions
```
┌─────────────────────────────────────────┐
│ Selected: 3 recommendations             │
│ [Accept Selected] [Reject Selected]     │
│                                         │
│ All: [Accept All] [Reject All]         │
└─────────────────────────────────────────┘
```

### Accept Workflow

1. **Preview Mode** (Optional)
   - Apply changes temporarily
   - User can listen to result
   - Can undo preview

2. **Accept Action**
   - Generate REAPER actions from recommendation
   - Apply changes to track
   - Log change in history

3. **Modify Action**
   - User can adjust parameters
   - Re-generate recommendation based on modifications
   - Loop until user accepts

4. **Reject Action**
   - Dismiss recommendation
   - Optionally provide feedback ("not helpful", "too aggressive")

## Mastering Mode

### Master Bus Analysis

When `mode: "master"`:

```json
{
  "mode": "master",
  "analysis_data": {
    // Master bus DSP analysis
    "loudness": {
      "integrated_lufs": -14.2,
      "peak_lufs": -12.8,
      "true_peak": -1.5
    },
    "frequency_balance": {...},
    "stereo_width": 0.92,
    "dynamic_range": 10.5
  },
  "context": {
    "target": "streaming",  // or "cd", "vinyl", etc.
    "genre": "electronic",
    "user_request": "Master to streaming standards"
  }
}
```

### Mastering Recommendations

```json
{
  "recommendations": [
    {
      "description": "Increase loudness to -14 LUFS for streaming",
      "action": {
        "type": "add_fx",
        "fx_name": "ReaLimit",
        "track": "master",
        "preset": {
          "limit": -1.0,
          "ceiling": -1.0,
          "release": 50.0
        }
      }
    },
    {
      "description": "Slight high-end boost for clarity",
      "action": {
        "type": "modify_fx_param",
        "fx_name": "ReaEQ",
        "band": 4,
        "gain": 1.5,
        "freq": 8000
      }
    }
  ]
}
```

## Implementation Architecture

### Components

1. **Audio Bounce Service** (REAPER Extension)
   - `BounceTrack(track_index, time_range)` → audio file
   - `BounceAllTracks(time_range)` → multiple audio files
   - `BounceMaster(time_range)` → master bus audio

2. **DSP Analysis Service** (JSFX)
   - Real-time analysis plugin
   - Export analysis data as JSON
   - Support for multi-track relationship analysis

3. **Mix/Analysis Agent** (Go Service)
   - Receives analysis data
   - Generates recommendations
   - Supports track, multi-track, and master modes

4. **Review UI** (REAPER Extension)
   - Display recommendations
   - Accept/reject/modify workflow
   - Preview mode

5. **Action Generator**
   - Converts recommendations to REAPER actions
   - Applies changes to tracks

### File Structure

```
magda-reaper/
├── src/
│   ├── magda_bounce.cpp          # Audio bouncing
│   ├── magda_analyzer.cpp        # DSP analysis integration
│   └── magda_mix_ui.cpp          # Review & accept UI

magda-agents-go/
├── agents/
│   └── mix/
│       ├── mix_agent.go          # Unified mix/analysis agent
│       ├── analysis_handler.go   # DSP analysis processing
│       └── recommendation.go     # Recommendation generation

aideas-api/
├── internal/
│   └── api/
│       └── handlers/
│           └── mix.go            # Mix analysis endpoint
```

## API Endpoints

### Analyze Track
```
POST /api/v1/magda/mix/analyze
Content-Type: application/json

{
  "mode": "track",
  "analysis_data": {...},
  "context": {...}
}
```

### Analyze Multi-Track
```
POST /api/v1/magda/mix/analyze
Content-Type: application/json

{
  "mode": "multi_track",
  "analysis_data": {
    "tracks": [{...}, {...}],
    "master": {...},
    "relationships": {...}
  },
  "context": {...}
}
```

### Analyze Master
```
POST /api/v1/magda/mix/analyze
Content-Type: application/json

{
  "mode": "master",
  "analysis_data": {...},
  "context": {
    "target": "streaming"
  }
}
```

## Benefits of Unified Agent

1. **Single Source of Truth**: One agent understands all analysis contexts
2. **Context Awareness**: Agent can distinguish between mixing and mastering needs
3. **Consistent Recommendations**: Same underlying logic across use cases
4. **Simplified Architecture**: One agent to maintain and improve
5. **Better Context**: Agent sees full picture (track → mix → master)

## Future Enhancements

1. **Learning from Feedback**: Track accept/reject patterns
2. **Reference Track Comparison**: Analyze reference track and match characteristics
3. **Genre-Specific Recommendations**: Adjust recommendations based on genre
4. **Real-Time Analysis**: Live analysis during playback (more complex)
5. **Custom Analysis Parameters**: User-configurable analysis depth

