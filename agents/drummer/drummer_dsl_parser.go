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

	grammar := llm.GetDrummerDSLGrammar()
	larkParser := gs.NewLarkParser()

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

	p.actions = make([]map[string]any, 0)

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

// Pattern handles pattern() calls - creates a drum_pattern action
func (d *DrummerDSL) Pattern(args gs.Args) error {
	p := d.parser

	drumName := ""
	if drumValue, ok := args["drum"]; ok && drumValue.Kind == gs.ValueString {
		drumName = drumValue.Str
	}
	if drumName == "" {
		return fmt.Errorf("pattern: missing drum name")
	}

	grid := ""
	if gridValue, ok := args["grid"]; ok && gridValue.Kind == gs.ValueString {
		grid = strings.Trim(gridValue.Str, "\"")
	}
	if grid == "" {
		return fmt.Errorf("pattern: missing grid")
	}

	velocity := 100
	if velValue, ok := args["velocity"]; ok && velValue.Kind == gs.ValueNumber {
		velocity = int(velValue.Num)
	}

	action := map[string]any{
		"type":     "drum_pattern",
		"drum":     drumName,
		"grid":     grid,
		"velocity": velocity,
	}

	p.actions = append(p.actions, action)
	log.Printf("🥁 Pattern: drum=%s, grid=%s (%d hits)", drumName, grid, countHits(grid))

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
