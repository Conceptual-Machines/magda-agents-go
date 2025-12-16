package drummer

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Conceptual-Machines/grammar-school-go/gs"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/Conceptual-Machines/magda-agents-go/models"
)

// Default MIDI note mappings for General MIDI drums
// These can be overridden by user drum mappings
var DefaultDrumMIDINotes = map[string]int{
	"kick":        36,
	"snare":       38,
	"snare_rim":   40,
	"snare_xstick": 37,
	"hat":         42,
	"hat_open":    46,
	"hat_pedal":   44,
	"tom_high":    50,
	"tom_mid":     47,
	"tom_low":     45,
	"crash":       49,
	"ride":        51,
	"ride_bell":   53,
	"china":       52,
	"splash":      55,
	"cowbell":     56,
	"tambourine":  54,
	"clap":        39,
	"snap":        43,
	"shaker":      82,
	"conga_high":  62,
	"conga_low":   63,
	"bongo_high":  60,
	"bongo_low":   61,
}

// DrumPattern represents a single drum track pattern
type DrumPattern struct {
	Drum      string // Canonical drum name (kick, snare, etc.)
	Grid      string // Grid notation string
	Velocity  int    // Base velocity (default 100)
	Humanize  int    // Timing variation 0-100
}

// DrumBeat represents a complete drum beat with multiple patterns
type DrumBeat struct {
	Patterns    []DrumPattern
	Bars        int     // Number of bars
	Tempo       float64 // BPM (optional, for reference)
	Swing       int     // Swing amount 0-100
	Subdivision int     // Grid subdivision (8, 16, 32)
}

// DrummerDSLParser parses Drummer DSL code
type DrummerDSLParser struct {
	engine     *gs.Engine
	drummerDSL *DrummerDSL
	beat       *DrumBeat
	rawDSL     string
}

// DrummerDSL implements the DSL methods
type DrummerDSL struct {
	parser *DrummerDSLParser
}

// NewDrummerDSLParser creates a new drummer DSL parser
func NewDrummerDSLParser() (*DrummerDSLParser, error) {
	parser := &DrummerDSLParser{
		drummerDSL: &DrummerDSL{},
		beat:       &DrumBeat{Subdivision: 16, Bars: 1},
	}

	parser.drummerDSL.parser = parser

	// Get Drummer DSL grammar
	grammar := llm.GetDrummerDSLGrammar()

	// Use generic Lark parser from grammar-school
	larkParser := gs.NewLarkParser()

	// Create Engine with DrummerDSL instance and parser
	engine, err := gs.NewEngine(grammar, parser.drummerDSL, larkParser)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	parser.engine = engine

	return parser, nil
}

// ParseDSL parses DSL code and returns a DrumBeat
func (p *DrummerDSLParser) ParseDSL(dslCode string) (*DrumBeat, error) {
	if dslCode == "" {
		return nil, fmt.Errorf("empty DSL code")
	}

	// Store raw DSL for manual parsing if needed
	p.rawDSL = dslCode

	// Reset beat for new parse
	p.beat = &DrumBeat{Subdivision: 16, Bars: 1}

	// Execute DSL code using Grammar School Engine
	ctx := context.Background()
	if err := p.engine.Execute(ctx, dslCode); err != nil {
		return nil, fmt.Errorf("failed to execute DSL: %w", err)
	}

	if len(p.beat.Patterns) == 0 {
		return nil, fmt.Errorf("no patterns found in DSL code")
	}

	log.Printf("✅ Drummer DSL Parser: Translated %d patterns from DSL", len(p.beat.Patterns))
	return p.beat, nil
}

