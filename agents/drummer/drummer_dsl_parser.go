package drummer

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Conceptual-Machines/grammar-school-go/gs"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
)

// DrummerDSLParser parses Drummer DSL code using Grammar School
type DrummerDSLParser struct {
	engine     *gs.Engine
	drummerDSL *DrummerDSL
	actions    []map[string]any
	rawDSL     string
}

// DrummerDSL implements the DSL side-effect methods
type DrummerDSL struct {
	parser *DrummerDSLParser
}

// NewDrummerDSLParser creates a new drummer DSL parser
func NewDrummerDSLParser() (*DrummerDSLParser, error) {
	parser := &DrummerDSLParser{
		drummerDSL: &DrummerDSL{},
		actions:    make([]map[string]any, 0),
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

// ParseDSL parses DSL code and returns actions
func (p *DrummerDSLParser) ParseDSL(dslCode string) ([]map[string]any, error) {
	if dslCode == "" {
		return nil, fmt.Errorf("empty DSL code")
	}

	// Store raw DSL for manual parsing if needed
	p.rawDSL = dslCode

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

	log.Printf("✅ Drummer DSL Parser: Translated %d actions from DSL", len(p.actions))
	return p.actions, nil
}

// ========== DSL Side-Effect Methods (DrummerDSL) ==========

// Pattern handles pattern() calls - creates a drum_pattern action
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

	// Create action map (like DAW agent does)
	action := map[string]any{
		"type":     "drum_pattern",
		"drum":     drumName,
		"grid":     grid,
		"velocity": velocity,
		"humanize": humanize,
	}

	p.actions = append(p.actions, action)
	log.Printf("🥁 Pattern: drum=%s, grid=%s (%d hits)", drumName, grid, countHits(grid))

	return nil
}

// Beat handles beat() calls - creates a drum_beat action with metadata
func (d *DrummerDSL) Beat(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"type":        "drum_beat",
		"bars":        1,
		"subdivision": 16,
	}

	// Extract bars
	if barsValue, ok := args["bars"]; ok && barsValue.Kind == gs.ValueNumber {
		action["bars"] = int(barsValue.Num)
	}

	// Extract tempo
	if tempoValue, ok := args["tempo"]; ok && tempoValue.Kind == gs.ValueNumber {
		action["tempo"] = tempoValue.Num
	}

	// Extract swing
	if swingValue, ok := args["swing"]; ok && swingValue.Kind == gs.ValueNumber {
		action["swing"] = int(swingValue.Num)
	}

	// Extract subdivision
	if subValue, ok := args["subdivision"]; ok && subValue.Kind == gs.ValueNumber {
		action["subdivision"] = int(subValue.Num)
	}

	p.actions = append(p.actions, action)
	log.Printf("🥁 Beat: bars=%v, subdivision=%v", action["bars"], action["subdivision"])

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
