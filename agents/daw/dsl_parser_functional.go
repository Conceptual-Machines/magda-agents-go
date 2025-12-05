package daw

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Conceptual-Machines/grammar-school-go/gs"
)

// FunctionalDSLParser parses MAGDA DSL code with functional method support.
// Uses Grammar School Engine for parsing and supports filter, map, etc.
type FunctionalDSLParser struct {
	engine            *gs.Engine
	reaperDSL         *ReaperDSL
	currentTrackIndex int
	trackCounter      int
	state             map[string]interface{}
	data              map[string]interface{} // Storage for collections
	iterationContext  map[string]interface{} // Current iteration variables (track, fx, clip, etc.)
	actions           []map[string]interface{}
}

// ReaperDSL implements the DSL methods for REAPER operations.
type ReaperDSL struct {
	parser *FunctionalDSLParser
}

// magdaParser implements gs.Parser interface for MAGDA DSL
type magdaParser struct {
	parent *FunctionalDSLParser
}

// Parse implements gs.Parser interface
func (p *magdaParser) Parse(input string) (*gs.CallChain, error) {
	// Simple parser implementation that converts DSL to CallChain
	// This is a workaround for grammar-school-go requiring a parser
	// TODO: Implement proper grammar-based parsing when grammar-school-go supports it

	// For now, we'll use the existing ParseDSL logic to extract actions
	// and convert them to CallChain format
	// This is a temporary solution until grammar-school-go is fixed

	// Since we're using the engine.Execute to parse, we need a basic parser
	// that just returns an empty CallChain and let the engine handle it
	// Actually, we can't do that because Execute needs the CallChain...

	// Let's create a simple parser that extracts method calls
	return p.parseSimpleDSL(input)
}

// parseSimpleDSL parses simple DSL like track(...).method(...)
func (p *magdaParser) parseSimpleDSL(input string) (*gs.CallChain, error) {
	// This is a minimal implementation - grammar-school-go was supposed to handle nil parser
	// but recent changes broke it. This is a workaround.

	// Parse the input manually into a CallChain
	chain := &gs.CallChain{Calls: []gs.Call{}}

	// Simple regex-based parsing for now
	// Split by dots to get method calls
	parts := splitMethodCalls(input)

	for _, part := range parts {
		call := parseMethodCall(part)
		if call != nil {
			chain.Calls = append(chain.Calls, *call)
		}
	}

	return chain, nil
}

