package llm

// GetJSFXDSLGrammar returns the Lark grammar definition for JSFX DSL
// The DSL allows AI to generate JSFX audio effects for REAPER
// Based on: https://www.reaper.fm/sdk/js/js.php
//
// JSFX is based on EEL2 language with audio-specific extensions
// The DSL defines structure; actual EEL2 code goes in code strings
func GetJSFXDSLGrammar() string {
	return `
// JSFX DSL Grammar - AI-assisted JSFX effect generation
// SYNTAX:
//   effect(name="My Effect", tags="dynamics compressor")
//   slider(id=1, default=0, min=-60, max=0, step=1, name="Gain (dB)")
//   pin(type=in, name="Left", channel=0)
//   pin(type=out, name="Left", channel=0)
//   import(file="some_library.jsfx-inc")
//   option(name="no_meter")
//   init_code(code="gain = 1;")
//   slider_code(code="gain = 10^(slider1/20);")
//   block_code(code="// per-block processing")
//   sample_code(code="spl0 *= gain; spl1 *= gain;")
//   serialize_code(code="file_var(0, my_state);")
//   gfx_code(code="gfx_drawstr(\"Hello\");", width=400, height=200)

// ---------- Start rule ----------
start: statement (";" statement)*

// ---------- Statements ----------
statement: effect_call
         | slider_call
         | pin_call
         | import_call
         | option_call
         | filename_call
         | init_code_call
         | slider_code_call
         | block_code_call
         | sample_code_call
         | serialize_code_call
         | gfx_code_call

// ---------- Effect metadata ----------
effect_call: "effect" "(" effect_params ")"
effect_params: effect_param ("," SP effect_param)*
effect_param: "name" "=" STRING
            | "desc" "=" STRING
            | "tags" "=" STRING
            | "type" "=" EFFECT_TYPE

EFFECT_TYPE: "compressor" | "limiter" | "eq" | "filter" | "distortion" 
           | "delay" | "reverb" | "chorus" | "flanger" | "phaser"
           | "gate" | "expander" | "saturator" | "utility" | "analyzer"
           | "midi" | "synth" | "sampler" | "pitchshift" | "vocoder"
           | "multiband" | "stereo" | "dynamics" | "modulation"

// ---------- Slider (parameter) definition ----------
slider_call: "slider" "(" slider_params ")"
slider_params: slider_param ("," SP slider_param)*
slider_param: "id" "=" NUMBER
            | "default" "=" NUMBER
            | "min" "=" NUMBER
            | "max" "=" NUMBER
            | "step" "=" NUMBER
            | "name" "=" STRING
            | "var" "=" IDENTIFIER
            | "hidden" "=" BOOLEAN
            | "options" "=" STRING  // comma-separated list for dropdown

// ---------- Pin (input/output) definition ----------
pin_call: "pin" "(" pin_params ")"
pin_params: pin_param ("," SP pin_param)*
pin_param: "type" "=" PIN_TYPE
         | "name" "=" STRING
         | "channel" "=" NUMBER

PIN_TYPE: "in" | "out"

// ---------- Import statement ----------
import_call: "import" "(" "file" "=" STRING ")"

// ---------- Option statement ----------
option_call: "option" "(" option_params ")"
option_params: option_param ("," SP option_param)*
option_param: "name" "=" STRING
            | "value" "=" STRING

// ---------- Filename statement ----------
filename_call: "filename" "(" filename_params ")"
filename_params: filename_param ("," SP filename_param)*
filename_param: "id" "=" NUMBER
              | "path" "=" STRING
              | "name" "=" STRING

// ---------- Code sections ----------
init_code_call: "init_code" "(" "code" "=" STRING ")"
slider_code_call: "slider_code" "(" "code" "=" STRING ")"
block_code_call: "block_code" "(" "code" "=" STRING ")"
sample_code_call: "sample_code" "(" "code" "=" STRING ")"
serialize_code_call: "serialize_code" "(" "code" "=" STRING ")"
gfx_code_call: "gfx_code" "(" gfx_params ")"
gfx_params: gfx_param ("," SP gfx_param)*
gfx_param: "code" "=" STRING
         | "width" "=" NUMBER
         | "height" "=" NUMBER

// ---------- Terminals ----------
SP: " "+
STRING: /"[^"]*"/
NUMBER: /-?\d+(\.\d+)?/
BOOLEAN: "true" | "false"
IDENTIFIER: /[a-zA-Z_][a-zA-Z0-9_]*/
`
}

