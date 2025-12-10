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

	// Use generic Lark parser from grammar-school
	larkParser := gs.NewLarkParser()

	// Create Engine with ReaperDSL instance and parser
	engine, err := gs.NewEngine(grammar, parser.reaperDSL, larkParser)
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

			// Extract all clips from all tracks into a global clips collection
			// This allows filter(clips, ...) to work on all clips across all tracks
			allClips := make([]interface{}, 0)
			for _, trackInterface := range tracks {
				if track, ok := trackInterface.(map[string]interface{}); ok {
					if clips, ok := track["clips"].([]interface{}); ok {
						// Add track index to each clip for reference
						trackIndex, _ := track["index"].(int)
						if trackIndexFloat, ok := track["index"].(float64); ok {
							trackIndex = int(trackIndexFloat)
						}
						for _, clip := range clips {
							if clipMap, ok := clip.(map[string]interface{}); ok {
								// Ensure clip has track reference
								clipMap["track"] = trackIndex
							}
							allClips = append(allClips, clip)
						}
					}
				}
			}
			if len(allClips) > 0 {
				p.data["clips"] = allClips
				log.Printf("📦 Extracted %d clips from %d tracks into global clips collection", len(allClips), len(tracks))
			}
		}
		// Also check for top-level clips collection (if state provides it directly)
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