// splitMethodCalls splits "track(...).method(...)" into ["track(...)", "method(...)"]
func splitMethodCalls(input string) []string {
	var parts []string
	var current strings.Builder
	depth := 0

	for _, r := range input {
		char := string(r)

		if char == "(" {
			depth++
			current.WriteRune(r)
		} else if char == ")" {
			depth--
			current.WriteRune(r)
		} else if char == "." && depth == 0 {
			if current.Len() > 0 {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, strings.TrimSpace(current.String()))
	}

	return parts
}

// parseMethodCall parses "method(param=value)" into a Call
func parseMethodCall(input string) *gs.Call {
	input = strings.TrimSpace(input)

	// Find method name and params
	parenIndex := strings.Index(input, "(")
	if parenIndex == -1 {
		// No params
		methodName := strings.TrimSpace(input)
		// Capitalize first letter for Go method names (track -> Track, set_selected -> SetSelected)
		methodName = capitalizeMethodName(methodName)
		return &gs.Call{
			Name: methodName,
			Args: []gs.Arg{},
		}
	}

	methodName := strings.TrimSpace(input[:parenIndex])
	// Capitalize first letter and convert snake_case to CamelCase
	methodName = capitalizeMethodName(methodName)
	paramsStr := strings.TrimSpace(input[parenIndex+1:])

	// Remove trailing )
	paramsStr = strings.TrimSuffix(paramsStr, ")")

	// Parse params
	args := parseArgs(paramsStr)

	return &gs.Call{
		Name: methodName,
		Args: args,
	}
}

// parseArgs parses "param1=value1, param2=value2" into []Arg
func parseArgs(paramsStr string) []gs.Arg {
	if paramsStr == "" {
		return []gs.Arg{}
	}

	var args []gs.Arg

	// Split by comma, but respect string quotes
	var current strings.Builder
	depth := 0
	inString := false

	for _, r := range paramsStr {
		char := string(r)

		if char == "\"" {
			inString = !inString
			current.WriteRune(r)
		} else if char == "(" {
			depth++
			current.WriteRune(r)
		} else if char == ")" {
			depth--
			current.WriteRune(r)
		} else if char == "," && depth == 0 && !inString {
			argStr := strings.TrimSpace(current.String())
			if argStr != "" {
				args = append(args, parseArg(argStr))
			}
			current.Reset()
		} else {
			current.WriteRune(r)
		}
	}

	argStr := strings.TrimSpace(current.String())
	if argStr != "" {
		args = append(args, parseArg(argStr))
	}

	return args
}

// parseArg parses "name=value" into Arg
func parseArg(argStr string) gs.Arg {
	parts := strings.SplitN(argStr, "=", 2)
	if len(parts) != 2 {
		return gs.Arg{
			Name:  "",
			Value: gs.Value{Kind: gs.ValueString, Str: argStr},
		}
	}

	name := strings.TrimSpace(parts[0])
	valueStr := strings.TrimSpace(parts[1])

	value := parseValue(valueStr)

	return gs.Arg{
		Name:  name,
		Value: value,
	}
}

// capitalizeMethodName converts snake_case to CamelCase (track -> Track, set_selected -> SetSelected)
func capitalizeMethodName(name string) string {
	if name == "" {
		return name
	}

	// Convert snake_case to CamelCase
	parts := strings.Split(name, "_")
	var result strings.Builder
	for _, part := range parts {
		if part != "" {
			result.WriteString(strings.ToUpper(part[:1]) + strings.ToLower(part[1:]))
		}
	}

	return result.String()
}

// parseValue parses a value string into Value
func parseValue(valueStr string) gs.Value {
	valueStr = strings.TrimSpace(valueStr)

	// Check if it's a string
	if strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"") {
		return gs.Value{
			Kind: gs.ValueString,
			Str:  valueStr[1 : len(valueStr)-1],
		}
	}

	// Check if it's a boolean
	if valueStr == "true" {
		return gs.Value{Kind: gs.ValueBool, Bool: true}
	}
	if valueStr == "false" {
		return gs.Value{Kind: gs.ValueBool, Bool: false}
	}

	// Check if it's a number
	if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return gs.Value{Kind: gs.ValueNumber, Num: num}
	}

	// Default to string
	return gs.Value{Kind: gs.ValueString, Str: valueStr}
}

// NewFunctionalDSLParser creates a new functional DSL parser.
func NewFunctionalDSLParser() (*FunctionalDSLParser, error) {
	parser := &FunctionalDSLParser{
		reaperDSL:         &ReaperDSL{},
		currentTrackIndex: -1,
		trackCounter:      0,
		data:              make(map[string]interface{}),
		iterationContext:  make(map[string]interface{}),
		actions:           make([]map[string]interface{}, 0),
	}

	parser.reaperDSL.parser = parser

	// Get MAGDA DSL grammar
	grammar := GetMagdaDSLGrammarForFunctional()

	// Create a parser implementation for grammar-school-go
	// This is a workaround - grammar-school-go was supposed to support nil parser
	// but recent changes broke it
	magdaP := &magdaParser{parent: parser}

	// Create Engine with ReaperDSL instance and parser
	engine, err := gs.NewEngine(grammar, parser.reaperDSL, magdaP)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	parser.engine = engine

	return parser, nil
}

// SetState sets the current REAPER state.
func (p *FunctionalDSLParser) SetState(state map[string]interface{}) {
	p.state = state
	// Populate data with collections from state
	if state != nil {
		stateMap, ok := state["state"].(map[string]interface{})
		if !ok {
			stateMap = state
		}
		if tracks, ok := stateMap["tracks"].([]interface{}); ok {
			p.data["tracks"] = tracks
		}
		if clips, ok := stateMap["clips"].([]interface{}); ok {
			p.data["clips"] = clips
		}
	}
}

