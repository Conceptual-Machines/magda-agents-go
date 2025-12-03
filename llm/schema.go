package llm

const (
	// MIDI note number constraints
	midiNoteNumberMin = 0
	midiNoteNumberMax = 127

	// Velocity constraints
	velocityMin     = 1
	velocityMax     = 127
	velocityDefault = 100

	// Duration constraints
	durationBeatsMin = 0.01
)

// GetMusicalOutputSchema returns the JSON schema for musical output
// This schema defines the structure of the AI's musical generation output
func GetMusicalOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"choices": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"description": map[string]any{"type": "string"},
						"notes": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"midiNoteNumber": map[string]any{"type": "integer", "minimum": midiNoteNumberMin, "maximum": midiNoteNumberMax},
									"velocity":       map[string]any{"type": "integer", "minimum": velocityMin, "maximum": velocityMax, "default": velocityDefault},
									"startBeats":     map[string]any{"type": "number", "minimum": 0},
									"durationBeats":  map[string]any{"type": "number", "minimum": durationBeatsMin},
								},
								"required":             []string{"midiNoteNumber", "velocity", "startBeats", "durationBeats"},
								"additionalProperties": false,
							},
						},
					},
					"required":             []string{"description", "notes"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"choices"},
		"additionalProperties": false,
	}
}

// GetMagdaActionsSchema returns the JSON schema for MAGDA REAPER actions
// This schema defines the structure of actions that the AI should generate
// Note: OpenAI requires additionalProperties: false, which means all properties must be in 'required'
// Since we have action-specific optional fields, we include all possible fields but make most optional
func GetMagdaActionsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"actions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"action": map[string]any{
							"type": "string",
							"enum": []string{
								"create_track",
								"create_clip",
								"create_clip_at_bar",
								"add_instrument",
								"add_track_fx",
								"set_track_name",
								"set_track_volume",
								"set_track_pan",
								"set_track_mute",
								"set_track_solo",
							},
						},
						"track": map[string]any{
							"type":        []any{"integer", "null"},
							"description": "Track index (0-based). Use null if not applicable.",
						},
						"name": map[string]any{
							"type":        []any{"string", "null"},
							"description": "Track or clip name. Use null if not applicable.",
						},
						"index": map[string]any{
							"type":        []any{"integer", "null"},
							"description": "Track index to insert at (for create_track). Use null if not applicable.",
						},
						"position": map[string]any{
							"type":        []any{"number", "null"},
							"description": "Start position in seconds (for create_clip). Use null if not applicable.",
						},
						"length": map[string]any{
							"type":        []any{"number", "null"},
							"description": "Clip length in seconds (for create_clip). Use null if not applicable.",
						},
						"bar": map[string]any{
							"type":        []any{"integer", "null"},
							"description": "Bar number (1-based) for create_clip_at_bar. Use null if not applicable.",
							"minimum":     1,
						},
						"length_bars": map[string]any{
							"type":        []any{"integer", "null"},
							"description": "Clip length in bars (for create_clip_at_bar). Use null if not applicable.",
							"minimum":     1,
						},
						"fxname": map[string]any{
							"type":        []any{"string", "null"},
							"description": "FX name (e.g., 'VSTi: Serum', 'ReaEQ'). Use null if not applicable.",
						},
						"instrument": map[string]any{
							"type": []any{"string", "null"},
							//nolint:lll // Documentation string
							"description": "Instrument name (e.g., 'VSTi: Serum', 'VST3:ReaSynth'). Can be used with create_track to add an instrument when creating the track. Use null if not applicable.",
						},
						"volume_db": map[string]any{
							"type":        []any{"number", "null"},
							"description": "Volume in dB (for set_track_volume). Use null if not applicable.",
						},
						"pan": map[string]any{
							"type":        []any{"number", "null"},
							"description": "Pan value from -1.0 (left) to 1.0 (right). Use null if not applicable.",
							"minimum":     -1.0,
							"maximum":     1.0,
						},
						"mute": map[string]any{
							"type":        []any{"boolean", "null"},
							"description": "Mute state (for set_track_mute). Use null if not applicable.",
						},
						"solo": map[string]any{
							"type":        []any{"boolean", "null"},
							"description": "Solo state (for set_track_solo). Use null if not applicable.",
						},
					},
					// OpenAI requires additionalProperties: false AND all properties in 'required'
					// We include all properties here - the AI should only populate relevant fields for each action
					// The prompt will guide which fields to use for each action type
					"required": []string{
						"action",
						"track",
						"name",
						"index",
						"position",
						"length",
						"bar",
						"length_bars",
						"fxname",
						"instrument",
						"volume_db",
						"pan",
						"mute",
						"solo",
					},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"actions"},
		"additionalProperties": false,
	}
}
