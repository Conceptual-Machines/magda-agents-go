package daw

import (
	"reflect"
	"testing"
)

func TestDSLParser_ParseDSL(t *testing.T) {
	tests := []struct {
		name    string
		dslCode string
		want    []map[string]any
		wantErr bool
	}{
		{
			name:    "simple track creation",
			dslCode: `track(instrument="Serum")`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
			},
			wantErr: false,
		},
		{
			name:    "track with name",
			dslCode: `track(instrument="Serum", name="Bass")`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"name":       "Bass",
					"index":      0,
				},
			},
			wantErr: false,
		},
		{
			name:    "track with index",
			dslCode: `track(instrument="Serum", index=2)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      2,
				},
			},
			wantErr: false,
		},
		{
			name:    "track with clip chain",
			dslCode: `track(instrument="Serum").newClip(bar=3, length_bars=4)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action":      "create_clip_at_bar",
					"track":       0,
					"bar":         3,
					"length_bars": 4,
				},
			},
			wantErr: false,
		},
		{
			name:    "track with time-based clip",
			dslCode: `track(instrument="Serum").newClip(start=1.5, length=2.0)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action":   "create_clip",
					"track":    0,
					"position": 1.5,
					"length":   2.0,
				},
			},
			wantErr: false,
		},
		{
			name:    "track with clip and volume",
			dslCode: `track(instrument="Serum").newClip(bar=3, length_bars=4).setVolume(volume_db=-3.0)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action":      "create_clip_at_bar",
					"track":       0,
					"bar":         3,
					"length_bars": 4,
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
			name:    "track with pan",
			dslCode: `track(instrument="Serum").setPan(pan=0.5)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action": "set_track",
					"track":  0,
					"pan":    0.5,
				},
			},
			wantErr: false,
		},
		{
			name:    "track with mute",
			dslCode: `track(instrument="Serum").setMute(mute=true)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action": "set_track",
					"track":  0,
					"mute":   true,
				},
			},
			wantErr: false,
		},
		{
			name:    "track with solo",
			dslCode: `track(instrument="Serum").setSolo(solo=false)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action": "set_track",
					"track":  0,
					"solo":   false,
				},
			},
			wantErr: false,
		},
		{
			name:    "track with name setter",
			dslCode: `track(instrument="Serum").setName(name="My Track")`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action": "set_track",
					"track":  0,
					"name":   "My Track",
				},
			},
			wantErr: false,
		},
		{
			name:    "track with FX",
			dslCode: `track(instrument="Serum").addFX(fxname="ReaEQ")`,
			want: []map[string]any{
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
			name:    "multiple tracks",
			dslCode: `track(instrument="Serum").newClip(bar=1, length_bars=4) track(instrument="Piano").newClip(bar=2, length_bars=4)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action":      "create_clip_at_bar",
					"track":       0,
					"bar":         1,
					"length_bars": 4,
				},
				{
					"action":     "create_track",
					"instrument": "Piano",
					"index":      1,
				},
				{
					"action":      "create_clip_at_bar",
					"track":       1,
					"bar":         2,
					"length_bars": 4,
				},
			},
			wantErr: false,
		},
		{
			name:    "clip with default length",
			dslCode: `track(instrument="Serum").newClip(bar=3)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action":      "create_clip_at_bar",
					"track":       0,
					"bar":         3,
					"length_bars": 4, // Default
				},
			},
			wantErr: false,
		},
		{
			name:    "empty DSL",
			dslCode: ``,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid clip without track",
			dslCode: `.newClip(bar=3)`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "clip without bar or start",
			dslCode: `track(instrument="Serum").newClip(length_bars=4)`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewDSLParser()
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

func TestDSLParser_extractParams(t *testing.T) {
	tests := []struct {
		name     string
		call     string
		want     map[string]string
		wantKeys []string
	}{
		{
			name:     "single string param",
			call:     `track(instrument="Serum")`,
			wantKeys: []string{"instrument"},
		},
		{
			name:     "multiple params",
			call:     `track(instrument="Serum", name="Bass", index=2)`,
			wantKeys: []string{"instrument", "name", "index"},
		},
		{
			name:     "params with spaces",
			call:     `track( instrument = "Serum" , name = "Bass" )`,
			wantKeys: []string{"instrument", "name"},
		},
		{
			name:     "empty params",
			call:     `track()`,
			wantKeys: []string{},
		},
		{
			name:     "numeric params",
			call:     `newClip(bar=3, length_bars=4)`,
			wantKeys: []string{"bar", "length_bars"},
		},
		{
			name:     "boolean params",
			call:     `setMute(mute=true)`,
			wantKeys: []string{"mute"},
		},
		{
			name:     "float params",
			call:     `setVolume(volume_db=-3.5)`,
			wantKeys: []string{"volume_db"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewDSLParser()
			got := parser.extractParams(tt.call)

			// Check that all expected keys are present
			for _, key := range tt.wantKeys {
				if _, ok := got[key]; !ok {
					t.Errorf("extractParams() missing key %s, got %v", key, got)
				}
			}

			// Check specific values for known cases
			if tt.name == "single string param" {
				if got["instrument"] != TestInstrumentName {
					t.Errorf("extractParams() instrument = %v, want %s", got["instrument"], TestInstrumentName)
				}
			}
			if tt.name == "multiple params" {
				if got["instrument"] != TestInstrumentName {
					t.Errorf("extractParams() instrument = %v, want %s", got["instrument"], TestInstrumentName)
				}
				if got["name"] != "Bass" {
					t.Errorf("extractParams() name = %v, want Bass", got["name"])
				}
				if got["index"] != "2" {
					t.Errorf("extractParams() index = %v, want 2", got["index"])
				}
			}
		})
	}
}

func TestDSLParser_splitMethodChains(t *testing.T) {
	tests := []struct {
		name      string
		dslCode   string
		wantLen   int
		wantFirst string
	}{
		{
			name:      "single method",
			dslCode:   `track(instrument="Serum")`,
			wantLen:   1,
			wantFirst: `track(instrument="Serum")`,
		},
		{
			name:      "two methods",
			dslCode:   `track(instrument="Serum").newClip(bar=3)`,
			wantLen:   2,
			wantFirst: `track(instrument="Serum")`,
		},
		{
			name:      "three methods",
			dslCode:   `track(instrument="Serum").newClip(bar=3).setVolume(volume_db=-3.0)`,
			wantLen:   3,
			wantFirst: `track(instrument="Serum")`,
		},
		{
			name:      "nested strings",
			dslCode:   `track(instrument="VSTi: Serum (Xfer Records)").newClip(bar=3)`,
			wantLen:   2,
			wantFirst: `track(instrument="VSTi: Serum (Xfer Records)")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewDSLParser()
			got := parser.splitMethodChains(tt.dslCode)

			if len(got) != tt.wantLen {
				t.Errorf("splitMethodChains() len = %v, want %v", len(got), tt.wantLen)
			}

			if len(got) > 0 && got[0] != tt.wantFirst {
				t.Errorf("splitMethodChains() first = %v, want %v", got[0], tt.wantFirst)
			}
		})
	}
}