// ParseDSL parses DSL code and returns REAPER API actions.
func (p *FunctionalDSLParser) ParseDSL(dslCode string) ([]map[string]interface{}, error) {
	if dslCode == "" {
		return nil, fmt.Errorf("empty DSL code")
	}

	// Reset actions for new parse
	p.actions = make([]map[string]interface{}, 0)
	p.currentTrackIndex = -1
	p.trackCounter = 0
	p.clearIterationContext()

	// Execute DSL code using Grammar School Engine
	ctx := context.Background()
	if err := p.engine.Execute(ctx, dslCode); err != nil {
		return nil, fmt.Errorf("failed to execute DSL: %w", err)
	}

	if len(p.actions) == 0 {
		return nil, fmt.Errorf("no actions found in DSL code")
	}

	log.Printf("✅ Functional DSL Parser: Translated %d actions from DSL", len(p.actions))
	return p.actions, nil
}

// setIterationContext sets the current iteration variables.
func (p *FunctionalDSLParser) setIterationContext(context map[string]interface{}) {
	p.iterationContext = context
}

// clearIterationContext clears iteration context.
func (p *FunctionalDSLParser) clearIterationContext() {
	p.iterationContext = make(map[string]interface{})
}

// getIterVarFromCollection derives iteration variable name from collection name.
// tracks -> track, fx_chain -> fx, clips -> clip
func (p *FunctionalDSLParser) getIterVarFromCollection(collectionName string) string {
	// Remove common suffixes
	varName := collectionName
	if len(varName) > 1 && varName[len(varName)-1] == 's' {
		varName = varName[:len(varName)-1]
	}
	if len(varName) > 6 && varName[len(varName)-6:] == "_chain" {
		varName = varName[:len(varName)-6]
	}
	if varName == "" || len(varName) < 2 {
		return "item"
	}
	return varName
}

// resolveCollection resolves a collection name to actual data.
func (p *FunctionalDSLParser) resolveCollection(name string) ([]interface{}, error) {
	// Check if it's in data storage
	if collection, ok := p.data[name]; ok {
		if list, ok := collection.([]interface{}); ok {
			return list, nil
		}
		return nil, fmt.Errorf("collection %s is not a list", name)
	}

	// Check if it's a literal identifier
	return nil, fmt.Errorf("collection %s not found", name)
}

// ========== Side-effect methods (ReaperDSL) ==========

// Track handles track() calls.
func (r *ReaperDSL) Track(args gs.Args) error {
	p := r.parser

	// Check if this is a track reference by ID
	if idValue, ok := args["id"]; ok && idValue.Kind == gs.ValueNumber {
		trackNum := int(idValue.Num)
		p.currentTrackIndex = trackNum - 1
		return nil
	}

	// Check if this is selected track reference
	if selectedValue, ok := args["selected"]; ok && selectedValue.Kind == gs.ValueBool {
		if selectedValue.Bool {
			selectedIndex := p.getSelectedTrackIndex()
			if selectedIndex >= 0 {
				p.currentTrackIndex = selectedIndex
				return nil
			}
			return fmt.Errorf("no selected track found in state")
		}
	}

	// This is a track creation
	action := map[string]interface{}{
		"action": "create_track",
	}

	if instrumentValue, ok := args["instrument"]; ok && instrumentValue.Kind == gs.ValueString {
		action["instrument"] = instrumentValue.Str
	}
	if nameValue, ok := args["name"]; ok && nameValue.Kind == gs.ValueString {
		action["name"] = nameValue.Str
	}

	if indexValue, ok := args["index"]; ok && indexValue.Kind == gs.ValueNumber {
		action["index"] = int(indexValue.Num)
		p.trackCounter = int(indexValue.Num) + 1
	} else {
		action["index"] = p.trackCounter
		p.trackCounter++
	}

	p.currentTrackIndex = action["index"].(int)
	p.actions = append(p.actions, action)

	return nil
}

