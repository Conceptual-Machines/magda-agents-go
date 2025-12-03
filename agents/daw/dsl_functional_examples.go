package daw

import (
	"fmt"
	"log"
)

// ExampleREAPERState is example REAPER state structure for testing.
var ExampleREAPERState = map[string]interface{}{
	"state": map[string]interface{}{
		"tracks": []interface{}{
			map[string]interface{}{
				"index":    0,
				"name":     "Drums",
				"selected": false,
				"fx": []interface{}{
					map[string]interface{}{"name": "ReaEQ", "enabled": true},
					map[string]interface{}{"name": "ReaComp", "enabled": false},
				},
				"clips": []interface{}{
					map[string]interface{}{"start": 0.0, "length": 4.0, "name": "Kick"},
					map[string]interface{}{"start": 4.0, "length": 4.0, "name": "Snare"},
				},
			},
			map[string]interface{}{
				"index":    1,
				"name":     "FX",
				"selected": true,
				"fx": []interface{}{
					map[string]interface{}{"name": "ReaVerb", "enabled": true},
				},
				"clips": []interface{}{},
			},
			map[string]interface{}{
				"index":    2,
				"name":     "Bass",
				"selected": false,
				"fx":       []interface{}{},
				"clips": []interface{}{
					map[string]interface{}{"start": 0.0, "length": 8.0, "name": "BassLine"},
				},
			},
		},
	},
}

// ExampleFilterTracks shows low-level filter implementation for tracks.
// This demonstrates what happens internally when calling: filter(tracks, track.name == "FX")
func ExampleFilterTracks() {
	stateMap := ExampleREAPERState["state"].(map[string]interface{})
	tracks := stateMap["tracks"].([]interface{})
	filtered := make([]interface{}, 0)

	// This is what happens internally in filter():
	for _, trackInterface := range tracks {
		track := trackInterface.(map[string]interface{})

		// Set iteration context: track = current track dict
		iterationContext := map[string]interface{}{
			"track": track,
		}
		_ = iterationContext // Use in predicate evaluation

		// Evaluate predicate: track.name == "FX"
		// 1. Resolve "track" from iteration context -> {...}
		trackObj := iterationContext["track"].(map[string]interface{})

		// 2. Access property "name" -> "FX"
		trackName := trackObj["name"].(string)

		// 3. Compare: "FX" == "FX" -> true
		predicateResult := trackName == "FX"

		// 4. If True, include in result
		if predicateResult {
			filtered = append(filtered, track)
		}
	}

	trackNames := make([]string, len(filtered))
	for i, t := range filtered {
		trackNames[i] = t.(map[string]interface{})["name"].(string)
	}
	log.Printf("Filtered tracks: %v", trackNames)
	// Output: Filtered tracks: [FX]
}

// ExampleFilterFXChain shows low-level filter implementation for FX chain.
// This demonstrates: filter(fx_chain, fx.enabled == true)
func ExampleFilterFXChain() {
	stateMap := ExampleREAPERState["state"].(map[string]interface{})
	tracks := stateMap["tracks"].([]interface{})
	track := tracks[0].(map[string]interface{})
	fxChain := track["fx"].([]interface{})
	filtered := make([]interface{}, 0)

	// This is what happens internally in filter():
	for _, fxInterface := range fxChain {
		fx := fxInterface.(map[string]interface{})

		// Set iteration context: fx = current FX dict
		iterationContext := map[string]interface{}{
			"fx": fx,
		}
		_ = iterationContext

		// Evaluate predicate: fx.enabled == true
		fxObj := iterationContext["fx"].(map[string]interface{})
		fxEnabled := fxObj["enabled"].(bool)
		predicateResult := fxEnabled == true

		if predicateResult {
			filtered = append(filtered, fx)
		}
	}

	fxNames := make([]string, len(filtered))
	for i, f := range filtered {
		fxNames[i] = f.(map[string]interface{})["name"].(string)
	}
	log.Printf("Enabled FX: %v", fxNames)
	// Output: Enabled FX: [ReaEQ]
}

// ExampleFilterClips shows low-level filter implementation for clips.
// This demonstrates: filter(clips, clip.start < 4.0)
func ExampleFilterClips() {
	stateMap := ExampleREAPERState["state"].(map[string]interface{})
	tracks := stateMap["tracks"].([]interface{})
	track := tracks[0].(map[string]interface{})
	clips := track["clips"].([]interface{})
	filtered := make([]interface{}, 0)

	// This is what happens internally in filter():
	for _, clipInterface := range clips {
		clip := clipInterface.(map[string]interface{})

		// Set iteration context: clip = current clip dict
		iterationContext := map[string]interface{}{
			"clip": clip,
		}
		_ = iterationContext

		// Evaluate predicate: clip.start < 4.0
		clipObj := iterationContext["clip"].(map[string]interface{})
		clipStart := clipObj["start"].(float64)
		predicateResult := clipStart < 4.0

		if predicateResult {
			filtered = append(filtered, clip)
		}
	}

	clipNames := make([]string, len(filtered))
	for i, c := range filtered {
		clipNames[i] = c.(map[string]interface{})["name"].(string)
	}
	log.Printf("Clips starting before 4.0: %v", clipNames)
	// Output: Clips starting before 4.0: [Kick]
}

