# Sound Designer Agent Specification

## Overview

An AI agent that generates DSP code in FAUST, enabling users to create custom audio effects and synthesizers through natural language. Can be deployed as:

1. **API endpoint** - Generate FAUST code for export
2. **Standalone JUCE VST plugin** - Live DSP with built-in FAUST compiler

---

## 1. Architecture

### 1.1 As API Service

```
┌─────────────────────────────────────────────────────────────────────┐
│  USER REQUEST                                                        │
│  "Create a warm analog-style low-pass filter with resonance"        │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  LLM + CFG (FAUST Grammar)                                          │
│                                                                      │
│  Constrained to valid FAUST syntax                                  │
│  Outputs well-formed DSP code                                       │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  FAUST CODE OUTPUT                                                   │
│                                                                      │
│  import("stdfaust.lib");                                            │
│  freq = hslider("Cutoff", 1000, 20, 20000, 1);                     │
│  res = hslider("Resonance", 0.5, 0, 1, 0.01);                      │
│  process = fi.resonlp(freq, res, 1);                                │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
            User exports to FAUST IDE / compiles externally
```

### 1.2 As Standalone JUCE VST Plugin

```
┌─────────────────────────────────────────────────────────────────────┐
│  JUCE VST PLUGIN                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────┐    ┌──────────────────┐    ┌────────────────┐ │
│  │  Chat UI        │───▶│  API Client      │───▶│  FAUST Code    │ │
│  │  (text input)   │    │  (to cloud/local)│    │  Editor View   │ │
│  └─────────────────┘    └──────────────────┘    └────────────────┘ │
│                                                          │          │
│                                                          ▼          │
│  ┌─────────────────┐    ┌──────────────────┐    ┌────────────────┐ │
│  │  Audio I/O      │◀───│  LLVM JIT        │◀───│  FAUST         │ │
│  │  (process)      │    │  Compiled DSP    │    │  Compiler      │ │
│  └─────────────────┘    └──────────────────┘    └────────────────┘ │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Dynamic Parameter UI (generated from FAUST metadata)        │   │
│  │  [Cutoff: ████████░░] [Resonance: ████░░░░░░]               │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. FAUST Language Overview

### 2.1 Why FAUST?

- **Functional**: Clean syntax, no side effects
- **Well-defined grammar**: Perfect for CFG constraints
- **Rich library**: `stdfaust.lib` has filters, oscillators, effects
- **Multi-target**: Compiles to C++, LLVM, WebAssembly, etc.
- **Live compilation**: LLVM JIT enables real-time code updates

### 2.2 Basic FAUST Syntax

```faust
// Simple gain
process = _ * 0.5;

// Stereo
process = _, _;

// With UI control
gain = hslider("Gain", 0.5, 0, 1, 0.01);
process = _ * gain;

// Using library
import("stdfaust.lib");
process = fi.lowpass(2, 1000);  // 2nd order LP at 1kHz
```

### 2.3 Common Patterns

```faust
// Filter with controls
import("stdfaust.lib");
freq = hslider("Frequency[Hz]", 1000, 20, 20000, 1) : si.smoo;
q = hslider("Q", 1, 0.1, 10, 0.1);
process = fi.resonlp(freq, q, 1);

// Oscillator
import("stdfaust.lib");
freq = hslider("Frequency", 440, 20, 2000, 1);
process = os.osc(freq);

// Delay effect
import("stdfaust.lib");
delay_time = hslider("Delay[ms]", 200, 1, 1000, 1) * ma.SR / 1000;
feedback = hslider("Feedback", 0.5, 0, 0.95, 0.01);
process = + ~ (@(int(delay_time)) * feedback);

// Distortion
import("stdfaust.lib");
drive = hslider("Drive", 1, 1, 100, 0.1);
process = _ * drive : ef.cubicnl(0.5, 0) : _ / drive;
```

---

## 3. CFG Grammar for FAUST

### 3.1 Simplified Grammar (Subset)

```ebnf
program: import_stmt* definition* "process" "=" expression ";"

import_stmt: "import" "(" STRING ")" ";"

definition: IDENT "=" expression ";"

expression: term (("+" | "-" | "*" | "/" | ":" | "~" | ",") term)*

term: NUMBER
    | STRING  
    | IDENT
    | IDENT "(" arg_list? ")"
    | "(" expression ")"
    | "_"                          // Identity/wire
    | "!"                          // Cut
    | ui_element

ui_element: slider | button | checkbox | nentry

slider: ("hslider" | "vslider") "(" STRING "," NUMBER "," NUMBER "," NUMBER "," NUMBER ")"

button: "button" "(" STRING ")"

checkbox: "checkbox" "(" STRING ")"

nentry: "nentry" "(" STRING "," NUMBER "," NUMBER "," NUMBER "," NUMBER ")"

arg_list: expression ("," expression)*

// Library functions (subset)
lib_func: "fi." filter_func
        | "os." osc_func
        | "ef." effect_func
        | "de." delay_func
        | "an." analysis_func
        | "si." signal_func
        | "ma." math_func

