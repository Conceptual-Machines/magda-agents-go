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

func TestArrangerIntegration_TwoBarArpeggio(t *testing.T) {
	// Test: "2 bar E minor arpeggio with 16th notes"
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// 2 bars = 8 beats, 16th notes = 0.25 beats per note
	dsl := `arpeggio(symbol=Em, note_duration=0.25, length=8)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// 8 beats / 0.25 beats per note = 32 notes
	if len(noteEvents) != 32 {
		t.Errorf("Expected 32 notes (16th notes filling 2 bars), got %d", len(noteEvents))
	}

	// Last note should end at beat 8
	lastNote := noteEvents[len(noteEvents)-1]
	expectedLastEnd := 8.0
	actualLastEnd := lastNote.StartBeats + lastNote.DurationBeats
	if actualLastEnd != expectedLastEnd {
		t.Errorf("Expected last note to end at %.1f, got %.4f", expectedLastEnd, actualLastEnd)
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
		0, 0, 0, // C chord at beat 0
		4, 4, 4, // Am chord at beat 4
		8, 8, 8, // F chord at beat 8
		12, 12, 12, // G chord at beat 12
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

// ===== COMPREHENSIVE ARPEGGIO INTEGRATION TESTS =====

func TestArrangerIntegration_ArpeggioQuarterNotes(t *testing.T) {
	// Quarter notes (1 beat each) filling 1 bar = 4 notes
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=C, note_duration=1.0, length=4)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// 4 beats / 1 beat per note = 4 notes
	if len(noteEvents) != 4 {
		t.Errorf("Expected 4 notes (quarter notes in 1 bar), got %d", len(noteEvents))
	}

	for i, note := range noteEvents {
		if note.DurationBeats != 1.0 {
			t.Errorf("Note %d: expected duration 1.0 (quarter note), got %.4f", i, note.DurationBeats)
		}
	}

	// Verify sequential timing: 0, 1, 2, 3
	expectedStarts := []float64{0, 1, 2, 3}
	for i, note := range noteEvents {
		if note.StartBeats != expectedStarts[i] {
			t.Errorf("Note %d: expected start %.1f, got %.4f", i, expectedStarts[i], note.StartBeats)
		}
	}
}

func TestArrangerIntegration_ArpeggioFourBars(t *testing.T) {
	// 4 bars = 16 beats with 16th notes
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=Dm, note_duration=0.25, length=16)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// 16 beats / 0.25 beats per note = 64 notes
	if len(noteEvents) != 64 {
		t.Errorf("Expected 64 notes (16th notes in 4 bars), got %d", len(noteEvents))
	}

	// First note at 0, last note should end at 16
	lastNote := noteEvents[len(noteEvents)-1]
	lastEnd := lastNote.StartBeats + lastNote.DurationBeats
	if lastEnd != 16.0 {
		t.Errorf("Expected last note to end at 16.0, got %.4f", lastEnd)
	}

	// Verify D minor pitches (D=50, F=53, A=57 in octave 4)
	expectedPitches := []int{50, 53, 57}
	for i, note := range noteEvents {
		expectedPitch := expectedPitches[i%3]
		if note.MidiNoteNumber != expectedPitch {
			t.Errorf("Note %d: expected pitch %d (D minor), got %d", i, expectedPitch, note.MidiNoteNumber)
		}
	}
}

func TestArrangerIntegration_ArpeggioMajor7thChord(t *testing.T) {
	// Test with Cmaj7 chord (4 notes: C, E, G, B)
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=Cmaj7, note_duration=0.5, length=4)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// 4 beats / 0.5 beats per note = 8 notes
	if len(noteEvents) != 8 {
		t.Errorf("Expected 8 notes (8th notes in 1 bar), got %d", len(noteEvents))
	}

	// Cmaj7 = C(48), E(52), G(55), B(59)
	expectedPitches := []int{48, 52, 55, 59}
	for i, note := range noteEvents {
		expectedPitch := expectedPitches[i%4]
		if note.MidiNoteNumber != expectedPitch {
			t.Errorf("Note %d: expected pitch %d (Cmaj7), got %d", i, expectedPitch, note.MidiNoteNumber)
		}
	}
}

func TestArrangerIntegration_ArpeggioWithOctave(t *testing.T) {
	// Test arpeggio in octave 5
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=Em, note_duration=0.25, length=4, octave=5)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// E minor in octave 5: E=64, G=67, B=71
	expectedPitches := []int{64, 67, 71}
	for i, note := range noteEvents {
		expectedPitch := expectedPitches[i%3]
		if note.MidiNoteNumber != expectedPitch {
			t.Errorf("Note %d: expected pitch %d (Em octave 5), got %d", i, expectedPitch, note.MidiNoteNumber)
		}
	}
}

func TestArrangerIntegration_ArpeggioWithVelocity(t *testing.T) {
	// Test arpeggio with custom velocity
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=Am, note_duration=0.5, length=4, velocity=80)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	for i, note := range noteEvents {
		if note.Velocity != 80 {
			t.Errorf("Note %d: expected velocity 80, got %d", i, note.Velocity)
		}
	}
}

func TestArrangerIntegration_ArpeggioStartOffset(t *testing.T) {
	// Test arpeggio with start offset (e.g., after a chord)
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=G, note_duration=0.25, length=4)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	// Start at beat 8 (after 2 bars of other content)
	startOffset := 8.0
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, startOffset)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// First note should start at offset
	if noteEvents[0].StartBeats != startOffset {
		t.Errorf("First note: expected start %.1f, got %.4f", startOffset, noteEvents[0].StartBeats)
	}

	// Last note should end at 8 + 4 = 12
	lastNote := noteEvents[len(noteEvents)-1]
	lastEnd := lastNote.StartBeats + lastNote.DurationBeats
	expectedEnd := startOffset + 4.0
	if lastEnd != expectedEnd {
		t.Errorf("Expected last note to end at %.1f, got %.4f", expectedEnd, lastEnd)
	}
}

func TestArrangerIntegration_ArpeggioMinor7th(t *testing.T) {
	// Test Am7 arpeggio (A, C, E, G)
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=Am7, note_duration=0.5, length=4)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// Am7 in octave 4: A=57, C=60, E=64, G=67
	expectedPitches := []int{57, 60, 64, 67}
	for i, note := range noteEvents {
		expectedPitch := expectedPitches[i%4]
		if note.MidiNoteNumber != expectedPitch {
			t.Errorf("Note %d: expected pitch %d (Am7), got %d", i, expectedPitch, note.MidiNoteNumber)
		}
	}
}

func TestArrangerIntegration_ArpeggioDiminished(t *testing.T) {
	// Test Bdim arpeggio (B, D, F)
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=Bdim, note_duration=0.25, length=4)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// Bdim in octave 4: B=59, D=62, F=65
	expectedPitches := []int{59, 62, 65}
	for i, note := range noteEvents {
		expectedPitch := expectedPitches[i%3]
		if note.MidiNoteNumber != expectedPitch {
			t.Errorf("Note %d: expected pitch %d (Bdim), got %d", i, expectedPitch, note.MidiNoteNumber)
		}
	}
}

func TestArrangerIntegration_ArpeggioTimingExact(t *testing.T) {
	// Verify exact timing of each note in 16th note arpeggio
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=Em, note_duration=0.25, length=2)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// 2 beats / 0.25 = 8 notes
	if len(noteEvents) != 8 {
		t.Fatalf("Expected 8 notes, got %d", len(noteEvents))
	}

	// Verify exact timing: 0, 0.25, 0.5, 0.75, 1.0, 1.25, 1.5, 1.75
	expectedStarts := []float64{0, 0.25, 0.5, 0.75, 1.0, 1.25, 1.5, 1.75}
	for i, note := range noteEvents {
		if note.StartBeats != expectedStarts[i] {
			t.Errorf("Note %d: expected start %.2f, got %.4f", i, expectedStarts[i], note.StartBeats)
		}
		if note.DurationBeats != 0.25 {
			t.Errorf("Note %d: expected duration 0.25, got %.4f", i, note.DurationBeats)
		}
	}
}

func TestArrangerIntegration_ArpeggioSharpFlat(t *testing.T) {
	// Test arpeggios with sharps and flats
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Test F# minor
	dsl := `arpeggio(symbol=F#m, note_duration=0.5, length=4)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// F#m in octave 4: F#=54, A=57, C#=61
	expectedPitches := []int{54, 57, 61}
	for i, note := range noteEvents {
		expectedPitch := expectedPitches[i%3]
		if note.MidiNoteNumber != expectedPitch {
			t.Errorf("Note %d: expected pitch %d (F#m), got %d", i, expectedPitch, note.MidiNoteNumber)
		}
	}
}

func TestArrangerIntegration_ArpeggioFlatKey(t *testing.T) {
	// Test Bb major arpeggio
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `arpeggio(symbol=Bb, note_duration=0.5, length=4)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// Bb major in octave 4: Bb=58, D=62, F=65
	expectedPitches := []int{58, 62, 65}
	for i, note := range noteEvents {
		expectedPitch := expectedPitches[i%3]
		if note.MidiNoteNumber != expectedPitch {
			t.Errorf("Note %d: expected pitch %d (Bb), got %d", i, expectedPitch, note.MidiNoteNumber)
		}
	}
}

// ===== SINGLE NOTE INTEGRATION TESTS =====
// These tests verify the new note() DSL for single sustained notes

func TestArrangerIntegration_SingleNoteE1Sustained(t *testing.T) {
	// Test: "sustained E1" should produce a single E1 note for 4 beats
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `note(pitch="E1", duration=4)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	action := actions[0]
	if action["type"] != "note" {
		t.Errorf("Expected type 'note', got %v", action["type"])
	}

	// Convert to NoteEvents
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// Should produce exactly 1 note
	if len(noteEvents) != 1 {
		t.Fatalf("Expected 1 note event for sustained note, got %d", len(noteEvents))
	}

	note := noteEvents[0]

	// E1 = MIDI 28 (using formula: (octave+1)*12 + semitone, E=4 semitones, so (1+1)*12 + 4 = 28)
	if note.MidiNoteNumber != 28 {
		t.Errorf("Expected MIDI note 28 (E1), got %d", note.MidiNoteNumber)
	}

	// Duration should be 4 beats (sustained/whole note)
	if note.DurationBeats != 4.0 {
		t.Errorf("Expected duration 4.0 beats, got %.2f", note.DurationBeats)
	}

	// Start at beat 0
	if note.StartBeats != 0.0 {
		t.Errorf("Expected start at 0.0, got %.2f", note.StartBeats)
	}

	// Default velocity should be 100
	if note.Velocity != 100 {
		t.Errorf("Expected velocity 100, got %d", note.Velocity)
	}
}

func TestArrangerIntegration_SingleNoteC4MiddleC(t *testing.T) {
	// Test: middle C (C4 = MIDI 60)
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `note(pitch="C4", duration=2, velocity=80)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	if len(noteEvents) != 1 {
		t.Fatalf("Expected 1 note event, got %d", len(noteEvents))
	}

	note := noteEvents[0]

	// C4 = MIDI 60 (middle C)
	if note.MidiNoteNumber != 60 {
		t.Errorf("Expected MIDI note 60 (C4), got %d", note.MidiNoteNumber)
	}

	if note.DurationBeats != 2.0 {
		t.Errorf("Expected duration 2.0 beats, got %.2f", note.DurationBeats)
	}

	if note.Velocity != 80 {
		t.Errorf("Expected velocity 80, got %d", note.Velocity)
	}
}

func TestArrangerIntegration_SingleNoteSharp(t *testing.T) {
	// Test: F#3 with sharp accidental
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `note(pitch="F#3", duration=1)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	if len(noteEvents) != 1 {
		t.Fatalf("Expected 1 note event, got %d", len(noteEvents))
	}

	note := noteEvents[0]

	// F#3 = MIDI 54 ((3+1)*12 + 6 = 54)
	if note.MidiNoteNumber != 54 {
		t.Errorf("Expected MIDI note 54 (F#3), got %d", note.MidiNoteNumber)
	}
}

func TestArrangerIntegration_SingleNoteFlat(t *testing.T) {
	// Test: Bb2 with flat accidental
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `note(pitch="Bb2", duration=8)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	if len(noteEvents) != 1 {
		t.Fatalf("Expected 1 note event, got %d", len(noteEvents))
	}

	note := noteEvents[0]

	// Bb2 = MIDI 46 ((2+1)*12 + 10 = 46)
	if note.MidiNoteNumber != 46 {
		t.Errorf("Expected MIDI note 46 (Bb2), got %d", note.MidiNoteNumber)
	}

	// 8 beats = 2 bars
	if note.DurationBeats != 8.0 {
		t.Errorf("Expected duration 8.0 beats (2 bars), got %.2f", note.DurationBeats)
	}
}

func TestArrangerIntegration_SingleNoteWithStartOffset(t *testing.T) {
	// Test: note starting at beat 4 (second bar)
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	dsl := `note(pitch="A4", duration=4)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]

	// Start offset at beat 4 (second bar)
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 4.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	note := noteEvents[0]

	// A4 = MIDI 69 (440 Hz)
	if note.MidiNoteNumber != 69 {
		t.Errorf("Expected MIDI note 69 (A4), got %d", note.MidiNoteNumber)
	}

	// Should start at beat 4
	if note.StartBeats != 4.0 {
		t.Errorf("Expected start at 4.0, got %.2f", note.StartBeats)
	}

	// Should end at beat 8
	endBeat := note.StartBeats + note.DurationBeats
	if endBeat != 8.0 {
		t.Errorf("Expected note to end at 8.0, got %.2f", endBeat)
	}
}

