package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Conceptual-Machines/grammar-school-go/gs"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
)

// ArrangerDSLParser parses Arranger DSL code with chord symbols.
// Uses Grammar School Engine for parsing.
type ArrangerDSLParser struct {
	engine    *gs.Engine
	arrangerDSL *ArrangerDSL
	actions   []map[string]any
}

// ArrangerDSL implements the DSL methods for musical composition.
type ArrangerDSL struct {
	parser *ArrangerDSLParser
}

// NewArrangerDSLParser creates a new arranger DSL parser.
func NewArrangerDSLParser() (*ArrangerDSLParser, error) {
	parser := &ArrangerDSLParser{
		arrangerDSL: &ArrangerDSL{},
		actions:     make([]map[string]any, 0),
	}

	parser.arrangerDSL.parser = parser

	// Get Arranger DSL grammar
	grammar := llm.GetArrangerDSLGrammar()

	// Use generic Lark parser from grammar-school
	larkParser := gs.NewLarkParser()

	// Create Engine with ArrangerDSL instance and parser
	engine, err := gs.NewEngine(grammar, parser.arrangerDSL, larkParser)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	parser.engine = engine

	return parser, nil
}

// ParseDSL parses DSL code and returns arranger actions.
func (p *ArrangerDSLParser) ParseDSL(dslCode string) ([]map[string]any, error) {
	if dslCode == "" {
		return nil, fmt.Errorf("empty DSL code")
	}

	// Reset actions for new parse
	p.actions = make([]map[string]any, 0)

	// Execute DSL code using Grammar School Engine
	ctx := context.Background()
	if err := p.engine.Execute(ctx, dslCode); err != nil {
		return nil, fmt.Errorf("failed to execute DSL: %w", err)
	}

	if len(p.actions) == 0 {
		return nil, fmt.Errorf("no actions found in DSL code")
	}

	log.Printf("✅ Arranger DSL Parser: Translated %d actions from DSL", len(p.actions))
	return p.actions, nil
}

// ========== Side-effect methods (ArrangerDSL) ==========

// Arpeggio handles arpeggio() calls.
// Example: arpeggio("Em", length=2, repeat=4)
func (a *ArrangerDSL) Arpeggio(args gs.Args) error {
	p := a.parser

	// Extract chord symbol
	chordSymbol := ""
	if symbolValue, ok := args["symbol"]; ok && symbolValue.Kind == gs.ValueString {
		chordSymbol = symbolValue.Str
	} else if chordValue, ok := args["chord"]; ok && chordValue.Kind == gs.ValueString {
		chordSymbol = chordValue.Str
	} else if posValue, ok := args[""]; ok && posValue.Kind == gs.ValueString {
		// First positional arg (empty key)
		chordSymbol = posValue.Str
	} else {
		// Last resort: find first string value
		for _, v := range args {
			if v.Kind == gs.ValueString {
				chordSymbol = v.Str
				break
			}
		}
	}

	if chordSymbol == "" {
		return fmt.Errorf("arpeggio: missing chord symbol")
	}

	// Parse bass note from chord symbol for arpeggios too (e.g., "Emin/G")
	bassNote := ""
	if strings.Contains(chordSymbol, "/") {
		parts := strings.Split(chordSymbol, "/")
		if len(parts) == 2 {
			bassNote = strings.TrimSpace(parts[1])
			// Keep the base chord without the bass note in the chord field
			chordSymbol = strings.TrimSpace(parts[0])
		}
	}

	// Extract length (default: 4 beats = 1 bar)
	length := 4.0
	if lengthValue, ok := args["length"]; ok && lengthValue.Kind == gs.ValueNumber {
		length = lengthValue.Num
	} else if durationValue, ok := args["duration"]; ok && durationValue.Kind == gs.ValueNumber {
		length = durationValue.Num
	} else if len(args) > 1 {
		// Second positional arg might be length
		count := 0
		for _, v := range args {
			if v.Kind == gs.ValueString && count == 0 {
				count++
				continue
			}
			if v.Kind == gs.ValueNumber && count == 1 {
				length = v.Num
				break
			}
		}
	}

	// Extract repeat (default: 1)
	repeat := 1
	if repeatValue, ok := args["repeat"]; ok && repeatValue.Kind == gs.ValueNumber {
		repeat = int(repeatValue.Num)
	} else if repetitionsValue, ok := args["repetitions"]; ok && repetitionsValue.Kind == gs.ValueNumber {
		repeat = int(repetitionsValue.Num)
	}

	// Extract optional parameters
	velocity := 100
	if velocityValue, ok := args["velocity"]; ok && velocityValue.Kind == gs.ValueNumber {
		velocity = int(velocityValue.Num)
	}

	octave := 4
	if octaveValue, ok := args["octave"]; ok && octaveValue.Kind == gs.ValueNumber {
		octave = int(octaveValue.Num)
	}

	direction := "up"
	if directionValue, ok := args["direction"]; ok && directionValue.Kind == gs.ValueString {
		direction = directionValue.Str
	}

	pattern := ""
	if patternValue, ok := args["pattern"]; ok && patternValue.Kind == gs.ValueString {
		pattern = patternValue.Str
	}

	// Create action
	action := map[string]any{
		"type":        "arpeggio",
		"chord":       chordSymbol,
		"length":      length,
		"repeat":      repeat,
		"velocity":    velocity,
		"octave":      octave,
		"direction":   direction,
	}
	if pattern != "" {
		action["pattern"] = pattern
	}
	if bassNote != "" {
		action["bass"] = bassNote
	}

	p.actions = append(p.actions, action)
	return nil
}

