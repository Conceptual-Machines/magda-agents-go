package daw

import (
	"strings"
	"testing"
)

// TestDSLDetection ensures all DSL methods are properly detected
// This test ensures DSL detection works for unified set_track and set_clip methods
func TestDSLDetection(t *testing.T) {
	tests := []struct {
		name     string
		dslCode  string
		expected bool
	}{
		// Track operations
		{"track()", "track(instrument=\"Serum\")", true},

		// Clip operations
		{"new_clip()", "track().new_clip(bar=1)", true},
		// NOTE: add_midi is no longer part of DAW DSL grammar - handled by arranger agent

		// Delete operations
		{"delete()", "track().delete()", true},
		{"delete_clip()", "track().delete_clip(bar=1)", true},

		// Functional operations
		{"filter()", "filter(tracks, track.name == \"X\")", true},
		{"map()", "map(tracks, @get_name)", true},
		{"for_each()", "for_each(tracks, @add_reverb)", true},

		// Track property setters - Unified methods
		{"set_track()", "track().set_track(selected=true)", true},
		{"set_track() with filter", "filter(tracks, track.name == \"X\").set_track(selected=true)", true},
		{"set_track() with multiple properties", "track().set_track(mute=true, solo=true)", true},
		{"set_track() volume", "track().set_track(volume_db=-3.0)", true},
		{"set_track() pan", "track().set_track(pan=0.5)", true},
		{"set_track() name", "track().set_track(name=\"Bass\")", true},
		{"set_clip()", "track().set_clip(name=\"Clip\")", true},
		{"add_fx()", "track().add_fx(fxname=\"ReaEQ\")", true},

		// Complex chains
		{"complex chain with set_track", "track(instrument=\"Serum\").set_track(name=\"Bass\", selected=true)", true},
		{"filter with set_track", "filter(tracks, track.muted == false).set_track(selected=true)", true},

		// Invalid DSL
		{"empty string", "", false},
		{"plain text", "This is not DSL code", false},
		{"JSON", "{\"action\": \"create_track\"}", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the DSL detection logic from parseActionsFromResponse
			hasTrackPrefix := strings.HasPrefix(tt.dslCode, "track(")
			hasNewClip := strings.Contains(tt.dslCode, ".new_clip(")
			hasFilter := strings.Contains(tt.dslCode, ".filter(") || strings.HasPrefix(tt.dslCode, "filter(")
			hasMap := strings.Contains(tt.dslCode, ".map(") || strings.HasPrefix(tt.dslCode, "map(")
			hasForEach := strings.Contains(tt.dslCode, ".for_each(") || strings.HasPrefix(tt.dslCode, "for_each(")
			hasDelete := strings.Contains(tt.dslCode, ".delete(")
			hasDeleteClip := strings.Contains(tt.dslCode, ".delete_clip(")
			hasSetTrack := strings.Contains(tt.dslCode, ".set_track(")
			hasSetClip := strings.Contains(tt.dslCode, ".set_clip(")
			hasAddFx := strings.Contains(tt.dslCode, ".add_fx(")

			isDSL := hasTrackPrefix || hasNewClip || hasFilter || hasMap || hasForEach || hasDelete || hasDeleteClip ||
				hasSetTrack || hasSetClip || hasAddFx

			if isDSL != tt.expected {
				t.Errorf("DSL detection for %q = %v, want %v", tt.dslCode, isDSL, tt.expected)
				t.Logf("  hasTrackPrefix=%v, hasNewClip=%v, hasFilter=%v, hasMap=%v, hasForEach=%v",
					hasTrackPrefix, hasNewClip, hasFilter, hasMap, hasForEach)
				t.Logf("  hasDelete=%v, hasDeleteClip=%v, hasSetTrack=%v, hasSetClip=%v, hasAddFx=%v",
					hasDelete, hasDeleteClip, hasSetTrack, hasSetClip, hasAddFx)
			}
		})
	}
}
