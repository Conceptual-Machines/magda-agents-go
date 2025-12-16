# Automation Agent Specification

## Overview

Extension to the DAW agent for drawing automation curves on track parameters, plugin parameters, and FX. Supports common shapes (linear, exponential, sine) and custom point-based curves.

**Note**: This can be integrated directly into the existing DAW agent rather than as a separate agent.

---

## 1. DSL Syntax

### 1.1 Basic Pattern

```
track(<selector>).get_plugin(<plugin>).get_param(<param>).automation(<curve_spec>)
```

### 1.2 Examples

```python
# Fade in track volume over 4 bars
track(index=0).automation(
    param="volume",
    start=0,
    end=4,
    curve="linear",
    from=-inf,
    to=0
)

# Fade out at end of project
track(name="Lead Synth").automation(
    param="volume",
    start=32,
    end=36,
    curve="exponential",
    from=0,
    to=-inf
)

# Automate plugin parameter
track(index=0).get_plugin("Serum").get_param("Filter Cutoff").automation(
    start=0,
    end=8,
    curve="sine",
    min=0.2,
    max=0.8,
    cycles=4
)

# Pan automation - LFO style
track(index=1).automation(
    param="pan",
    start=0,
    end=16,
    curve="triangle",
    min=-1,
    max=1,
    cycles=8
)

# Custom envelope points
track(index=0).get_plugin("ReaComp").get_param("Threshold").automation(
    start=0,
    end=8,
    curve="points",
    points=[
        {pos: 0, value: 0.5},
        {pos: 2, value: 0.8},
        {pos: 4, value: 0.3},
        {pos: 8, value: 0.5}
    ]
)
```

---

## 2. Curve Types

### 2.1 Simple Curves

| Curve | Description | Parameters |
|-------|-------------|------------|
| `linear` | Straight line | `from`, `to` |
| `exponential` | Exp curve (fast start, slow end) | `from`, `to` |
| `logarithmic` | Log curve (slow start, fast end) | `from`, `to` |
| `scurve` | S-curve (smooth ease in/out) | `from`, `to` |

### 2.2 Periodic Curves (LFO-style)

| Curve | Description | Parameters |
|-------|-------------|------------|
| `sine` | Sine wave | `min`, `max`, `cycles`, `phase` |
| `triangle` | Triangle wave | `min`, `max`, `cycles`, `phase` |
| `square` | Square wave | `min`, `max`, `cycles`, `duty` |
| `sawtooth` | Sawtooth wave | `min`, `max`, `cycles`, `phase` |
| `random` | Random (S&H style) | `min`, `max`, `steps` |

### 2.3 Custom Points

| Curve | Description | Parameters |
|-------|-------------|------------|
| `points` | Point-based envelope | `points: [{pos, value, curve?}]` |

Point curve types between points:
- `linear` (default)
- `smooth` (bezier/spline)
- `step` (instant change)

---

## 3. Parameter Targets

### 3.1 Track Parameters (built-in)

| Parameter | Range | Description |
|-----------|-------|-------------|
| `volume` | -inf to +12 dB | Track volume |
| `pan` | -1 to +1 | Track pan (L/R) |
| `width` | 0 to 1 | Stereo width |
| `mute` | 0 or 1 | Mute state |

### 3.2 Plugin Parameters

Access via:
```
track().get_plugin(<name>).get_param(<param_name>)
```

Parameter names are plugin-specific. Values typically 0.0 to 1.0 (normalized).

### 3.3 Send/Receive Levels

```
track().get_send(<index>).automation(...)
track().get_receive(<index>).automation(...)
```

---

## 4. Time/Position Units

### 4.1 Bar-based (default)

```
start=0, end=4   // Bars 0-4
start=8.5, end=9 // Bar 8 beat 3 to bar 9
```

### 4.2 Time-based (seconds)

```
start_time=0, end_time=10.5  // Seconds
```

### 4.3 Relative to Selection

```
position="selection"  // Use current time selection
```

---

## 5. Grammar Extension

Add to existing Magda DSL grammar:

```
// New chain methods
automation_chain: ".automation" "(" automation_params ")"

automation_params: param ("," param)*

param: param_name "=" param_value

// Automation-specific params
param_name: "param" | "start" | "end" | "start_time" | "end_time"
          | "curve" | "from" | "to" | "min" | "max"
          | "cycles" | "phase" | "duty" | "steps" | "points"

// Plugin/param access chain
plugin_chain: ".get_plugin" "(" STRING ")"
param_chain: ".get_param" "(" STRING ")"

// Full automation expression
automation_expr: track_ref plugin_chain? param_chain? automation_chain
```

---

## 6. Output Actions

### 6.1 Action: `create_automation_envelope`

```json
{
  "action": "create_automation_envelope",
  "track": 0,
  "param_type": "track",     // "track" | "plugin" | "send"
  "param_name": "volume",    // or plugin param name
  "plugin_name": null,       // for plugin params
  "plugin_index": null,      // alternative to name
  "points": [
    {"position": 0.0, "value": 0.0, "shape": 0},
    {"position": 4.0, "value": 1.0, "shape": 0}
  ]
}
```

### 6.2 Action: `set_automation_points`

For adding points to existing envelope:

```json
{
  "action": "set_automation_points",
  "track": 0,
  "envelope_name": "Volume",
  "points": [...]
}
```

### 6.3 Action: `clear_automation`

```json
{
  "action": "clear_automation",
  "track": 0,
  "param_name": "volume",
  "start": 0,
  "end": 8
}
```

---

## 7. Curve Generation (Parser)

### 7.1 Linear

