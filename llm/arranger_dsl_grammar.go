package llm

// GetArrangerDSLGrammar returns the Lark grammar definition for Arranger DSL
// The DSL uses chord symbols (Em, C, Am7) and arpeggios instead of discrete notes
// Supports both relative timing (length, note_duration) and explicit rhythm timing (start, duration)
// Format: arpeggio(symbol=Em, note_duration=0.25) or chord(symbol=C, start=0, duration=4)
func GetArrangerDSLGrammar() string {
	return `
// Arranger DSL Grammar - Chord symbol-based musical composition
// SIMPLE SYNTAX ONLY - one call per statement:
//   arpeggio(symbol=Em, note_duration=0.25) - for arpeggios with specific note duration
//   arpeggio(symbol=Em, start=0.0, note_duration=0.25) - with explicit start time
//   chord(symbol=C, length=4) - for chords (simultaneous notes) with relative timing
//   chord(symbol=C, start=0, duration=4) - for chords with explicit rhythm timing
//   progression(chords=[C, Am, F, G], length=16) - for chord progressions
// Note: Supports both relative timing (length) and explicit rhythm timing (start, duration)

// ---------- Start rule ----------
start: statement

// ---------- Statements - ONE call only, no chaining ----------
statement: arpeggio_call
         | chord_call
         | progression_call

// ---------- Arpeggio: SEQUENTIAL notes ----------
arpeggio_call: "arpeggio" "(" arpeggio_params ")"

arpeggio_params: arpeggio_named_params

arpeggio_named_params: arpeggio_named_param ("," SP arpeggio_named_param)*
arpeggio_named_param: "symbol" "=" chord_symbol
                    | "chord" "=" chord_symbol
                    | "length" "=" NUMBER
                    | "start" "=" NUMBER  // Explicit start time in beats (for rhythm timing)
                    | "duration" "=" NUMBER  // Explicit duration in beats (for rhythm timing)
                    | "note_duration" "=" NUMBER  // REQUIRED for note length: 0.25=16th, 0.5=8th, 1=quarter
                    | "rhythm" "=" STRING  // Rhythm template name (swing, bossa, syncopated, etc.)
                    | "repeat" "=" NUMBER
                    | "velocity" "=" NUMBER
                    | "octave" "=" NUMBER
                    | "direction" "=" ("up" | "down" | "updown")

// ---------- Chord: SIMULTANEOUS notes ----------
chord_call: "chord" "(" chord_params ")"

chord_params: chord_named_params

chord_named_params: chord_named_param ("," SP chord_named_param)*
chord_named_param: "symbol" "=" chord_symbol
                 | "chord" "=" chord_symbol
                 | "length" "=" NUMBER
                 | "start" "=" NUMBER  // Explicit start time in beats (for rhythm timing)
                 | "duration" "=" NUMBER  // Explicit duration in beats (for rhythm timing)
                 | "rhythm" "=" STRING  // Rhythm template name (swing, bossa, syncopated, etc.)
                 | "repeat" "=" NUMBER
                 | "velocity" "=" NUMBER
                 | "inversion" "=" NUMBER

// ---------- Progression: sequence of chords ----------
progression_call: "progression" "(" progression_params ")"

progression_params: progression_named_params

progression_named_params: progression_named_param ("," SP progression_named_param)*
progression_named_param: "chords" "=" chords_array
                       | "length" "=" NUMBER
                       | "start" "=" NUMBER  // Explicit start time in beats (for rhythm timing)
                       | "repeat" "=" NUMBER

chords_array: "[" (chord_symbol ("," SP chord_symbol)*)? "]"

// ---------- Chord symbol (supports Em, C, Am7, Cmaj7, etc.) ----------
chord_symbol: CHORD_ROOT CHORD_QUALITY? CHORD_EXTENSION? CHORD_BASS?
CHORD_ROOT: /[A-G][#b]?/
CHORD_QUALITY: "m" | "dim" | "aug" | "sus2" | "sus4"
CHORD_EXTENSION: /[0-9]+/ | "maj7" | "min7" | "dim7" | "aug7" | "add9" | "add11" | "add13"
CHORD_BASS: "/" CHORD_ROOT

// ---------- Terminals ----------
SP: " "+
STRING: /"[^"]*"/
NUMBER: /-?\d+(\.\d+)?/
`
}