// NewClip handles .new_clip() calls.
func (r *ReaperDSL) NewClip(args gs.Args) error {
	p := r.parser

	trackIndex := p.currentTrackIndex
	if trackIndex < 0 {
		trackIndex = p.getSelectedTrackIndex()
		if trackIndex < 0 {
			return fmt.Errorf("no track context for clip call")
		}
	}

	action := map[string]interface{}{
		"track": trackIndex,
	}

	if barValue, ok := args["bar"]; ok && barValue.Kind == gs.ValueNumber {
		action["action"] = "create_clip_at_bar"
		action["bar"] = int(barValue.Num)
		if lengthBarsValue, ok := args["length_bars"]; ok && lengthBarsValue.Kind == gs.ValueNumber {
			action["length_bars"] = int(lengthBarsValue.Num)
		} else {
			action["length_bars"] = 4
		}
	} else if startValue, ok := args["start"]; ok && startValue.Kind == gs.ValueNumber {
		action["action"] = "create_clip"
		action["position"] = startValue.Num
		if lengthValue, ok := args["length"]; ok && lengthValue.Kind == gs.ValueNumber {
			action["length"] = lengthValue.Num
		} else {
			action["length"] = 4.0
		}
	} else if positionValue, ok := args["position"]; ok && positionValue.Kind == gs.ValueNumber {
		action["action"] = "create_clip"
		action["position"] = positionValue.Num
		if lengthValue, ok := args["length"]; ok && lengthValue.Kind == gs.ValueNumber {
			action["length"] = lengthValue.Num
		} else {
			action["length"] = 4.0
		}
	} else {
		return fmt.Errorf("clip call must specify bar, start, or position")
	}

	p.actions = append(p.actions, action)
	return nil
}

// AddMidi handles .add_midi() calls.
func (r *ReaperDSL) AddMidi(args gs.Args) error {
	p := r.parser

	if p.currentTrackIndex < 0 {
		return fmt.Errorf("no track context for midi call")
	}

	action := map[string]interface{}{
		"action": "add_midi",
		"track":  p.currentTrackIndex,
		"notes":  []interface{}{},
	}

	if _, ok := args["notes"]; ok {
		// Handle notes array (would need more complex Value type)
		// For now, placeholder
		action["notes"] = []interface{}{}
	}

	p.actions = append(p.actions, action)
	return nil
}

// AddFX handles .add_fx() calls.
func (r *ReaperDSL) AddFX(args gs.Args) error {
	p := r.parser

	if p.currentTrackIndex < 0 {
		return fmt.Errorf("no track context for FX call")
	}

	action := map[string]interface{}{
		"track": p.currentTrackIndex,
	}

	if fxnameValue, ok := args["fxname"]; ok && fxnameValue.Kind == gs.ValueString {
		action["action"] = "add_track_fx"
		action["fxname"] = fxnameValue.Str
	} else if instrumentValue, ok := args["instrument"]; ok && instrumentValue.Kind == gs.ValueString {
		action["action"] = "add_instrument"
		action["fxname"] = instrumentValue.Str
	} else {
		return fmt.Errorf("FX call must specify fxname or instrument")
	}

	p.actions = append(p.actions, action)
	return nil
}

// SetVolume handles .set_volume() calls.
func (r *ReaperDSL) SetVolume(args gs.Args) error {
	p := r.parser
	if p.currentTrackIndex < 0 {
		return fmt.Errorf("no track context for volume call")
	}
	volumeValue, ok := args["volume_db"]
	if !ok || volumeValue.Kind != gs.ValueNumber {
		return fmt.Errorf("volume_db must be a number")
	}
	action := map[string]interface{}{
		"action":    "set_track_volume",
		"track":     p.currentTrackIndex,
		"volume_db": volumeValue.Num,
	}
	p.actions = append(p.actions, action)
	return nil
}

