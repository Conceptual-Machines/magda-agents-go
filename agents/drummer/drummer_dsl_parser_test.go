package drummer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrummerDSLParser_ParsePattern(t *testing.T) {
	tests := []struct {
		name         string
		dsl          string
		expectedDrum string
		expectedGrid string
		expectedHits int
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
			name:         "off-beat hat (four on floor)",
			dsl:          `pattern(drum=hat, grid="-x-x-x-x-x-x-x-x")`,
			expectedDrum: "hat",
			expectedGrid: "-x-x-x-x-x-x-x-x",
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
			// Create new parser for each test to reset state
			p, err := NewDrummerDSLParser()
			require.NoError(t, err)

			actions, err := p.ParseDSL(tt.dsl)
			require.NoError(t, err)
			require.Len(t, actions, 1)

			action := actions[0]
			assert.Equal(t, "drum_pattern", action["type"])
			assert.Equal(t, tt.expectedDrum, action["drum"])
			assert.Equal(t, tt.expectedGrid, action["grid"])
			assert.Equal(t, tt.expectedHits, countHits(action["grid"].(string)))
		})
	}
}

func TestDrummerDSLParser_ActionFormat(t *testing.T) {
	parser, err := NewDrummerDSLParser()
	require.NoError(t, err)

	// Parse a pattern with velocity
	dsl := `pattern(drum=kick, grid="x---x---", velocity=110)`
	actions, err := parser.ParseDSL(dsl)
	require.NoError(t, err)
	require.Len(t, actions, 1)

	action := actions[0]

	// Check action has all expected fields
	assert.Equal(t, "drum_pattern", action["type"])
	assert.Equal(t, "kick", action["drum"])
	assert.Equal(t, "x---x---", action["grid"])
	assert.Equal(t, 110, action["velocity"])
	assert.Equal(t, 0, action["humanize"])
}

func TestDrummerDSLParser_FourOnTheFloor(t *testing.T) {
	// Four on the floor = kick on every beat + hat on off-beats
	// This tests parsing multiple pattern calls (would need multi-statement support)

	parser, err := NewDrummerDSLParser()
	require.NoError(t, err)

	// Test kick pattern (every beat)
	dsl := `pattern(drum=kick, grid="x---x---x---x---")`
	actions, err := parser.ParseDSL(dsl)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, "kick", actions[0]["drum"])
	assert.Equal(t, 4, countHits(actions[0]["grid"].(string)))

	// Test off-beat hat pattern
	parser2, _ := NewDrummerDSLParser()
	dsl2 := `pattern(drum=hat, grid="-x-x-x-x-x-x-x-x")`
	actions2, err := parser2.ParseDSL(dsl2)
	require.NoError(t, err)
	require.Len(t, actions2, 1)
	assert.Equal(t, "hat", actions2[0]["drum"])
	assert.Equal(t, 8, countHits(actions2[0]["grid"].(string)))
}

func TestCountHits(t *testing.T) {
	tests := []struct {
		grid     string
		expected int
	}{
		{"x---x---x---x---", 4},
		{"-x-x-x-x-x-x-x-x", 8},
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