func TestArrangerIntegration_SingleNoteBassNote(t *testing.T) {
	// Test: Low bass note (typical for bass synth like Serum)
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// E1 is a common bass note
	dsl := `note(pitch="E1", duration=4, velocity=100)`

	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	action := actions[0]
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	note := noteEvents[0]

	// E1 = MIDI 28 (low bass)
	if note.MidiNoteNumber != 28 {
		t.Errorf("Expected MIDI note 28 (E1 bass), got %d", note.MidiNoteNumber)
	}

	// This is a sustained bass note
	if note.DurationBeats != 4.0 {
		t.Errorf("Expected duration 4.0 beats (sustained), got %.2f", note.DurationBeats)
	}
}

func TestArrangerIntegration_SingleNoteVsArpeggio(t *testing.T) {
	// Critical test: Verify single note produces 1 event, arpeggio produces many
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Single sustained E note
	noteDSL := `note(pitch="E1", duration=4)`
	noteActions, err := parser.ParseDSL(noteDSL)
	if err != nil {
		t.Fatalf("ParseDSL (note) failed: %v", err)
	}
	noteEvents, err := ConvertArrangerActionToNoteEvents(noteActions[0], 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents (note) failed: %v", err)
	}

	// Re-create parser for second parse
	parser2, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser 2: %v", err)
	}

	// E minor arpeggio (many notes)
	arpDSL := `arpeggio(symbol=Em, note_duration=0.25, length=4)`
	arpActions, err := parser2.ParseDSL(arpDSL)
	if err != nil {
		t.Fatalf("ParseDSL (arpeggio) failed: %v", err)
	}
	arpEvents, err := ConvertArrangerActionToNoteEvents(arpActions[0], 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents (arpeggio) failed: %v", err)
	}

	// Single note should produce exactly 1 event
	if len(noteEvents) != 1 {
		t.Errorf("Single note should produce 1 event, got %d", len(noteEvents))
	}

	// Arpeggio should produce many events (16 for 16th notes in 1 bar)
	if len(arpEvents) != 16 {
		t.Errorf("Arpeggio should produce 16 events, got %d", len(arpEvents))
	}

	// Single note should be sustained (4 beats)
	if noteEvents[0].DurationBeats != 4.0 {
		t.Errorf("Single note should be 4 beats, got %.2f", noteEvents[0].DurationBeats)
	}

	// Arpeggio notes should be short (0.25 beats each)
	for i, event := range arpEvents {
		if event.DurationBeats != 0.25 {
			t.Errorf("Arpeggio note %d should be 0.25 beats, got %.2f", i, event.DurationBeats)
		}
	}
}