// SetPan, SetMute, SetSolo, SetName methods (similar pattern)
func (r *ReaperDSL) SetPan(args gs.Args) error {
	p := r.parser
	if p.currentTrackIndex < 0 {
		return fmt.Errorf("no track context for pan call")
	}
	panValue, ok := args["pan"]
	if !ok || panValue.Kind != gs.ValueNumber {
		return fmt.Errorf("pan must be a number")
	}
	action := map[string]interface{}{
		"action": "set_track_pan",
		"track":  p.currentTrackIndex,
		"pan":    panValue.Num,
	}
	p.actions = append(p.actions, action)
	return nil
}

func (r *ReaperDSL) SetMute(args gs.Args) error {
	p := r.parser
	if p.currentTrackIndex < 0 {
		return fmt.Errorf("no track context for mute call")
	}
	muteValue, ok := args["mute"]
	if !ok || muteValue.Kind != gs.ValueBool {
		return fmt.Errorf("mute must be a boolean")
	}
	action := map[string]interface{}{
		"action": "set_track_mute",
		"track":  p.currentTrackIndex,
		"mute":   muteValue.Bool,
	}
	p.actions = append(p.actions, action)
	return nil
}

func (r *ReaperDSL) SetSolo(args gs.Args) error {
	p := r.parser
	if p.currentTrackIndex < 0 {
		return fmt.Errorf("no track context for solo call")
	}
	soloValue, ok := args["solo"]
	if !ok || soloValue.Kind != gs.ValueBool {
		return fmt.Errorf("solo must be a boolean")
	}
	action := map[string]interface{}{
		"action": "set_track_solo",
		"track":  p.currentTrackIndex,
		"solo":   soloValue.Bool,
	}
	p.actions = append(p.actions, action)
	return nil
}

func (r *ReaperDSL) SetName(args gs.Args) error {
	p := r.parser
	if p.currentTrackIndex < 0 {
		return fmt.Errorf("no track context for name call")
	}
	nameValue, ok := args["name"]
	if !ok || nameValue.Kind != gs.ValueString {
		return fmt.Errorf("name must be a string")
	}
	action := map[string]interface{}{
		"action": "set_track_name",
		"track":  p.currentTrackIndex,
		"name":   nameValue.Str,
	}
	p.actions = append(p.actions, action)
	return nil
}

// SetSelected handles .set_selected() calls.
// If there's a filtered collection, applies to all items; otherwise uses currentTrackIndex.
func (r *ReaperDSL) SetSelected(args gs.Args) error {
	p := r.parser
	selectedValue, ok := args["selected"]
	if !ok || selectedValue.Kind != gs.ValueBool {
		return fmt.Errorf("selected must be a boolean")
	}
	selected := selectedValue.Bool

	// Check if we have a filtered collection to apply to
	if filteredCollection, hasFiltered := p.data["current_filtered"]; hasFiltered {
		if filtered, ok := filteredCollection.([]interface{}); ok && len(filtered) > 0 {
			// Apply to all filtered tracks
			for _, item := range filtered {
				trackMap, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				trackIndex, ok := trackMap["index"].(int)
				if !ok {
					// Try float64 (JSON numbers are float64)
					if trackIndexFloat, ok := trackMap["index"].(float64); ok {
						trackIndex = int(trackIndexFloat)
					} else {
						continue
					}
				}
				action := map[string]interface{}{
					"action":   "set_track_selected",
					"track":    trackIndex,
					"selected": selected,
				}
				p.actions = append(p.actions, action)
			}
			// Clear filtered collection after applying
			delete(p.data, "current_filtered")
			log.Printf("Applied set_selected to %d filtered tracks", len(filtered))
			return nil
		}
	}

	// Normal single-track operation
	if p.currentTrackIndex < 0 {
		return fmt.Errorf("no track context for selected call")
	}
	action := map[string]interface{}{
		"action":   "set_track_selected",
		"track":    p.currentTrackIndex,
		"selected": selected,
	}
	p.actions = append(p.actions, action)
	return nil
}

// ========== Functional methods ==========

