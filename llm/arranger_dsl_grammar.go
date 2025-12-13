package llm

// GetArrangerDSLGrammar returns the Lark grammar definition for Arranger DSL
// The DSL uses chord symbols (Em, C, Am7) and arpeggios instead of discrete notes
// Timing is relative - only length and repetitions. Absolute positioning handled by DAW agent.
// Format: arpeggio("Em", length=2) or chord("C", length=1, repeat=4)
func GetArrangerDSLGrammar() string {
	return `
// Arranger DSL Grammar - Chord symbol-based musical composition
// Syntax: arpeggio("Em", length=2) or chord("C", length=1, repeat=4)
// Note: Timing is relative - no start times. DAW agent handles absolute positioning.

// ---------- Start rule ----------
start: statement+

// ---------- Statements ----------
statement: composition
         | arpeggio_call
         | chord_call
         | progression_call

// ---------- Composition structure ----------
composition: "composition" "(" ")" chain_item+
           | "choice" "(" choice_params ")"

chain_item: ".add_arpeggio" "(" arpeggio_params ")"
          | ".add_chord" "(" chord_params ")"
          | ".add_progression" "(" progression_params ")"

// ---------- Arpeggio operations ----------
arpeggio_call: "arpeggio" "(" arpeggio_params ")"

arpeggio_params: chord_symbol "," NUMBER  // positional: symbol, length
                | arpeggio_named_params

arpeggio_named_params: arpeggio_named_param ("," SP arpeggio_named_param)*
arpeggio_named_param: "symbol" "=" chord_symbol
                    | "chord" "=" chord_symbol
                    | "length" "=" NUMBER
                    | "duration" "=" NUMBER  // alias for length
                    | "note_duration" "=" NUMBER  // duration of each note (e.g., 0.25 for 16th)
                    | "repeat" "=" NUMBER
                    | "repetitions" "=" NUMBER  // alias for repeat
                    | "velocity" "=" NUMBER
                    | "octave" "=" NUMBER
                    | "direction" "=" ("up" | "down" | "updown")
                    | "pattern" "=" STRING

// ---------- Chord operations ----------
chord_call: "chord" "(" chord_params ")"

chord_params: chord_symbol "," NUMBER  // positional: symbol, length
             | chord_named_params

chord_named_params: chord_named_param ("," SP chord_named_param)*
chord_named_param: "symbol" "=" chord_symbol
                 | "chord" "=" chord_symbol
                 | "length" "=" NUMBER
                 | "duration" "=" NUMBER  // alias for length
                 | "repeat" "=" NUMBER
                 | "repetitions" "=" NUMBER  // alias for repeat
                 | "velocity" "=" NUMBER
                 | "voicing" "=" STRING
                 | "inversion" "=" NUMBER

// ---------- Progression operations ----------
progression_call: "progression" "(" progression_params ")"

progression_params: progression_named_params

progression_named_params: progression_named_param ("," SP progression_named_param)*
progression_named_param: "chords" "=" chords_array
                       | "length" "=" NUMBER
                       | "duration" "=" NUMBER  // alias for length
                       | "repeat" "=" NUMBER
                       | "repetitions" "=" NUMBER  // alias for repeat
                       | "pattern" "=" STRING

chords_array: "[" (chord_symbol ("," SP chord_symbol)*)? "]"

// ---------- Choice parameters ----------
choice_params: STRING "," musical_content
             | "description" "=" STRING "," "content" "=" musical_content
             | "description" "=" STRING ("," choice_param)*

choice_param: "content" "=" musical_content
            | "arpeggios" "=" arpeggios_array
            | "chords" "=" chords_array
            | "progressions" "=" progressions_array

musical_content: arpeggios_array | chords_array | progressions_array | mixed_content

arpeggios_array: "[" (arpeggio_item ("," SP arpeggio_item)*)? "]"
arpeggio_item: "arpeggio" "(" arpeggio_params ")"

progressions_array: "[" (progression_item ("," SP progression_item)*)? "]"
progression_item: "progression" "(" progression_params ")"

mixed_content: "[" (musical_item ("," SP musical_item)*)? "]"
musical_item: arpeggio_item | chord_item

chord_item: "chord" "(" chord_params ")"

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

