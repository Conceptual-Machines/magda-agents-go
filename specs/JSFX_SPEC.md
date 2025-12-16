# JSFX Sound Designer Specification

## Overview

Generate JSFX (Reaper's native effect format) using LLM + CFG. Zero external dependencies - code is saved directly to Reaper's Effects folder and appears instantly in the FX browser.

**Advantages over FAUST:**
- Native Reaper integration (no compilation step)
- Hot reload (edit → save → instant update)
- Access to Reaper APIs (tempo, transport, MIDI)
- Simpler deployment (just a text file)

---

## 1. JSFX Structure

### 1.1 Basic Template

```
desc:Effect Name
// Description and metadata

slider1:0.5<0,1,0.01>Parameter Name

@init
// Initialization code (runs once)

@slider
// Runs when slider changes

@block
// Runs once per audio block

@sample
// Runs for each sample
spl0 = spl0 * slider1;  // Left channel
spl1 = spl1 * slider1;  // Right channel
```

### 1.2 Complete Example - Resonant Filter

```
desc:Resonant Low-pass Filter
//tags: filter

slider1:1000<20,20000,1:log>Cutoff (Hz)
slider2:0.5<0,1,0.01>Resonance
slider3:0<-12,12,0.1>Output Gain (dB)

@init
// Filter coefficients
a1 = a2 = b0 = b1 = b2 = 0;
x1l = x2l = y1l = y2l = 0;
x1r = x2r = y1r = y2r = 0;

@slider
// Calculate coefficients when slider changes
freq = slider1;
q = 0.5 + slider2 * 9.5;  // Q from 0.5 to 10
gain = 10^(slider3/20);

omega = 2 * $pi * freq / srate;
sin_omega = sin(omega);
cos_omega = cos(omega);
alpha = sin_omega / (2 * q);

b0 = (1 - cos_omega) / 2;
b1 = 1 - cos_omega;
b2 = (1 - cos_omega) / 2;
a0 = 1 + alpha;
a1 = -2 * cos_omega;
a2 = 1 - alpha;

// Normalize
b0 /= a0; b1 /= a0; b2 /= a0;
a1 /= a0; a2 /= a0;

@sample
// Biquad filter - Left
out_l = b0 * spl0 + b1 * x1l + b2 * x2l - a1 * y1l - a2 * y2l;
x2l = x1l; x1l = spl0;
y2l = y1l; y1l = out_l;
spl0 = out_l * gain;

// Biquad filter - Right
out_r = b0 * spl1 + b1 * x1r + b2 * x2r - a1 * y1r - a2 * y2r;
x2r = x1r; x1r = spl1;
y2r = y1r; y1r = out_r;
spl1 = out_r * gain;
```

---

## 2. CFG Grammar for JSFX

### 2.1 Top-Level Structure

```ebnf
jsfx_program: header slider_defs sections

header: desc_line options_line* tag_line?

desc_line: "desc:" TEXT NEWLINE

options_line: "options:" option ("," option)* NEWLINE
option: IDENT "=" value

tag_line: "//tags:" tag ("," tag)* NEWLINE
tag: IDENT

slider_defs: slider_def*

slider_def: "slider" NUMBER ":" default_value slider_range slider_name NEWLINE

slider_range: "<" NUMBER "," NUMBER "," NUMBER (":" scale)? ">"
scale: "log" | "sqr"  // logarithmic or squared scaling

slider_name: TEXT

sections: init_section? slider_section? block_section? sample_section

init_section: "@init" NEWLINE statement*
slider_section: "@slider" NEWLINE statement*
block_section: "@block" NEWLINE statement*
sample_section: "@sample" NEWLINE statement*
```

### 2.2 Statement Grammar

```ebnf
statement: assignment
         | if_statement
         | while_statement
         | function_call
         | COMMENT

assignment: variable "=" expression ";"?

variable: IDENT
        | IDENT "[" expression "]"  // Array access
        | "spl" NUMBER              // Audio sample
        | "slider" NUMBER           // Slider value

expression: term (binary_op term)*

term: NUMBER
    | variable
    | function_call
    | "(" expression ")"
    | unary_op term

binary_op: "+" | "-" | "*" | "/" | "%" | "^"  // ^ is power
         | "&" | "|" | "~"                     // bitwise
         | "==" | "!=" | "<" | ">" | "<=" | ">="
         | "&&" | "||"

unary_op: "-" | "!"

function_call: IDENT "(" arg_list? ")"
arg_list: expression ("," expression)*

if_statement: "(" condition ")" "?" statement (":" statement)?

while_statement: "while" "(" condition ")" statement
               | "loop" "(" expression "," statement ")"
```

### 2.3 Built-in Functions

```ebnf
builtin_func: math_func | trig_func | audio_func | util_func

math_func: "abs" | "sign" | "min" | "max" | "floor" | "ceil" 
         | "sqrt" | "pow" | "exp" | "log" | "log10"

trig_func: "sin" | "cos" | "tan" | "asin" | "acos" | "atan" | "atan2"

audio_func: "spl" | "spl0" | "spl1" | "srate" | "samplesblock"

util_func: "rand" | "time" | "tempo" | "play_state" | "beat_position"
```

### 2.4 Built-in Variables

```
// Audio
spl0, spl1, spl2, ...   // Sample values (in/out)
srate                    // Sample rate
samplesblock             // Samples per block
num_ch                   // Number of channels

// Transport (Reaper-specific)
tempo                    // Current tempo
play_state               // Playing/stopped/recording
beat_position            // Position in beats

// Sliders
slider1, slider2, ...    // Slider values
```

---

## 3. Common Patterns (Templates)

### 3.1 Simple Gain

```
desc:Simple Gain
slider1:0<-60,12,0.1>Gain (dB)

@slider
gain = 10^(slider1/20);

@sample
spl0 *= gain;
spl1 *= gain;
```

### 3.2 Delay Effect

```
desc:Simple Delay
slider1:250<1,1000,1>Delay (ms)
slider2:0.5<0,1,0.01>Feedback
slider3:0.5<0,1,0.01>Mix

@init
bufsize = srate * 2;  // 2 second max
buf_l = 0;
buf_r = bufsize;
pos = 0;

@slider
delay_samples = (slider1 / 1000) * srate;

@sample
// Read from buffer
del_l = buf_l[pos];
del_r = buf_r[pos];

// Write to buffer with feedback
buf_l[pos] = spl0 + del_l * slider2;
buf_r[pos] = spl1 + del_r * slider2;

// Increment position
pos += 1;
pos >= delay_samples ? pos = 0;

// Mix
spl0 = spl0 * (1 - slider3) + del_l * slider3;
spl1 = spl1 * (1 - slider3) + del_r * slider3;
```

### 3.3 Soft Clipper / Saturation

```
desc:Soft Saturator
slider1:1<1,10,0.1>Drive
slider2:0<-12,12,0.1>Output (dB)

@slider
drive = slider1;
out_gain = 10^(slider2/20);

@sample
// Soft clipping with tanh
spl0 = tanh(spl0 * drive) / tanh(drive) * out_gain;
spl1 = tanh(spl1 * drive) / tanh(drive) * out_gain;
```

### 3.4 Tremolo

```
desc:Tremolo
slider1:4<0.1,20,0.1>Rate (Hz)
slider2:0.5<0,1,0.01>Depth
slider3:0<0,1,1{Sine,Triangle,Square}>Shape

@init
phase = 0;

@sample
// LFO
slider3 == 0 ? (
  lfo = (sin(phase) + 1) / 2;
) : slider3 == 1 ? (
  lfo = abs(phase / $pi - 1);
) : (
  lfo = phase < $pi ? 1 : 0;
);

// Apply modulation
mod = 1 - slider2 + slider2 * lfo;
spl0 *= mod;
spl1 *= mod;

// Increment phase
phase += 2 * $pi * slider1 / srate;
phase >= 2 * $pi ? phase -= 2 * $pi;
```

### 3.5 Parametric EQ Band

```
desc:Parametric EQ
slider1:1000<20,20000,1:log>Frequency
slider2:0<-24,24,0.1>Gain (dB)
slider3:1<0.1,10,0.1>Q

@init
x1l = x2l = y1l = y2l = 0;
x1r = x2r = y1r = y2r = 0;

@slider
A = 10^(slider2/40);
omega = 2 * $pi * slider1 / srate;
sin_omega = sin(omega);
cos_omega = cos(omega);
alpha = sin_omega / (2 * slider3);

b0 = 1 + alpha * A;
b1 = -2 * cos_omega;
b2 = 1 - alpha * A;
a0 = 1 + alpha / A;
a1 = -2 * cos_omega;
a2 = 1 - alpha / A;

// Normalize
b0 /= a0; b1 /= a0; b2 /= a0;
a1 /= a0; a2 /= a0;

@sample
// Biquad
out_l = b0 * spl0 + b1 * x1l + b2 * x2l - a1 * y1l - a2 * y2l;
x2l = x1l; x1l = spl0; y2l = y1l; y1l = out_l;
spl0 = out_l;

out_r = b0 * spl1 + b1 * x1r + b2 * x2r - a1 * y1r - a2 * y2r;
x2r = x1r; x1r = spl1; y2r = y1r; y1r = out_r;
spl1 = out_r;
```

---

## 4. LLM System Prompt

```
You are a JSFX effect designer for Reaper DAW.

JSFX STRUCTURE:
```
desc:Effect Name
slider1:default<min,max,step>Name
slider2:default<min,max,step:log>Name (Log Scale)

@init
// Run once at startup

@slider  
// Run when slider changes

@sample
// Run for each audio sample
spl0 = ...; // Left channel
spl1 = ...; // Right channel
```

KEY VARIABLES:
- spl0, spl1: Audio samples (read/write)
- srate: Sample rate
- slider1, slider2, ...: Parameter values
- $pi: Pi constant

COMMON PATTERNS:
- Gain in dB: gain = 10^(slider_db/20)
- Filter coefficient: omega = 2*$pi*freq/srate
- LFO: sin(phase), phase += 2*$pi*rate/srate
- Soft clip: tanh(x*drive)/tanh(drive)
- Delay buffer: buf[pos], pos = (pos+1) % bufsize

TIPS:
- Initialize variables in @init
- Calculate coefficients in @slider (not @sample)
- Keep @sample minimal for CPU efficiency
- Use ternary: condition ? true_val : false_val

EXAMPLE - Tremolo:
```
desc:Tremolo
slider1:4<0.1,20,0.1>Rate (Hz)
slider2:0.5<0,1,0.01>Depth

@init
phase = 0;

@sample
lfo = (sin(phase) + 1) / 2;
mod = 1 - slider2 + slider2 * lfo;
spl0 *= mod;
spl1 *= mod;
phase += 2 * $pi * slider1 / srate;
phase >= 2 * $pi ? phase -= 2 * $pi;
```
```

---

## 5. Integration with Reaper

### 5.1 File Deployment

Generated JSFX saved to:
```
<Reaper Resource Path>/Effects/MAGDA/<effect_name>.jsfx
```

Instantly appears in FX browser under: `JS: MAGDA/<effect_name>`

### 5.2 API Endpoint

```
POST /api/v1/sound-designer/jsfx/generate

Body:
{
  "prompt": "warm analog filter with resonance",
  "stereo": true
}

Response:
{
  "jsfx_code": "desc:Warm Analog Filter\n...",
  "filename": "warm_analog_filter.jsfx",
  "description": "Resonant low-pass filter with analog warmth"
}
```

### 5.3 Reaper Action (C++)

```cpp
void DeployJSFX(const std::string& code, const std::string& filename) {
    // Get Reaper resource path
    char path[1024];
    GetResourcePath(path, sizeof(path));
    
    // Create MAGDA subfolder
    std::string folder = std::string(path) + "/Effects/MAGDA";
    CreateDirectory(folder.c_str(), nullptr);
    
    // Write JSFX file
    std::string filepath = folder + "/" + filename;
    std::ofstream file(filepath);
    file << code;
    file.close();
    
    // Refresh FX browser
    Main_OnCommand(40886, 0);  // Refresh FX list
}
```

### 5.4 DSL Action

```python
# Generate and add JSFX to track
create_jsfx(
    prompt="warm filter",
    name="my_filter"
).add_to_track(0)

# Or via standard track chain
track(0).add_fx(jsfx="warm filter")  # Inline generation
```

---

## 6. Comparison: JSFX vs FAUST

| Aspect | JSFX | FAUST |
|--------|------|-------|
| **Syntax** | C-like (EEL2) | Functional |
| **Compilation** | Interpreted (JIT) | LLVM compile |
| **Integration** | Native Reaper | External plugin |
| **Hot Reload** | Yes (instant) | Requires recompile |
| **Reaper APIs** | Full access | None |
| **Portability** | Reaper only | Any DAW (VST/AU) |
| **Performance** | Good | Excellent |
| **Grammar Complexity** | Medium | Lower |

**Recommendation**: 
- **JSFX** for Reaper-specific features, quick iteration
- **FAUST** for portable plugins, complex DSP

---

## 7. Grammar Complexity Estimate

| Component | Difficulty | Notes |
|-----------|------------|-------|
| Top structure | Easy | Fixed sections |
| Slider definitions | Easy | Regular pattern |
| Expressions | Medium | Standard arithmetic |
| Control flow | Medium | Ternary-heavy syntax |
| Built-in functions | Easy | Finite set |
| Memory/buffers | Medium | Array syntax |

**Overall: Medium complexity** - Simpler than full C, more complex than FAUST's functional style.

---

## 8. Implementation Files

```
agents/sounddesigner/
├── jsfx_grammar.go           // CFG grammar
├── jsfx_templates.go         // Common patterns
├── jsfx_generator.go         // LLM integration
├── jsfx_validator.go         // Syntax checking
├── JSFX_SPEC.md             // This file
```

---

## 9. Future Extensions

- **Graphics**: JSFX supports @gfx for custom UI
- **MIDI processing**: @block with midisend/midirecv
- **Sidechain**: Multiple input handling
- **Preset system**: Save/load parameter states
- **Effect chains**: Generate multi-effect JSFX
- **Learning mode**: Explain generated code line by line