// Filter filters a collection using a predicate.
// For Go, we'll use a simpler approach since we don't have expression evaluation yet.
// The predicate can be a function reference or we evaluate simple comparisons.
//
// Example: filter(tracks, @is_fx_track) or filter(tracks, "name", "==", "FX")
func (r *ReaperDSL) Filter(args gs.Args) error {
	p := r.parser

	// Get collection name or value
	var collection []interface{}
	var collectionName string

	if collectionValue, ok := args["collection"]; ok {
		if collectionValue.Kind == gs.ValueString {
			collectionName = collectionValue.Str
			var err error
			collection, err = p.resolveCollection(collectionName)
			if err != nil {
				return fmt.Errorf("failed to resolve collection: %w", err)
			}
		}
	} else if collectionValue, ok := args["_positional"]; ok {
		// First positional argument is collection
		if collectionValue.Kind == gs.ValueString {
			collectionName = collectionValue.Str
			var err error
			collection, err = p.resolveCollection(collectionName)
			if err != nil {
				return fmt.Errorf("failed to resolve collection: %w", err)
			}
		}
	} else {
		return fmt.Errorf("filter requires a collection argument")
	}

	if collection == nil {
		return fmt.Errorf("collection not found or is empty")
	}

	// Derive iteration variable name
	iterVar := p.getIterVarFromCollection(collectionName)

	// Filter the collection
	// For now, we'll use a simple predicate evaluation
	// In a full implementation, you'd evaluate expressions here
	filtered := make([]interface{}, 0)

	for _, item := range collection {
		// Set iteration context
		p.setIterationContext(map[string]interface{}{
			iterVar: item,
		})

		// Evaluate predicate - support property_access comparison_op value format
		// Example: filter(tracks, track.name == "foo")
		predicateMatched := false

		// Try to find predicate components from parsed args
		// The grammar should parse "track.name == \"foo\"" into property, operator, value
		if propValue, ok := args["property"]; ok && propValue.Kind == gs.ValueString {
			// Property access like "track.name"
				if opValue, ok := args["operator"]; ok && opValue.Kind == gs.ValueString {
					if compareValue, ok := args["value"]; ok {
					// Extract property name from "track.name" -> "name"
					propParts := strings.Split(propValue.Str, ".")
					var propName string
					if len(propParts) > 1 {
						// track.name -> name
						propName = propParts[len(propParts)-1]
					} else {
						propName = propValue.Str
					}
					predicateMatched = evaluateSimplePredicate(item, propName, opValue.Str, compareValue)
				}
			}
		} else if predicateValue, ok := args["predicate"]; ok {
			// Handle function reference predicate (future extension)
			if predicateValue.Kind == gs.ValueFunction {
				// Function reference - would need to call it
				// For now, include all items as placeholder
				predicateMatched = true
			}
		}

		if predicateMatched {
			filtered = append(filtered, item)
		}

		p.clearIterationContext()
	}

	// Store filtered result - return the filtered collection name for chaining
	resultName := collectionName + "_filtered"
	p.data[resultName] = filtered

	// Also store as "current_filtered" for potential chaining
	p.data["current_filtered"] = filtered

	// Set the current collection context so chained methods can operate on filtered results
	p.currentTrackIndex = -1 // Reset, will be set per item in map/for_each

	log.Printf("Filtered %d items to %d", len(collection), len(filtered))
	return nil
}

// Map maps a function over a collection.
func (r *ReaperDSL) Map(args gs.Args) error {
	p := r.parser

	// Get collection
	var collection []interface{}
	var collectionName string

	if collectionValue, ok := args["collection"]; ok && collectionValue.Kind == gs.ValueString {
		collectionName = collectionValue.Str
		var err error
		collection, err = p.resolveCollection(collectionName)
		if err != nil {
			return fmt.Errorf("failed to resolve collection: %w", err)
		}
	} else {
		return fmt.Errorf("map requires a collection argument")
	}

	// Get function reference
	if funcValue, ok := args["func"]; ok && funcValue.Kind == gs.ValueFunction {
		_ = funcValue.Str // funcName for future use
		iterVar := p.getIterVarFromCollection(collectionName)

		mapped := make([]interface{}, 0, len(collection))

		for _, item := range collection {
			p.setIterationContext(map[string]interface{}{
				iterVar: item,
			})

			// Apply function to item
			// Would need to call the function handler here
			// For now, just pass through
			mapped = append(mapped, item)

			p.clearIterationContext()
		}

		resultName := collectionName + "_mapped"
		p.data[resultName] = mapped
		log.Printf("Mapped %d items", len(collection))
		return nil
	}

	return fmt.Errorf("map requires a function argument")
}

