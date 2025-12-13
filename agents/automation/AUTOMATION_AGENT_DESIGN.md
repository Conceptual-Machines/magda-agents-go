# Automation Agent Design

## Overview

The **Automation Agent** is a specialized agent that understands how to draw automation curves for volume, pan, FX parameters, and other automatable properties in REAPER.

Unlike simple "set volume to -3dB" commands, automation involves:
- Drawing curves over time
- Interpolation between points
- Complex shapes (swells, fades, ramps, s-curves)
- Musical timing context (bars, beats, measures)

## Use Cases

### Volume Automation
- "Volume swell from bar 5 to bar 9"
- "Fade out the last 4 bars"
- "Duck the bass during the kick drum hits"

### Pan Automation
- "Pan the guitar from left to right over 8 bars"
- "Auto-pan the synth at 2Hz"

### FX Parameter Automation
- "Sweep the filter cutoff from 1000Hz to 8000Hz over the bridge"
- "Gradually increase reverb mix during the chorus"
- "Add wobble effect that speeds up during the drop"

### Complex Curves
- "Smooth volume envelope that follows the melody"
- "Exponential fade in for the intro"
- "Logarithmic reverb decay automation"

## Automation Agent Responsibilities

### 1. Curve Understanding
- **Linear**: Constant rate of change
- **Exponential**: Faster change at beginning/end
- **Logarithmic**: Slower change at beginning/end
- **S-Curve**: Smooth acceleration then deceleration
- **Bezier**: Custom control points
- **Stepped**: No interpolation (discrete values)
- **Sine/Cosine**: Oscillating patterns
- **Custom**: User-described shapes

### 2. Musical Context
- Convert "from bar 5 to bar 9" → time positions
- Understand musical phrasing (intro, verse, chorus, bridge)
- Align with beat grid
- Respect time signature and tempo

### 3. Parameter Mapping
- "Volume" → track volume envelope
- "Pan" → track pan envelope
- "Filter cutoff" → specific FX parameter envelope
- "Reverb mix" → FX parameter envelope

### 4. Value Interpretation
- **Absolute**: "Set to -3dB" (exact value)
- **Relative**: "Increase by 6dB" (delta from current)
- **Normalized**: "0.0 to 1.0" (normalized range)
- **Musical**: "piano to forte" (dynamic markings)
- **Percentage**: "50% to 100%" (percent of range)

## Automation Data Structure

```go
type AutomationPoint struct {
    Time      float64 `json:"time"`      // Time in seconds (or beats)
    Value     float64 `json:"value"`     // Parameter value
    Shape     string  `json:"shape"`     // "linear", "square", "slow", "fast", "bezier"
    Tension   float64 `json:"tension"`   // Bezier tension (0.0-1.0, optional)
}

type AutomationCurve struct {
    Parameter string            `json:"parameter"` // "volume", "pan", "fx_param:0:cutoff"
    Points    []AutomationPoint `json:"points"`
    Interpolation string        `json:"interpolation"` // "linear", "square", "slow", "fast", "bezier"
    Shape     string            `json:"shape"`         // Overall curve shape
}

type AutomationRequest struct {
    Description string          `json:"description"` // "Volume swell from bar 5 to bar 9"
    Target      string          `json:"target"`      // Track index, FX instance, etc.
    Curve       AutomationCurve `json:"curve"`
    TimeMode    string          `json:"time_mode"`   // "seconds", "beats", "bars"
}
```

## Coordination with Other Agents

### Workflow

```
User Request: "Volume swell from bar 5 to bar 9 on track 1"
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 1: DAW Agent                                           │
│ - Detects: "Volume swell from bar 5 to bar 9 on track 1"   │
│ - Generates:                                                │
│   • set_track_automation(                                   │
│       track: 1,                                             │
│       parameter: "volume",                                  │
│       curve: <PLACEHOLDER_AUTOMATION_CURVE>                 │
│     )                                                       │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 2: Automation Agent                                    │
│ - Parses: "Volume swell from bar 5 to bar 9"               │
│ - Determines:                                               │
│   • Start: bar 5, value: -inf dB (or current value)        │
│   • End: bar 9, value: 0 dB (or target value)              │
│   • Shape: "swell" → exponential/logarithmic curve          │
│ - Generates AutomationCurve:                                │
│   {                                                         │
│     parameter: "volume",                                    │
│     points: [                                               │
│       {time: 20.0, value: -60.0, shape: "slow"},          │
│       {time: 36.0, value: 0.0, shape: "fast"}             │
│     ],                                                      │
│     interpolation: "bezier",                                │
│     shape: "swell"                                          │
│   }                                                         │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 3: Orchestrator (Injection)                            │
│ - Replaces <PLACEHOLDER_AUTOMATION_CURVE> with curve data   │
└─────────────────────────────────────────────────────────────┘
```

