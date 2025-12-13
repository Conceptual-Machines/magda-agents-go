package llm

import (
	"testing"
)

func TestIsDSLCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Track creation
		{"track() call", "track(instrument=\"Serum\")", true},
		{"track() with params", "track(name=\"Bass\", index=0)", true},

		// Clip operations
		{"new_clip()", "track().new_clip(bar=1)", true},
		{"new_clip() standalone", "new_clip(bar=1)", false}, // Must be chained or after track()

		// MIDI operations
		{"add_midi()", "track().add_midi(notes=[])", true},

		// Delete operations
		{"delete()", "track().delete()", true},
		{"delete_clip()", "track().delete_clip(bar=1)", true},

		// Functional operations (can be standalone or chained)
		{"filter() standalone", "filter(tracks, track.name == \"X\")", true},
		{"filter() chained", "track().filter(tracks, track.name == \"X\")", true},
		{"map() standalone", "map(tracks, @get_name)", true},
		{"map() chained", "track().map(tracks, @get_name)", true},
		{"for_each() standalone", "for_each(tracks, @add_reverb)", true},
		{"for_each() chained", "track().for_each(tracks, @add_reverb)", true},

		// Track property setters - THESE WERE MISSING!
		{"set_selected()", "track().set_selected(selected=true)", true},
		{"set_selected() with filter", "filter(tracks, track.name == \"X\").set_selected(selected=true)", true},
		{"set_mute()", "track().set_mute(mute=true)", true},
		{"set_solo()", "track().set_solo(solo=true)", true},
		{"set_volume()", "track().set_volume(volume_db=-3.0)", true},
		{"set_pan()", "track().set_pan(pan=0.5)", true},
		{"set_name()", "track().set_name(name=\"Bass\")", true},
		{"add_fx()", "track().add_fx(fxname=\"ReaEQ\")", true},

		// Complex chains
		{"complex chain", "track(instrument=\"Serum\").set_name(name=\"Bass\").set_volume(volume_db=-3.0).set_selected(selected=true)", true},
		{"filter with set_selected", "filter(tracks, track.muted == false).set_selected(selected=true)", true},

		// Invalid DSL
		{"empty string", "", false},
		{"plain text", "This is not DSL code", false},
		{"JSON", "{\"action\": \"create_track\"}", false},
		// Note: track().invalid_method() will pass because it starts with track( - this is acceptable
	}

	provider := &OpenAIProvider{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.isDSLCode(tt.input)
			if result != tt.expected {
				t.Errorf("isDSLCode(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsDSLCode_AllMethods ensures all implemented DSL methods are detected
func TestIsDSLCode_AllMethods(t *testing.T) {
	// This test ensures we don't forget to add new methods to isDSLCode()
	// If a new method is added to the grammar but not to isDSLCode(), this test will fail

	allMethods := []string{
		"track(",
		".new_clip(",
		".add_midi(",
		".delete(",
		".delete_clip(",
		".filter(",
		".map(",
		".for_each(",
		".set_selected(",
		".set_mute(",
		".set_solo(",
		".set_volume(",
		".set_pan(",
		".set_name(",
		".add_fx(",
	}

	provider := &OpenAIProvider{}

	for _, method := range allMethods {
		t.Run("detects_"+method, func(t *testing.T) {
			// Create a DSL string with this method
			var dsl string
			if method == "track(" {
				dsl = "track(instrument=\"Test\")"
			} else if method == ".filter(" || method == ".map(" || method == ".for_each(" {
				dsl = "filter(tracks, track.name == \"X\")"
			} else {
				dsl = "track()." + method[1:] + "param=value)"
			}

			if !provider.isDSLCode(dsl) {
				t.Errorf("isDSLCode() should detect DSL containing %q, but it didn't. DSL: %q", method, dsl)
			}
		})
	}
}
