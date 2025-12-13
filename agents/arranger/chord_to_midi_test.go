package services

import (
	"testing"
)

func TestChordToMIDI(t *testing.T) {
	tests := []struct {
		name          string
		chordSymbol   string
		octave        int
		expectedNotes []int
		expectError   bool
	}{
		{
			name:          "C major",
			chordSymbol:   "C",
			octave:        4,
			expectedNotes: []int{48, 52, 55}, // C4, E4, G4
			expectError:   false,
		},
		{
			name:          "E minor",
			chordSymbol:   "Em",
			octave:        4,
			expectedNotes: []int{52, 55, 59}, // E4, G4, B4
			expectError:   false,
		},
		{
			name:          "A minor",
			chordSymbol:   "Am",
			octave:        4,
			expectedNotes: []int{57, 60, 64}, // A4, C5, E5
			expectError:   false,
		},
		{
			name:          "G major",
			chordSymbol:   "G",
			octave:        4,
			expectedNotes: []int{55, 59, 62}, // G4, B4, D5
			expectError:   false,
		},
		{
			name:          "F major",
			chordSymbol:   "F",
			octave:        4,
			expectedNotes: []int{53, 57, 60}, // F4, A4, C5
			expectError:   false,
		},
		{
			name:          "A minor 7th",
			chordSymbol:   "Am7",
			octave:        4,
			expectedNotes: []int{57, 60, 64, 67}, // A4, C5, E5, G5
			expectError:   false,
		},
		{
			name:          "C major 7th",
			chordSymbol:   "Cmaj7",
			octave:        4,
			expectedNotes: []int{48, 52, 55, 59}, // C4, E4, G4, B4
			expectError:   false,
		},
		{
			name:          "octave 3",
			chordSymbol:   "C",
			octave:        3,
			expectedNotes: []int{36, 40, 43}, // C3, E3, G3
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes, err := ChordToMIDI(tt.chordSymbol, tt.octave)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("ChordToMIDI failed: %v", err)
			}

			if len(notes) != len(tt.expectedNotes) {
				t.Errorf("Expected %d notes, got %d", len(tt.expectedNotes), len(notes))
			}

			for i, expected := range tt.expectedNotes {
				if i < len(notes) && notes[i] != expected {
					t.Errorf("Note %d: expected MIDI %d, got %d", i, expected, notes[i])
				}
			}
		})
	}
}

func TestChordToMIDI_Inversion(t *testing.T) {
	// Test chord with bass note (inversion)
	notes, err := ChordToMIDI("Em/G", 4)
	if err != nil {
		t.Fatalf("ChordToMIDI failed: %v", err)
	}

	// Should have bass G in octave 3 prepended
	// G3 = 43, then Em chord (E4=52, G4=55, B4=59)
	if len(notes) < 4 {
		t.Fatalf("Expected at least 4 notes with bass, got %d", len(notes))
	}

	// First note should be bass G (lower octave)
	if notes[0] != 43 { // G3
		t.Errorf("Expected bass note G3 (43), got %d", notes[0])
	}
}

func TestConvertArrangerActionToNoteEvents_Arpeggio(t *testing.T) {
	action := map[string]any{
		"type":     "arpeggio",
		"chord":    "Em",
		"length":   4.0, // 1 bar
		"velocity": 100,
		"octave":   4,
		// No repeat specified = auto-fill the bar with 16th notes
	}

	events, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// Em triad has 3 notes, default 16th notes (0.25 beats)
	// 4 beats / 0.25 = 16 notes to fill the bar
	// 16 notes = 5 full cycles of 3 notes (15) + 1 more note = 16
	if len(events) != 16 {
		t.Errorf("Expected 16 events (filling 1 bar with 16th notes), got %d", len(events))
	}

	// Each note should be 0.25 beats (16th note)
	for i, event := range events {
		if event.DurationBeats != 0.25 {
			t.Errorf("Event %d: expected duration 0.25 (16th note), got %.4f", i, event.DurationBeats)
		}
	}

	// Should be sequential (start times should increase)
	for i := 1; i < len(events); i++ {
		if events[i].StartBeats <= events[i-1].StartBeats {
			t.Errorf("Arpeggio notes should be sequential: event %d starts at %.2f, event %d starts at %.2f",
				i-1, events[i-1].StartBeats, i, events[i].StartBeats)
		}
	}
}

