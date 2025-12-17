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
chain: clip_chain | fx_chain | volume_chain | pan_chain | mute_chain | solo_chain | name_chain

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