```go
func GenerateLinear(start, end, from, to float64, resolution int) []Point {
    points := make([]Point, resolution)
    for i := 0; i < resolution; i++ {
        t := float64(i) / float64(resolution-1)
        points[i] = Point{
            Position: start + t*(end-start),
            Value:    from + t*(to-from),
        }
    }
    return points
}
```

### 7.2 Sine Wave

```go
func GenerateSine(start, end, min, max float64, cycles int, phase float64) []Point {
    // Points per cycle for smooth curve
    pointsPerCycle := 32
    totalPoints := cycles * pointsPerCycle
    
    points := make([]Point, totalPoints)
    for i := 0; i < totalPoints; i++ {
        t := float64(i) / float64(totalPoints)
        angle := 2*math.Pi*float64(cycles)*t + phase
        value := (math.Sin(angle) + 1) / 2  // 0-1
        value = min + value*(max-min)        // Scale to range
        
        points[i] = Point{
            Position: start + t*(end-start),
            Value:    value,
        }
    }
    return points
}
```

### 7.3 Exponential

```go
func GenerateExponential(start, end, from, to float64, resolution int) []Point {
    points := make([]Point, resolution)
    for i := 0; i < resolution; i++ {
        t := float64(i) / float64(resolution-1)
        // Exponential curve: t^2 for ease-out, sqrt(t) for ease-in
        expT := t * t  // Ease out (fast start, slow end)
        points[i] = Point{
            Position: start + t*(end-start),
            Value:    from + expT*(to-from),
        }
    }
    return points
}
```

---

## 8. Reaper API Mapping

### 8.1 Key Functions

```cpp
// Get/create envelope
TrackEnvelope* GetTrackEnvelopeByName(MediaTrack* track, const char* envname);
TrackEnvelope* GetFXEnvelope(MediaTrack* track, int fxindex, int paramindex, bool create);

// Envelope manipulation
bool InsertEnvelopePoint(TrackEnvelope* env, double time, double value, 
                         int shape, double tension, bool selected, bool* noSort);
bool DeleteEnvelopePointRange(TrackEnvelope* env, double start, double end);
bool Envelope_SortPoints(TrackEnvelope* env);

// Shape values for InsertEnvelopePoint:
// 0 = linear
// 1 = square
// 2 = slow start/end
// 3 = fast start
// 4 = fast end
// 5 = bezier
```

### 8.2 Action Implementation (C++)

```cpp
void ExecuteCreateAutomationEnvelope(const json& action) {
    int trackIdx = action["track"];
    MediaTrack* track = GetTrack(0, trackIdx);
    
    TrackEnvelope* env = nullptr;
    
    if (action["param_type"] == "track") {
        env = GetTrackEnvelopeByName(track, action["param_name"].c_str());
    } else if (action["param_type"] == "plugin") {
        int fxIdx = TrackFX_GetByName(track, action["plugin_name"].c_str(), false);
        int paramIdx = TrackFX_GetParamFromIdent(track, fxIdx, action["param_name"].c_str());
        env = GetFXEnvelope(track, fxIdx, paramIdx, true);
    }
    
    if (!env) return;
    
    // Clear existing points in range
    double start = action["points"][0]["position"];
    double end = action["points"].back()["position"];
    DeleteEnvelopePointRange(env, start, end);
    
    // Insert new points
    for (const auto& pt : action["points"]) {
        InsertEnvelopePoint(env, 
            pt["position"].get<double>(),
            pt["value"].get<double>(),
            pt.value("shape", 0),
            0.0,  // tension
            false, // selected
            nullptr
        );
    }
    
    Envelope_SortPoints(env);
}
```

---

## 9. LLM Prompt Examples

### System Prompt Addition

```
AUTOMATION:
You can create automation curves using the .automation() method.

Examples:
- Fade in: track(0).automation(param="volume", start=0, end=4, curve="linear", from=-inf, to=0)
- Filter sweep: track(0).get_plugin("Serum").get_param("Filter Cutoff").automation(start=0, end=8, curve="sine", min=0.2, max=0.8, cycles=2)
- Pan LFO: track(1).automation(param="pan", start=0, end=16, curve="triangle", min=-0.5, max=0.5, cycles=4)

Curve types: linear, exponential, logarithmic, scurve, sine, triangle, square, sawtooth, points
```

### User Requests → DSL

| Request | DSL Output |
|---------|------------|
| "fade in the drums" | `track(name="Drums").automation(param="volume", start=0, end=4, curve="linear", from=-inf, to=0)` |
| "add a filter sweep to track 1" | `track(1).get_plugin("*EQ*").get_param("*freq*").automation(start=0, end=8, curve="exponential", from=0.1, to=0.9)` |
| "make the synth pan side to side" | `track(name="Synth").automation(param="pan", start=0, end=16, curve="sine", min=-0.8, max=0.8, cycles=4)` |

---

## 10. Integration with DAW Agent

### 10.1 Grammar Extension

Add automation methods to existing functional DSL in `agents/daw/`:

```go
// In functional_dsl_parser.go
case "automation":
    return p.parseAutomationCall(args, currentContext)
```

### 10.2 New Files (if separate)

If kept as extension to DAW agent:
```
agents/daw/
  automation_parser.go      // Parse .automation() calls
  automation_curves.go      // Generate curve points
  automation_curves_test.go
```

Or if separate agent:
```
agents/automation/
  automation_agent.go
  curves.go
  parser.go
  AUTOMATION_AGENT_SPEC.md
```

---

## 11. Future Extensions

- **Tempo sync**: Cycles synced to tempo (1 cycle = 1 bar)
- **Modulation sources**: Link automation to audio input, sidechain
- **Envelope templates**: Save/load common shapes
- **Multi-param**: Automate multiple params with one call
- **Relative automation**: Add to existing envelope values
- **Automation recording**: Record from controller input

