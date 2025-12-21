package jsfx

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Conceptual-Machines/grammar-school-go/gs"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
)

// JSFXDSLParser parses JSFX DSL code using Grammar School
type JSFXDSLParser struct {
	engine  *gs.Engine
	jsfxDSL *JSFXDSL
	actions []map[string]any
}

// JSFXDSL implements the DSL side-effect methods
type JSFXDSL struct {
	parser *JSFXDSLParser
}

// NewJSFXDSLParser creates a new JSFX DSL parser
func NewJSFXDSLParser() (*JSFXDSLParser, error) {
	parser := &JSFXDSLParser{
		jsfxDSL: &JSFXDSL{},
		actions: make([]map[string]any, 0),
	}

	parser.jsfxDSL.parser = parser

	grammar := llm.GetJSFXDSLGrammar()
	larkParser := gs.NewLarkParser()

	engine, err := gs.NewEngine(grammar, parser.jsfxDSL, larkParser)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	parser.engine = engine
	return parser, nil
}

// ParseDSL parses DSL code and returns actions
func (p *JSFXDSLParser) ParseDSL(dslCode string) ([]map[string]any, error) {
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

	log.Printf("✅ JSFX DSL Parser: Translated %d actions from DSL", len(p.actions))
	return p.actions, nil
}

// Effect handles effect() calls - defines effect metadata
func (d *JSFXDSL) Effect(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "effect",
	}

	if nameValue, ok := args["name"]; ok && nameValue.Kind == gs.ValueString {
		action["name"] = strings.Trim(nameValue.Str, "\"")
	}

	if descValue, ok := args["desc"]; ok && descValue.Kind == gs.ValueString {
		action["desc"] = strings.Trim(descValue.Str, "\"")
	}

	if tagsValue, ok := args["tags"]; ok && tagsValue.Kind == gs.ValueString {
		action["tags"] = strings.Trim(tagsValue.Str, "\"")
	}

	if typeValue, ok := args["type"]; ok && typeValue.Kind == gs.ValueString {
		action["type"] = strings.Trim(typeValue.Str, "\"")
	}

	p.actions = append(p.actions, action)
	log.Printf("🔧 Effect: name=%v, tags=%v", action["name"], action["tags"])

	return nil
}

// Slider handles slider() calls - defines a parameter
func (d *JSFXDSL) Slider(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "slider",
	}

	if idValue, ok := args["id"]; ok && idValue.Kind == gs.ValueNumber {
		action["id"] = idValue.Num
	}

	if defaultValue, ok := args["default"]; ok && defaultValue.Kind == gs.ValueNumber {
		action["default"] = defaultValue.Num
	}

	if minValue, ok := args["min"]; ok && minValue.Kind == gs.ValueNumber {
		action["min"] = minValue.Num
	}

	if maxValue, ok := args["max"]; ok && maxValue.Kind == gs.ValueNumber {
		action["max"] = maxValue.Num
	}

	if stepValue, ok := args["step"]; ok && stepValue.Kind == gs.ValueNumber {
		action["step"] = stepValue.Num
	}

	if nameValue, ok := args["name"]; ok && nameValue.Kind == gs.ValueString {
		action["name"] = strings.Trim(nameValue.Str, "\"")
	}

	if varValue, ok := args["var"]; ok && varValue.Kind == gs.ValueString {
		action["var"] = strings.Trim(varValue.Str, "\"")
	}

	if hiddenValue, ok := args["hidden"]; ok && hiddenValue.Kind == gs.ValueBool {
		action["hidden"] = hiddenValue.Bool
	}

	if optionsValue, ok := args["options"]; ok && optionsValue.Kind == gs.ValueString {
		action["options"] = strings.Trim(optionsValue.Str, "\"")
	}

	p.actions = append(p.actions, action)
	log.Printf("🎚️ Slider: id=%v, name=%v, range=[%v, %v]",
		action["id"], action["name"], action["min"], action["max"])

	return nil
}

// InitCode handles init_code() calls - @init section
func (d *JSFXDSL) InitCode(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "init_code",
	}

	if codeValue, ok := args["code"]; ok && codeValue.Kind == gs.ValueString {
		action["code"] = codeValue.Str
	}

	p.actions = append(p.actions, action)
	log.Printf("📝 Init code: %d chars", len(action["code"].(string)))

	return nil
}

// SliderCode handles slider_code() calls - @slider section
func (d *JSFXDSL) SliderCode(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "slider_code",
	}

	if codeValue, ok := args["code"]; ok && codeValue.Kind == gs.ValueString {
		action["code"] = codeValue.Str
	}

	p.actions = append(p.actions, action)
	log.Printf("📝 Slider code: %d chars", len(action["code"].(string)))

	return nil
}