func TestArrangerIntegration_SingleNoteAllOctaves(t *testing.T) {
	// Test note names across different octaves
	tests := []struct {
		pitch        string
		expectedMIDI int
	}{
		{"C0", 12},
		{"C1", 24},
		{"C2", 36},
		{"C3", 48},
		{"C4", 60}, // Middle C
		{"C5", 72},
		{"C6", 84},
		{"E1", 28}, // User's original request
		{"A4", 69}, // 440 Hz
		{"G2", 43}, // Bass range
	}

	for _, tt := range tests {
		t.Run(tt.pitch, func(t *testing.T) {
			parser, err := NewArrangerDSLParser()
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			dsl := `note(pitch="` + tt.pitch + `", duration=4)`
			actions, err := parser.ParseDSL(dsl)
			if err != nil {
				t.Fatalf("ParseDSL failed for %s: %v", tt.pitch, err)
			}

			noteEvents, err := ConvertArrangerActionToNoteEvents(actions[0], 0.0)
			if err != nil {
				t.Fatalf("ConvertArrangerActionToNoteEvents failed for %s: %v", tt.pitch, err)
			}

			if len(noteEvents) != 1 {
				t.Fatalf("Expected 1 note for %s, got %d", tt.pitch, len(noteEvents))
			}

			if noteEvents[0].MidiNoteNumber != tt.expectedMIDI {
				t.Errorf("Expected MIDI %d for %s, got %d", tt.expectedMIDI, tt.pitch, noteEvents[0].MidiNoteNumber)
			}
		})
	}
}