func TestConvertArrangerActionToNoteEvents_ArpeggioWithNoteDuration(t *testing.T) {
	action := map[string]any{
		"type":          "arpeggio",
		"chord":         "Em",
		"note_duration": 0.25, // 16th notes
		"repeat":        4,
		"velocity":      100,
		"octave":        4,
	}

	events, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// With note_duration=0.25 and repeat=4, should have 3*4=12 notes
	if len(events) != 12 {
		t.Errorf("Expected 12 events (3 notes * 4 repeats), got %d", len(events))
	}

	// Each note should be 0.25 beats
	for i, event := range events {
		if event.DurationBeats != 0.25 {
			t.Errorf("Event %d: expected duration 0.25, got %.4f", i, event.DurationBeats)
		}
	}
}

func TestConvertArrangerActionToNoteEvents_Chord(t *testing.T) {
	action := map[string]any{
		"type":     "chord",
		"chord":    "C",
		"length":   4.0,
		"repeat":   1,
		"velocity": 100,
		"octave":   4,
	}

	events, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// C major triad has 3 notes
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	// All notes should start at the same time (simultaneous)
	for i, event := range events {
		if event.StartBeats != 0.0 {
			t.Errorf("Chord note %d: expected start 0.0, got %.2f", i, event.StartBeats)
		}
		if event.DurationBeats != 4.0 {
			t.Errorf("Chord note %d: expected duration 4.0, got %.2f", i, event.DurationBeats)
		}
	}
}

func TestConvertArrangerActionToNoteEvents_Progression(t *testing.T) {
	action := map[string]any{
		"type":     "progression",
		"chords":   []string{"C", "Am", "F", "G"},
		"length":   16.0, // 4 beats per chord
		"repeat":   1,
		"velocity": 100,
		"octave":   4,
	}

	events, err := ConvertArrangerActionToNoteEvents(action, 0.0)
	if err != nil {
		t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
	}

	// 4 chords * 3 notes each = 12 notes
	if len(events) != 12 {
		t.Errorf("Expected 12 events (4 chords * 3 notes), got %d", len(events))
	}

	// First 3 notes should start at 0, next 3 at 4, etc.
	expectedStarts := []float64{0, 0, 0, 4, 4, 4, 8, 8, 8, 12, 12, 12}
	for i, event := range events {
		if i < len(expectedStarts) && event.StartBeats != expectedStarts[i] {
			t.Errorf("Event %d: expected start %.1f, got %.1f", i, expectedStarts[i], event.StartBeats)
		}
	}

	// Each chord's notes should have duration of 4 beats
	for i, event := range events {
		if event.DurationBeats != 4.0 {
			t.Errorf("Event %d: expected duration 4.0, got %.2f", i, event.DurationBeats)
		}
	}
}

func TestChordQualities(t *testing.T) {
	tests := []struct {
		name          string
		chordSymbol   string
		intervals     []int // expected intervals from root
	}{
		{"major", "C", []int{0, 4, 7}},
		{"minor", "Cm", []int{0, 3, 7}},
		{"diminished", "Cdim", []int{0, 3, 6}},
		{"augmented", "Caug", []int{0, 4, 8}},
		{"sus2", "Csus2", []int{0, 2, 7}},
		{"sus4", "Csus4", []int{0, 5, 7}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes, err := ChordToMIDI(tt.chordSymbol, 4)
			if err != nil {
				t.Fatalf("ChordToMIDI failed: %v", err)
			}

			rootMIDI := 48 // C4
			for i, expectedInterval := range tt.intervals {
				if i < len(notes) {
					actualInterval := notes[i] - rootMIDI
					if actualInterval != expectedInterval {
						t.Errorf("Note %d: expected interval %d, got %d", i, expectedInterval, actualInterval)
					}
				}
			}
		})
	}
}