// Chord handles chord() calls.
// Example: chord("C", length=1, repeat=4)
func (a *ArrangerDSL) Chord(args gs.Args) error {
	p := a.parser

	// Extract chord symbol
	chordSymbol := ""
	if symbolValue, ok := args["symbol"]; ok && symbolValue.Kind == gs.ValueString {
		chordSymbol = symbolValue.Str
	} else if chordValue, ok := args["chord"]; ok && chordValue.Kind == gs.ValueString {
		chordSymbol = chordValue.Str
	} else if posValue, ok := args[""]; ok && posValue.Kind == gs.ValueString {
		// First positional arg (empty key)
		chordSymbol = posValue.Str
	} else {
		// Last resort: find first string value
		for _, v := range args {
			if v.Kind == gs.ValueString {
				chordSymbol = v.Str
				break
			}
		}
	}

	if chordSymbol == "" {
		return fmt.Errorf("chord: missing chord symbol")
	}

	// Extract length (default: 4 beats = 1 bar)
	length := 4.0
	if lengthValue, ok := args["length"]; ok && lengthValue.Kind == gs.ValueNumber {
		length = lengthValue.Num
	} else if durationValue, ok := args["duration"]; ok && durationValue.Kind == gs.ValueNumber {
		length = durationValue.Num
	} else {
		// Check for positional number args (after the string arg)
		// Grammar School may pass positional args in order
		positionalCount := 0
		for _, v := range args {
			if v.Kind == gs.ValueString && v.Str == chordSymbol {
				positionalCount++
			} else if v.Kind == gs.ValueNumber && positionalCount > 0 {
				length = v.Num
				break
			}
		}
	}

	// Extract repeat (default: 1)
	repeat := 1
	if repeatValue, ok := args["repeat"]; ok && repeatValue.Kind == gs.ValueNumber {
		repeat = int(repeatValue.Num)
	} else if repetitionsValue, ok := args["repetitions"]; ok && repetitionsValue.Kind == gs.ValueNumber {
		repeat = int(repetitionsValue.Num)
	}

	// Extract optional parameters
	velocity := 100
	if velocityValue, ok := args["velocity"]; ok && velocityValue.Kind == gs.ValueNumber {
		velocity = int(velocityValue.Num)
	}

	voicing := ""
	if voicingValue, ok := args["voicing"]; ok && voicingValue.Kind == gs.ValueString {
		voicing = voicingValue.Str
	}

	inversion := 0
	if inversionValue, ok := args["inversion"]; ok && inversionValue.Kind == gs.ValueNumber {
		inversion = int(inversionValue.Num)
	}

	// Parse bass note from chord symbol (e.g., "Emin/G" -> bass note is "G")
	bassNote := ""
	if strings.Contains(chordSymbol, "/") {
		parts := strings.Split(chordSymbol, "/")
		if len(parts) == 2 {
			bassNote = strings.TrimSpace(parts[1])
			// Keep the base chord without the bass note in the chord field
			chordSymbol = strings.TrimSpace(parts[0])
		}
	}

	// Create action
	action := map[string]any{
		"type":     "chord",
		"chord":    chordSymbol,
		"length":   length,
		"repeat":   repeat,
		"velocity": velocity,
	}
	if voicing != "" {
		action["voicing"] = voicing
	}
	if inversion != 0 {
		action["inversion"] = inversion
	}
	if bassNote != "" {
		action["bass"] = bassNote
	}

	p.actions = append(p.actions, action)
	return nil
}

