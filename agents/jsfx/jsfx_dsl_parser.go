package jsfx

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Conceptual-Machines/grammar-school-go/gs"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
)

// JSFXDSLParser parses JSFX DSL code using Grammar School and builds JSFX directly
type JSFXDSLParser struct {
	engine  *gs.Engine
	jsfxDSL *JSFXDSL
}

// JSFXDSL builds JSFX code directly as DSL methods are called
type JSFXDSL struct {
	// Header
	effectName string
	effectTags string

	// Pins
	inPins  []string
	outPins []string

	// Imports and options
	imports   []string
	options   []string // "name" or "name=value"
	filenames []string // "id,path"

	// Sliders
	sliders []string // formatted slider lines

	// Code sections
	initCode      string
	sliderCode    string
	blockCode     string
	sampleCode    string
	serializeCode string
	gfxCode       string
	gfxWidth      int
	gfxHeight     int
}

// NewJSFXDSLParser creates a new JSFX DSL parser
func NewJSFXDSLParser() (*JSFXDSLParser, error) {
	jsfxDSL := &JSFXDSL{
		effectName: "AI Generated Effect",
		effectTags: "utility",
		inPins:     make([]string, 0),
		outPins:    make([]string, 0),
		imports:    make([]string, 0),
		options:    make([]string, 0),
		filenames:  make([]string, 0),
		sliders:    make([]string, 0),
	}

	parser := &JSFXDSLParser{
		jsfxDSL: jsfxDSL,
	}

	grammar := llm.GetJSFXDSLGrammar()
	larkParser := gs.NewLarkParser()

	engine, err := gs.NewEngine(grammar, jsfxDSL, larkParser)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	parser.engine = engine
	return parser, nil
}

// ParseDSL parses DSL code and returns the generated JSFX code
func (p *JSFXDSLParser) ParseDSL(dslCode string) (string, error) {
	if dslCode == "" {
		return "", fmt.Errorf("empty DSL code")
	}

	// Reset state for new parse
	p.jsfxDSL.reset()

	ctx := context.Background()
	if err := p.engine.Execute(ctx, dslCode); err != nil {
		return "", fmt.Errorf("failed to execute DSL: %w", err)
	}

	// Build and return JSFX code
	jsfxCode := p.jsfxDSL.buildJSFX()
	log.Printf("✅ JSFX DSL Parser: Generated %d bytes of JSFX code", len(jsfxCode))

	return jsfxCode, nil
}

// reset clears the state for a new parse
func (d *JSFXDSL) reset() {
	d.effectName = "AI Generated Effect"
	d.effectTags = "utility"
	d.inPins = make([]string, 0)
	d.outPins = make([]string, 0)
	d.imports = make([]string, 0)
	d.options = make([]string, 0)
	d.filenames = make([]string, 0)
	d.sliders = make([]string, 0)
	d.initCode = ""
	d.sliderCode = ""
	d.blockCode = ""
	d.sampleCode = ""
	d.serializeCode = ""
	d.gfxCode = ""
	d.gfxWidth = 0
	d.gfxHeight = 0
}

// buildJSFX assembles the final JSFX code from collected components
func (d *JSFXDSL) buildJSFX() string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("desc:%s\n", d.effectName))
	sb.WriteString(fmt.Sprintf("tags:%s\n", d.effectTags))

	// Imports
	for _, imp := range d.imports {
		sb.WriteString(fmt.Sprintf("import %s\n", imp))
	}

	// Options
	for _, opt := range d.options {
		sb.WriteString(fmt.Sprintf("options:%s\n", opt))
	}

	// Filenames
	for _, fn := range d.filenames {
		sb.WriteString(fmt.Sprintf("filename:%s\n", fn))
	}

	// Input pins
	for _, pin := range d.inPins {
		sb.WriteString(fmt.Sprintf("in_pin:%s\n", pin))
	}

	// Output pins
	for _, pin := range d.outPins {
		sb.WriteString(fmt.Sprintf("out_pin:%s\n", pin))
	}

	sb.WriteString("\n")

	// Sliders
	for _, slider := range d.sliders {
		sb.WriteString(slider + "\n")
	}

	// @init section
	if d.initCode != "" {
		sb.WriteString("\n@init\n")
		sb.WriteString(unescapeCode(d.initCode))
		sb.WriteString("\n")
	}

	// @slider section
	if d.sliderCode != "" {
		sb.WriteString("\n@slider\n")
		sb.WriteString(unescapeCode(d.sliderCode))
		sb.WriteString("\n")
	}

	// @block section
	if d.blockCode != "" {
		sb.WriteString("\n@block\n")
		sb.WriteString(unescapeCode(d.blockCode))
		sb.WriteString("\n")
	}

	// @sample section
	if d.sampleCode != "" {
		sb.WriteString("\n@sample\n")
		sb.WriteString(unescapeCode(d.sampleCode))
		sb.WriteString("\n")
	}

	// @serialize section
	if d.serializeCode != "" {
		sb.WriteString("\n@serialize\n")
		sb.WriteString(unescapeCode(d.serializeCode))
		sb.WriteString("\n")
	}

	// @gfx section
	if d.gfxCode != "" {
		if d.gfxWidth > 0 && d.gfxHeight > 0 {
			sb.WriteString(fmt.Sprintf("\n@gfx %d %d\n", d.gfxWidth, d.gfxHeight))
		} else {
			sb.WriteString("\n@gfx\n")
		}
		sb.WriteString(unescapeCode(d.gfxCode))
		sb.WriteString("\n")
	}

	return sb.String()
}

