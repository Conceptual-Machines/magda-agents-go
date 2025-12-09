package daw

import (
	"reflect"
	"testing"
)

// TestTrackCreation tests track() creation with various parameters
func TestTrackCreation(t *testing.T) {
	tests := []struct {
		state   map[string]interface{}
		name    string
		dslCode string
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:    "create track with instrument",
			dslCode: `track(instrument="Serum")`,
			state:   nil,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
			},
			wantErr: false,
		},
		{
			name:    "create track with name",
			dslCode: `track(name="Bass Track")`,
			state:   nil,
			want: []map[string]interface{}{
				{
					"action": "create_track",
					"name":   "Bass Track",
					"index":  0,
				},
			},
			wantErr: false,
		},
		{
			name:    "create track with instrument and name",
			dslCode: `track(instrument="Piano", name="Piano Track")`,
			state:   nil,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Piano",
					"name":       "Piano Track",
					"index":      0,
				},
			},
			wantErr: false,
		},
		{
			name:    "create track with index",
			dslCode: `track(instrument="Drums", index=2)`,
			state:   nil,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Drums",
					"index":      2,
				},
			},
			wantErr: false,
		},
		{
			name:    "reference track by id",
			dslCode: `track(id=1).set_name(name="Renamed")`,
			state: map[string]interface{}{
				"tracks": []interface{}{
					map[string]interface{}{
						"index": 0,
						"name":  "Track 1",
					},
				},
			},
			want: []map[string]interface{}{
				{
					"action": "set_track_name",
					"track":  0,
					"name":   "Renamed",
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

// TestNewClip tests .new_clip() with various parameters
func TestNewClip(t *testing.T) {
	tests := []struct {
		name    string
		dslCode string
		state   map[string]interface{}
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:    "create clip at bar",
			dslCode: `track(instrument="Serum").new_clip(bar=1)`,
			want: []map[string]interface{}{
				{
					"action":      "create_track",
					"instrument":  "Serum",
					"index":       0,
				},
				{
					"action":      "create_clip_at_bar",
					"track":       0,
					"bar":         1,
					"length_bars": 4,
				},
			},
			wantErr: false,
		},
		{
			name:    "create clip at bar with length",
			dslCode: `track(instrument="Piano").new_clip(bar=2, length_bars=8)`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Piano",
					"index":      0,
				},
				{
					"action":      "create_clip_at_bar",
					"track":       0,
					"bar":         2,
					"length_bars": 8,
				},
			},
			wantErr: false,
		},
		{
			name:    "create clip at position",
			dslCode: `track(instrument="Drums").new_clip(position=10.5)`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Drums",
					"index":      0,
				},
				{
					"action":   "create_clip",
					"track":    0,
					"position": 10.5,
					"length":   4.0,
				},
			},
			wantErr: false,
		},
		{
			name:    "create clip at position with length",
			dslCode: `track(instrument="Bass").new_clip(position=5.0, length=2.0)`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Bass",
					"index":      0,
				},
				{
					"action":   "create_clip",
					"track":    0,
					"position": 5.0,
					"length":   2.0,
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

// TestAddMidi tests .add_midi() method
func TestAddMidi(t *testing.T) {
	tests := []struct {
		state   map[string]interface{}
		name    string
		dslCode string
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:    "add midi to track",
			dslCode: `track(instrument="Serum").add_midi(notes=[])`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action": "add_midi",
					"track":  0,
					"notes":  []interface{}{},
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

// TestAddFX tests .add_fx() method
func TestAddFX(t *testing.T) {
	tests := []struct {
		state   map[string]interface{}
		name    string
		dslCode string
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:    "add fx by name",
			dslCode: `track(instrument="Serum").add_fx(fxname="ReaEQ")`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action": "add_track_fx",
					"track":  0,
					"fxname": "ReaEQ",
				},
			},
			wantErr: false,
		},
		{
			name:    "add instrument",
			dslCode: `track().add_fx(instrument="Serum")`,
			want: []map[string]interface{}{
				{
					"action": "create_track",
					"index":  0,
				},
				{
					"action":     "add_instrument",
					"track":      0,
					"fxname":     "Serum",
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

// TestTrackProperties tests all track property setters
func TestTrackProperties(t *testing.T) {
	tests := []struct {
		state   map[string]interface{}
		name    string
		dslCode string
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:    "set volume",
			dslCode: `track(instrument="Serum").set_volume(volume_db=-3.0)`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action":    "set_track_volume",
					"track":     0,
					"volume_db": -3.0,
				},
			},
			wantErr: false,
		},
		{
			name:    "set pan",
			dslCode: `track(instrument="Piano").set_pan(pan=0.5)`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Piano",
					"index":      0,
				},
				{
					"action": "set_track_pan",
					"track":  0,
					"pan":    0.5,
				},
			},
			wantErr: false,
		},
		{
			name:    "set mute",
			dslCode: `track(instrument="Drums").set_mute(mute=true)`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Drums",
					"index":      0,
				},
				{
					"action": "set_track_mute",
					"track":  0,
					"mute":   true,
				},
			},
			wantErr: false,
		},
		{
			name:    "set solo",
			dslCode: `track(instrument="Bass").set_solo(solo=true)`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Bass",
					"index":      0,
				},
				{
					"action": "set_track_solo",
					"track":  0,
					"solo":   true,
				},
			},
			wantErr: false,
		},
		{
			name:    "set name",
			dslCode: `track(instrument="Serum").set_name(name="Lead")`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action": "set_track_name",
					"track":  0,
					"name":   "Lead",
				},
			},
			wantErr: false,
		},
		{
			name:    "set selected",
			dslCode: `track(instrument="Piano").set_selected(selected=true)`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Piano",
					"index":      0,
				},
				{
					"action":   "set_track_selected",
					"track":    0,
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

// TestDeleteOperations tests delete operations
func TestDeleteOperations(t *testing.T) {
	tests := []struct {
		state   map[string]interface{}
		name    string
		dslCode string
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:    "delete track by id",
			dslCode: `track(id=1).delete()`,
			state: map[string]interface{}{
				"tracks": []interface{}{
					map[string]interface{}{
						"index": 0,
						"name":  "Track 1",
					},
				},
			},
			want: []map[string]interface{}{
				{
					"action": "delete_track",
					"track":  0,
				},
			},
			wantErr: false,
		},
		{
			name:    "delete clip by bar",
			dslCode: `track(id=1).delete_clip(bar=2)`,
			state: map[string]interface{}{
				"tracks": []interface{}{
					map[string]interface{}{
						"index": 0,
						"name":  "Track 1",
					},
				},
			},
			want: []map[string]interface{}{
				{
					"action": "delete_clip",
					"track":  0,
					"bar":    2,
				},
			},
			wantErr: false,
		},
		{
			name:    "delete clip by position",
			dslCode: `track(id=1).delete_clip(position=10.5)`,
			state: map[string]interface{}{
				"tracks": []interface{}{
					map[string]interface{}{
						"index": 0,
						"name":  "Track 1",
					},
				},
			},
			want: []map[string]interface{}{
				{
					"action":   "delete_clip",
					"track":    0,
					"position": 10.5,
				},
			},
			wantErr: false,
		},
		{
			name:    "delete clip by clip index",
			dslCode: `track(id=1).delete_clip(clip=0)`,
			state: map[string]interface{}{
				"tracks": []interface{}{
					map[string]interface{}{
						"index": 0,
						"name":  "Track 1",
					},
				},
			},
			want: []map[string]interface{}{
				{
					"action": "delete_clip",
					"track":  0,
					"clip":   0,
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

			if tt.state != nil {
				parser.SetState(tt.state)
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

// TestFilterOperations tests filter() functional operations
func TestFilterOperations(t *testing.T) {
	tests := []struct {
		state   map[string]interface{}
		name    string
		dslCode string
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:    "filter tracks by name and delete",
			dslCode: `filter(tracks, track.name=="Nebula Drift").delete()`,
			state: map[string]interface{}{
				"tracks": []interface{}{
					map[string]interface{}{
						"index": 0,
						"name":  "Nebula Drift",
					},
					map[string]interface{}{
						"index": 1,
						"name":  "Other Track",
					},
				},
			},
			want: []map[string]interface{}{
				{
					"action": "delete_track",
					"track":  0,
				},
			},
			wantErr: false,
		},
		{
			name:    "filter tracks by name and set selected",
			dslCode: `filter(tracks, track.name=="FX Track").set_selected(selected=true)`,
			state: map[string]interface{}{
				"tracks": []interface{}{
					map[string]interface{}{
						"index": 0,
						"name":  "FX Track",
					},
					map[string]interface{}{
						"index": 1,
						"name":  "Other Track",
					},
				},
			},
			want: []map[string]interface{}{
				{
					"action":   "set_track_selected",
					"track":    0,
					"selected": true,
				},
			},
			wantErr: false,
		},
		{
			name:    "filter tracks by mute status",
			dslCode: `filter(tracks, track.muted==true).set_mute(mute=false)`,
			state: map[string]interface{}{
				"tracks": []interface{}{
					map[string]interface{}{
						"index": 0,
						"name":  "Track 1",
						"muted": true,
					},
					map[string]interface{}{
						"index": 1,
						"name":  "Track 2",
						"muted": false,
					},
				},
			},
			want: []map[string]interface{}{
				{
					"action": "set_track_mute",
					"track":  0,
					"mute":   false,
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

			if tt.state != nil {
				parser.SetState(tt.state)
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

// TestMethodChaining tests complex method chaining
func TestMethodChaining(t *testing.T) {
	tests := []struct {
		state   map[string]interface{}
		name    string
		dslCode string
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:    "create track with multiple operations",
			dslCode: `track(instrument="Serum", name="Lead").new_clip(bar=1, length_bars=4).add_midi(notes=[]).set_volume(volume_db=-3.0).set_pan(pan=0.5)`,
			want: []map[string]interface{}{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"name":       "Lead",
					"index":      0,
				},
				{
					"action":      "create_clip_at_bar",
					"track":       0,
					"bar":         1,
					"length_bars": 4,
				},
				{
					"action": "add_midi",
					"track":  0,
					"notes":  []interface{}{},
				},
				{
					"action":    "set_track_volume",
					"track":     0,
					"volume_db": -3.0,
				},
				{
					"action": "set_track_pan",
					"track":  0,
					"pan":    0.5,
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