filter_func: "lowpass" | "highpass" | "bandpass" | "resonlp" | "resonhp" | "peak_eq"
osc_func: "osc" | "sawtooth" | "square" | "triangle" | "lf_sawpos"
effect_func: "cubicnl" | "compressor_mono" | "gate_mono"
delay_func: "delay" | "fdelay" | "sdelay"
```

### 3.2 Full Grammar File

Would be more extensive, covering:
- All standard library functions
- Pattern matching
- Recursion
- Parallel/sequential composition
- Route expressions

---

## 4. LLM System Prompt

```
You are a sound designer AI that creates audio effects and synthesizers using FAUST DSP.

FAUST BASICS:
- `process = ...;` is the main audio processing function
- `_` is the input signal (wire)
- `:` is sequential composition (connect output to input)
- `,` is parallel composition (side by side)
- `~` is feedback loop
- `@(n)` is delay by n samples

COMMON LIBRARY FUNCTIONS:
Filters (fi.):
  fi.lowpass(order, freq)     - Low-pass filter
  fi.highpass(order, freq)    - High-pass filter  
  fi.resonlp(freq, q, gain)   - Resonant low-pass
  fi.peak_eq(freq, gain, q)   - Parametric EQ band

Oscillators (os.):
  os.osc(freq)       - Sine wave
  os.sawtooth(freq)  - Sawtooth wave
  os.square(freq)    - Square wave

Effects (ef.):
  ef.cubicnl(drive, offset)  - Soft clipping distortion
  ef.echo(time, feedback)    - Echo effect

Delays (de.):
  de.delay(max, time)        - Variable delay
  de.fdelay(max, time)       - Fractional delay

UI CONTROLS:
  hslider("Name", default, min, max, step)  - Horizontal slider
  vslider("Name", default, min, max, step)  - Vertical slider  
  button("Name")                             - Momentary button
  checkbox("Name")                           - Toggle

SMOOTHING:
  : si.smoo          - Smooth parameter changes (prevent clicks)

EXAMPLE - Resonant Filter:
```faust
import("stdfaust.lib");
freq = hslider("Cutoff[Hz]", 1000, 20, 20000, 1) : si.smoo;
res = hslider("Resonance", 0.7, 0, 1, 0.01);
process = fi.resonlp(freq, res * 5, 1);
```

EXAMPLE - Stereo Chorus:
```faust
import("stdfaust.lib");
rate = hslider("Rate[Hz]", 0.5, 0.1, 5, 0.01);
depth = hslider("Depth", 0.5, 0, 1, 0.01);
chorus = _ <: _, de.fdelay(1024, 512 + 256 * depth * os.osc(rate)) :> _ / 2;
process = chorus, chorus;
```

Always:
1. Start with `import("stdfaust.lib");`
2. Define parameters with sliders before `process`
3. Use `si.smoo` on parameters to prevent clicks
4. Keep it simple - avoid overly complex expressions
```

---

## 5. Request → FAUST Examples

| User Request | Generated FAUST |
|--------------|-----------------|
| "warm low-pass filter" | `fi.resonlp(freq, 0.5, 1)` with freq slider |
| "simple delay effect" | `+ ~ (@(delay_time) * feedback)` |
| "soft distortion" | `ef.cubicnl(drive, 0)` with drive slider |
| "tremolo effect" | `_ * (0.5 + 0.5 * os.osc(rate))` |
| "ping pong delay" | Stereo cross-feedback delay |
| "4-band EQ" | Four `fi.peak_eq` bands in series |
| "basic synth" | `os.sawtooth` + filter + envelope |

---

## 6. JUCE Plugin Architecture

### 6.1 Components

```
MagdaSoundDesigner/
├── Source/
│   ├── PluginProcessor.cpp      // Audio processing
│   ├── PluginProcessor.h
│   ├── PluginEditor.cpp         // UI
│   ├── PluginEditor.h
│   ├── FaustCompiler.cpp        // FAUST → LLVM JIT
│   ├── FaustCompiler.h
│   ├── ChatComponent.cpp        // Text input UI
│   ├── ChatComponent.h
│   ├── CodeEditor.cpp           // FAUST code display
│   ├── CodeEditor.h
│   ├── ApiClient.cpp            // HTTP client to AI backend
│   └── ApiClient.h
├── JuceLibraryCode/
├── Builds/
└── MagdaSoundDesigner.jucer
```

### 6.2 Key Classes

```cpp
// FaustCompiler.h
class FaustCompiler {
public:
    FaustCompiler();
    ~FaustCompiler();
    
    // Compile FAUST code to executable DSP
    bool compile(const juce::String& faustCode);
    
    // Get compiled DSP
    dsp* getDSP() { return m_dsp; }
    
    // Get UI parameters
    std::vector<FaustParam> getParameters();
    
    // Error handling
    juce::String getLastError() { return m_lastError; }
    
private:
    llvm_dsp_factory* m_factory = nullptr;
    dsp* m_dsp = nullptr;
    juce::String m_lastError;
};

