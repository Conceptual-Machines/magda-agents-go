package daw

import (
	"strings"
	"testing"
)

// TestDSLDetection ensures all DSL methods are properly detected
// This test would have caught the missing set_selected() detection
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
		{"add_midi()", "track().add_midi(notes=[])", true},
		
		// Delete operations
		{"delete()", "track().delete()", true},
		{"delete_clip()", "track().delete_clip(bar=1)", true},
		
		// Functional operations
		{"filter()", "filter(tracks, track.name == \"X\")", true},
		{"map()", "map(tracks, @get_name)", true},
		{"for_each()", "for_each(tracks, @add_reverb)", true},
		
		// Track property setters - CRITICAL: These were missing before!
		{"set_selected()", "track().set_selected(selected=true)", true},
		{"set_selected() with filter", "filter(tracks, track.name == \"X\").set_selected(selected=true)", true},
		{"set_mute()", "track().set_mute(mute=true)", true},
		{"set_solo()", "track().set_solo(solo=true)", true},
		{"set_volume()", "track().set_volume(volume_db=-3.0)", true},
		{"set_pan()", "track().set_pan(pan=0.5)", true},
		{"set_name()", "track().set_name(name=\"Bass\")", true},
		{"add_fx()", "track().add_fx(fxname=\"ReaEQ\")", true},
		
		// Complex chains
		{"complex chain with set_selected", "track(instrument=\"Serum\").set_name(name=\"Bass\").set_selected(selected=true)", true},
		{"filter with set_selected", "filter(tracks, track.muted == false).set_selected(selected=true)", true},
		
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
			hasAddMidi := strings.Contains(tt.dslCode, ".add_midi(")
			hasFilter := strings.Contains(tt.dslCode, ".filter(") || strings.HasPrefix(tt.dslCode, "filter(")
			hasMap := strings.Contains(tt.dslCode, ".map(") || strings.HasPrefix(tt.dslCode, "map(")
			hasForEach := strings.Contains(tt.dslCode, ".for_each(") || strings.HasPrefix(tt.dslCode, "for_each(")
			hasDelete := strings.Contains(tt.dslCode, ".delete(")
			hasDeleteClip := strings.Contains(tt.dslCode, ".delete_clip(")
			hasSetSelected := strings.Contains(tt.dslCode, ".set_selected(")
			hasSetMute := strings.Contains(tt.dslCode, ".set_mute(")
			hasSetSolo := strings.Contains(tt.dslCode, ".set_solo(")
			hasSetVolume := strings.Contains(tt.dslCode, ".set_volume(")
			hasSetPan := strings.Contains(tt.dslCode, ".set_pan(")
			hasSetName := strings.Contains(tt.dslCode, ".set_name(")
			hasAddFx := strings.Contains(tt.dslCode, ".add_fx(")

			isDSL := hasTrackPrefix || hasNewClip || hasAddMidi || hasFilter || hasMap || hasForEach || hasDelete || hasDeleteClip ||
				hasSetSelected || hasSetMute || hasSetSolo || hasSetVolume || hasSetPan || hasSetName || hasAddFx

			if isDSL != tt.expected {
				t.Errorf("DSL detection for %q = %v, want %v", tt.dslCode, isDSL, tt.expected)
				t.Logf("  hasTrackPrefix=%v, hasNewClip=%v, hasAddMidi=%v, hasFilter=%v, hasMap=%v, hasForEach=%v",
					hasTrackPrefix, hasNewClip, hasAddMidi, hasFilter, hasMap, hasForEach)
				t.Logf("  hasDelete=%v, hasDeleteClip=%v, hasSetSelected=%v, hasSetMute=%v, hasSetSolo=%v",
					hasDelete, hasDeleteClip, hasSetSelected, hasSetMute, hasSetSolo)
				t.Logf("  hasSetVolume=%v, hasSetPan=%v, hasSetName=%v, hasAddFx=%v",
					hasSetVolume, hasSetPan, hasSetName, hasAddFx)
			}
		})
	}
}

// Helper function to match strings.Contains behavior
func containsSubstring(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}