// Store stores a value in data storage.
func (r *ReaperDSL) Store(args gs.Args) error {
	p := r.parser

	nameValue, ok := args["name"]
	if !ok || nameValue.Kind != gs.ValueString {
		return fmt.Errorf("store requires a name argument")
	}

	// Get value (would need to handle different types)
	if valueValue, ok := args["value"]; ok {
		// Convert Value to interface{}
		var value interface{}
		switch valueValue.Kind {
		case gs.ValueString:
			value = valueValue.Str
		case gs.ValueNumber:
			value = valueValue.Num
		case gs.ValueBool:
			value = valueValue.Bool
		default:
			value = nil
		}
		p.data[nameValue.Str] = value
		log.Printf("Stored %s = %v", nameValue.Str, value)
		return nil
	}

	return fmt.Errorf("store requires a value argument")
}

// GetTracks gets all tracks from state.
func (r *ReaperDSL) GetTracks(args gs.Args) error {
	p := r.parser

	if p.state == nil {
		return nil
	}

	stateMap, ok := p.state["state"].(map[string]interface{})
	if !ok {
		stateMap = p.state
	}

	if tracks, ok := stateMap["tracks"].([]interface{}); ok {
		p.data["tracks"] = tracks
	}

	return nil
}

// GetFXChain gets FX chain for current track.
func (r *ReaperDSL) GetFXChain(args gs.Args) error {
	p := r.parser

	trackIndex := p.currentTrackIndex
	if trackIndex < 0 || p.state == nil {
		return nil
	}

	stateMap, ok := p.state["state"].(map[string]interface{})
	if !ok {
		stateMap = p.state
	}

	tracks, ok := stateMap["tracks"].([]interface{})
	if !ok || trackIndex >= len(tracks) {
		return nil
	}

	track, ok := tracks[trackIndex].(map[string]interface{})
	if !ok {
		return nil
	}

	if fxChain, ok := track["fx"].([]interface{}); ok {
		p.data["fx_chain"] = fxChain
	}

	return nil
}

// Helper functions

func (p *FunctionalDSLParser) getSelectedTrackIndex() int {
	if p.state == nil {
		return -1
	}

	stateMap, ok := p.state["state"].(map[string]interface{})
	if !ok {
		stateMap = p.state
	}

	tracks, ok := stateMap["tracks"].([]interface{})
	if !ok {
		return -1
	}

	for i, track := range tracks {
		trackMap, ok := track.(map[string]interface{})
		if !ok {
			continue
		}
		if selected, ok := trackMap["selected"].(bool); ok && selected {
			return i
		}
	}

	return -1
}

// evaluateSimplePredicate evaluates a simple property-based predicate.
func evaluateSimplePredicate(item interface{}, propName, operator string, compareValue gs.Value) bool {
	itemMap, ok := item.(map[string]interface{})
	if !ok {
		return false
	}

	itemValue, ok := itemMap[propName]
	if !ok {
		return false
	}

	switch operator {
	case "==":
		return compareValues(itemValue, compareValue) == 0
	case "!=":
		return compareValues(itemValue, compareValue) != 0
	case "<":
		return compareValues(itemValue, compareValue) < 0
	case ">":
		return compareValues(itemValue, compareValue) > 0
	case "<=":
		return compareValues(itemValue, compareValue) <= 0
	case ">=":
		return compareValues(itemValue, compareValue) >= 0
	default:
		return false
	}
}

