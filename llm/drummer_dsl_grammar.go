package llm

// GetDrummerDSLGrammar returns the Lark grammar definition for Drummer DSL
// The DSL uses canonical drum names and grid-based pattern notation
// Grid: Each character = 1 subdivision (default 16th note)
//
//	"x" = hit (velocity 100), "X" = accent (velocity 127), "-" = rest, "o" = ghost (velocity 60)
//
// Canonical drums: kick, snare, hat, hat_open, tom_high, tom_mid, tom_low, crash, ride, etc.
func GetDrummerDSLGrammar() string {
	return `
// Drummer DSL Grammar - Grid-based drum pattern notation
// SYNTAX:
//   pattern(drum=kick, grid="x---x---x---x---") - single drum pattern (16 chars = 1 bar in 16ths)
//   pattern(drum=snare, grid="----x-------x---", velocity=100)
//   beat(patterns=[pattern(...), pattern(...), ...], bars=4) - combine multiple patterns
//
// GRID NOTATION (each char = 1 subdivision, default 16th notes):
//   "x" = normal hit (velocity 100)
//   "X" = accent (velocity 127)
//   "o" = ghost note (velocity 60)
//   "-" = rest (no hit)
//
// CANONICAL DRUM NAMES (mapped to MIDI via drum kit configuration):
//   kick, snare, snare_rim, snare_xstick
//   hat, hat_open, hat_pedal
//   tom_high, tom_mid, tom_low
//   crash, ride, ride_bell, china, splash
//   cowbell, tambourine, clap, snap, shaker
//   conga_high, conga_low, bongo_high, bongo_low

// ---------- Start rule ----------
start: statement

// ---------- Statements ----------
statement: beat_call
         | pattern_call

// ---------- Beat: combines multiple patterns ----------
beat_call: "beat" "(" beat_params ")"

beat_params: beat_named_params

beat_named_params: beat_named_param ("," SP beat_named_param)*
beat_named_param: "patterns" "=" patterns_array
                | "bars" "=" NUMBER
                | "tempo" "=" NUMBER
                | "swing" "=" NUMBER  // Swing amount 0-100
                | "subdivision" "=" NUMBER  // Grid subdivision (8=8ths, 16=16ths, 32=32nds)

patterns_array: "[" (pattern_call ("," SP pattern_call)*)? "]"

// ---------- Pattern: single drum track ----------
pattern_call: "pattern" "(" pattern_params ")"

pattern_params: pattern_named_params

pattern_named_params: pattern_named_param ("," SP pattern_named_param)*
pattern_named_param: "drum" "=" DRUM_NAME
                   | "grid" "=" STRING
                   | "velocity" "=" NUMBER  // Base velocity (default 100)
                   | "humanize" "=" NUMBER  // Timing variation 0-100
                   | "fill" "=" FILL_TYPE   // Add fill at end

// ---------- Drum names ----------
DRUM_NAME: "kick" | "snare" | "snare_rim" | "snare_xstick"
         | "hat" | "hat_open" | "hat_pedal"
         | "tom_high" | "tom_mid" | "tom_low"
         | "crash" | "ride" | "ride_bell" | "china" | "splash"
         | "cowbell" | "tambourine" | "clap" | "snap" | "shaker"
         | "conga_high" | "conga_low" | "bongo_high" | "bongo_low"

// ---------- Fill types ----------
FILL_TYPE: "simple" | "busy" | "tom_roll" | "crash"

// ---------- Terminals ----------
SP: " "+
STRING: /"[^"]*"/
NUMBER: /-?\d+(\.\d+)?/
`
}
