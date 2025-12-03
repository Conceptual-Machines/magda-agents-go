package models

// NoteEvent represents a single musical note with timing and pitch information
type NoteEvent struct {
	MidiNoteNumber int     `json:"midiNoteNumber"`
	Velocity       int     `json:"velocity"`
	StartBeats     float64 `json:"startBeats"`
	DurationBeats  float64 `json:"durationBeats"`
}

// ChordEvent represents a chord with timing information
type ChordEvent struct {
	ChordSymbol   string  `json:"chordSymbol"`
	StartBeats    float64 `json:"startBeats"`
	DurationBeats float64 `json:"durationBeats"`
}

// MusicalChoice represents a complete musical sequence choice
type MusicalChoice struct {
	Description string       `json:"description"`
	Notes       []NoteEvent  `json:"notes"`
	Chords      []ChordEvent `json:"chords,omitempty"`
}

// MusicalOutput represents the complete structured output from the LLM
type MusicalOutput struct {
	Choices []MusicalChoice `json:"choices"`
}
