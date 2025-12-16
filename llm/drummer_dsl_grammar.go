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
//   pattern(drum=kick, grid="x---x---x---x---")
//   pattern(drum=snare, grid="----x-------x---", velocity=100)
//
// GRID NOTATION (each char = 1 16th note):
//   "x" = hit (velocity 100)
//   "X" = accent (velocity 127)
//   "o" = ghost note (velocity 60)
//   "-" = rest
//
// DRUMS: kick, snare, hat, hat_open, tom_high, tom_mid, tom_low, crash, ride

// ---------- Start rule ----------
start: pattern_call (";" pattern_call)*

// ---------- Pattern ----------
pattern_call: "pattern" "(" pattern_params ")"

pattern_params: pattern_named_params

pattern_named_params: pattern_named_param ("," SP pattern_named_param)*
pattern_named_param: "drum" "=" DRUM_NAME
                   | "grid" "=" STRING
                   | "velocity" "=" NUMBER

// ---------- Drum names ----------
DRUM_NAME: "kick" | "snare" | "snare_rim" | "snare_xstick"
         | "hat" | "hat_open" | "hat_pedal"
         | "tom_high" | "tom_mid" | "tom_low"
         | "crash" | "ride" | "ride_bell" | "china" | "splash"
         | "cowbell" | "tambourine" | "clap" | "snap" | "shaker"

// ---------- Terminals ----------
SP: " "+
STRING: /"[^"]*"/
NUMBER: /-?\d+(\.\d+)?/
`
}
