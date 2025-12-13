package daw

import (
	"reflect"
	"testing"
)

func TestFunctionalDSLParser_SetTrack(t *testing.T) {
	tests := []struct {
		name    string
		dslCode string
		want    []map[string]any
		wantErr bool
	}{
		{
			name:    "track with set_track selected true",
			dslCode: `track(instrument="Serum").set_track(selected=true)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action":   "set_track",
					"track":    0,
					"selected": true,
				},
			},
			wantErr: false,
		},
		{
			name:    "track with set_track selected false",
			dslCode: `track(instrument="Piano").set_track(selected=false)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Piano",
					"index":      0,
				},
				{
					"action":   "set_track",
					"track":    0,
					"selected": false,
				},
			},
			wantErr: false,
		},
		{
			name:    "track with multiple operations including selection",
			dslCode: `track(instrument="Serum").set_track(name="Bass", selected=true)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action":   "set_track",
					"track":    0,
					"name":     "Bass",
					"selected": true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewFunctionalDSLParser()
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			got, err := parser.ParseDSL(tt.dslCode)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDSL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDSL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFunctionalDSLParser_SetClipLength(t *testing.T) {
	parser, err := NewFunctionalDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	state := map[string]any{
		"tracks": []any{
			map[string]any{
				"index": 0,
				"name":  "Track 1",
				"clips": []any{
					map[string]any{
						"index":    0,
						"position": 0.0,
						"length":   2.0,
						"track":    0,
					},
					map[string]any{
						"index":    1,
						"position": 4.0,
						"length":   2.0,
						"track":    0,
					},
				},
			},
		},
	}
	parser.SetState(state)

	// Test setting clip length via filter
	dslCode := `filter(clips, clip.length < 3.0).set_clip(length=4.0)`
	actions, err := parser.ParseDSL(dslCode)
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("Should generate at least one action")
	}

	// Check that we got set_clip actions with length property
	hasSetClipLength := false
	for _, action := range actions {
		if actionType, ok := action["action"].(string); ok && actionType == "set_clip" {
			if length, ok := action["length"].(float64); ok {
				if length != 4.0 {
					t.Errorf("Length should be 4.0, got %v", length)
				}
				hasSetClipLength = true
			}
		}
	}
	if !hasSetClipLength {
		t.Error("Should have set_clip action with length property")
	}
}
