package services

import (
	"testing"
)

// Integration tests for the complete arranger DSL flow:
// DSL string → Parser → Actions → NoteEvents

func TestArrangerIntegration_ArpeggioWith16thNotes(t *testing.T) {
	// Test: "E minor arpeggio with 16th notes" should produce 16 sequential notes
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// The simplified grammar only allows: arpeggio(symbol=Em, note_duration=0.25)
	dsl := `arpeggio(symbol=Em, note_duration=0.25)`
	
	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	action := actions[0]
	if action["type"] != "arpeggio" {
		t.Errorf("Expected type 'arpeggio', got %v", action["type"])
	}

	// Convert to NoteEvents
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// Should produce 16 notes (16th notes filling 1 bar = 4 beats)
	// 4 beats / 0.25 beats per note = 16 notes
	if len(noteEvents) != 16 {
		t.Errorf("Expected 16 notes (16th notes filling 1 bar), got %d", len(noteEvents))
	}

	// All notes should be 0.25 beats (16th notes)
	for i, note := range noteEvents {
		if note.DurationBeats != 0.25 {
			t.Errorf("Note %d: expected duration 0.25, got %.4f", i, note.DurationBeats)
		}
	}

	// Notes should be sequential (increasing start times)
	for i := 1; i < len(noteEvents); i++ {
		if noteEvents[i].StartBeats <= noteEvents[i-1].StartBeats {
			t.Errorf("Notes should be sequential: note %d starts at %.2f, note %d starts at %.2f",
				i-1, noteEvents[i-1].StartBeats, i, noteEvents[i].StartBeats)
		}
	}

	// Check E minor notes (E=52, G=55, B=59 in octave 4)
	expectedPitches := []int{52, 55, 59} // E, G, B pattern repeating
	for i, note := range noteEvents {
		expectedPitch := expectedPitches[i%3]
		if note.MidiNoteNumber != expectedPitch {
			t.Errorf("Note %d: expected pitch %d, got %d", i, expectedPitch, note.MidiNoteNumber)
		}
	}
}

func TestArrangerIntegration_ArpeggioWith8thNotes(t *testing.T) {
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=Am, note_duration=0.5)`
	
	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// 8th notes (0.5 beats) filling 1 bar (4 beats) = 8 notes
	if len(noteEvents) != 8 {
		t.Errorf("Expected 8 notes (8th notes filling 1 bar), got %d", len(noteEvents))
	}

	for i, note := range noteEvents {
		if note.DurationBeats != 0.5 {
			t.Errorf("Note %d: expected duration 0.5, got %.4f", i, note.DurationBeats)
		}
	}
}

func TestArrangerIntegration_ChordSimultaneous(t *testing.T) {
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `chord(symbol=C, length=4)`
	
	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	if action["type"] != "chord" {
		t.Errorf("Expected type 'chord', got %v", action["type"])
	}

	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// C major triad = 3 notes
	if len(noteEvents) != 3 {
		t.Errorf("Expected 3 notes (C major triad), got %d", len(noteEvents))
	}

	// All notes should start at the same time (simultaneous = chord)
	for i, note := range noteEvents {
		if note.StartBeats != 0.0 {
			t.Errorf("Chord note %d: expected start 0.0, got %.2f", i, note.StartBeats)
		}
		if note.DurationBeats != 4.0 {
			t.Errorf("Chord note %d: expected duration 4.0, got %.2f", i, note.DurationBeats)
		}
	}

	// Check C major notes (C=48, E=52, G=55 in octave 4)
	expectedPitches := []int{48, 52, 55}
	for i, note := range noteEvents {
		if note.MidiNoteNumber != expectedPitches[i] {
			t.Errorf("Note %d: expected pitch %d, got %d", i, expectedPitches[i], note.MidiNoteNumber)
		}
	}
}

func TestArrangerIntegration_Progression(t *testing.T) {
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `progression(chords=[C, Am, F, G], length=16)`
	
	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	if action["type"] != "progression" {
		t.Errorf("Expected type 'progression', got %v", action["type"])
	}

	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// 4 chords × 3 notes each = 12 notes
	if len(noteEvents) != 12 {
		t.Errorf("Expected 12 notes (4 chords × 3 notes), got %d", len(noteEvents))
	}

	// 16 beats / 4 chords = 4 beats per chord
	expectedStarts := []float64{
		0, 0, 0,       // C chord at beat 0
		4, 4, 4,       // Am chord at beat 4
		8, 8, 8,       // F chord at beat 8
		12, 12, 12,    // G chord at beat 12
	}
	
	for i, note := range noteEvents {
		if note.StartBeats != expectedStarts[i] {
			t.Errorf("Note %d: expected start %.1f, got %.1f", i, expectedStarts[i], note.StartBeats)
		}
		if note.DurationBeats != 4.0 {
			t.Errorf("Note %d: expected duration 4.0, got %.2f", i, note.DurationBeats)
		}
	}
}

func TestArrangerIntegration_FilterRedundantChords(t *testing.T) {
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Even if we somehow get old-style DSL with both chord and arpeggio,
	// the filter should remove the redundant chord
	// Note: This tests the filterRedundantChords function
	
	// Simulate what the filter does by creating actions directly
	actions := []map[string]any{
		{"type": "chord", "chord": "Em", "length": 4.0},
		{"type": "arpeggio", "chord": "Em", "note_duration": 0.25, "length": 4.0},
	}
	
	filtered := parser.filterRedundantChords(actions)
	
	// Should only have the arpeggio
	if len(filtered) != 1 {
		t.Errorf("Expected 1 action after filtering, got %d", len(filtered))
	}
	
	if filtered[0]["type"] != "arpeggio" {
		t.Errorf("Expected arpeggio to remain, got %v", filtered[0]["type"])
	}
}

func TestArrangerIntegration_DefaultNoteDuration(t *testing.T) {
	// When note_duration is not specified, arpeggio should default to 16th notes
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=Em)`
	
	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// Default: 16th notes (0.25 beats) filling 1 bar (4 beats) = 16 notes
	if len(noteEvents) != 16 {
		t.Errorf("Expected 16 notes with default 16th notes, got %d", len(noteEvents))
	}

	for i, note := range noteEvents {
		if note.DurationBeats != 0.25 {
			t.Errorf("Note %d: expected default duration 0.25 (16th note), got %.4f", i, note.DurationBeats)
		}
	}
}

func TestArrangerIntegration_ArpeggioNoChordGenerated(t *testing.T) {
	// Critical test: Ensure arpeggio ONLY produces sequential notes, never simultaneous
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=Em, note_duration=0.25)`
	
	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// Check that NO two notes start at the same time (that would be a chord)
	startTimes := make(map[float64]int)
	for _, note := range noteEvents {
		startTimes[note.StartBeats]++
	}

	for startTime, count := range startTimes {
		if count > 1 {
			t.Errorf("Found %d notes starting at %.2f - arpeggio should have sequential notes only!", 
				count, startTime)
		}
	}
}

func TestArrangerIntegration_ChordAllSimultaneous(t *testing.T) {
	// Critical test: Ensure chord produces ONLY simultaneous notes
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `chord(symbol=Am7, length=4)`
	
	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// All notes should start at exactly 0.0
	for i, note := range noteEvents {
		if note.StartBeats != 0.0 {
			t.Errorf("Chord note %d starts at %.2f - all chord notes should start at same time!", 
				i, note.StartBeats)
		}
	}
}

