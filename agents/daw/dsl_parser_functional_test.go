package daw

import (
	"reflect"
	"testing"
)

func TestFunctionalDSLParser_SetSelected(t *testing.T) {
	tests := []struct {
		name    string
		dslCode string
		want    []map[string]any
		wantErr bool
	}{
		{
			name:    "track with set_selected true",
			dslCode: `track(instrument="Serum").set_selected(selected=true)`,
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
			name:    "track with set_selected false",
			dslCode: `track(instrument="Piano").set_selected(selected=false)`,
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
			dslCode: `track(instrument="Serum").set_name(name="Bass").set_selected(selected=true)`,
			want: []map[string]any{
				{
					"action":     "create_track",
					"instrument": "Serum",
					"index":      0,
				},
				{
					"action": "set_track",
					"track":  0,
					"name":   "Bass",
				},
				{
					"action":   "set_track",
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