func TestDSLParser_parseTrackCall(t *testing.T) {
	tests := []struct {
		name        string
		call        string
		wantAction  string
		wantIndex   int
		wantErr     bool
		checkFields map[string]any
	}{
		{
			name:       "basic track",
			call:       `track()`,
			wantAction: "create_track",
			wantIndex:  0,
			wantErr:    false,
		},
		{
			name:       "track with instrument",
			call:       `track(instrument="Serum")`,
			wantAction: "create_track",
			wantIndex:  0,
			wantErr:    false,
			checkFields: map[string]any{
				"instrument": "Serum",
			},
		},
		{
			name:       "track with name",
			call:       `track(name="Bass")`,
			wantAction: "create_track",
			wantIndex:  0,
			wantErr:    false,
			checkFields: map[string]any{
				"name": "Bass",
			},
		},
		{
			name:       "track with index",
			call:       `track(index=5)`,
			wantAction: "create_track",
			wantIndex:  5,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewDSLParser()
			got, gotIndex, err := parser.parseTrackCall(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTrackCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got["action"] != tt.wantAction {
				t.Errorf("parseTrackCall() action = %v, want %v", got["action"], tt.wantAction)
			}
			if gotIndex != tt.wantIndex {
				t.Errorf("parseTrackCall() index = %v, want %v", gotIndex, tt.wantIndex)
			}
			for key, wantValue := range tt.checkFields {
				if got[key] != wantValue {
					t.Errorf("parseTrackCall() %s = %v, want %v", key, got[key], wantValue)
				}
			}
		})
	}
}

func TestDSLParser_parseClipCall(t *testing.T) {
	tests := []struct {
		name        string
		call        string
		trackIndex  int
		wantAction  string
		wantErr     bool
		checkFields map[string]any
	}{
		{
			name:       "clip with bar",
			call:       `.newClip(bar=3, length_bars=4)`,
			trackIndex: 0,
			wantAction: "create_clip_at_bar",
			wantErr:    false,
			checkFields: map[string]any{
				"track":       0,
				"bar":         3,
				"length_bars": 4,
			},
		},
		{
			name:       "clip with start",
			call:       `.newClip(start=1.5, length=2.0)`,
			trackIndex: 0,
			wantAction: "create_clip",
			wantErr:    false,
			checkFields: map[string]any{
				"track":    0,
				"position": 1.5,
				"length":   2.0,
			},
		},
		{
			name:       "clip with position",
			call:       `.newClip(position=2.5, length=1.0)`,
			trackIndex: 1,
			wantAction: "create_clip",
			wantErr:    false,
			checkFields: map[string]any{
				"track":    1,
				"position": 2.5,
				"length":   1.0,
			},
		},
		{
			name:       "clip default length",
			call:       `.newClip(bar=3)`,
			trackIndex: 0,
			wantAction: "create_clip_at_bar",
			wantErr:    false,
			checkFields: map[string]any{
				"track":       0,
				"bar":         3,
				"length_bars": 4, // Default
			},
		},
		{
			name:       "no track context",
			call:       `.newClip(bar=3)`,
			trackIndex: -1,
			wantErr:    true,
		},
		{
			name:       "missing bar and start",
			call:       `.newClip(length_bars=4)`,
			trackIndex: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewDSLParser()
			got, err := parser.parseClipCall(tt.call, tt.trackIndex)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseClipCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got["action"] != tt.wantAction {
					t.Errorf("parseClipCall() action = %v, want %v", got["action"], tt.wantAction)
				}
				for key, wantValue := range tt.checkFields {
					if got[key] != wantValue {
						t.Errorf("parseClipCall() %s = %v, want %v", key, got[key], wantValue)
					}
				}
			}
		})
	}
}

