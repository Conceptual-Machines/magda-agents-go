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
		name        string
		chordSymbol string
		intervals   []int // expected intervals from root
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

// TestNoteNameToMIDI tests the note name to MIDI conversion
func TestNoteNameToMIDI(t *testing.T) {
	tests := []struct {
		name         string
		noteName     string
		expectedMIDI int
		expectError  bool
	}{
		// Standard notes (C4 = middle C = MIDI 60)
		// Formula: (octave + 1) * 12 + semitone
		{"C4 (middle C)", "C4", 60, false},
		{"C0", "C0", 12, false},
		{"C-1", "C-1", 0, false},
		{"C5", "C5", 72, false},
		// E1 (the user's request): (1+1)*12 + 4 = 28
		{"E1", "E1", 28, false},
		// Other common notes
		{"A4 (440Hz)", "A4", 69, false}, // (4+1)*12 + 9 = 69
		{"G3", "G3", 55, false},         // (3+1)*12 + 7 = 55
		{"D2", "D2", 38, false},         // (2+1)*12 + 2 = 38
		// Sharp notes
		{"C#4", "C#4", 61, false}, // 60 + 1 = 61
		{"F#3", "F#3", 54, false}, // (3+1)*12 + 6 = 54
		{"G#2", "G#2", 44, false}, // (2+1)*12 + 8 = 44
		// Flat notes (Bb = A# = 10 semitones)
		{"Bb2", "Bb2", 46, false}, // (2+1)*12 + 10 = 46
		{"Eb4", "Eb4", 63, false}, // (4+1)*12 + 3 = 63
		{"Ab3", "Ab3", 56, false}, // (3+1)*12 + 8 = 56
		// Edge cases
		{"B0", "B0", 23, false}, // (0+1)*12 + 11 = 23
		{"A0", "A0", 21, false}, // (0+1)*12 + 9 = 21
		// Lowercase should work too (case insensitive)
		{"lowercase e1", "e1", 28, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			midiNote, err := NoteNameToMIDI(tt.noteName)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("NoteNameToMIDI failed: %v", err)
			}

			if midiNote != tt.expectedMIDI {
				t.Errorf("NoteNameToMIDI(%s) = %d, want %d", tt.noteName, midiNote, tt.expectedMIDI)
			}
		})
	}
}

// TestConvertSingleNoteToNoteEvents tests conversion of single note actions
func TestConvertSingleNoteToNoteEvents(t *testing.T) {
	tests := []struct {
		name             string
		action           map[string]any
		startBeat        float64
		expectedMIDI     int
		expectedDuration float64
		expectedVelocity int
		expectedStart    float64
		expectError      bool
	}{
		{
			name: "sustained E1",
			action: map[string]any{
				"type":     "note",
				"pitch":    "E1",
				"duration": 4.0,
				"velocity": 100,
			},
			startBeat:        0.0,
			expectedMIDI:     28, // E1
			expectedDuration: 4.0,
			expectedVelocity: 100,
			expectedStart:    0.0,
			expectError:      false,
		},
		{
			name: "C4 for 2 bars",
			action: map[string]any{
				"type":     "note",
				"pitch":    "C4",
				"duration": 8.0,
				"velocity": 80,
			},
			startBeat:        4.0, // starts at beat 4
			expectedMIDI:     60,  // C4
			expectedDuration: 8.0,
			expectedVelocity: 80,
			expectedStart:    4.0,
			expectError:      false,
		},
		{
			name: "F#3 with explicit start",
			action: map[string]any{
				"type":     "note",
				"pitch":    "F#3",
				"duration": 2.0,
				"velocity": 100,
				"start":    8.0, // explicit start overrides
			},
			startBeat:        0.0, // this gets overridden by action["start"]
			expectedMIDI:     54,  // F#3 = (3+1)*12 + 6 = 54
			expectedDuration: 2.0,
			expectedVelocity: 100,
			expectedStart:    8.0, // explicit start wins
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := ConvertArrangerActionToNoteEvents(tt.action, tt.startBeat)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("ConvertArrangerActionToNoteEvents failed: %v", err)
			}

			if len(events) != 1 {
				t.Fatalf("Expected 1 event, got %d", len(events))
			}

			event := events[0]
			if event.MidiNoteNumber != tt.expectedMIDI {
				t.Errorf("Expected MIDI %d, got %d", tt.expectedMIDI, event.MidiNoteNumber)
			}
			if event.DurationBeats != tt.expectedDuration {
				t.Errorf("Expected duration %.1f, got %.1f", tt.expectedDuration, event.DurationBeats)
			}
			if event.Velocity != tt.expectedVelocity {
				t.Errorf("Expected velocity %d, got %d", tt.expectedVelocity, event.Velocity)
			}
			if event.StartBeats != tt.expectedStart {
				t.Errorf("Expected start %.1f, got %.1f", tt.expectedStart, event.StartBeats)
			}
		})
	}
}