// ExampleMapOverTracks shows low-level map implementation.
// This demonstrates: map(@get_name, tracks)
func ExampleMapOverTracks() {
	getName := func(track interface{}) interface{} {
		trackMap := track.(map[string]interface{})
		return trackMap["name"]
	}

	stateMap := ExampleREAPERState["state"].(map[string]interface{})
	tracks := stateMap["tracks"].([]interface{})
	mapped := make([]interface{}, 0, len(tracks))

	// This is what happens internally in map():
	for _, trackInterface := range tracks {
		track := trackInterface

		// Set iteration context
		iterationContext := map[string]interface{}{
			"track": track,
		}
		_ = iterationContext

		// Call function with current item
		result := getName(track)

		// Store result
		mapped = append(mapped, result)
	}

	trackNames := make([]string, len(mapped))
	for i, m := range mapped {
		trackNames[i] = m.(string)
	}
	log.Printf("Mapped track names: %v", trackNames)
	// Output: Mapped track names: [Drums FX Bass]
}

// ExampleForEachWithSideEffects shows low-level for_each implementation.
// This demonstrates: for_each(filter(tracks, track.selected == true), @add_fx)
func ExampleForEachWithSideEffects() {
	stateMap := ExampleREAPERState["state"].(map[string]interface{})
	tracks := stateMap["tracks"].([]interface{})
	actions := make([]map[string]interface{}, 0)

	addFX := func(track interface{}) {
		trackMap := track.(map[string]interface{})
		action := map[string]interface{}{
			"action":  "add_track_fx",
			"track":   trackMap["index"],
			"fxname":  "ReaVerb",
		}
		actions = append(actions, action)
	}

	// Filter tracks first
	filteredTracks := make([]interface{}, 0)
	for _, trackInterface := range tracks {
		track := trackInterface.(map[string]interface{})
		if selected, ok := track["selected"].(bool); ok && selected {
			filteredTracks = append(filteredTracks, track)
		}
	}

	// Then for each filtered track, execute action function
	for _, trackInterface := range filteredTracks {
		track := trackInterface

		// Set iteration context
		iterationContext := map[string]interface{}{
			"track": track,
		}
		_ = iterationContext

		// Execute action function
		addFX(track)
	}

	log.Printf("Actions emitted: %v", actions)
	// Output: Actions emitted: [map[action:add_track_fx fxname:ReaVerb track:1]]
}

// ExampleComplexFilterExpression shows complex filter with multiple conditions.
// This demonstrates: filter(tracks, track.name == "FX" && track.selected == true)
func ExampleComplexFilterExpression() {
	stateMap := ExampleREAPERState["state"].(map[string]interface{})
	tracks := stateMap["tracks"].([]interface{})
	filtered := make([]interface{}, 0)

	for _, trackInterface := range tracks {
		track := trackInterface.(map[string]interface{})
		iterationContext := map[string]interface{}{
			"track": track,
		}
		_ = iterationContext

		trackObj := track

		// Evaluate: track.name == "FX" && track.selected == true
		condition1 := trackObj["name"].(string) == "FX"
		condition2 := trackObj["selected"].(bool) == true
		predicateResult := condition1 && condition2

		if predicateResult {
			filtered = append(filtered, track)
		}
	}

	trackNames := make([]string, len(filtered))
	for i, t := range filtered {
		trackNames[i] = t.(map[string]interface{})["name"].(string)
	}
	log.Printf("Complex filter result: %v", trackNames)
	// Output: Complex filter result: [FX]
}

// ExampleNestedPropertyAccess shows nested property access in predicate.
// This demonstrates: filter(tracks, track.fx[0].name == "ReaEQ")
func ExampleNestedPropertyAccess() {
	stateMap := ExampleREAPERState["state"].(map[string]interface{})
	tracks := stateMap["tracks"].([]interface{})
	filtered := make([]interface{}, 0)

	for _, trackInterface := range tracks {
		track := trackInterface.(map[string]interface{})
		iterationContext := map[string]interface{}{
			"track": track,
		}
		_ = iterationContext

		trackObj := track

		// Evaluate: track.fx[0].name == "ReaEQ"
		// Navigate nested properties
		fxList, ok := trackObj["fx"].([]interface{})
		if ok && len(fxList) > 0 {
			firstFX := fxList[0].(map[string]interface{})
			fxName := firstFX["name"].(string)
			predicateResult := fxName == "ReaEQ"

			if predicateResult {
				filtered = append(filtered, track)
			}
		}
	}

	trackNames := make([]string, len(filtered))
	for i, t := range filtered {
		trackNames[i] = t.(map[string]interface{})["name"].(string)
	}
	log.Printf("Tracks with ReaEQ as first FX: %v", trackNames)
	// Output: Tracks with ReaEQ as first FX: [Drums]
}

// RunAllExamples runs all functional examples.
func RunAllExamples() {
	fmt.Println("=== Filter Tracks ===")
	ExampleFilterTracks()

	fmt.Println("\n=== Filter FX Chain ===")
	ExampleFilterFXChain()

	fmt.Println("\n=== Filter Clips ===")
	ExampleFilterClips()

	fmt.Println("\n=== Map Over Tracks ===")
	ExampleMapOverTracks()

	fmt.Println("\n=== For Each with Side Effects ===")
	ExampleForEachWithSideEffects()

	fmt.Println("\n=== Complex Filter ===")
	ExampleComplexFilterExpression()

	fmt.Println("\n=== Nested Property Access ===")
	ExampleNestedPropertyAccess()
}

