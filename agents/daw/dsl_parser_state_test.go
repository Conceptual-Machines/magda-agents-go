package daw

import (
	"testing"
)

func TestFunctionalDSLParser_TrackIndexWithExistingTracks(t *testing.T) {
	tests := []struct {
		name          string
		state         map[string]any
		dslCode       string
		expectedIndex int
	}{
		{
			name:          "no existing tracks",
			state:         nil,
			dslCode:       `track(name="New Track")`,
			expectedIndex: 0,
		},
		{
			name: "one existing track",
			state: map[string]any{
				"tracks": []any{
					map[string]any{"index": 0, "name": "Existing Track"},
				},
			},
			dslCode:       `track(name="New Track")`,
			expectedIndex: 1,
		},
		{
			name: "two existing tracks",
			state: map[string]any{
				"tracks": []any{
					map[string]any{"index": 0, "name": "Track 1"},
					map[string]any{"index": 1, "name": "Track 2"},
				},
			},
			dslCode:       `track(name="New Track")`,
			expectedIndex: 2,
		},
		{
			name: "nested state structure",
			state: map[string]any{
				"state": map[string]any{
					"tracks": []any{
						map[string]any{"index": 0, "name": "Track 1"},
						map[string]any{"index": 1, "name": "Track 2"},
						map[string]any{"index": 2, "name": "Track 3"},
					},
				},
			},
			dslCode:       `track(name="New Track")`,
			expectedIndex: 3,
		},
		{
			name: "multiple new tracks start from correct index",
			state: map[string]any{
				"tracks": []any{
					map[string]any{"index": 0, "name": "Existing"},
				},
			},
			dslCode:       `track(name="First New") track(name="Second New")`,
			expectedIndex: 1, // First new track at index 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewFunctionalDSLParser()
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			if tt.state != nil {
				parser.SetState(tt.state)
			}

			actions, err := parser.ParseDSL(tt.dslCode)
			if err != nil {
				t.Fatalf("ParseDSL failed: %v", err)
			}

			if len(actions) == 0 {
				t.Fatal("No actions returned")
			}

			// Check first create_track action has correct index
			firstAction := actions[0]
			if firstAction["action"] != "create_track" {
				t.Errorf("Expected first action to be create_track, got %v", firstAction["action"])
			}

			index, ok := firstAction["index"].(int)
			if !ok {
				t.Fatalf("Index not found or not an int: %v", firstAction["index"])
			}

			if index != tt.expectedIndex {
				t.Errorf("Expected track index %d, got %d", tt.expectedIndex, index)
			}
		})
	}
}

func TestFunctionalDSLParser_ChainedActionsUseCorrectTrackIndex(t *testing.T) {
	// Test that when creating a track with existing tracks,
	// subsequent chained actions use the correct track index
	parser, err := NewFunctionalDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	state := map[string]any{
		"tracks": []any{
			map[string]any{"index": 0, "name": "Existing Track"},
		},
	}
	parser.SetState(state)

	actions, err := parser.ParseDSL(`track(name="New Track").new_clip(bar=1, length_bars=2)`)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	if len(actions) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(actions))
	}

	// First action: create_track at index 1 (after existing track 0)
	createTrack := actions[0]
	if createTrack["action"] != "create_track" {
		t.Errorf("Expected create_track, got %v", createTrack["action"])
	}
	if createTrack["index"] != 1 {
		t.Errorf("Expected track index 1, got %v", createTrack["index"])
	}

	// Second action: create_clip_at_bar referencing track 1
	createClip := actions[1]
	if createClip["action"] != "create_clip_at_bar" {
		t.Errorf("Expected create_clip_at_bar, got %v", createClip["action"])
	}
	if createClip["track"] != 1 {
		t.Errorf("Expected clip to reference track 1, got %v", createClip["track"])
	}
}