// compareValues compares two values and returns -1, 0, or 1.
func compareValues(a interface{}, b gs.Value) int {
	switch b.Kind {
	case gs.ValueString:
		aStr, ok := a.(string)
		if !ok {
			return -1
		}
		if aStr < b.Str {
			return -1
		} else if aStr > b.Str {
			return 1
		}
		return 0
	case gs.ValueNumber:
		aNum, ok := a.(float64)
		if !ok {
			if aInt, ok := a.(int); ok {
				aNum = float64(aInt)
			} else {
				return -1
			}
		}
		if aNum < b.Num {
			return -1
		} else if aNum > b.Num {
			return 1
		}
		return 0
	case gs.ValueBool:
		aBool, ok := a.(bool)
		if !ok {
			return -1
		}
		if aBool == b.Bool {
			return 0
		} else if !aBool && b.Bool {
			return -1
		}
		return 1
	default:
		return -1
	}
}

// GetMagdaDSLGrammarForFunctional returns the grammar with functional methods added.
// This is the grammar used for CFG generation to allow the LLM to generate functional DSL code.
func GetMagdaDSLGrammarForFunctional() string {
	// Start with base grammar
	baseGrammar := `
// MAGDA DSL Grammar - Functional scripting for REAPER operations
// Syntax: track().new_clip().add_midi() with method chaining

start: statement+

statement: track_call chain?
         | functional_call

track_call: "track" "(" track_params? ")"
track_params: track_param ("," SP track_param)*
           | NUMBER
track_param: "instrument" "=" STRING
           | "name" "=" STRING
           | "index" "=" NUMBER
           | "id" "=" NUMBER
           | "selected" "=" BOOLEAN

chain: clip_chain | midi_chain | fx_chain | volume_chain | pan_chain | mute_chain | solo_chain | name_chain | selected_chain

clip_chain: ".new_clip" "(" clip_params? ")"
clip_params: clip_param ("," SP clip_param)*
clip_param: "bar" "=" NUMBER
          | "start" "=" NUMBER
          | "length_bars" "=" NUMBER
          | "length" "=" NUMBER
          | "position" "=" NUMBER

midi_chain: ".add_midi" "(" midi_params? ")"
midi_params: "notes" "=" array

fx_chain: ".add_fx" "(" fx_params? ")"
fx_params: "fxname" "=" STRING
         | "instrument" "=" STRING

volume_chain: ".set_volume" "(" "volume_db" "=" NUMBER ")"
pan_chain: ".set_pan" "(" "pan" "=" NUMBER ")"
mute_chain: ".set_mute" "(" "mute" "=" BOOLEAN ")"
solo_chain: ".set_solo" "(" "solo" "=" BOOLEAN ")"
name_chain: ".set_name" "(" "name" "=" STRING ")"
selected_chain: ".set_selected" "(" "selected" "=" BOOLEAN ")"

// Functional operations
functional_call: filter_call chain?
                 | map_call
                 | for_each_call

filter_call: "filter" "(" IDENTIFIER "," filter_predicate ")"
filter_predicate: property_access comparison_op value
                | property_access comparison_op BOOLEAN
                | property_access "==" STRING
                | property_access "!=" STRING
                | property_access "==" BOOLEAN

map_call: "map" "(" IDENTIFIER "," function_ref ")"
          | "map" "(" IDENTIFIER "," method_call ")"

for_each_call: "for_each" "(" IDENTIFIER "," function_ref ")"
               | "for_each" "(" IDENTIFIER "," method_call ")"

method_call: IDENTIFIER "." IDENTIFIER "(" method_params? ")"
method_params: method_param ("," SP method_param)*
method_param: IDENTIFIER "=" (STRING | NUMBER | BOOLEAN)

property_access: IDENTIFIER "." IDENTIFIER
               | IDENTIFIER "." IDENTIFIER "[" NUMBER "]"

comparison_op: "==" | "!=" | "<" | ">" | "<=" | ">="

function_ref: "@" IDENTIFIER

array: "[" (value ("," SP value)*)? "]"
value: STRING | NUMBER | BOOLEAN | array

SP: " "
STRING: /"[^"]*"/
NUMBER: /-?\d+(\.\d+)?/
BOOLEAN: "true" | "false"
IDENTIFIER: /[a-zA-Z_][a-zA-Z0-9_]*/
`

	return baseGrammar
}
