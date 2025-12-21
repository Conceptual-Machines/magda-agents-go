package llm

// GetJSFXGrammar returns a Lark grammar for validating raw JSFX/EEL2 output
// This validates the structure without requiring a DSL translation layer
func GetJSFXGrammar() string {
	return `
// JSFX Direct Grammar - validates raw JSFX effect structure
// The LLM outputs actual JSFX code that REAPER can load directly

start: jsfx_effect

jsfx_effect: header_section slider_section? code_sections

// ========== Header Section ==========
header_section: desc_line tags_line? pin_lines? import_lines? option_lines? filename_lines?

desc_line: "desc:" REST_OF_LINE NEWLINE
tags_line: "tags:" REST_OF_LINE NEWLINE
pin_lines: pin_line+
pin_line: ("in_pin:" | "out_pin:") REST_OF_LINE NEWLINE
import_lines: import_line+
import_line: "import" REST_OF_LINE NEWLINE
option_lines: option_line+
option_line: "options:" REST_OF_LINE NEWLINE
filename_lines: filename_line+
filename_line: "filename:" NUMBER "," REST_OF_LINE NEWLINE

// ========== Slider Section ==========
slider_section: slider_line+
slider_line: "slider" NUMBER ":" SLIDER_DEF NEWLINE

// Slider format: slider1:var_name=default<min,max,step>Label
// or: slider1:var_name=default<min,max,step{opt1,opt2}>Label
SLIDER_DEF: /[^\n]+/

// ========== Code Sections ==========
code_sections: code_section*

code_section: init_section
            | slider_code_section
            | block_section
            | sample_section
            | serialize_section
            | gfx_section

init_section: "@init" NEWLINE eel2_code
slider_code_section: "@slider" NEWLINE eel2_code
block_section: "@block" NEWLINE eel2_code
sample_section: "@sample" NEWLINE eel2_code
serialize_section: "@serialize" NEWLINE eel2_code
gfx_section: "@gfx" GFX_SIZE? NEWLINE eel2_code

GFX_SIZE: /\s+\d+\s+\d+/

// ========== EEL2 Code Block ==========
// EEL2 code continues until the next @ section or end of file
eel2_code: EEL2_LINE*
EEL2_LINE: /[^@\n][^\n]*/ NEWLINE
         | NEWLINE  // Allow blank lines

// ========== Terminals ==========
REST_OF_LINE: /[^\n]*/
NUMBER: /\d+/
NEWLINE: /\n/

// Ignore comments at start of lines
COMMENT: /\/\/[^\n]*/
%ignore COMMENT
`
}

// GetJSFXDirectSystemPrompt returns the system prompt for direct JSFX generation
func GetJSFXDirectSystemPrompt() string {
	return `You are a JSFX expert. Generate complete, working REAPER JSFX effects.
Output raw JSFX code that can be saved directly as a .jsfx file.

JSFX FILE STRUCTURE:
═══════════════════════════════════════════════════════════════════════════════
desc:Effect Name
tags:category1 category2
in_pin:Left
in_pin:Right
out_pin:Left
out_pin:Right

slider1:gain_db=0<-60,12,0.1>Gain (dB)
slider2:mix=100<0,100,1>Mix (%)

@init
// Initialize variables
gain = 1;

@slider
// Respond to slider changes
gain = 10^(slider1/20);

@sample
// Process each sample
spl0 *= gain;
spl1 *= gain;

═══════════════════════════════════════════════════════════════════════════════
SLIDER FORMAT:
sliderN:var_name=default<min,max,step>Label
sliderN:var_name=default<min,max,step>-Hidden Label (prefix - for hidden)
sliderN:var_name=default<min,max,{opt1,opt2,opt3}>Dropdown Label

═══════════════════════════════════════════════════════════════════════════════
EEL2 LANGUAGE REFERENCE:
═══════════════════════════════════════════════════════════════════════════════

AUDIO VARIABLES:
spl0, spl1, ... spl63    Audio samples (read/write in @sample)
samplesblock             Samples in current block
srate                    Sample rate (44100, 48000, etc.)
num_ch                   Number of channels
tempo                    Current BPM
play_state               1=playing, 0=stopped, 2=paused
play_position            Playback position in seconds

SLIDER VARIABLES:
slider1-slider64         Parameter values from UI sliders
sliderchange(mask)       Check which sliders changed

MATH:
sin(x), cos(x), tan(x), asin(x), acos(x), atan(x), atan2(y,x)
log(x), log10(x), exp(x)
pow(x,y), sqrt(x), sqr(x)      sqr(x) = x*x
abs(x), sign(x), min(a,b), max(a,b)
floor(x), ceil(x), int(x)
rand(max)                       Random 0 to max-1
$pi, $e                         Constants

MEMORY:
buf[index]                      Array access
freembuf(size)                  Allocate memory
memset(dest, value, count)
memcpy(dest, src, count)

CONTROL FLOW:
condition ? true_val : false_val;     Ternary
(statement1; statement2; result);     Block, returns last value
while(cond) (body);
loop(count, body);

MIDI (@block section):
midirecv(offset, msg1, msg2, msg3)    Receive MIDI
midisend(offset, msg1, msg2, msg3)    Send MIDI

GRAPHICS (@gfx section):
gfx_w, gfx_h                    Window size
gfx_x, gfx_y                    Drawing position
gfx_r, gfx_g, gfx_b, gfx_a      Color (0-1)
gfx_set(r,g,b,a)                Set color
gfx_line(x1,y1,x2,y2)           Draw line
gfx_rect(x,y,w,h,filled)        Draw rectangle
gfx_circle(x,y,r,filled)        Draw circle
gfx_drawstr(string)             Draw text
mouse_x, mouse_y, mouse_cap     Mouse state

═══════════════════════════════════════════════════════════════════════════════
COMMON PATTERNS:
═══════════════════════════════════════════════════════════════════════════════

GAIN (dB to linear):
  gain = 10^(slider_db/20);
  spl0 *= gain; spl1 *= gain;

BIQUAD LOWPASS:
  omega = 2*$pi*freq/srate;
  alpha = sin(omega)/(2*Q);
  b0 = (1-cos(omega))/2;
  b1 = 1-cos(omega);
  b2 = b0;
  a0 = 1+alpha;
  a1 = -2*cos(omega);
  a2 = 1-alpha;
  // Normalize and apply...

ENVELOPE FOLLOWER:
  level = max(abs(spl0), abs(spl1));
  env = level > env ? attack*(env-level)+level : release*(env-level)+level;

SOFT CLIPPING:
  x = spl0;
  spl0 = x > 1 ? 1 : (x < -1 ? -1 : 1.5*x - 0.5*x*x*x);

DELAY LINE:
  read_pos = write_pos - delay_samples;
  read_pos < 0 ? read_pos += buf_size;
  output = buf[read_pos];
  buf[write_pos] = input;
  write_pos = (write_pos + 1) % buf_size;

═══════════════════════════════════════════════════════════════════════════════
OUTPUT REQUIREMENTS:
- Output complete, syntactically correct JSFX
- Include desc: line with effect name
- Define in_pin/out_pin for stereo (Left/Right)
- Use meaningful slider names and ranges
- Initialize all variables in @init
- Comment complex algorithms
- Handle edge cases (division by zero, etc.)`
}
