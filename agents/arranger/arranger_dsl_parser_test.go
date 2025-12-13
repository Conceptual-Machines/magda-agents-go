package services

import (
	"testing"
)

func TestArrangerDSLParser_Arpeggio(t *testing.T) {
	tests := []struct {
		name           string
		dsl            string
		expectedChord  string
		expectedLength float64
		expectError    bool
	}{
		{
			name:           "arpeggio with symbol parameter",
			dsl:            `arpeggio(symbol=Em, length=2)`,
			expectedChord:  "Em",
			expectedLength: 2.0,
			expectError:    false,
		},
		{
			name:           "arpeggio with note_duration",
			dsl:            `arpeggio(symbol=Em, note_duration=0.25, repeat=4)`,
			expectedChord:  "Em",
			expectedLength: 4.0, // default when no length specified
			expectError:    false,
		},
		{
			name:           "composition with arpeggio",
			dsl:            `composition().add_arpeggio(symbol=Em, length=4)`,
			expectedChord:  "Em",
			expectedLength: 4.0,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewArrangerDSLParser()
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			actions, err := parser.ParseDSL(tt.dsl)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseDSL failed: %v", err)
			}

			if len(actions) == 0 {
				t.Fatal("Expected at least one action")
			}

			// Find the arpeggio action
			var arpeggioAction map[string]any
			for _, action := range actions {
				if action["type"] == "arpeggio" {
					arpeggioAction = action
					break
				}
			}

			if arpeggioAction == nil {
				t.Fatal("No arpeggio action found")
			}

			if chord, ok := arpeggioAction["chord"].(string); !ok || chord != tt.expectedChord {
				t.Errorf("Expected chord %s, got %v", tt.expectedChord, arpeggioAction["chord"])
			}

			if length, ok := arpeggioAction["length"].(float64); !ok || length != tt.expectedLength {
				t.Errorf("Expected length %f, got %v", tt.expectedLength, arpeggioAction["length"])
			}
		})
	}
}

func TestArrangerDSLParser_Chord(t *testing.T) {
	tests := []struct {
		name           string
		dsl            string
		expectedChord  string
		expectedLength float64
		expectError    bool
	}{
		{
			name:           "chord with symbol parameter",
			dsl:            `chord(symbol=Am7, length=2)`,
			expectedChord:  "Am7",
			expectedLength: 2.0,
			expectError:    false,
		},
		{
			name:           "minor chord",
			dsl:            `chord(symbol=Em, length=4, repeat=2)`,
			expectedChord:  "Em",
			expectedLength: 4.0,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewArrangerDSLParser()
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			actions, err := parser.ParseDSL(tt.dsl)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseDSL failed: %v", err)
			}

			if len(actions) == 0 {
				t.Fatal("Expected at least one action")
			}

			action := actions[0]
			if action["type"] != "chord" {
				t.Errorf("Expected type 'chord', got %v", action["type"])
			}

			if chord, ok := action["chord"].(string); !ok || chord != tt.expectedChord {
				t.Errorf("Expected chord %s, got %v", tt.expectedChord, action["chord"])
			}

			if length, ok := action["length"].(float64); !ok || length != tt.expectedLength {
				t.Errorf("Expected length %f, got %v", tt.expectedLength, action["length"])
			}
		})
	}
}

func TestArrangerDSLParser_Progression(t *testing.T) {
	tests := []struct {
		name           string
		dsl            string
		expectedChords []string
		expectedLength float64
		expectError    bool
	}{
		{
			name:           "basic progression",
			dsl:            `progression(chords=[C, Am, F, G], length=16)`,
			expectedChords: []string{"C", "Am", "F", "G"},
			expectedLength: 16.0,
			expectError:    false,
		},
		{
			name:           "progression with quotes",
			dsl:            `progression(chords=["C", "Am", "F", "G"], length=16)`,
			expectedChords: []string{"C", "Am", "F", "G"},
			expectedLength: 16.0,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewArrangerDSLParser()
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			actions, err := parser.ParseDSL(tt.dsl)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseDSL failed: %v", err)
			}

			if len(actions) == 0 {
				t.Fatal("Expected at least one action")
			}

			action := actions[0]
			if action["type"] != "progression" {
				t.Errorf("Expected type 'progression', got %v", action["type"])
			}

			chords, ok := action["chords"].([]string)
			if !ok {
				t.Fatalf("Expected chords to be []string, got %T", action["chords"])
			}

			if len(chords) != len(tt.expectedChords) {
				t.Errorf("Expected %d chords, got %d", len(tt.expectedChords), len(chords))
			}

			for i, expected := range tt.expectedChords {
				if i < len(chords) && chords[i] != expected {
					t.Errorf("Chord %d: expected %s, got %s", i, expected, chords[i])
				}
			}

			if length, ok := action["length"].(float64); !ok || length != tt.expectedLength {
				t.Errorf("Expected length %f, got %v", tt.expectedLength, action["length"])
			}
		})
	}
}

func TestArrangerDSLParser_NoteDuration(t *testing.T) {
	parser, err := NewArrangerDSLParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Test with explicit note_duration parameter
	dsl := `arpeggio(symbol=Em, note_duration=0.25, repeat=4)`
	actions, err := parser.ParseDSL(dsl)
	if err != nil {
		t.Fatalf("ParseDSL failed: %v", err)
	}

	if len(actions) == 0 {
		t.Fatal("Expected at least one action")
	}

	action := actions[0]

	// Check note_duration is captured
	noteDuration, ok := action["note_duration"].(float64)
	if !ok {
		t.Fatalf("Expected note_duration to be float64, got %T", action["note_duration"])
	}

	if noteDuration != 0.25 {
		t.Errorf("Expected note_duration 0.25, got %f", noteDuration)
	}
}