### Integration Points

**With DAW Agent:**
- DAW agent knows track structure, FX instances, parameter IDs
- DAW agent creates placeholder actions: `set_track_automation`, `set_fx_automation`

**With Arranger Agent:**
- Arranger may provide musical context: "during the chorus", "on beat 3"
- Automation agent can align curves with musical events

## Automation Types

### 1. Volume Automation

**Simple Fade:**
```
"Fade in from silence over 2 bars"
→ Points: [{time: 0, value: -inf}, {time: 8, value: 0}]
→ Shape: logarithmic
```

**Swell:**
```
"Volume swell from bar 5 to bar 9"
→ Points: [{time: 20, value: -20}, {time: 36, value: 0}]
→ Shape: exponential (faster at end)
```

**Dynamic Ducking:**
```
"Duck bass by 6dB during kick hits"
→ Requires: Kick timing from audio analysis or MIDI
→ Points: Multiple points at kick times
→ Shape: stepped or fast attack/release
```

### 2. Pan Automation

**Sweep:**
```
"Pan guitar from left to right over 8 bars"
→ Points: [{time: 0, value: -1.0}, {time: 32, value: 1.0}]
→ Shape: linear or s-curve
```

**Auto-Pan:**
```
"Auto-pan synth at 2Hz"
→ Points: Sine wave points over time
→ Shape: sine interpolation
```

### 3. FX Parameter Automation

**Filter Sweep:**
```
"Sweep filter cutoff from 1000Hz to 8000Hz during bridge"
→ Parameter: "fx_param:0:cutoff" (FX index 0, parameter "cutoff")
→ Points: [{time: 80, value: 1000}, {time: 112, value: 8000}]
→ Shape: exponential
```

**Reverb Mix:**
```
"Gradually increase reverb mix during chorus"
→ Parameter: "fx_param:1:wet_mix"
→ Points: [{time: 64, value: 0.1}, {time: 96, value: 0.8}]
→ Shape: linear or logarithmic
```

### 4. Complex Patterns

**Following Musical Shape:**
```
"Volume envelope that follows the melody contour"
→ Requires: MIDI note analysis
→ Points: Derived from note velocities and timing
→ Shape: Custom bezier curves
```

**Modulation Patterns:**
```
"LFO-style modulation on delay feedback"
→ Points: Sine/cosine wave pattern
→ Shape: Sine interpolation with configurable frequency
```

## Natural Language Parsing

### Time References
- **Bars**: "from bar 5 to bar 9", "bars 1-4", "the intro"
- **Beats**: "on beat 3", "over 4 beats"
- **Musical Sections**: "during the chorus", "in the verse", "at the bridge"
- **Duration**: "over 8 bars", "for 4 measures", "lasting 32 beats"

### Value References
- **Absolute**: "to -3dB", "set to 50%", "at 8000Hz"
- **Relative**: "increase by 6dB", "reduce by 50%"
- **Musical**: "from silence to full volume", "piano to forte"
- **Range**: "from 20% to 80%", "between 0.5 and 1.0"

### Curve Shapes
- **Fade**: "fade in", "fade out"
- **Swell**: "swell", "crescendo", "build"
- **Drop**: "drop", "decrescendo", "diminuendo"
- **Smooth**: "smooth", "gradual", "gentle"
- **Sharp**: "sharp", "sudden", "quick"
- **Oscillating**: "auto-pan", "wobble", "LFO", "modulation"
- **Following**: "follow the melody", "track the kick"

## REAPER API Mapping

### Envelope Creation
```cpp
// Track envelope
TrackEnvelope* GetTrackEnvelope(MediaTrack* track, int envelopeindex);
// 0 = volume, 1 = pan, 2 = width, etc.

// FX parameter envelope
TrackEnvelope* GetFXEnvelope(MediaTrack* track, int fxindex, int parameterindex, bool create);
```

### Point Insertion
```cpp
int InsertEnvelopePoint(TrackEnvelope* envelope, double time, double value, int shape, double tension, bool selected);
// shape: 0=linear, 1=square, 2=slow, 3=fast, 4=bezier
```

### Point Deletion/Modification
```cpp
bool DeleteEnvelopePoint(TrackEnvelope* envelope, int pointindex);
bool SetEnvelopePoint(TrackEnvelope* envelope, int pointindex, double* timeInOptional, double* valueInOptional, int* shapeInOptional, double* tensionInOptional, bool* selectedInOptional);
```

## Action Format

### DAW Action with Placeholder
```json
{
  "action": "set_track_automation",
  "track": 1,
  "parameter": "volume",
  "curve": {
    "_placeholder": true,
    "_type": "automation_curve",
    "_description": "Volume swell from bar 5 to bar 9",
    "_time_mode": "bars"
  }
}
```