// Progression handles progression() calls.
// Example: progression(chords=["C", "Am", "F", "G"], length=4, repeat=2)
func (a *ArrangerDSL) Progression(args gs.Args) error {
	p := a.parser

	// DEBUG: Log all args to see what Grammar School passes
	log.Printf("🎵 Progression called with args: %+v", args)
	for k, v := range args {
		log.Printf("🎵   arg[%q] = Kind:%d, Str:%q, Num:%f", k, v.Kind, v.Str, v.Num)
	}

	// Extract chords array
	// Grammar School may pass arrays as strings that need parsing
	chords := []string{}
	if chordsValue, ok := args["chords"]; ok {
		log.Printf("🎵 Found 'chords' arg: Kind=%d, Str=%q", chordsValue.Kind, chordsValue.Str)
		if chordsValue.Kind == gs.ValueString {
			// Parse string representation like ["C", "Am", "F", "G"] or [C, Am, F, G]
			chordsStr := strings.TrimSpace(chordsValue.Str)
			chordsStr = strings.TrimPrefix(chordsStr, "[")
			chordsStr = strings.TrimSuffix(chordsStr, "]")
			if chordsStr != "" {
				// Split by comma and clean up quotes/spaces
				parts := strings.Split(chordsStr, ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					// Remove quotes if present, but keep the chord symbol
					part = strings.Trim(part, "\"'")
					if part != "" {
						chords = append(chords, part)
					}
				}
			}
		}
	}

	if len(chords) == 0 {
		return fmt.Errorf("progression: missing chords array")
	}

	// Extract length (default: number of chords * 4 beats = 1 bar per chord)
	length := float64(len(chords)) * 4.0
	if lengthValue, ok := args["length"]; ok && lengthValue.Kind == gs.ValueNumber {
		length = lengthValue.Num
	} else if durationValue, ok := args["duration"]; ok && durationValue.Kind == gs.ValueNumber {
		length = durationValue.Num
	}

	// Extract repeat (default: 1)
	repeat := 1
	if repeatValue, ok := args["repeat"]; ok && repeatValue.Kind == gs.ValueNumber {
		repeat = int(repeatValue.Num)
	} else if repetitionsValue, ok := args["repetitions"]; ok && repetitionsValue.Kind == gs.ValueNumber {
		repeat = int(repetitionsValue.Num)
	}

	pattern := ""
	if patternValue, ok := args["pattern"]; ok && patternValue.Kind == gs.ValueString {
		pattern = patternValue.Str
	}

	// Create action
	action := map[string]any{
		"type":    "progression",
		"chords":  chords,
		"length":  length,
		"repeat":  repeat,
	}
	if pattern != "" {
		action["pattern"] = pattern
	}

	p.actions = append(p.actions, action)
	return nil
}

// Composition handles composition() calls with chaining.
// Example: composition().add_arpeggio("Em", length=2).add_chord("C", length=1)
func (a *ArrangerDSL) Composition(args gs.Args) error {
	// Composition() itself doesn't create an action, it's just a container
	// The chain items will create actions
	return nil
}

// AddArpeggio handles .add_arpeggio() chain calls.
func (a *ArrangerDSL) AddArpeggio(args gs.Args) error {
	return a.Arpeggio(args)
}

// AddChord handles .add_chord() chain calls.
func (a *ArrangerDSL) AddChord(args gs.Args) error {
	return a.Chord(args)
}

// AddProgression handles .add_progression() chain calls.
func (a *ArrangerDSL) AddProgression(args gs.Args) error {
	return a.Progression(args)
}

// Choice handles choice() calls (single choice format).
// Example: choice("E minor arpeggio", [arpeggio("Em", length=2)])
func (a *ArrangerDSL) Choice(args gs.Args) error {
	// Extract description
	description := ""
	if descValue, ok := args["description"]; ok && descValue.Kind == gs.ValueString {
		description = descValue.Str
	} else if len(args) > 0 {
		// First positional arg might be description
		for _, v := range args {
			if v.Kind == gs.ValueString {
				description = v.Str
				break
			}
		}
	}

	// Extract content (arpeggios, chords, or progressions)
	// The content will be parsed as separate statements, so we just mark this as a choice
	if description != "" {
		// Store description for later use (could be used in choice metadata)
		log.Printf("📝 Choice description: %s", description)
	}

	// Content items will be parsed as separate statements
	return nil
}