// SampleCode handles sample_code() calls - @sample section
func (d *JSFXDSL) SampleCode(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "sample_code",
	}

	if codeValue, ok := args["code"]; ok && codeValue.Kind == gs.ValueString {
		action["code"] = codeValue.Str
	}

	p.actions = append(p.actions, action)
	log.Printf("📝 Sample code: %d chars", len(action["code"].(string)))

	return nil
}

// BlockCode handles block_code() calls - @block section
func (d *JSFXDSL) BlockCode(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "block_code",
	}

	if codeValue, ok := args["code"]; ok && codeValue.Kind == gs.ValueString {
		action["code"] = codeValue.Str
	}

	p.actions = append(p.actions, action)
	log.Printf("📝 Block code: %d chars", len(action["code"].(string)))

	return nil
}

// GfxCode handles gfx_code() calls - @gfx section
func (d *JSFXDSL) GfxCode(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "gfx_code",
	}

	if codeValue, ok := args["code"]; ok && codeValue.Kind == gs.ValueString {
		action["code"] = codeValue.Str
	}

	if widthValue, ok := args["width"]; ok && widthValue.Kind == gs.ValueNumber {
		action["width"] = widthValue.Num
	}

	if heightValue, ok := args["height"]; ok && heightValue.Kind == gs.ValueNumber {
		action["height"] = heightValue.Num
	}

	p.actions = append(p.actions, action)
	log.Printf("📝 GFX code: %d chars", len(action["code"].(string)))

	return nil
}

// Pin handles pin() calls - defines audio input/output pins
func (d *JSFXDSL) Pin(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "pin",
	}

	if typeValue, ok := args["type"]; ok && typeValue.Kind == gs.ValueString {
		action["type"] = strings.Trim(typeValue.Str, "\"")
	}

	if nameValue, ok := args["name"]; ok && nameValue.Kind == gs.ValueString {
		action["name"] = strings.Trim(nameValue.Str, "\"")
	}

	if channelValue, ok := args["channel"]; ok && channelValue.Kind == gs.ValueNumber {
		action["channel"] = channelValue.Num
	}

	p.actions = append(p.actions, action)
	log.Printf("📍 Pin: type=%v, name=%v, channel=%v", action["type"], action["name"], action["channel"])

	return nil
}

// Import handles import() calls - includes other JSFX files
func (d *JSFXDSL) Import(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "import",
	}

	if fileValue, ok := args["file"]; ok && fileValue.Kind == gs.ValueString {
		action["file"] = strings.Trim(fileValue.Str, "\"")
	}

	p.actions = append(p.actions, action)
	log.Printf("📦 Import: file=%v", action["file"])

	return nil
}

// Option handles option() calls - sets JSFX options
func (d *JSFXDSL) Option(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "option",
	}

	if nameValue, ok := args["name"]; ok && nameValue.Kind == gs.ValueString {
		action["name"] = strings.Trim(nameValue.Str, "\"")
	}

	if valueValue, ok := args["value"]; ok && valueValue.Kind == gs.ValueString {
		action["value"] = strings.Trim(valueValue.Str, "\"")
	}

	p.actions = append(p.actions, action)
	log.Printf("⚙️ Option: name=%v, value=%v", action["name"], action["value"])

	return nil
}

// Filename handles filename() calls - references external files
func (d *JSFXDSL) Filename(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "filename",
	}

	if idValue, ok := args["id"]; ok && idValue.Kind == gs.ValueNumber {
		action["id"] = idValue.Num
	}

	if pathValue, ok := args["path"]; ok && pathValue.Kind == gs.ValueString {
		action["path"] = strings.Trim(pathValue.Str, "\"")
	}

	if nameValue, ok := args["name"]; ok && nameValue.Kind == gs.ValueString {
		action["name"] = strings.Trim(nameValue.Str, "\"")
	}

	p.actions = append(p.actions, action)
	log.Printf("📁 Filename: id=%v, path=%v", action["id"], action["path"])

	return nil
}

// SerializeCode handles serialize_code() calls - @serialize section
func (d *JSFXDSL) SerializeCode(args gs.Args) error {
	p := d.parser

	action := map[string]any{
		"action": "serialize_code",
	}

	if codeValue, ok := args["code"]; ok && codeValue.Kind == gs.ValueString {
		action["code"] = codeValue.Str
	}

	p.actions = append(p.actions, action)
	log.Printf("💾 Serialize code: %d chars", len(action["code"].(string)))

	return nil
}