### Automation Agent Output
```json
{
  "parameter": "volume",
  "points": [
    {
      "time": 20.0,
      "value": -60.0,
      "shape": "slow",
      "tension": 0.5
    },
    {
      "time": 36.0,
      "value": 0.0,
      "shape": "fast",
      "tension": 0.5
    }
  ],
  "interpolation": "bezier",
  "shape": "swell"
}
```

### Final Injected Action
```json
{
  "action": "set_track_automation",
  "track": 1,
  "parameter": "volume",
  "curve": {
    "parameter": "volume",
    "points": [
      {"time": 20.0, "value": -60.0, "shape": "slow", "tension": 0.5},
      {"time": 36.0, "value": 0.0, "shape": "fast", "tension": 0.5}
    ],
    "interpolation": "bezier",
    "shape": "swell"
  }
}
```

## Implementation Requirements

### 1. Automation Agent Service

**File**: `agents/automation/automation_agent.go`

```go
package automation

type AutomationAgent struct {
    provider llm.Provider
    // ... config
}

type AutomationResult struct {
    Curve AutomationCurve `json:"curve"`
    Usage any             `json:"usage"`
}

func (a *AutomationAgent) GenerateCurve(
    ctx context.Context,
    description string,
    target string,  // "track:1", "fx:0:param:cutoff"
    timeMode string, // "bars", "beats", "seconds"
    projectContext map[string]interface{}, // BPM, time signature, etc.
) (*AutomationResult, error) {
    // Use LLM to parse description and generate curve points
    // Consider musical context, curve shapes, interpolation types
}
```

### 2. Orchestrator Extension

Update `agents/coordination/orchestrator.go` to handle automation placeholders:

```go
func (o *Orchestrator) ProcessRequest(...) {
    // ... existing code for musical content
    
    // Also check for automation placeholders
    automationPlaceholders := findAutomationPlaceholders(dawResult.Actions)
    for _, placeholder := range automationPlaceholders {
        curve, err := o.automationAgent.GenerateCurve(
            ctx,
            placeholder["_description"].(string),
            placeholder["_target"].(string),
            placeholder["_time_mode"].(string),
            state,
        )
        injectAutomationCurve(dawResult.Actions, placeholder, curve.Curve)
    }
}
```

### 3. DAW Parser Support

Update `agents/daw/dsl_parser*.go` to detect automation requests and create placeholders:

```go
// Detect patterns like:
// - "fade in", "fade out", "swell"
// - "pan from left to right"
// - "automate [parameter] from [value] to [value]"
```

## Examples

### Example 1: Simple Volume Fade
**Input**: "Fade in track 1 over 4 bars"

**DAW Action**:
```json
{
  "action": "set_track_automation",
  "track": 1,
  "parameter": "volume",
  "curve": {"_placeholder": true, "_description": "Fade in over 4 bars"}
}
```

**Automation Agent Output**:
```json
{
  "parameter": "volume",
  "points": [
    {"time": 0.0, "value": -60.0, "shape": "slow"},
    {"time": 16.0, "value": 0.0, "shape": "slow"}
  ],
  "interpolation": "bezier",
  "shape": "fade_in"
}
```

### Example 2: FX Parameter Sweep
**Input**: "Sweep filter cutoff on track 2 from 1000Hz to 8000Hz during bars 5-8"

**DAW Action**:
```json
{
  "action": "set_fx_automation",
  "track": 2,
  "fx_index": 0,
  "parameter": "cutoff",
  "curve": {"_placeholder": true, "_description": "Sweep from 1000Hz to 8000Hz during bars 5-8"}
}
```

**Automation Agent Output**:
```json
{
  "parameter": "cutoff",
  "points": [
    {"time": 20.0, "value": 1000.0, "shape": "slow"},
    {"time": 36.0, "value": 8000.0, "shape": "fast"}
  ],
  "interpolation": "exponential",
  "shape": "sweep"
}
```

### Example 3: Complex Pattern
**Input**: "Auto-pan the synth at 2Hz with 50% depth"

**Automation Agent Output**:
```json
{
  "parameter": "pan",
  "points": [
    {"time": 0.0, "value": -0.5, "shape": "sine"},
    {"time": 0.5, "value": 0.5, "shape": "sine"},
    {"time": 1.0, "value": -0.5, "shape": "sine"},
    // ... repeated pattern
  ],
  "interpolation": "sine",
  "shape": "auto_pan",
  "frequency": 2.0,
  "depth": 0.5
}
```

## Next Steps

1. ✅ Design automation agent architecture
2. ⏳ Implement automation curve generation
3. ⏳ Add REAPER envelope API support in `magda-reaper`
4. ⏳ Update orchestrator for automation placeholders
5. ⏳ Test with examples: fades, sweeps, modulations