func TestDSLParser_parseVolumeCall(t *testing.T) {
	tests := []struct {
		name       string
		call       string
		trackIndex int
		wantErr    bool
		wantVolume float64
	}{
		{
			name:       "positive volume",
			call:       `.setVolume(volume_db=3.0)`,
			trackIndex: 0,
			wantErr:    false,
			wantVolume: 3.0,
		},
		{
			name:       "negative volume",
			call:       `.setVolume(volume_db=-3.5)`,
			trackIndex: 0,
			wantErr:    false,
			wantVolume: -3.5,
		},
		{
			name:       "no track context",
			call:       `.setVolume(volume_db=0.0)`,
			trackIndex: -1,
			wantErr:    true,
		},
		{
			name:       "missing volume_db",
			call:       `.setVolume()`,
			trackIndex: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewDSLParser()
			got, err := parser.parseVolumeCall(tt.call, tt.trackIndex)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVolumeCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got["action"] != "set_track_volume" {
					t.Errorf("parseVolumeCall() action = %v, want set_track_volume", got["action"])
				}
				if got["volume_db"] != tt.wantVolume {
					t.Errorf("parseVolumeCall() volume_db = %v, want %v", got["volume_db"], tt.wantVolume)
				}
			}
		})
	}
}

func TestDSLParser_parsePanCall(t *testing.T) {
	tests := []struct {
		name       string
		call       string
		trackIndex int
		wantErr    bool
		wantPan    float64
	}{
		{
			name:       "center pan",
			call:       `.setPan(pan=0.0)`,
			trackIndex: 0,
			wantErr:    false,
			wantPan:    0.0,
		},
		{
			name:       "right pan",
			call:       `.setPan(pan=1.0)`,
			trackIndex: 0,
			wantErr:    false,
			wantPan:    1.0,
		},
		{
			name:       "left pan",
			call:       `.setPan(pan=-0.5)`,
			trackIndex: 0,
			wantErr:    false,
			wantPan:    -0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewDSLParser()
			got, err := parser.parsePanCall(tt.call, tt.trackIndex)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePanCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got["pan"] != tt.wantPan {
					t.Errorf("parsePanCall() pan = %v, want %v", got["pan"], tt.wantPan)
				}
			}
		})
	}
}

func TestDSLParser_parseMuteCall(t *testing.T) {
	tests := []struct {
		name       string
		call       string
		trackIndex int
		wantErr    bool
		wantMute   bool
	}{
		{
			name:       "mute true",
			call:       `.setMute(mute=true)`,
			trackIndex: 0,
			wantErr:    false,
			wantMute:   true,
		},
		{
			name:       "mute false",
			call:       `.setMute(mute=false)`,
			trackIndex: 0,
			wantErr:    false,
			wantMute:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewDSLParser()
			got, err := parser.parseMuteCall(tt.call, tt.trackIndex)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMuteCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got["mute"] != tt.wantMute {
					t.Errorf("parseMuteCall() mute = %v, want %v", got["mute"], tt.wantMute)
				}
			}
		})
	}
}

func TestDSLParser_parseFXCall(t *testing.T) {
	tests := []struct {
		name       string
		call       string
		trackIndex int
		wantAction string
		wantErr    bool
		wantFXName string
	}{
		{
			name:       "add FX",
			call:       `.addFX(fxname="ReaEQ")`,
			trackIndex: 0,
			wantAction: "add_track_fx",
			wantErr:    false,
			wantFXName: "ReaEQ",
		},
		{
			name:       "add instrument",
			call:       `.addInstrument(instrument="Serum")`,
			trackIndex: 0,
			wantAction: "add_instrument",
			wantErr:    false,
			wantFXName: "Serum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewDSLParser()
			got, err := parser.parseFXCall(tt.call, tt.trackIndex)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFXCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got["action"] != tt.wantAction {
					t.Errorf("parseFXCall() action = %v, want %v", got["action"], tt.wantAction)
				}
				if got["fxname"] != tt.wantFXName {
					t.Errorf("parseFXCall() fxname = %v, want %v", got["fxname"], tt.wantFXName)
				}
			}
		})
	}
}
