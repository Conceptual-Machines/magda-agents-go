package llm

// GetJSFXDSLGrammar returns the Lark grammar definition for JSFX DSL
// The DSL allows AI to generate JSFX audio effects for REAPER
// Based on: https://www.reaper.fm/sdk/js/js.php
//
// JSFX sections: @init, @slider, @sample, @block, @gfx, @serialize
// Sample vars: spl0, spl1, ... spl63
// Slider vars: slider1, slider2, ... slider256
func GetJSFXDSLGrammar() string {
	return `
// JSFX DSL Grammar - AI-assisted JSFX effect generation
// SYNTAX:
//   effect(name="My Effect", tags="dynamics compressor")
//   slider(id=1, default=0, min=-60, max=0, step=1, name="Gain (dB)")
//   init_code(code="gain = 1;")
//   slider_code(code="gain = 10^(slider1/20);")
//   sample_code(code="spl0 *= gain; spl1 *= gain;")
//
// EFFECT TYPES: compressor, limiter, eq, filter, distortion, delay, reverb, chorus, flanger, phaser, gate, expander

// ---------- Start rule ----------
start: statement (";" statement)*

// ---------- Statements ----------
statement: effect_call
         | slider_call
         | init_code_call
         | slider_code_call
         | sample_code_call
         | block_code_call
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
           | "midi" | "synth" | "sampler"

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

// ---------- Code sections ----------
init_code_call: "init_code" "(" "code" "=" STRING ")"
slider_code_call: "slider_code" "(" "code" "=" STRING ")"
sample_code_call: "sample_code" "(" "code" "=" STRING ")"
block_code_call: "block_code" "(" "code" "=" STRING ")"
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

// GetJSFXSystemPrompt returns the system prompt for JSFX generation
func GetJSFXSystemPrompt() string {
	return `You are a JSFX expert. Generate REAPER JSFX effects using the DSL grammar.

JSFX REFERENCE (https://www.reaper.fm/sdk/js/js.php):

AUDIO VARIABLES:
- spl0, spl1, ... spl63: Audio samples for each channel
- samplesblock: Number of samples in current block
- srate: Sample rate
- num_ch: Number of channels
- tempo, ts_num, ts_denom: Tempo and time signature
- play_state, play_position: Playback state and position

SLIDER VARIABLES:
- slider1, slider2, ... slider256: Parameter values
- slider_show(mask, value): Show/hide sliders
- sliderchange(mask): Get changed sliders bitmap

MATH FUNCTIONS:
- sin, cos, tan, asin, acos, atan, atan2
- log, log10, exp, pow, sqrt, abs
- min, max, floor, ceil, sign
- rand(max): Random 0 to max-1

MEMORY:
- slider1[] (array access), mdct_ctx[]
- memset(dest, value, length)
- memcpy(dest, src, length)
- freembuf(size): Allocate/get memory

FFT:
- fft(buffer, size), ifft(buffer, size)
- fft_real(buffer, size), ifft_real(buffer, size)
- convolve_c(dest, src, size)

COMMON PATTERNS:

1. GAIN/VOLUME:
   @slider: gain = 10^(slider1/20);  // dB to linear
   @sample: spl0 *= gain; spl1 *= gain;

2. FILTER (biquad):
   @init: a0=a1=a2=b1=b2=x1=x2=y1=y2=0;
   @slider: // calculate coefficients based on freq, Q
   @sample: y0 = a0*spl0 + a1*x1 + a2*x2 - b1*y1 - b2*y2;
            x2=x1; x1=spl0; y2=y1; y1=y0; spl0=y0;

3. COMPRESSOR:
   @init: env = 0;
   @slider: thresh_lin = 10^(thresh_db/20); ratio_inv = 1/ratio;
   @sample: input_level = max(abs(spl0), abs(spl1));
            env = input_level > env ? attack_coef*(env-input_level)+input_level 
                                   : release_coef*(env-input_level)+input_level;
            gain_reduction = env > thresh_lin ? (thresh_lin/env)^(1-ratio_inv) : 1;
            spl0 *= gain_reduction; spl1 *= gain_reduction;

4. DELAY:
   @init: buf_size = srate*2; delay_buf = 0; write_pos = 0;
   @slider: delay_samples = floor(delay_ms * srate / 1000);
   @sample: read_pos = write_pos - delay_samples;
            read_pos < 0 ? read_pos += buf_size;
            delayed = delay_buf[read_pos];
            delay_buf[write_pos] = spl0 + delayed*feedback;
            spl0 = spl0*(1-mix) + delayed*mix;
            write_pos = (write_pos+1) % buf_size;

OUTPUT FORMAT:
Use the DSL to define the effect structure, then provide full JSFX code in sample_code/init_code etc.

effect(name="Effect Name", tags="category tags");
slider(id=1, default=0, min=-60, max=0, step=0.1, name="Gain (dB)");
init_code(code="gain = 1;");
slider_code(code="gain = 10^(slider1/20);");
sample_code(code="spl0 *= gain; spl1 *= gain;")
`
}

