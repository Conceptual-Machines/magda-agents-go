package llm

// GetMagdaDSLGrammar returns the Lark grammar definition for MAGDA DSL
// The DSL allows chaining operations like: track(instrument="Serum").newClip(bar=3, length_bars=4)
// NOTE: MIDI notes are handled by the ARRANGER agent, NOT the DAW agent
func GetMagdaDSLGrammar() string {
	return `
// MAGDA DSL Grammar - Functional scripting for REAPER operations
// Syntax: track().newClip() with method chaining
// NOTE: Musical content (notes, chords, arpeggios) is handled by ARRANGER agent

// ---------- Start rule ----------
start: statement+

// ---------- Statements ----------
statement: track_call chain?

// ---------- Track creation or reference ----------
track_call: "track" "(" track_params? ")"
track_params: track_param ("," SP track_param)*
           | NUMBER  // track(1) references existing track 1
track_param: "instrument" "=" STRING
           | "name" "=" STRING
           | "index" "=" NUMBER
           | "id" "=" NUMBER  // track(id=1) references existing track 1
           | "selected" "=" BOOLEAN  // track(selected=true) references currently selected track

// ---------- Method chaining ----------
chain: clip_chain | fx_chain | volume_chain | pan_chain | mute_chain | solo_chain | name_chain | automation_chain

// ---------- Clip operations ----------
clip_chain: ".newClip" "(" clip_params? ")" (fx_chain | volume_chain | pan_chain | mute_chain | solo_chain | name_chain)?
clip_params: clip_param ("," SP clip_param)*
clip_param: "bar" "=" NUMBER
          | "start" "=" NUMBER
          | "end" "=" NUMBER
          | "length_bars" "=" NUMBER
          | "length" "=" NUMBER
          | "position" "=" NUMBER

// ---------- FX operations ----------
fx_chain: ".addFX" "(" fx_params? ")"
fx_params: "fxname" "=" STRING
         | "instrument" "=" STRING

// ---------- Track control operations ----------
volume_chain: ".setVolume" "(" "volume_db" "=" NUMBER ")"
pan_chain: ".setPan" "(" "pan" "=" NUMBER ")"
mute_chain: ".setMute" "(" "mute" "=" BOOLEAN ")"
solo_chain: ".setSolo" "(" "solo" "=" BOOLEAN ")"
name_chain: ".setName" "(" "name" "=" STRING ")"

// ---------- Automation operations ----------
// Two modes: curve-based (recommended) or point-based
// Curve-based example: track(id=1).addAutomation(param="volume", curve="fade_in", start=0, end=4)
// Point-based example: track(id=1).addAutomation(param="volume", points=[{time=0, value=-60}, {time=4, value=0}])
automation_chain: ".addAutomation" "(" automation_params ")"
automation_params: automation_param ("," SP automation_param)*
automation_param: "param" "=" STRING           // "volume", "pan", "mute", or FX param like "Serum:Cutoff"
                | "curve" "=" STRING           // "fade_in", "fade_out", "ramp", "sine", "saw", "square", "exp_in", "exp_out"
                | "start" "=" NUMBER           // start time in beats
                | "end" "=" NUMBER             // end time in beats
                | "start_bar" "=" NUMBER       // start bar (alternative to beats)
                | "end_bar" "=" NUMBER         // end bar (alternative to beats)
                | "from" "=" NUMBER            // starting value (for ramp)
                | "to" "=" NUMBER              // ending value (for ramp)
                | "freq" "=" NUMBER            // frequency for oscillators (cycles per bar)
                | "amplitude" "=" NUMBER       // amplitude for oscillators (0-1 range, scaled to param)
                | "phase" "=" NUMBER           // phase offset for oscillators (0-1)
                | "points" "=" automation_points  // manual points (legacy/advanced)
                | "shape" "=" NUMBER           // curve shape: 0=linear, 1=square, 2=slow, 3=fast start, 4=fast end, 5=bezier
automation_points: "[" automation_point ("," SP automation_point)* "]"
automation_point: "{" automation_point_fields "}"
automation_point_fields: automation_point_field ("," SP automation_point_field)*
automation_point_field: "time" "=" NUMBER      // time in beats (0 = start, 4 = beat 4)
                      | "bar" "=" NUMBER       // bar number (1 = first bar)
                      | "value" "=" NUMBER     // value: dB for volume, -1 to 1 for pan, 0-1 for others

// ---------- Arrays ----------
array: "[" (value ("," SP value)*)? "]"
value: STRING | NUMBER | BOOLEAN | array

// ---------- Terminals ----------
SP: " "
STRING: /"[^"]*"/
NUMBER: /-?\d+(\.\d+)?/
BOOLEAN: "true" | "false"
`
}