// unescapeCode converts escaped strings back to normal code
func unescapeCode(code string) string {
	code = strings.Trim(code, "\"")
	code = strings.ReplaceAll(code, "\\n", "\n")
	code = strings.ReplaceAll(code, "\\t", "\t")
	code = strings.ReplaceAll(code, "\\\"", "\"")
	return code
}

// trimQuotes removes surrounding quotes from a string
func trimQuotes(s string) string {
	return strings.Trim(s, "\"")
}

// === Grammar School DSL Methods ===
// These are called by Grammar School as it parses the DSL

// Effect handles effect() calls - defines effect metadata
func (d *JSFXDSL) Effect(args gs.Args) error {
	if nameValue, ok := args["name"]; ok && nameValue.Kind == gs.ValueString {
		d.effectName = trimQuotes(nameValue.Str)
	}
	if descValue, ok := args["desc"]; ok && descValue.Kind == gs.ValueString {
		d.effectName = trimQuotes(descValue.Str) // desc overrides name
	}
	if tagsValue, ok := args["tags"]; ok && tagsValue.Kind == gs.ValueString {
		d.effectTags = trimQuotes(tagsValue.Str)
	}
	log.Printf("🔧 Effect: name=%s, tags=%s", d.effectName, d.effectTags)
	return nil
}

// Slider handles slider() calls - defines a parameter
func (d *JSFXDSL) Slider(args gs.Args) error {
	id := 1
	defaultVal := 0.0
	minVal := 0.0
	maxVal := 1.0
	step := 1.0
	name := "Parameter"
	varName := ""
	hidden := false

	if v, ok := args["id"]; ok && v.Kind == gs.ValueNumber {
		id = int(v.Num)
	}
	if v, ok := args["default"]; ok && v.Kind == gs.ValueNumber {
		defaultVal = v.Num
	}
	if v, ok := args["min"]; ok && v.Kind == gs.ValueNumber {
		minVal = v.Num
	}
	if v, ok := args["max"]; ok && v.Kind == gs.ValueNumber {
		maxVal = v.Num
	}
	if v, ok := args["step"]; ok && v.Kind == gs.ValueNumber {
		step = v.Num
	}
	if v, ok := args["name"]; ok && v.Kind == gs.ValueString {
		name = trimQuotes(v.Str)
	}
	if v, ok := args["var"]; ok && v.Kind == gs.ValueString {
		varName = trimQuotes(v.Str)
	}
	if v, ok := args["hidden"]; ok && v.Kind == gs.ValueBool {
		hidden = v.Bool
	}

	// Build slider line: slider1:var=0.00<-60.00,12.00,0.1000>-Name
	hiddenPrefix := ""
	if hidden {
		hiddenPrefix = "-"
	}
	varPrefix := ""
	if varName != "" {
		varPrefix = varName + "="
	}

	sliderLine := fmt.Sprintf("slider%d:%s%.2f<%.2f,%.2f,%.4f>%s%s",
		id, varPrefix, defaultVal, minVal, maxVal, step, hiddenPrefix, name)
	d.sliders = append(d.sliders, sliderLine)

	log.Printf("🎚️ Slider: id=%d, name=%s, range=[%.2f, %.2f]", id, name, minVal, maxVal)
	return nil
}