// PluginProcessor - audio callback
void MagdaSoundDesignerProcessor::processBlock(AudioBuffer<float>& buffer, MidiBuffer&)
{
    if (m_dsp) {
        // FAUST processing
        float* inputs[] = { buffer.getWritePointer(0), buffer.getWritePointer(1) };
        float* outputs[] = { buffer.getWritePointer(0), buffer.getWritePointer(1) };
        m_dsp->compute(buffer.getNumSamples(), inputs, outputs);
    }
}

// Hot-reload on new code
void MagdaSoundDesignerProcessor::setFaustCode(const juce::String& code)
{
    FaustCompiler newCompiler;
    if (newCompiler.compile(code)) {
        // Swap DSP (thread-safe)
        std::lock_guard<std::mutex> lock(m_dspMutex);
        m_compiler = std::move(newCompiler);
        m_dsp = m_compiler.getDSP();
        updateParameters();
    }
}
```

### 6.3 UI Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  MAGDA Sound Designer                                          [×]  │
├─────────────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │  Chat                                                           │ │
│ │  ┌─────────────────────────────────────────────────────────────┐│ │
│ │  │ You: make a warm analog filter                              ││ │
│ │  │ AI: Created resonant low-pass filter with warmth control    ││ │
│ │  └─────────────────────────────────────────────────────────────┘│ │
│ │  [Type request...                                        ] [Send]│ │
│ └─────────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │  FAUST Code                                         [Edit] [Copy]│ │
│ │  ┌─────────────────────────────────────────────────────────────┐│ │
│ │  │ import("stdfaust.lib");                                     ││ │
│ │  │ freq = hslider("Cutoff", 800, 20, 8000, 1) : si.smoo;      ││ │
│ │  │ warmth = hslider("Warmth", 0.3, 0, 1, 0.01);               ││ │
│ │  │ process = fi.resonlp(freq, warmth * 3, 1);                 ││ │
│ │  └─────────────────────────────────────────────────────────────┘│ │
│ └─────────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │  Parameters (auto-generated from FAUST UI)                      │ │
│ │                                                                  │ │
│ │  Cutoff     [▓▓▓▓▓▓▓░░░░░░░░░░░░░] 800 Hz                      │ │
│ │  Warmth     [▓▓▓░░░░░░░░░░░░░░░░░] 0.30                        │ │
│ │                                                                  │ │
│ └─────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 7. API Endpoints

### 7.1 Generate FAUST Code

```
POST /api/v1/sound-designer/generate

Body:
{
  "prompt": "warm analog low-pass filter with resonance",
  "type": "effect",  // "effect" | "synth" | "analyzer"
  "stereo": true
}

Response:
{
  "faust_code": "import(\"stdfaust.lib\");\n...",
  "description": "Resonant low-pass filter with warmth control",
  "parameters": [
    {"name": "Cutoff", "type": "hslider", "default": 800, "min": 20, "max": 8000},
    {"name": "Warmth", "type": "hslider", "default": 0.3, "min": 0, "max": 1}
  ]
}
```

### 7.2 Validate FAUST Code

```
POST /api/v1/sound-designer/validate

Body:
{
  "faust_code": "import(\"stdfaust.lib\");\nprocess = fi.lowpass(2, 1000);"
}

Response:
{
  "valid": true,
  "errors": [],
  "warnings": ["Consider adding si.smoo to frequency parameter"]
}
```

### 7.3 Get Templates

```
GET /api/v1/sound-designer/templates

Response:
{
  "templates": [
    {"id": "lowpass", "name": "Low-pass Filter", "code": "..."},
    {"id": "delay", "name": "Delay Effect", "code": "..."},
    {"id": "distortion", "name": "Soft Distortion", "code": "..."}
  ]
}
```

---

## 8. Dependencies

### 8.1 For JUCE Plugin

- **JUCE Framework** (GPL/Commercial)
- **libfaust** (embedded FAUST compiler)
- **LLVM** (for JIT compilation)

### 8.2 For API Service

- **faust2** CLI (for validation/compilation)
- Standard Go dependencies

---

## 9. Project Structure

### 9.1 Go Agent (API)

```
agents/sounddesigner/
├── sounddesigner_agent.go
├── faust_grammar.go          // CFG grammar for FAUST
├── faust_templates.go        // Common templates
├── faust_validator.go        // Syntax validation
├── SOUND_DESIGNER_AGENT_SPEC.md
```

### 9.2 JUCE Plugin (Separate Repo)

```
magda-sound-designer/
├── Source/
├── Resources/
├── Builds/
├── JuceLibraryCode/
├── CMakeLists.txt
├── MagdaSoundDesigner.jucer
└── README.md
```

---

## 10. Future Extensions

- **Preset browser**: Save/load generated effects
- **Chain builder**: Combine multiple effects
- **Visual feedback**: Waveform/spectrum display
- **MIDI control**: Map MIDI CC to parameters
- **Export formats**: VST3, AU, LV2, WebAudio
- **Collaborative**: Share creations, community library
- **Learn mode**: Explain what the generated code does
- **Optimization**: Suggest performance improvements

