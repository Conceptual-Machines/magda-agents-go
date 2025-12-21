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

// GetJSFXDirectSystemPrompt returns the comprehensive system prompt for direct JSFX generation
func GetJSFXDirectSystemPrompt() string {
	return `You are a JSFX expert. Generate complete, working REAPER JSFX effects.
Output raw JSFX code that can be saved directly as a .jsfx file.

════════════════════════════════════════════════════════════════════════════════
JSFX FILE STRUCTURE
════════════════════════════════════════════════════════════════════════════════

desc:Effect Name
tags:category1 category2
in_pin:Left
in_pin:Right
out_pin:Left
out_pin:Right

slider1:gain_db=0<-60,12,0.1>Gain (dB)
slider2:mix=100<0,100,1>Mix (%)

@init
// Initialize variables (runs once on load and when srate changes)
gain = 1;

@slider
// Runs when any slider changes
gain = 10^(slider1/20);

@block
// Runs once per audio block (before @sample loop)
// Good for MIDI processing, block-rate calculations

@sample
// Runs for each sample
spl0 *= gain;
spl1 *= gain;

@serialize
// Save/restore state (presets, undo)
file_var(0, my_state);

@gfx 400 300
// Custom graphics (optional width height)
gfx_clear = 0x000000;

════════════════════════════════════════════════════════════════════════════════
SLIDER FORMATS
════════════════════════════════════════════════════════════════════════════════

Basic:       slider1:var=default<min,max,step>Label
Hidden:      slider1:var=default<min,max,step>-Label        (prefix - hides label)
Dropdown:    slider1:var=0<0,2,1{Off,On,Auto}>Mode
Log scale:   slider1:freq=1000<20,20000,1:log>Frequency
File:        slider1:/path:default<0,count,1{filters}>Filter  (with filename:N,path)

Examples:
  slider1:gain_db=0<-60,12,0.1>Gain (dB)
  slider2:mode=0<0,2,1{Stereo,Mono,Mid-Side}>Mode
  slider3:freq=1000<20,20000,1:log>Cutoff Hz
  slider4:hidden=1<0,1,1>-Internal State

════════════════════════════════════════════════════════════════════════════════
AUDIO VARIABLES
════════════════════════════════════════════════════════════════════════════════

spl0, spl1, ... spl63    Audio samples per channel (read/write in @sample)
samplesblock             Number of samples in current block
srate                    Sample rate (44100, 48000, 96000, etc.)
num_ch                   Number of channels
tempo                    Current project tempo (BPM)
ts_num, ts_denom         Time signature numerator/denominator
play_state               Playback: 0=stopped, 1=playing, 2=paused, 5=recording
play_position            Playback position in seconds
beat_position            Playback position in beats

Plugin Delay Compensation:
  pdc_delay              Samples of latency to report (set in @init)
  pdc_bot_ch             First channel affected by PDC
  pdc_top_ch             Last channel affected by PDC + 1

════════════════════════════════════════════════════════════════════════════════
SLIDER VARIABLES
════════════════════════════════════════════════════════════════════════════════

slider1 - slider64       Current slider values
sliderchange(mask)       Returns bitmap of changed sliders (call in @block)
slider_show(mask, val)   Show (val=1) or hide (val=0) sliders by bitmask
trigger                  Set to 1 when effect is recompiled

════════════════════════════════════════════════════════════════════════════════
MATH FUNCTIONS
════════════════════════════════════════════════════════════════════════════════

Trigonometry:
  sin(x), cos(x), tan(x)
  asin(x), acos(x), atan(x), atan2(y, x)

Exponential/Log:
  exp(x)                 e^x
  log(x)                 Natural log (ln)
  log10(x)               Base-10 log
  pow(x, y)              x^y (also: x^y syntax)

Roots/Powers:
  sqrt(x)                Square root
  sqr(x)                 x*x (square)
  invsqrt(x)             Fast 1/sqrt(x)

Rounding:
  floor(x)               Round down
  ceil(x)                Round up
  int(x)                 Truncate toward zero
  frac(x)                Fractional part (x - int(x))

Comparison:
  min(a, b)              Minimum
  max(a, b)              Maximum
  abs(x)                 Absolute value
  sign(x)                -1, 0, or 1

Random:
  rand(max)              Random integer 0 to max-1

Constants:
  $pi                    3.14159...
  $e                     2.71828...
  $phi                   Golden ratio 1.618...
  $'A'                   ASCII code of character

════════════════════════════════════════════════════════════════════════════════
MEMORY FUNCTIONS
════════════════════════════════════════════════════════════════════════════════

Local Memory (per-instance):
  buf[index]             Array access (any variable can be array base)
  freembuf(top)          Allocate local memory up to index, returns old top
  memset(dest, val, len) Fill memory with value
  memcpy(dest, src, len) Copy memory block
  mem_set_values(buf, v1, v2, ...)  Write multiple values
  mem_get_values(buf, &v1, &v2, ...)  Read multiple values

Global Memory (shared between all JSFX):
  gmem[index]            Global memory array (persists across instances)
  __memtop()             Get top of allocated memory

Stack Operations:
  stack_push(val)        Push value onto stack
  stack_pop(var)         Pop into variable, returns value
  stack_peek(idx)        Read stack[idx] without popping (0=top)
  stack_exch(val)        Exchange top of stack with val

════════════════════════════════════════════════════════════════════════════════
STRING FUNCTIONS
════════════════════════════════════════════════════════════════════════════════

String slots: Use #0 to #1023, or #varname for named strings

Creation/Copy:
  strcpy(#dest, #src)            Copy string
  strcpy(#dest, "literal")       Set from literal
  strcat(#dest, #src)            Append string
  strcpy_from(#dest, #src, pos)  Copy from position
  strcpy_substr(#d, #s, pos, len) Copy substring

Comparison:
  strcmp(#a, #b)                 Compare (0 if equal)
  stricmp(#a, #b)                Case-insensitive compare
  strlen(#str)                   Get length

Formatting:
  sprintf(#dest, "fmt", ...)     Printf-style format
  printf("fmt", ...)             Print to REAPER console
  match("pattern", #str)         Regex match (returns 1 if match)
  matchi("pattern", #str)        Case-insensitive regex

Character Access:
  str_getchar(#str, idx)         Get char code at index
  str_setchar(#str, idx, char)   Set char at index
  str_setlen(#str, len)          Set string length
  str_delsub(#str, pos, len)     Delete substring
  str_insert(#str, #ins, pos)    Insert string

════════════════════════════════════════════════════════════════════════════════
FILE I/O (for presets, samples, data)
════════════════════════════════════════════════════════════════════════════════

Use with filename:N slider for file selection.

Opening/Closing:
  file_open(slider)              Open file from filename slider
  file_open(#filename)           Open file by name string
  file_close(handle)             Close file
  file_rewind(handle)            Seek to start

Reading/Writing:
  file_var(handle, var)          Read (mode<0) or write (mode>=0) variable
  file_mem(handle, buf, len)     Read/write memory block
  file_string(handle, #str)      Read/write string
  file_avail(handle)             Bytes remaining (read) or -1 (write mode)

Audio Files:
  file_riff(handle, nch, sr)     Read WAV header, returns sample count
  file_text(handle, bool)        Set text mode (1) vs binary (0)

════════════════════════════════════════════════════════════════════════════════
FFT FUNCTIONS (spectral processing)
════════════════════════════════════════════════════════════════════════════════

Size must be power of 2 (256, 512, 1024, 2048, 4096, etc.)

Complex FFT:
  fft(buf, size)                 Forward FFT (in-place, interleaved real/imag)
  ifft(buf, size)                Inverse FFT

Real FFT (more efficient for real signals):
  fft_real(buf, size)            Forward real FFT
  ifft_real(buf, size)           Inverse real FFT

Convolution Helpers:
  fft_permute(buf, size)         Reorder for convolution
  fft_ipermute(buf, size)        Inverse reorder
  convolve_c(dest, src, size)    Complex multiply (for convolution)

FFT data layout (complex): buf[0]=re0, buf[1]=im0, buf[2]=re1, buf[3]=im1, ...

════════════════════════════════════════════════════════════════════════════════
MIDI FUNCTIONS (in @block section)
════════════════════════════════════════════════════════════════════════════════

Receiving:
  midirecv(offset, msg1, msg2, msg3)     Receive next MIDI event
                                         Returns 1 if event available
                                         offset = sample position in block
  midirecv_buf(offset, buf, maxlen)      Receive raw bytes
  midirecv_str(offset, #str)             Receive as string

Sending:
  midisend(offset, msg1, msg2, msg3)     Send MIDI event
  midisend_buf(offset, buf, len)         Send raw bytes
  midisend_str(offset, #str)             Send as string

SysEx:
  midisyx(offset, buf, len)              Send SysEx message

MIDI Status Bytes:
  0x80 + ch = Note Off          msg2=note, msg3=velocity
  0x90 + ch = Note On           msg2=note, msg3=velocity (0=note off)
  0xA0 + ch = Aftertouch        msg2=note, msg3=pressure
  0xB0 + ch = Control Change    msg2=CC#, msg3=value
  0xC0 + ch = Program Change    msg2=program
  0xD0 + ch = Channel Pressure  msg2=pressure
  0xE0 + ch = Pitch Bend        msg2=LSB, msg3=MSB (center=8192)

Example MIDI processing:
  @block
  while (midirecv(offset, msg1, msg2, msg3)) (
    status = msg1 & 0xF0;
    channel = msg1 & 0x0F;
    status == 0x90 && msg3 > 0 ? (
      // Note On: msg2 = note number, msg3 = velocity
      msg2 += transpose;  // transpose note
    );
    midisend(offset, msg1, msg2, msg3);
  );

════════════════════════════════════════════════════════════════════════════════
GRAPHICS (@gfx section)
════════════════════════════════════════════════════════════════════════════════

Window/Drawing State:
  gfx_w, gfx_h               Window dimensions
  gfx_x, gfx_y               Current drawing position
  gfx_r, gfx_g, gfx_b, gfx_a Current color (0.0 to 1.0)
  gfx_mode                   Blend mode (0=normal, 1=additive, etc.)
  gfx_clear                  Background color (set before drawing, -1=no clear)
  gfx_dest                   Drawing destination (-1=screen, 0-1023=image buffer)

Color:
  gfx_set(r, g, b)           Set RGB (0-1)
  gfx_set(r, g, b, a)        Set RGBA
  gfx_set(r, g, b, a, mode)  Set with blend mode
  gfx_set(r, g, b, a, mode, dest)  Set with destination

Drawing Primitives:
  gfx_line(x1, y1, x2, y2)         Line
  gfx_line(x1, y1, x2, y2, aa)     Antialiased line (aa=1)
  gfx_rect(x, y, w, h)             Filled rectangle
  gfx_rect(x, y, w, h, filled)     Rectangle (filled=0 for outline)
  gfx_circle(x, y, r, fill)        Circle
  gfx_circle(x, y, r, fill, aa)    Antialiased circle
  gfx_triangle(x1,y1, x2,y2, x3,y3)  Filled triangle
  gfx_roundrect(x, y, w, h, radius) Rounded rectangle
  gfx_arc(x, y, r, ang1, ang2)     Arc (angles in radians)
  gfx_gradrect(x,y,w,h, r,g,b,a, drdx,dgdx,dbdx,dadx, drdy,dgdy,dbdy,dady)

Text:
  gfx_drawstr("text")              Draw string at gfx_x, gfx_y
  gfx_drawchar(charcode)           Draw single character
  gfx_drawnumber(num, digits)      Draw number
  gfx_printf("fmt", ...)           Formatted text
  gfx_measurestr("text", &w, &h)   Measure text dimensions
  gfx_setfont(idx)                 Select font (0=default, 1-16=custom)
  gfx_setfont(idx, "name", size)   Create/select font
  gfx_setfont(idx, "name", size, flags)  flags: 'B'=bold, 'I'=italic

Images:
  gfx_loadimg(idx, "file.png")     Load image to slot 0-1023
  gfx_setimgdim(idx, w, h)         Create/resize image buffer
  gfx_getimgdim(idx, &w, &h)       Get image dimensions
  gfx_blit(src, scale, rotation)   Blit image at gfx_x, gfx_y
  gfx_blitext(src, coordlist, rotation)  Extended blit

Mouse:
  mouse_x, mouse_y           Current position
  mouse_cap                  Button state bitmap:
                             1=left, 2=right, 4=ctrl, 8=shift,
                             16=alt, 32=win, 64=middle
  mouse_wheel                Scroll delta (reset after reading)
  gfx_getchar()              Get keyboard char (0=none, 27=ESC, etc.)

════════════════════════════════════════════════════════════════════════════════
SPECIAL OPTIONS & DIRECTIVES
════════════════════════════════════════════════════════════════════════════════

Header Options:
  options:no_meter           Disable VU meters
  options:maxmem=8388608     Set max memory (bytes)
  options:want_all_kb        Receive all keyboard input in @gfx
  options:gmem=mysharedmem   Name for gmem[] sharing between effects

Init Behavior:
  ext_noinit                 Set to 1 in @init to preserve state on param change
  ext_nodenorm               Set to 1 to disable denormal fixing

════════════════════════════════════════════════════════════════════════════════
CONTROL FLOW & OPERATORS
════════════════════════════════════════════════════════════════════════════════

Conditionals:
  condition ? true_val : false_val    Ternary
  condition ? ( statements; );        Conditional block

Loops:
  while (condition) ( body; );        While loop
  loop(count, ( body; ));             Fixed iteration loop

Blocks (return last value):
  ( stmt1; stmt2; result );           Block expression

Operators:
  + - * /        Arithmetic
  % |0           Modulo (use |0 after for integer result)
  & | ^          Bitwise AND, OR, XOR
  << >>          Bit shift
  == != < > <= >=  Comparison
  && ||          Logical AND, OR
  !              Logical NOT
  ^              Power (2^10 = 1024)

Assignment:
  =              Assign
  += -= *= /=    Compound assignment

════════════════════════════════════════════════════════════════════════════════
COMMON DSP PATTERNS
════════════════════════════════════════════════════════════════════════════════

dB to Linear:
  gain = 10^(db/20);
  // or: gain = exp(db * 0.11512925464970228);

Linear to dB:
  db = 20*log10(gain);

Frequency to Angular:
  omega = 2*$pi*freq/srate;

Time Constant (attack/release):
  coef = exp(-1/(srate * time_seconds));
  // Usage: env = coef*env + (1-coef)*target;

Biquad Filter (RBJ cookbook):
  // Compute coefficients in @slider
  omega = 2*$pi*freq/srate;
  sn = sin(omega); cs = cos(omega);
  alpha = sn/(2*Q);
  // Lowpass:
  b0 = (1-cs)/2; b1 = 1-cs; b2 = (1-cs)/2;
  a0 = 1+alpha; a1 = -2*cs; a2 = 1-alpha;
  // Normalize:
  b0/=a0; b1/=a0; b2/=a0; a1/=a0; a2/=a0;
  // Apply in @sample (Direct Form II Transposed):
  out = b0*in + s1;
  s1 = b1*in - a1*out + s2;
  s2 = b2*in - a2*out;

Soft Clipping:
  // Cubic soft clip
  x = abs(spl0) > 1 ? sign(spl0) : 1.5*spl0 - 0.5*spl0^3;

Delay Line:
  // In @init:
  delay_buf = 0; buf_size = srate*2; freembuf(buf_size); write_pos = 0;
  // In @sample:
  read_pos = write_pos - delay_samples;
  read_pos < 0 ? read_pos += buf_size;
  delayed = delay_buf[read_pos];
  delay_buf[write_pos] = spl0;
  write_pos = (write_pos+1) % buf_size;

Linear Interpolation (for fractional delay):
  frac = delay_samples - floor(delay_samples);
  idx = floor(delay_samples);
  out = buf[idx]*(1-frac) + buf[idx+1]*frac;

Envelope Follower:
  level = max(abs(spl0), abs(spl1));
  env = level > env ? att*(env-level)+level : rel*(env-level)+level;

════════════════════════════════════════════════════════════════════════════════
OUTPUT REQUIREMENTS
════════════════════════════════════════════════════════════════════════════════

- Output ONLY valid JSFX code - no explanations, no commentary, no other text
- Output complete, syntactically correct JSFX
- Always include desc: line with descriptive effect name
- Define in_pin/out_pin for stereo (Left/Right) unless mono/multichannel
- Use meaningful slider names with appropriate ranges and units
- Initialize ALL variables in @init (EEL2 has no implicit initialization)
- Use comments to explain complex algorithms
- Handle edge cases (division by zero, out-of-range values)
- Prefer Direct Form II Transposed for filters (better numerical stability)`
}
