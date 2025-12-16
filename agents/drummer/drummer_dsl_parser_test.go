package drummer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrummerDSLParser_ParsePattern(t *testing.T) {
	parser, err := NewDrummerDSLParser()
	require.NoError(t, err)

	tests := []struct {
		name           string
		dsl            string
		expectedDrum   string
		expectedGrid   string
		expectedHits   int
	}{
		{
			name:         "basic kick pattern",
			dsl:          `pattern(drum=kick, grid="x---x---x---x---")`,
			expectedDrum: "kick",
			expectedGrid: "x---x---x---x---",
			expectedHits: 4,
		},
		{
			name:         "snare backbeat",
			dsl:          `pattern(drum=snare, grid="----x-------x---")`,
			expectedDrum: "snare",
			expectedGrid: "----x-------x---",
			expectedHits: 2,
		},
		{
			name:         "hi-hat 8ths",
			dsl:          `pattern(drum=hat, grid="x-x-x-x-x-x-x-x-")`,
			expectedDrum: "hat",
			expectedGrid: "x-x-x-x-x-x-x-x-",
			expectedHits: 8,
		},
		{
			name:         "accented pattern",
			dsl:          `pattern(drum=snare, grid="----X-------X---")`,
			expectedDrum: "snare",
			expectedGrid: "----X-------X---",
			expectedHits: 2,
		},
		{
			name:         "ghost notes",
			dsl:          `pattern(drum=snare, grid="o-o-x-o-o-o-x-o-")`,
			expectedDrum: "snare",
			expectedGrid: "o-o-x-o-o-o-x-o-",
			expectedHits: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beat, err := parser.ParseDSL(tt.dsl)
			require.NoError(t, err)
			require.Len(t, beat.Patterns, 1)

			pattern := beat.Patterns[0]
			assert.Equal(t, tt.expectedDrum, pattern.Drum)
			assert.Equal(t, tt.expectedGrid, pattern.Grid)
			assert.Equal(t, tt.expectedHits, countHits(pattern.Grid))
		})
	}
}

func TestDrummerDSLParser_ConvertToNoteEvents(t *testing.T) {
	parser, err := NewDrummerDSLParser()
	require.NoError(t, err)

	// Parse a simple kick pattern
	dsl := `pattern(drum=kick, grid="x---x---x---x---")`
	beat, err := parser.ParseDSL(dsl)
	require.NoError(t, err)

	// Convert to MIDI notes
	notes, err := parser.ConvertToNoteEvents(beat, nil)
	require.NoError(t, err)

	// Should have 4 notes (4 hits)
	assert.Len(t, notes, 4)

	// Check first note
	assert.Equal(t, 36, notes[0].MidiNoteNumber) // Kick = 36 in GM
	assert.Equal(t, 100, notes[0].Velocity)
	assert.Equal(t, 0.0, notes[0].StartBeats)

	// Check timing (16th notes = 0.25 beats apart)
	assert.Equal(t, 1.0, notes[1].StartBeats) // Beat 2 (4 * 0.25)
	assert.Equal(t, 2.0, notes[2].StartBeats) // Beat 3
	assert.Equal(t, 3.0, notes[3].StartBeats) // Beat 4
}

func TestDrummerDSLParser_VelocityLevels(t *testing.T) {
	parser, err := NewDrummerDSLParser()
	require.NoError(t, err)

	// Pattern with different velocity levels
	dsl := `pattern(drum=snare, grid="xXo-")`
	beat, err := parser.ParseDSL(dsl)
	require.NoError(t, err)

	notes, err := parser.ConvertToNoteEvents(beat, nil)
	require.NoError(t, err)

	assert.Len(t, notes, 3)
	assert.Equal(t, 100, notes[0].Velocity) // x = normal
	assert.Equal(t, 127, notes[1].Velocity) // X = accent
	assert.Equal(t, 60, notes[2].Velocity)  // o = ghost
}

func TestDrummerDSLParser_CustomMapping(t *testing.T) {
	parser, err := NewDrummerDSLParser()
	require.NoError(t, err)

	dsl := `pattern(drum=kick, grid="x---")`
	beat, err := parser.ParseDSL(dsl)
	require.NoError(t, err)

	// Use custom mapping (e.g., Addictive Drums)
	customMapping := map[string]int{
		"kick": 24, // Different from GM 36
	}

	notes, err := parser.ConvertToNoteEvents(beat, customMapping)
	require.NoError(t, err)

	assert.Len(t, notes, 1)
	assert.Equal(t, 24, notes[0].MidiNoteNumber) // Should use custom mapping
}

func TestDrummerDSLParser_Swing(t *testing.T) {
	parser, err := NewDrummerDSLParser()
	require.NoError(t, err)

	// Create beat with swing manually (since beat() parsing needs more work)
	beat := &DrumBeat{
		Patterns: []DrumPattern{
			{Drum: "hat", Grid: "xxxx"},
		},
		Swing:       50,
		Subdivision: 16,
	}

	notes, err := parser.ConvertToNoteEvents(beat, nil)
	require.NoError(t, err)

	assert.Len(t, notes, 4)

	// With 50% swing, odd notes (index 1, 3) should be delayed
	// Normal timing: 0, 0.25, 0.5, 0.75
	// With swing: 0, ~0.3125, 0.5, ~0.8125
	assert.Equal(t, 0.0, notes[0].StartBeats)
	assert.Greater(t, notes[1].StartBeats, 0.25) // Delayed by swing
	assert.Equal(t, 0.5, notes[2].StartBeats)
	assert.Greater(t, notes[3].StartBeats, 0.75) // Delayed by swing
}

func TestCountHits(t *testing.T) {
	tests := []struct {
		grid     string
		expected int
	}{
		{"x---x---x---x---", 4},
		{"xxxxxxxxxxxx", 12},
		{"----------------", 0},
		{"xXoX", 4},
		{"x-x-x-x-", 4},
	}

	for _, tt := range tests {
		t.Run(tt.grid, func(t *testing.T) {
			assert.Equal(t, tt.expected, countHits(tt.grid))
		})
	}
}