// AddFx handles .add_fx() calls.
// Note: Method name must be AddFx (not AddFX) for grammar-school camelCase conversion
func (r *ReaperDSL) AddFx(args gs.Args) error {
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
	muteValue, ok := args["mute"]
	if !ok || muteValue.Kind != gs.ValueBool {
		return fmt.Errorf("mute must be a boolean")
	}
	mute := muteValue.Bool

	// Check if we have a filtered collection to apply to
	if filteredCollection, hasFiltered := p.data["current_filtered"]; hasFiltered {
		log.Printf("🔍 SetMute: Found filtered collection (hasFiltered=%v)", hasFiltered)
		if filtered, ok := filteredCollection.([]interface{}); ok {
			log.Printf("🔍 SetMute: Filtered collection has %d items, mute=%v", len(filtered), mute)
			if len(filtered) > 0 {
				// Apply to all filtered tracks
				for _, item := range filtered {
					trackMap, ok := item.(map[string]interface{})
					if !ok {
						log.Printf("⚠️  SetMute: Item is not a map: %T", item)
						continue
					}
					trackIndex, ok := trackMap["index"].(int)
					if !ok {
						// Try float64 (JSON numbers are float64)
						if trackIndexFloat, ok := trackMap["index"].(float64); ok {
							trackIndex = int(trackIndexFloat)
						} else {
							log.Printf("⚠️  SetMute: Could not extract track index from %+v", trackMap)
							continue
						}
					}
					trackName, _ := trackMap["name"].(string)
					log.Printf("✅ SetMute: Adding action for track %d (name='%s'), mute=%v", trackIndex, trackName, mute)
					action := map[string]interface{}{
						"action": "set_track_mute",
						"track":  trackIndex,
						"mute":   mute,
					}
					p.actions = append(p.actions, action)
				}
				// Clear filtered collection after applying
				delete(p.data, "current_filtered")
				log.Printf("✅ SetMute: Applied set_mute to %d filtered tracks", len(filtered))
				return nil
			}
		}
	}

	// Normal single-track operation
	if p.currentTrackIndex < 0 {
		return fmt.Errorf("no track context for mute call")
	}
	action := map[string]interface{}{
		"action": "set_track_mute",
		"track":  p.currentTrackIndex,
		"mute":   mute,
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
	if !ok {
		return fmt.Errorf("selected parameter is required")
	}
	
	var selected bool
	if selectedValue.Kind == gs.ValueBool {
		selected = selectedValue.Bool
	} else if selectedValue.Kind == gs.ValueString {
		// Handle string "true"/"false" as fallback
		selectedStr := strings.ToLower(strings.TrimSpace(selectedValue.Str))
		if selectedStr == "true" {
			selected = true
		} else if selectedStr == "false" {
			selected = false
		} else {
			return fmt.Errorf("selected must be a boolean (got string: %q)", selectedValue.Str)
		}
	} else {
		return fmt.Errorf("selected must be a boolean (got kind: %v)", selectedValue.Kind)
	}

	// Check if we have a filtered collection to apply to
	if filteredCollection, hasFiltered := p.data["current_filtered"]; hasFiltered {
		log.Printf("🔍 SetSelected: Found filtered collection (hasFiltered=%v)", hasFiltered)
		if filtered, ok := filteredCollection.([]interface{}); ok {
			log.Printf("🔍 SetSelected: Filtered collection has %d items, selected=%v", len(filtered), selected)
			if len(filtered) > 0 {
				// Apply to all filtered tracks
				for _, item := range filtered {
					trackMap, ok := item.(map[string]interface{})
					if !ok {
						log.Printf("⚠️  SetSelected: Item is not a map: %T", item)
						continue
					}
					trackIndex, ok := trackMap["index"].(int)
					if !ok {
						// Try float64 (JSON numbers are float64)
						if trackIndexFloat, ok := trackMap["index"].(float64); ok {
							trackIndex = int(trackIndexFloat)
						} else {
							log.Printf("⚠️  SetSelected: Could not extract track index from %+v", trackMap)
							continue
						}
					}
					trackName, _ := trackMap["name"].(string)
					log.Printf("✅ SetSelected: Adding action for track %d (name='%s'), selected=%v", trackIndex, trackName, selected)
					action := map[string]interface{}{
						"action":   "set_track_selected",
						"track":    trackIndex,
						"selected": selected,
					}
					p.actions = append(p.actions, action)
				}
				// Clear filtered collection after applying
				delete(p.data, "current_filtered")
				log.Printf("✅ SetSelected: Applied set_selected to %d filtered tracks", len(filtered))
				return nil
			} else {
				log.Printf("⚠️  SetSelected: Filtered collection is empty! This means filter() returned 0 results.")
			}
		} else {
			log.Printf("⚠️  SetSelected: Filtered collection is not a []interface{}: %T", filteredCollection)
		}
	} else {
		log.Printf("🔍 SetSelected: No filtered collection found, using single-track mode (currentTrackIndex=%d)", p.currentTrackIndex)
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

// Delete handles .delete() calls to delete the current track.
// If there's a filtered collection, applies to all items; otherwise uses currentTrackIndex.
func (r *ReaperDSL) Delete(args gs.Args) error {
	p := r.parser

	// Check if we have a filtered collection to apply to
	if filteredCollection, hasFiltered := p.data["current_filtered"]; hasFiltered {
		log.Printf("🔍 Delete: Found filtered collection (hasFiltered=%v)", hasFiltered)
		if filtered, ok := filteredCollection.([]interface{}); ok {
			log.Printf("🔍 Delete: Filtered collection has %d items", len(filtered))
			if len(filtered) > 0 {
				// Apply to all filtered tracks
				for _, item := range filtered {
					trackMap, ok := item.(map[string]interface{})
					if !ok {
						log.Printf("⚠️  Delete: Item is not a map: %T", item)
						continue
					}
					trackIndex, ok := trackMap["index"].(int)
					if !ok {
						// Try float64 (JSON numbers are float64)
						if trackIndexFloat, ok := trackMap["index"].(float64); ok {
							trackIndex = int(trackIndexFloat)
						} else {
							log.Printf("⚠️  Delete: Could not extract track index from %+v", trackMap)
							continue
						}
					}
					trackName, _ := trackMap["name"].(string)
					log.Printf("✅ Delete: Adding action for track %d (name='%s')", trackIndex, trackName)
					action := map[string]interface{}{
						"action": "delete_track",
						"track":  trackIndex,
					}
					p.actions = append(p.actions, action)
				}
				// Clear filtered collection after applying
				delete(p.data, "current_filtered")
				log.Printf("✅ Delete: Applied delete_track to %d filtered tracks", len(filtered))
				return nil
			} else {
				log.Printf("⚠️  Delete: Filtered collection is empty! This means filter() returned 0 results.")
			}
		} else {
			log.Printf("⚠️  Delete: Filtered collection is not a []interface{}: %T", filteredCollection)
		}
	} else {
		log.Printf("🔍 Delete: No filtered collection found, using single-track mode (currentTrackIndex=%d)", p.currentTrackIndex)
	}

	// Normal single-track operation
	if p.currentTrackIndex < 0 {
		return fmt.Errorf("no track context for delete call")
	}
	action := map[string]interface{}{
		"action": "delete_track",
		"track":  p.currentTrackIndex,
	}
	p.actions = append(p.actions, action)
	return nil
}

// DeleteClip handles .deleteClip() calls to delete a clip from the current track.
func (r *ReaperDSL) DeleteClip(args gs.Args) error {
	p := r.parser
	if p.currentTrackIndex < 0 {
		return fmt.Errorf("no track context for deleteClip call")
	}
	action := map[string]interface{}{
		"action": "delete_clip",
		"track":  p.currentTrackIndex,
	}

	// Clip identification: clip index, position, or bar
	if clipValue, ok := args["clip"]; ok && clipValue.Kind == gs.ValueNumber {
		action["clip"] = int(clipValue.Num)
	} else if positionValue, ok := args["position"]; ok && positionValue.Kind == gs.ValueNumber {
		action["position"] = positionValue.Num
	} else if barValue, ok := args["bar"]; ok && barValue.Kind == gs.ValueNumber {
		action["bar"] = int(barValue.Num)
	} else {
		return fmt.Errorf("deleteClip requires one of: clip (index), position (seconds), or bar (number)")
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

	// Log all args for debugging
	log.Printf("🔍 Filter: Received args with %d keys: %v", len(args), getArgsKeys(args))
	for k, v := range args {
		log.Printf("   Filter arg[%s] = %+v (Kind: %v, Str: '%s', Num: %v)", k, v, v.Kind, v.Str, v.Num)
	}

	// Get collection name or value
	var collection []interface{}
	var collectionName string

	// Try multiple ways to find the collection argument
	// 1. Named argument "collection"
	if collectionValue, ok := args["collection"]; ok {
		if collectionValue.Kind == gs.ValueString {
			collectionName = collectionValue.Str
			var err error
			collection, err = p.resolveCollection(collectionName)
			if err == nil {
				log.Printf("✅ Filter: Found collection '%s' via named arg 'collection'", collectionName)
			} else {
				log.Printf("⚠️  Filter: Failed to resolve collection '%s' from named arg: %v", collectionName, err)
			}
		}
	}

	// 2. First positional argument (empty key or _positional)
	if collection == nil {
		if collectionValue, ok := args[""]; ok {
			if collectionValue.Kind == gs.ValueString {
				collectionName = collectionValue.Str
				var err error
				collection, err = p.resolveCollection(collectionName)
				if err == nil {
					log.Printf("✅ Filter: Found collection '%s' via positional arg (empty key)", collectionName)
				} else {
					log.Printf("⚠️  Filter: Failed to resolve collection '%s' from positional arg: %v", collectionName, err)
				}
			}
		} else if collectionValue, ok := args["_positional"]; ok {
			if collectionValue.Kind == gs.ValueString {
				collectionName = collectionValue.Str
				var err error
				collection, err = p.resolveCollection(collectionName)
				if err == nil {
					log.Printf("✅ Filter: Found collection '%s' via _positional key", collectionName)
				} else {
					log.Printf("⚠️  Filter: Failed to resolve collection '%s' from _positional: %v", collectionName, err)
				}
			}
		}
	}

	// 3. Last resort: iterate and find first string value that resolves to a collection
	if collection == nil {
		log.Printf("🔍 Filter: Trying to find collection by iterating all args...")
		for key, value := range args {
			// Skip known named arguments that are not collections
			if key == "predicate" || key == "property" || key == "operator" || key == "value" {
				continue
			}
			// Try to resolve as collection
			if value.Kind == gs.ValueString {
				potentialName := value.Str
				log.Printf("🔍 Filter: Trying to resolve '%s' (from key '%s') as collection...", potentialName, key)
				if resolved, err := p.resolveCollection(potentialName); err == nil && resolved != nil {
					collectionName = potentialName
					collection = resolved
					log.Printf("✅ Filter: Found collection '%s' via iteration (key: '%s')", collectionName, key)
					break
				} else {
					log.Printf("⚠️  Filter: '%s' is not a valid collection: %v", potentialName, err)
				}
			}
		}
	}

	// Check if we found a collection
	if collection == nil {
		log.Printf("❌ Filter: Could not find collection argument. Available data keys: %v", getDataKeys(p.data))
		return fmt.Errorf("filter requires a collection argument (got args: %v, available collections: %v)", args, getDataKeys(p.data))
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
				} else {
					log.Printf("⚠️  Filter: Missing 'value' in predicate args: %+v", args)
				}
			} else {
				log.Printf("⚠️  Filter: Missing 'operator' in predicate args: %+v", args)
			}
		} else if predicateValue, ok := args["predicate"]; ok {
			// Handle function reference predicate (future extension)
			if predicateValue.Kind == gs.ValueFunction {
				// Function reference - would need to call it
				// For now, include all items as placeholder
				predicateMatched = true
			}
		} else {
			// Try to manually parse predicate from args
			// The parser might have split the predicate across multiple args
			// Example: track.name=="Nebula Drift" might be parsed as:
			//   args["track.name"] = "=\"Nebula Drift\""
			// We need to reconstruct the full predicate

			// First, try to find a complete predicate string
			for key, value := range args {
				if value.Kind == gs.ValueString {
					predStr := strings.TrimSpace(value.Str)
					log.Printf("🔍 Filter: Checking predicate string '%s' (key: '%s')", predStr, key)
					// Check if it looks like a complete predicate: "track.name == \"value\"" or "track.name==\"value\""
					if strings.Contains(predStr, ".") && (strings.Contains(predStr, "==") || strings.Contains(predStr, "!=")) {
						log.Printf("🔍 Filter: Attempting to parse complete predicate: '%s'", predStr)
						// Try to parse it manually
						if matched := p.parseAndEvaluatePredicate(predStr, item, iterVar); matched {
							log.Printf("✅ Filter: Predicate matched for item: %v", item)
							predicateMatched = true
							break
						} else {
							log.Printf("❌ Filter: Predicate did not match for item: %v", item)
						}
					}
				}
			}

			// If no complete predicate found, try to reconstruct from split args
			// Look for args with keys like "track.name" and values starting with "=" or "!="
			if !predicateMatched {
				for key, value := range args {
					// Skip the collection argument (empty key)
					if key == "" {
						continue
					}
					if value.Kind == gs.ValueString {
						valueStr := strings.TrimSpace(value.Str)
						// Check if value starts with comparison operator (e.g., "=\"value\"" or "==\"value\"")
						if strings.HasPrefix(valueStr, "=") || strings.HasPrefix(valueStr, "!=") {
							// Reconstruct predicate: key + value
							// key is like "track.name", value is like "=\"Nebula Drift\"" or "=true"
							operator := "=="
							if strings.HasPrefix(valueStr, "!=") {
								operator = "!="
								valueStr = strings.TrimPrefix(valueStr, "!=")
							} else {
								valueStr = strings.TrimPrefix(valueStr, "=")
							}

							// Check if value is a boolean (true/false) - don't wrap in quotes
							valueStr = strings.TrimSpace(valueStr)
							isBoolean := valueStr == "true" || valueStr == "false"

							// Remove quotes if present (for string values)
							if !isBoolean {
								valueStr = strings.Trim(valueStr, "\"")
							}

							// Reconstruct predicate
							var reconstructedPred string
							if isBoolean {
								// For booleans: "track.muted == true" (no quotes)
								reconstructedPred = fmt.Sprintf("%s %s %s", key, operator, valueStr)
							} else {
								// For strings: "track.name == \"Nebula Drift\"" (with quotes)
								reconstructedPred = fmt.Sprintf("%s %s \"%s\"", key, operator, valueStr)
							}
							log.Printf("🔍 Filter: Reconstructed predicate from split args: '%s'", reconstructedPred)

							// Parse and evaluate
							if matched := p.parseAndEvaluatePredicate(reconstructedPred, item, iterVar); matched {
								log.Printf("✅ Filter: Reconstructed predicate matched for item: %v", item)
								predicateMatched = true
								break
							} else {
								// This is expected - predicate didn't match this item, continue checking
								log.Printf("🔍 Filter: Predicate did not match for item (this is normal): %v", item)
							}
						}
					}
				}
			}

			// Note: predicateMatched being false here is expected for items that don't match the predicate
			// We only log a warning if we couldn't even attempt to parse the predicate
			// (which would mean we didn't find any predicate-like args at all)
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

	log.Printf("✅ Filtered %d items from '%s' to %d matches", len(collection), collectionName, len(filtered))
	if len(filtered) == 0 {
		log.Printf("⚠️  WARNING: Filter returned 0 results! Args received: %v", getArgsKeys(args))
		// Log first item to debug
		if len(collection) > 0 {
			log.Printf("   First item in collection: %+v", collection[0])
		}
	}
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

// ForEach applies a function or method to each item in a collection (side effects).
// Grammar: for_each(collection, @function) or for_each(collection, item.method())
func (r *ReaperDSL) ForEach(args gs.Args) error {
	p := r.parser

	// Get collection - similar to Filter and Map
	var collection []interface{}
	var collectionName string

	// Try to get collection from various argument positions
	// Note: for_each(tracks, track.method()) has two positional args, both with Name=""
	// The second one overwrites the first in the map, so we need to check both
	if collectionValue, ok := args["collection"]; ok && collectionValue.Kind == gs.ValueString {
		collectionName = collectionValue.Str
		var err error
		collection, err = p.resolveCollection(collectionName)
		if err != nil {
			return fmt.Errorf("failed to resolve collection: %w", err)
		}
	} else {
		// Check positional argument (Name="")
		// For for_each(tracks, track.method()), the second arg overwrites the first
		// So args[""] will be the method call, not the collection name
		// We need to find the collection by checking which string value is a valid collection name
		for _, value := range args {
			if value.Kind == gs.ValueString {
				potentialName := value.Str
				// Skip if it looks like a method call (contains "." and "(")
				if strings.Contains(potentialName, ".") && strings.Contains(potentialName, "(") {
					continue // This is the method call, not the collection
				}
				// Try to resolve as collection
				if resolved, err := p.resolveCollection(potentialName); err == nil && resolved != nil {
					collectionName = potentialName
					collection = resolved
					break
				}
			}
		}
	}

	if collection == nil {
		return fmt.Errorf("for_each requires a collection argument (got args: %v, available collections: %v)", args, getDataKeys(p.data))
	}

	// Derive iteration variable name
	iterVar := p.getIterVarFromCollection(collectionName)

	// Get the function/method to execute
	var methodCallStr string
	var funcRef string

	// Log all arguments for debugging
	log.Printf("🔄 ForEach: Received args: %v", getArgsKeys(args))
	for key, value := range args {
		log.Printf("  ForEach arg[%s]: Kind=%s, Str=%s", key, value.Kind, value.Str)
	}

	// Check for function reference (@func_name)
	if funcValue, ok := args["func"]; ok && funcValue.Kind == gs.ValueFunction {
		funcRef = funcValue.Str
		log.Printf("🔄 ForEach: Found function reference: @%s", funcRef)
		// TODO: Implement function registry and execution
		// For now, function references are not yet supported
		return fmt.Errorf("function references (@%s) are not yet implemented in for_each", funcRef)
	}

	// Check for method call string (e.g., "track.add_fx(fxname=\"ReaEQ\")")
	// For for_each(tracks, track.method()), the method call is the second positional argument
	// which overwrites the first in args[""], so args[""] will contain the method call
	// Try positional argument first (this is where the method call will be)
	if value, ok := args[""]; ok && value.Kind == gs.ValueString {
		// Check if it looks like a method call (contains "." and "(")
		if strings.Contains(value.Str, ".") && strings.Contains(value.Str, "(") {
			methodCallStr = value.Str
			log.Printf("🔄 ForEach: Found method call in positional arg: %s", methodCallStr)
		}
	}

	// Try other argument names
	if methodCallStr == "" {
		for _, key := range []string{"method", "call", "action"} {
			if value, ok := args[key]; ok {
				if value.Kind == gs.ValueString {
					// Check if it looks like a method call (contains "." and "(")
					if strings.Contains(value.Str, ".") && strings.Contains(value.Str, "(") {
						methodCallStr = value.Str
						log.Printf("🔄 ForEach: Found method call string in arg[%s]: %s", key, methodCallStr)
						break
					}
				}
			}
		}
	}

	// Last resort: check all string values that look like method calls
	if methodCallStr == "" {
		for key, value := range args {
			if value.Kind == gs.ValueString {
				// Skip if it's the collection name
				if value.Str != collectionName && strings.Contains(value.Str, ".") && strings.Contains(value.Str, "(") {
					methodCallStr = value.Str
					log.Printf("🔄 ForEach: Found method call in arg[%s]: %s", key, methodCallStr)
					break
				}
			}
		}
	}

	log.Printf("🔄 ForEach: Iterating over %d items in collection '%s'", len(collection), collectionName)
	log.Printf("🔄 ForEach: methodCallStr='%s', collectionName='%s'", methodCallStr, collectionName)

	// If we have a method call, parse and execute it for each item
	if methodCallStr != "" {
		// Parse method call: track.add_fx(fxname="ReaEQ")
		// Extract method name and parameters
		methodName, methodArgs, err := p.parseMethodCallString(methodCallStr)
		if err != nil {
			return fmt.Errorf("failed to parse method call '%s': %w", methodCallStr, err)
		}

		log.Printf("  ForEach: Executing method '%s' on each item", methodName)

		// Execute method for each item
		for i, item := range collection {
			// Set iteration context
			p.setIterationContext(map[string]interface{}{
				iterVar: item,
			})

			// If item is a track, set currentTrackIndex for method execution
			if trackMap, ok := item.(map[string]interface{}); ok {
				if index, ok := trackMap["index"].(int); ok {
					p.currentTrackIndex = index
				} else if indexFloat, ok := trackMap["index"].(float64); ok {
					p.currentTrackIndex = int(indexFloat)
				}
			}

			// Execute the method
			if err := p.executeMethodOnItem(methodName, methodArgs); err != nil {
				log.Printf("  ⚠️  ForEach[%d]: Error executing method '%s': %v", i, methodName, err)
				// Continue with next item instead of failing completely
			}

			p.clearIterationContext()
		}

		log.Printf("✅ ForEach: Processed %d items from '%s' with method '%s'", len(collection), collectionName, methodName)
		return nil
	}

	// If no function or method specified, just iterate and set context (for chaining)
	log.Printf("⚠️  ForEach: No function or method specified, only setting iteration context")
	for i, item := range collection {
		p.setIterationContext(map[string]interface{}{
			iterVar: item,
		})

		if trackMap, ok := item.(map[string]interface{}); ok {
			if index, ok := trackMap["index"].(int); ok {
				p.currentTrackIndex = index
			} else if indexFloat, ok := trackMap["index"].(float64); ok {
				p.currentTrackIndex = int(indexFloat)
			}
		}

		log.Printf("  ForEach[%d]: Processing item (index=%d)", i, p.currentTrackIndex)
		p.clearIterationContext()
	}

	log.Printf("✅ ForEach: Processed %d items from '%s'", len(collection), collectionName)
	return nil
}

// parseMethodCallString parses a method call string like "track.add_fx(fxname=\"ReaEQ\")"
// Returns the method name (e.g., "add_fx") and parsed arguments
func (p *FunctionalDSLParser) parseMethodCallString(methodCallStr string) (string, gs.Args, error) {
	methodCallStr = strings.TrimSpace(methodCallStr)

	// Find the dot that separates object from method
	dotIndex := strings.Index(methodCallStr, ".")
	if dotIndex < 0 {
		return "", nil, fmt.Errorf("method call must contain a dot (e.g., track.add_fx(...))")
	}

	// Extract method name and parameters
	methodPart := methodCallStr[dotIndex+1:]
	
	// Find opening parenthesis
	parenIndex := strings.Index(methodPart, "(")
	if parenIndex < 0 {
		return "", nil, fmt.Errorf("method call must contain parentheses")
	}

	methodName := methodPart[:parenIndex]
	methodName = strings.TrimSpace(methodName)

	// Extract parameters string
	paramsStr := methodPart[parenIndex+1:]
	// Find matching closing parenthesis
	depth := 1
	closeIndex := -1
	for i, char := range paramsStr {
		if char == '(' {
			depth++
		} else if char == ')' {
			depth--
			if depth == 0 {
				closeIndex = i
				break
			}
		}
	}

	if closeIndex < 0 {
		return "", nil, fmt.Errorf("unclosed parentheses in method call")
	}

	paramsStr = paramsStr[:closeIndex]
	paramsStr = strings.TrimSpace(paramsStr)

	// Parse parameters into gs.Args
	args := make(gs.Args)
	if paramsStr != "" {
		// Simple parameter parsing: key="value" or key=value
		parts := strings.Split(paramsStr, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Split by = to get key and value
			eqIndex := strings.Index(part, "=")
			if eqIndex < 0 {
				continue
			}

			key := strings.TrimSpace(part[:eqIndex])
			valueStr := strings.TrimSpace(part[eqIndex+1:])

			// Parse value
			var value gs.Value
			if strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"") {
				// String value
				value = gs.Value{
					Kind: gs.ValueString,
					Str:  valueStr[1 : len(valueStr)-1], // Remove quotes
				}
			} else if valueStr == "true" {
				value = gs.Value{Kind: gs.ValueBool, Bool: true}
			} else if valueStr == "false" {
				value = gs.Value{Kind: gs.ValueBool, Bool: false}
			} else if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
				value = gs.Value{Kind: gs.ValueNumber, Num: num}
			} else {
				value = gs.Value{Kind: gs.ValueString, Str: valueStr}
			}

			args[key] = value
		}
	}

	return methodName, args, nil
}

// executeMethodOnItem executes a method on the current item in the iteration context
func (p *FunctionalDSLParser) executeMethodOnItem(methodName string, methodArgs gs.Args) error {
	// Convert snake_case to CamelCase for method name
	methodNameCamel := capitalizeMethodName(methodName)

	// Call the appropriate method on ReaperDSL
	// We need to use reflection or a switch statement
	switch methodNameCamel {
	case "SetSelected":
		return p.reaperDSL.SetSelected(methodArgs)
	case "SetMute":
		return p.reaperDSL.SetMute(methodArgs)
	case "SetSolo":
		return p.reaperDSL.SetSolo(methodArgs)
	case "SetVolume":
		return p.reaperDSL.SetVolume(methodArgs)
	case "SetPan":
		return p.reaperDSL.SetPan(methodArgs)
	case "SetName":
		return p.reaperDSL.SetName(methodArgs)
	case "AddFx":
		return p.reaperDSL.AddFx(methodArgs)
	case "AddMidi":
		return p.reaperDSL.AddMidi(methodArgs)
	case "NewClip":
		return p.reaperDSL.NewClip(methodArgs)
	case "Delete":
		return p.reaperDSL.Delete(methodArgs)
	case "DeleteClip":
		return p.reaperDSL.DeleteClip(methodArgs)
	default:
		return fmt.Errorf("unknown method: %s (converted from %s)", methodNameCamel, methodName)
	}
}

// capitalizeMethodName converts snake_case to CamelCase (track -> Track, set_selected -> SetSelected)
func capitalizeMethodName(name string) string {
	if name == "" {
		return name
	}

	parts := strings.Split(name, "_")
	var result strings.Builder
	for _, part := range parts {
		if part != "" {
			result.WriteString(strings.ToUpper(part[:1]) + strings.ToLower(part[1:]))
		}
	}

	return result.String()
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

// getArgsKeys returns a list of keys in the args map for debugging
func getArgsKeys(args gs.Args) []string {
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	return keys
}

func getDataKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}

// parseAndEvaluatePredicate parses a predicate string like "track.name == \"value\"" and evaluates it
func (p *FunctionalDSLParser) parseAndEvaluatePredicate(predStr string, item interface{}, iterVar string) bool {
	// Remove quotes and whitespace
	predStr = strings.TrimSpace(predStr)

	// Try to match patterns like:
	// - track.name == "value"
	// - track.name=="value"
	// - track.name != "value"

	// Find the operator
	var op string
	var opIndex int
	if idx := strings.Index(predStr, "=="); idx != -1 {
		op = "=="
		opIndex = idx
	} else if idx := strings.Index(predStr, "!="); idx != -1 {
		op = "!="
		opIndex = idx
	} else {
		return false
	}

	// Split into left (property) and right (value)
	left := strings.TrimSpace(predStr[:opIndex])
	right := strings.TrimSpace(predStr[opIndex+len(op):])

	// Extract property name from "track.name" or "iterVar.name"
	// The left side should be like "track.name" where "track" is the iterVar
	propParts := strings.Split(left, ".")
	if len(propParts) != 2 {
		return false
	}

	// Verify the first part matches the iteration variable (or is just the variable name)
	// For "track.name", we expect iterVar to be "track"
	if propParts[0] != iterVar && propParts[0] != "track" {
		return false
	}

	propName := propParts[1]

	// Check if right side is a boolean (true/false without quotes)
	rightTrimmed := strings.TrimSpace(right)
	isBooleanValue := rightTrimmed == "true" || rightTrimmed == "false"

	// Remove quotes from right side if present (for string values)
	if !isBooleanValue {
		right = strings.Trim(right, "\"")
	}

	// Get the property value from the item
	itemMap, ok := item.(map[string]interface{})
	if !ok {
		return false
	}

	itemValue, ok := itemMap[propName]
	if !ok {
		return false
	}

	// Handle boolean comparisons specially
	if isBooleanValue {
		expectedBool := rightTrimmed == "true"
		if itemBool, ok := itemValue.(bool); ok {
			if op == "==" {
				return itemBool == expectedBool
			} else if op == "!=" {
				return itemBool != expectedBool
			}
		}
		// If item value is not a bool, convert and compare as string
		itemValueStr := fmt.Sprintf("%t", itemValue)
		if op == "==" {
			return itemValueStr == rightTrimmed
		} else if op == "!=" {
			return itemValueStr != rightTrimmed
		}
		return false
	}

	// Compare values (for strings and numbers)
	var itemValueStr string
	switch v := itemValue.(type) {
	case string:
		itemValueStr = v
	case float64:
		itemValueStr = fmt.Sprintf("%g", v)
	case bool:
		itemValueStr = fmt.Sprintf("%t", v)
	default:
		itemValueStr = fmt.Sprintf("%v", v)
	}

	// Evaluate comparison
	if op == "==" {
		return itemValueStr == right
	} else if op == "!=" {
		return itemValueStr != right
	}

	return false
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

chain: clip_chain | midi_chain | fx_chain | volume_chain | pan_chain | mute_chain | solo_chain | name_chain | selected_chain | delete_chain | delete_clip_chain

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

// Deletion operations
delete_chain: ".delete" "(" ")"
delete_clip_chain: ".delete_clip" "(" delete_clip_params? ")"
delete_clip_params: delete_clip_param ("," SP delete_clip_param)*
delete_clip_param: "clip" "=" NUMBER
                 | "position" "=" NUMBER
                 | "bar" "=" NUMBER

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