// Pin handles pin() calls - defines audio input/output pins
func (d *JSFXDSL) Pin(args gs.Args) error {
	pinType := ""
	pinName := ""

	if v, ok := args["type"]; ok && v.Kind == gs.ValueString {
		pinType = trimQuotes(v.Str)
	}
	if v, ok := args["name"]; ok && v.Kind == gs.ValueString {
		pinName = trimQuotes(v.Str)
	}

	if pinType == "in" {
		d.inPins = append(d.inPins, pinName)
	} else if pinType == "out" {
		d.outPins = append(d.outPins, pinName)
	}

	log.Printf("📍 Pin: type=%s, name=%s", pinType, pinName)
	return nil
}

// Import handles import() calls - includes other JSFX files
func (d *JSFXDSL) Import(args gs.Args) error {
	if v, ok := args["file"]; ok && v.Kind == gs.ValueString {
		d.imports = append(d.imports, trimQuotes(v.Str))
		log.Printf("📦 Import: %s", trimQuotes(v.Str))
	}
	return nil
}

// Option handles option() calls - sets JSFX options
func (d *JSFXDSL) Option(args gs.Args) error {
	name := ""
	value := ""

	if v, ok := args["name"]; ok && v.Kind == gs.ValueString {
		name = trimQuotes(v.Str)
	}
	if v, ok := args["value"]; ok && v.Kind == gs.ValueString {
		value = trimQuotes(v.Str)
	}

	if value != "" {
		d.options = append(d.options, fmt.Sprintf("%s=%s", name, value))
	} else {
		d.options = append(d.options, name)
	}

	log.Printf("⚙️ Option: %s", name)
	return nil
}

// Filename handles filename() calls - references external files
func (d *JSFXDSL) Filename(args gs.Args) error {
	id := 0
	path := ""

	if v, ok := args["id"]; ok && v.Kind == gs.ValueNumber {
		id = int(v.Num)
	}
	if v, ok := args["path"]; ok && v.Kind == gs.ValueString {
		path = trimQuotes(v.Str)
	}

	d.filenames = append(d.filenames, fmt.Sprintf("%d,%s", id, path))
	log.Printf("📁 Filename: id=%d, path=%s", id, path)
	return nil
}

// InitCode handles init_code() calls - @init section
func (d *JSFXDSL) InitCode(args gs.Args) error {
	if v, ok := args["code"]; ok && v.Kind == gs.ValueString {
		d.initCode = v.Str
		log.Printf("📝 Init code: %d chars", len(v.Str))
	}
	return nil
}

// SliderCode handles slider_code() calls - @slider section
func (d *JSFXDSL) SliderCode(args gs.Args) error {
	if v, ok := args["code"]; ok && v.Kind == gs.ValueString {
		d.sliderCode = v.Str
		log.Printf("📝 Slider code: %d chars", len(v.Str))
	}
	return nil
}

// BlockCode handles block_code() calls - @block section
func (d *JSFXDSL) BlockCode(args gs.Args) error {
	if v, ok := args["code"]; ok && v.Kind == gs.ValueString {
		d.blockCode = v.Str
		log.Printf("📝 Block code: %d chars", len(v.Str))
	}
	return nil
}

// SampleCode handles sample_code() calls - @sample section
func (d *JSFXDSL) SampleCode(args gs.Args) error {
	if v, ok := args["code"]; ok && v.Kind == gs.ValueString {
		d.sampleCode = v.Str
		log.Printf("📝 Sample code: %d chars", len(v.Str))
	}
	return nil
}

// SerializeCode handles serialize_code() calls - @serialize section
func (d *JSFXDSL) SerializeCode(args gs.Args) error {
	if v, ok := args["code"]; ok && v.Kind == gs.ValueString {
		d.serializeCode = v.Str
		log.Printf("💾 Serialize code: %d chars", len(v.Str))
	}
	return nil
}

// GfxCode handles gfx_code() calls - @gfx section
func (d *JSFXDSL) GfxCode(args gs.Args) error {
	if v, ok := args["code"]; ok && v.Kind == gs.ValueString {
		d.gfxCode = v.Str
		log.Printf("📝 GFX code: %d chars", len(v.Str))
	}
	if v, ok := args["width"]; ok && v.Kind == gs.ValueNumber {
		d.gfxWidth = int(v.Num)
	}
	if v, ok := args["height"]; ok && v.Kind == gs.ValueNumber {
		d.gfxHeight = int(v.Num)
	}
	return nil
}
