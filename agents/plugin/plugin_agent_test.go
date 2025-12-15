package plugin

import (
	"context"
	"testing"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAliases_Programmatic(t *testing.T) {
	cfg := &config.Config{}
	agent := NewPluginAgent(cfg)

	tests := []struct {
		name     string
		plugins  []PluginInfo
		expected map[string]string // alias -> full_name
	}{
		{
			name: "Simple plugin name",
			plugins: []PluginInfo{
				{
					FullName:     "VST3: Serum (Xfer Records)",
					Name:         "Serum",
					Format:       "VST3",
					Manufacturer: "Xfer Records",
					IsInstrument: true,
				},
			},
			expected: map[string]string{
				"serum": "VST3: Serum (Xfer Records)",
			},
		},
		{
			name: "Plugin with version",
			plugins: []PluginInfo{
				{
					FullName:     "VST3: Kontakt 7 (Native Instruments)",
					Name:         "Kontakt 7",
					Format:       "VST3",
					Manufacturer: "Native Instruments",
					IsInstrument: true,
				},
			},
			expected: map[string]string{
				"kontakt":   "VST3: Kontakt 7 (Native Instruments)",
				"kontakt7":  "VST3: Kontakt 7 (Native Instruments)",
				"kontakt 7": "VST3: Kontakt 7 (Native Instruments)",
			},
		},
		{
			name: "CamelCase plugin (ReaEQ)",
			plugins: []PluginInfo{
				{
					FullName:     "JS: ReaEQ",
					Name:         "ReaEQ",
					Format:       "JS",
					Manufacturer: "",
					IsInstrument: false,
				},
			},
			expected: map[string]string{
				"reaeq":  "JS: ReaEQ",
				"rea-eq": "JS: ReaEQ",
				"rea eq": "JS: ReaEQ",
				"eq":     "JS: ReaEQ",
			},
		},
		{
			name: "Manufacturer prefix",
			plugins: []PluginInfo{
				{
					FullName:     "VST3: Serum (Xfer Records)",
					Name:         "Serum",
					Format:       "VST3",
					Manufacturer: "Xfer Records",
					IsInstrument: true,
				},
			},
			expected: map[string]string{
				"serum":      "VST3: Serum (Xfer Records)",
				"xfer serum": "VST3: Serum (Xfer Records)",
			},
		},
		{
			name: "Multiple plugins with conflicts",
			plugins: []PluginInfo{
				{
					FullName:     "JS: ReaEQ",
					Name:         "ReaEQ",
					Format:       "JS",
					Manufacturer: "",
					IsInstrument: false,
				},
				{
					FullName:     "VST3: FabFilter Pro-Q 3 (FabFilter)",
					Name:         "Pro-Q 3",
					Format:       "VST3",
					Manufacturer: "FabFilter",
					IsInstrument: false,
				},
			},
			expected: map[string]string{
				"reaeq":  "JS: ReaEQ", // First one wins
				"rea-eq": "JS: ReaEQ",
				"eq":     "JS: ReaEQ", // First one wins
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aliases, err := agent.GenerateAliases(context.Background(), tt.plugins)
			require.NoError(t, err)
			require.NotNil(t, aliases)

			// Check that all expected aliases exist
			for expectedAlias, expectedFullName := range tt.expected {
				if fullName, exists := aliases[expectedAlias]; exists {
					assert.Equal(t, expectedFullName, fullName,
						"Alias '%s' should map to '%s'", expectedAlias, expectedFullName)
				} else {
					t.Errorf("Expected alias '%s' not found in generated aliases", expectedAlias)
				}
			}

			// Verify we generated at least some aliases
			assert.Greater(t, len(aliases), 0, "Should generate at least one alias")
		})
	}
}

func TestExtractBaseName(t *testing.T) {
	cfg := &config.Config{}
	agent := NewPluginAgent(cfg)

	tests := []struct {
		fullName string
		expected string
	}{
		{"VST3: Serum (Xfer Records)", "Serum"},
		{"JS: ReaEQ", "ReaEQ"},
		{"VST: Kontakt 7 (Native Instruments)", "Kontakt 7"},
		{"AU: Logic Pro X (Apple)", "Logic Pro X"},
		{"ReaPlugs: ReaComp", "ReaComp"},
	}

	for _, tt := range tests {
		t.Run(tt.fullName, func(t *testing.T) {
			result := agent.extractBaseName(tt.fullName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