// ConvertToNoteEvents converts a DrumBeat to MIDI NoteEvents
// drumMapping allows custom MIDI note mappings (can be nil for defaults)
func (p *DrummerDSLParser) ConvertToNoteEvents(beat *DrumBeat, drumMapping map[string]int) ([]models.NoteEvent, error) {
	if beat == nil || len(beat.Patterns) == 0 {
		return nil, fmt.Errorf("empty drum beat")
	}

	// Use provided mapping or defaults
	mapping := DefaultDrumMIDINotes
	if drumMapping != nil {
		mapping = drumMapping
	}

	var allNotes []models.NoteEvent
	subdivision := beat.Subdivision
	if subdivision == 0 {
		subdivision = 16 // Default to 16th notes
	}

	// Calculate duration of one subdivision in beats
	// In 4/4: 1 bar = 4 beats, 16 subdivisions = 0.25 beats each
	subdivisonDuration := 4.0 / float64(subdivision)

	for _, pattern := range beat.Patterns {
		// Get MIDI note for this drum
		midiNote, ok := mapping[pattern.Drum]
		if !ok {
			log.Printf("⚠️ Unknown drum: %s, using default note 36", pattern.Drum)
			midiNote = 36
		}

		// Parse grid string
		grid := pattern.Grid
		baseVelocity := pattern.Velocity
		if baseVelocity == 0 {
			baseVelocity = 100
		}

		// Process each character in the grid
		for i, char := range grid {
			var velocity int
			switch char {
			case 'x': // Normal hit
				velocity = baseVelocity
			case 'X': // Accent
				velocity = 127
			case 'o': // Ghost note
				velocity = 60
			case '-', ' ': // Rest
				continue
			default:
				// Skip unknown characters
				continue
			}

			// Calculate start time
			startBeat := float64(i) * subdivisonDuration

			// Apply swing if set (affects even-numbered subdivisions)
			if beat.Swing > 0 && i%2 == 1 {
				swingOffset := (float64(beat.Swing) / 100.0) * subdivisonDuration * 0.5
				startBeat += swingOffset
			}

			note := models.NoteEvent{
				MidiNoteNumber: midiNote,
				Velocity:       velocity,
				StartBeats:     startBeat,
				DurationBeats:  subdivisonDuration * 0.9, // Slightly shorter for separation
			}
			allNotes = append(allNotes, note)
		}
	}

	return allNotes, nil
}

// ========== DSL Methods (DrummerDSL) ==========

// Pattern handles pattern() calls
func (d *DrummerDSL) Pattern(args gs.Args) error {
	p := d.parser

	// Extract drum name
	drumName := ""
	if drumValue, ok := args["drum"]; ok && drumValue.Kind == gs.ValueString {
		drumName = drumValue.Str
	}
	if drumName == "" {
		return fmt.Errorf("pattern: missing drum name")
	}

	// Extract grid
	grid := ""
	if gridValue, ok := args["grid"]; ok && gridValue.Kind == gs.ValueString {
		grid = strings.Trim(gridValue.Str, "\"")
	}
	if grid == "" {
		return fmt.Errorf("pattern: missing grid")
	}

	// Extract optional velocity
	velocity := 100
	if velValue, ok := args["velocity"]; ok && velValue.Kind == gs.ValueNumber {
		velocity = int(velValue.Num)
	}

	// Extract optional humanize
	humanize := 0
	if humValue, ok := args["humanize"]; ok && humValue.Kind == gs.ValueNumber {
		humanize = int(humValue.Num)
	}

	pattern := DrumPattern{
		Drum:     drumName,
		Grid:     grid,
		Velocity: velocity,
		Humanize: humanize,
	}

	p.beat.Patterns = append(p.beat.Patterns, pattern)
	log.Printf("🥁 Pattern: drum=%s, grid=%s (%d hits)", drumName, grid, countHits(grid))

	return nil
}

// Beat handles beat() calls
func (d *DrummerDSL) Beat(args gs.Args) error {
	p := d.parser

	// Extract bars
	if barsValue, ok := args["bars"]; ok && barsValue.Kind == gs.ValueNumber {
		p.beat.Bars = int(barsValue.Num)
	}

	// Extract tempo
	if tempoValue, ok := args["tempo"]; ok && tempoValue.Kind == gs.ValueNumber {
		p.beat.Tempo = tempoValue.Num
	}

	// Extract swing
	if swingValue, ok := args["swing"]; ok && swingValue.Kind == gs.ValueNumber {
		p.beat.Swing = int(swingValue.Num)
	}

	// Extract subdivision
	if subValue, ok := args["subdivision"]; ok && subValue.Kind == gs.ValueNumber {
		p.beat.Subdivision = int(subValue.Num)
	}

	// Patterns are handled by nested pattern() calls via patterns_array
	// Grammar School will call Pattern() for each pattern in the array

	log.Printf("🥁 Beat: bars=%d, tempo=%.0f, swing=%d, subdivision=%d",
		p.beat.Bars, p.beat.Tempo, p.beat.Swing, p.beat.Subdivision)

	return nil
}

// countHits counts the number of hits in a grid string
func countHits(grid string) int {
	count := 0
	for _, c := range grid {
		if c == 'x' || c == 'X' || c == 'o' {
			count++
		}
	}
	return count
}