func TestArrangerIntegration_SingleNoteFullWorkflow(t *testing.T) {
	// End-to-end test: simulate what happens when user says "add sustained E1"
	// This mimics the full workflow: DSL → Parser → Actions → NoteEvents

	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// This is what the LLM should generate for "sustained E1 at bar 2"
	dsl := `note(pitch="E1", duration=4)`

	// Step 1: Parse DSL
	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("Step 1 - ParseDSL failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("Step 1 - Expected 1 action, got %d", len(actions))
	}

	action := actions[0]

	// Verify action structure
	if action["type"] != "note" {
		t.Errorf("Action type should be 'note', got %v", action["type"])
	}
	if action["pitch"] != "E1" {
		t.Errorf("Action pitch should be 'E1', got %v", action["pitch"])
	}
	if action["duration"] != 4.0 {
		t.Errorf("Action duration should be 4.0, got %v", action["duration"])
	}

	// Step 2: Convert to NoteEvents (bar 2 = beat 4 offset)
	startBeat := 4.0 // Bar 2 starts at beat 4 (assuming 4/4 time)
	noteEvents, err := ConvertArrangerActionToNoteEvents(action, startBeat)
	if err != nil {
		t.Fatalf("Step 2 - ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// Step 3: Verify final output
	if len(noteEvents) != 1 {
		t.Fatalf("Step 3 - Expected 1 note event, got %d", len(noteEvents))
	}

	note := noteEvents[0]

	// Final verification
	if note.MidiNoteNumber != 28 {
		t.Errorf("Final MIDI should be 28 (E1), got %d", note.MidiNoteNumber)
	}
	if note.StartBeats != 4.0 {
		t.Errorf("Final start should be 4.0 (bar 2), got %.2f", note.StartBeats)
	}
	if note.DurationBeats != 4.0 {
		t.Errorf("Final duration should be 4.0 (sustained), got %.2f", note.DurationBeats)
	}
	if note.Velocity != 100 {
		t.Errorf("Final velocity should be 100, got %d", note.Velocity)
	}

	t.Logf("✅ Full workflow test passed: 'sustained E1 at bar 2' → MIDI 28 at beat 4 for 4 beats")
}