// GetJSFXSystemPrompt returns the comprehensive system prompt for JSFX generation
func GetJSFXSystemPrompt() string {
	return `You are a JSFX expert. Generate REAPER JSFX effects using the DSL grammar.
Output DSL statements separated by semicolons. Put actual EEL2 code in code="" strings.

JSFX REFERENCE (based on EEL2):

═══════════════════════════════════════════════════════════════════════════════
AUDIO VARIABLES
═══════════════════════════════════════════════════════════════════════════════
spl0, spl1, ... spl63    Audio samples for each channel (read/write in @sample)
samplesblock             Number of samples in current block
srate                    Sample rate (e.g., 44100, 48000)
num_ch                   Number of channels
tempo                    Current tempo in BPM
ts_num, ts_denom         Time signature numerator/denominator
play_state               1=playing, 0=stopped, 2=paused, 5=recording, 6=rec paused
play_position            Current playback position in seconds
beat_position            Current position in beats
pdc_delay                Plugin delay compensation (samples to report)
pdc_bot_ch, pdc_top_ch   PDC channel range

═══════════════════════════════════════════════════════════════════════════════
SLIDER VARIABLES
═══════════════════════════════════════════════════════════════════════════════
slider1, slider2, ... slider256    Parameter values from sliders
slider_show(mask, value)           Show/hide sliders dynamically
sliderchange(mask)                 Bitmap of which sliders changed
trigger                            Set to 1 when effect recompiled

═══════════════════════════════════════════════════════════════════════════════
MATH FUNCTIONS
═══════════════════════════════════════════════════════════════════════════════
sin(x), cos(x), tan(x)             Trigonometric
asin(x), acos(x), atan(x), atan2(y,x)
log(x), log10(x), exp(x)           Logarithmic/exponential
pow(x,y), sqrt(x), sqr(x)          Power functions (sqr = x*x)
abs(x), sign(x)                    Absolute value, sign (-1, 0, 1)
min(a,b), max(a,b)                 Min/max
floor(x), ceil(x), int(x)          Rounding
rand(max)                          Random 0 to max-1
invsqrt(x)                         Fast inverse square root

CONSTANTS: $pi, $phi (golden ratio), $e, $'A' (ASCII char code)

═══════════════════════════════════════════════════════════════════════════════
MEMORY FUNCTIONS
═══════════════════════════════════════════════════════════════════════════════
buf[index]                         Array access (any variable can be array base)
freembuf(top)                      Allocate memory, returns previous top
memset(dest, value, count)         Fill memory with value
memcpy(dest, src, count)           Copy memory
mem_get_values(buf, ...)           Read multiple values from buffer
mem_set_values(buf, ...)           Write multiple values to buffer
stack_push(val)                    Push to stack
stack_pop(val)                     Pop from stack (returns value)
stack_peek(idx)                    Read stack without popping
stack_exch(val)                    Exchange top of stack

GLOBAL MEMORY:
gmem[index]                        Shared between all JSFX instances
reg00-reg99                        Shared registers between effects

═══════════════════════════════════════════════════════════════════════════════
FFT FUNCTIONS
═══════════════════════════════════════════════════════════════════════════════
fft(buf, size)                     Complex FFT (size must be power of 2)
ifft(buf, size)                    Inverse complex FFT
fft_real(buf, size)                Real FFT
ifft_real(buf, size)               Inverse real FFT
fft_permute(buf, size)             Permute for convolution
fft_ipermute(buf, size)            Inverse permute
convolve_c(dest, src, size)        Complex multiply for convolution

═══════════════════════════════════════════════════════════════════════════════
MIDI FUNCTIONS
═══════════════════════════════════════════════════════════════════════════════
midisend(offset, msg1, msg2, msg3) Send MIDI (offset=sample position in block)
midirecv(offset, msg1, msg2, msg3) Receive MIDI (returns 1 if message available)
midisend_buf(offset, buf, len)     Send raw MIDI bytes
midirecv_buf(offset, buf, maxlen)  Receive raw MIDI bytes
midisend_str(offset, string)       Send MIDI as string
midirecv_str(offset, string)       Receive MIDI as string
midisyx(offset, msgptr, len)       Send SysEx

MIDI STATUS BYTES:
0x90 = Note On, 0x80 = Note Off, 0xB0 = CC, 0xE0 = Pitch Bend
0xC0 = Program Change, 0xD0 = Aftertouch

═══════════════════════════════════════════════════════════════════════════════
STRING FUNCTIONS
═══════════════════════════════════════════════════════════════════════════════
strcpy(dest, src)                  Copy string
strcat(dest, src)                  Concatenate strings
strlen(str)                        Get string length
strcmp(s1, s2)                     Compare strings (0 if equal)
stricmp(s1, s2)                    Case-insensitive compare
sprintf(dest, fmt, ...)            Format string (like C printf)
printf(fmt, ...)                   Print to console
match(needle, haystack)            Regex match
matchi(needle, haystack)           Case-insensitive regex match

STRING INDICES: #string or "string" syntax, slots 0-1023 available

═══════════════════════════════════════════════════════════════════════════════
FILE I/O
═══════════════════════════════════════════════════════════════════════════════
file_open(filename)                Open file (use slider to reference filename:N)
file_close(handle)                 Close file
file_rewind(handle)                Rewind to start
file_var(handle, var)              Read/write variable
file_mem(handle, offset, length)   Read/write memory block
file_avail(handle)                 Get remaining bytes (read) or -1 (write)
file_riff(handle, nch, srate)      Read RIFF/WAV header, returns length
file_text(handle, bool)            Set text mode (1) or binary mode (0)
file_string(handle, str)           Read/write string

═══════════════════════════════════════════════════════════════════════════════
GRAPHICS (@gfx)
═══════════════════════════════════════════════════════════════════════════════
gfx_w, gfx_h                       Current window size
gfx_x, gfx_y                       Current drawing position
gfx_r, gfx_g, gfx_b, gfx_a         Current color (0-1 range)
gfx_set(r, g, b, a, mode, dest)    Set color and blend mode
gfx_clear                          Clear color (set before @gfx, auto-clears)

DRAWING:
gfx_line(x1,y1,x2,y2,aa)          Draw line (aa=antialiased)
gfx_rect(x,y,w,h,filled)          Draw rectangle
gfx_circle(x,y,r,filled,aa)       Draw circle
gfx_triangle(x1,y1,x2,y2,x3,y3)   Draw triangle
gfx_roundrect(x,y,w,h,radius,aa)  Rounded rectangle
gfx_arc(x,y,r,a1,a2,aa)           Draw arc
gfx_gradrect(x,y,w,h,r,g,b,a,...)  Gradient rectangle

TEXT:
gfx_drawstr(string)               Draw string at gfx_x, gfx_y
gfx_drawchar(char)                Draw single character
gfx_drawNumber(num, digits)       Draw number
gfx_printf(fmt, ...)              Formatted text
gfx_measurestr(str, w, h)         Measure string dimensions
gfx_setfont(idx, name, size, flags) Set font (idx 0-16, flags: 'B'old, 'I'talic)

IMAGES:
gfx_loadimg(idx, filename)        Load image to slot (0-1023)
gfx_blit(src, scale, rotation)    Blit image
gfx_blitext(src, coords, rotation) Extended blit
gfx_setimgdim(idx, w, h)          Set image buffer size
gfx_getimgdim(idx, w, h)          Get image dimensions

MOUSE:
mouse_x, mouse_y                  Current mouse position
mouse_cap                         Button state: 1=L, 2=R, 4=Ctrl, 8=Shift, 16=Alt, 32=Win, 64=M
mouse_wheel                       Wheel delta (reset after reading)
gfx_getchar()                     Get keyboard char (0 if none)

═══════════════════════════════════════════════════════════════════════════════
COMMON EFFECT PATTERNS
═══════════════════════════════════════════════════════════════════════════════

1. GAIN/VOLUME:
   @slider: gain = 10^(slider1/20);  // dB to linear
   @sample: spl0 *= gain; spl1 *= gain;

2. BIQUAD FILTER:
   @init: a0=a1=a2=b1=b2=x1l=x2l=y1l=y2l=0;
   @slider: 
     omega = 2*$pi*freq/srate;
     sn = sin(omega); cs = cos(omega);
     alpha = sn/(2*Q);
     // Lowpass coefficients:
     b0 = (1-cs)/2; b1 = 1-cs; b2 = (1-cs)/2;
     a0 = 1+alpha; a1 = -2*cs; a2 = 1-alpha;
     // Normalize:
     b0 /= a0; b1 /= a0; b2 /= a0; a1 /= a0; a2 /= a0;
   @sample:
     y0 = b0*spl0 + b1*x1l + b2*x2l - a1*y1l - a2*y2l;
     x2l=x1l; x1l=spl0; y2l=y1l; y1l=y0; spl0=y0;

3. COMPRESSOR:
   @init: env = 0;
   @slider: 
     thresh_lin = 10^(thresh_db/20); 
     ratio_inv = 1/ratio;
     attack_coef = exp(-1/(srate*attack_ms/1000));
     release_coef = exp(-1/(srate*release_ms/1000));
   @sample: 
     input_level = max(abs(spl0), abs(spl1));
     env = input_level > env 
         ? attack_coef*(env-input_level)+input_level 
         : release_coef*(env-input_level)+input_level;
     gain = env > thresh_lin ? (thresh_lin/env)^(1-ratio_inv) : 1;
     spl0 *= gain; spl1 *= gain;

4. DELAY:
   @init: 
     buf_size = srate*2; // 2 seconds max
     delay_buf = 0;      // buffer starts at memory address 0
     write_pos = 0;
     freembuf(buf_size); // allocate memory
   @slider: delay_samples = floor(delay_ms * srate / 1000);
   @sample: 
     read_pos = write_pos - delay_samples;
     read_pos < 0 ? read_pos += buf_size;
     delayed = delay_buf[read_pos];
     delay_buf[write_pos] = spl0 + delayed * feedback;
     spl0 = spl0 * (1-mix) + delayed * mix;
     write_pos = (write_pos+1) % buf_size;

5. MIDI PROCESSOR:
   @block:
     while (midirecv(offset, msg1, msg2, msg3)) (
       status = msg1 & 0xF0;
       channel = msg1 & 0x0F;
       status == 0x90 ? ( // Note On
         // Process note: msg2=note, msg3=velocity
       );
       midisend(offset, msg1, msg2, msg3); // pass through
     );

═══════════════════════════════════════════════════════════════════════════════
OUTPUT FORMAT
═══════════════════════════════════════════════════════════════════════════════
Use DSL to define effect structure. Put actual EEL2 code in code="" strings.
Use \n for newlines in code strings. Escape quotes with \"

EXAMPLE:
effect(name="Stereo Gain", tags="utility volume");
pin(type=in, name="Left", channel=0);
pin(type=in, name="Right", channel=1);
pin(type=out, name="Left", channel=0);
pin(type=out, name="Right", channel=1);
slider(id=1, default=0, min=-60, max=12, step=0.1, name="Gain (dB)");
slider(id=2, default=0, min=-100, max=100, step=1, name="Balance");
init_code(code="gain_l = gain_r = 1;");
slider_code(code="gain = 10^(slider1/20);\nbal = slider2/100;\ngain_l = gain * min(1, 1-bal);\ngain_r = gain * min(1, 1+bal);");
sample_code(code="spl0 *= gain_l;\nspl1 *= gain_r;")
`
}
