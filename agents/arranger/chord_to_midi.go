package services

import (
	"fmt"
	"log"
	"strings"

	"github.com/Conceptual-Machines/magda-agents-go/models"
)

// ChordToMIDI converts chord symbols to MIDI note numbers
// Supports: C, Em, Am7, Cmaj7, Emin/G (inversions), etc.
// Returns slice of MIDI note numbers (0-127) for the chord
func ChordToMIDI(chordSymbol string, octave int) ([]int, error) {
	// Parse bass note if present (e.g., "Emin/G" -> chord="Emin", bass="G")
	baseChord := chordSymbol
	bassNote := ""
	if strings.Contains(chordSymbol, "/") {
		parts := strings.Split(chordSymbol, "/")
		if len(parts) == 2 {
			baseChord = strings.TrimSpace(parts[0])
			bassNote = strings.TrimSpace(parts[1])
		}
	}

	// Parse root note
	root, err := parseRootNote(baseChord)
	if err != nil {
		return nil, fmt.Errorf("invalid chord root: %w", err)
	}

	// Calculate root MIDI note (C4 = 60)
	rootMIDI := noteToMIDI(root, octave)

	// Determine chord quality and extensions
	quality := parseChordQuality(baseChord)
	extensions := parseExtensions(baseChord)

	// Build chord intervals (semitones from root)
	intervals := buildChordIntervals(quality, extensions)

	// Convert intervals to MIDI notes
	notes := make([]int, 0, len(intervals)+1)
	for _, interval := range intervals {
		midiNote := rootMIDI + interval
		if midiNote < 0 || midiNote > 127 {
			continue // Skip out-of-range notes
		}
		notes = append(notes, midiNote)
	}

	// Add bass note if specified (inversion)
	if bassNote != "" {
		bassRoot, err := parseRootNote(bassNote)
		if err == nil {
			// Bass note typically one octave lower
			bassMIDI := noteToMIDI(bassRoot, octave-1)
			if bassMIDI >= 0 && bassMIDI <= 127 {
				// Prepend bass note
				notes = append([]int{bassMIDI}, notes...)
			}
		}
	}

	if len(notes) == 0 {
		return nil, fmt.Errorf("no valid MIDI notes generated for chord: %s", chordSymbol)
	}

	return notes, nil
}

// ConvertArrangerActionToNoteEvents converts an arranger action to NoteEvent array
// Handles: arpeggios, chords, progressions
func ConvertArrangerActionToNoteEvents(action map[string]any, startBeat float64) ([]models.NoteEvent, error) {
	actionType, ok := action["type"].(string)
	if !ok {
		return nil, fmt.Errorf("action missing type field")
	}

	switch actionType {
	case "arpeggio":
		return convertArpeggioToNoteEvents(action, startBeat)
	case "chord":
		return convertChordToNoteEvents(action, startBeat)
	case "progression":
		return convertProgressionToNoteEvents(action, startBeat)
	default:
		return nil, fmt.Errorf("unknown action type: %s", actionType)
	}
}

// convertArpeggioToNoteEvents converts an arpeggio action to sequential NoteEvents
func convertArpeggioToNoteEvents(action map[string]any, startBeat float64) ([]models.NoteEvent, error) {
	chordSymbol, ok := action["chord"].(string)
	if !ok {
		return nil, fmt.Errorf("arpeggio missing chord field")
	}

	length, _ := getFloat(action, "length", 4.0) // Default: 1 bar (4 beats)
	repeat, _ := getInt(action, "repeat", 1)
	velocity, _ := getInt(action, "velocity", 100)
	octave, _ := getInt(action, "octave", 4)
	direction, _ := getString(action, "direction", "up")
	
	// Check for explicit note_duration (e.g., 0.25 for 16th notes)
	explicitNoteDuration, hasNoteDuration := getFloat(action, "note_duration", 0)

	// Get chord notes
	chordNotes, err := ChordToMIDI(chordSymbol, octave)
	if err != nil {
		return nil, err
	}

	// Apply direction
	if direction == "down" {
		chordNotes = reverseSlice(chordNotes)
	} else if direction == "updown" {
		// Up then down (excluding duplicate middle note)
		up := chordNotes
		down := reverseSlice(chordNotes[1:])
		chordNotes = append(up, down...)
	}

	// Calculate note duration
	noteCount := len(chordNotes)
	var noteDuration float64
	if hasNoteDuration && explicitNoteDuration > 0 {
		// Use explicit note duration (e.g., 0.25 for 16th notes)
		noteDuration = explicitNoteDuration
		log.Printf("🎵 Using explicit note_duration: %.4f beats", noteDuration)
	} else {
		// Divide length by number of notes
		noteDuration = length / float64(noteCount)
	}

	var noteEvents []models.NoteEvent
	currentBeat := startBeat

	for r := 0; r < repeat; r++ {
		for _, midiNote := range chordNotes {
			noteEvents = append(noteEvents, models.NoteEvent{
				MidiNoteNumber: midiNote,
				Velocity:       velocity,
				StartBeats:     currentBeat,
				DurationBeats:  noteDuration,
			})
			currentBeat += noteDuration
		}
	}

	return noteEvents, nil
}

// convertChordToNoteEvents converts a chord action to simultaneous NoteEvents
func convertChordToNoteEvents(action map[string]any, startBeat float64) ([]models.NoteEvent, error) {
	chordSymbol, ok := action["chord"].(string)
	if !ok {
		return nil, fmt.Errorf("chord missing chord field")
	}

	length, _ := getFloat(action, "length", 4.0) // Default: 1 bar (4 beats)
	repeat, _ := getInt(action, "repeat", 1)
	velocity, _ := getInt(action, "velocity", 100)
	octave, _ := getInt(action, "octave", 4)

	// Get chord notes
	chordNotes, err := ChordToMIDI(chordSymbol, octave)
	if err != nil {
		return nil, err
	}

	var noteEvents []models.NoteEvent
	currentBeat := startBeat

	for r := 0; r < repeat; r++ {
		// All notes start at the same time (simultaneous chord)
		for _, midiNote := range chordNotes {
			noteEvents = append(noteEvents, models.NoteEvent{
				MidiNoteNumber: midiNote,
				Velocity:       velocity,
				StartBeats:     currentBeat,
				DurationBeats:  length,
			})
		}
		currentBeat += length
	}

	return noteEvents, nil
}

// convertProgressionToNoteEvents converts a progression action to NoteEvents
func convertProgressionToNoteEvents(action map[string]any, startBeat float64) ([]models.NoteEvent, error) {
	log.Printf("🎵 convertProgressionToNoteEvents: action=%+v", action)
	
	chords, ok := action["chords"].([]string)
	if !ok {
		log.Printf("🎵 chords not []string, trying []interface{}")
		// Try to extract from interface{} slice
		if chordsInterface, ok := action["chords"].([]interface{}); ok {
			log.Printf("🎵 found []interface{} with %d items", len(chordsInterface))
			chords = make([]string, len(chordsInterface))
			for i, c := range chordsInterface {
				if str, ok := c.(string); ok {
					chords[i] = str
					log.Printf("🎵 chord[%d] = %s", i, str)
				} else {
					log.Printf("🎵 chord[%d] is not string: %T = %v", i, c, c)
				}
			}
		} else {
			log.Printf("🎵 chords field: %T = %v", action["chords"], action["chords"])
			return nil, fmt.Errorf("progression missing chords field")
		}
	}

	log.Printf("🎵 Extracted chords: %v (len=%d)", chords, len(chords))

	length, _ := getFloat(action, "length", float64(len(chords))*4.0) // Default: 1 bar per chord
	repeat, _ := getInt(action, "repeat", 1)
	velocity, _ := getInt(action, "velocity", 100)
	octave, _ := getInt(action, "octave", 4)

	log.Printf("🎵 Progression params: length=%.2f, repeat=%d, velocity=%d, octave=%d", length, repeat, velocity, octave)

	// Calculate chord duration
	chordDuration := length / float64(len(chords))

	log.Printf("🎵 chordDuration=%.2f (length %.2f / %d chords)", chordDuration, length, len(chords))

	var noteEvents []models.NoteEvent
	currentBeat := startBeat

	for r := 0; r < repeat; r++ {
		log.Printf("🎵 Repeat %d/%d", r+1, repeat)
		for chordIdx, chordSymbol := range chords {
			log.Printf("🎵 Processing chord %d/%d: %s", chordIdx+1, len(chords), chordSymbol)
			chordNotes, err := ChordToMIDI(chordSymbol, octave)
			if err != nil {
				log.Printf("🎵 ERROR: ChordToMIDI failed for %s: %v", chordSymbol, err)
				return nil, fmt.Errorf("invalid chord in progression: %s: %w", chordSymbol, err)
			}

			log.Printf("🎵 Chord %s => MIDI notes: %v", chordSymbol, chordNotes)

			// All notes of the chord start simultaneously
			for _, midiNote := range chordNotes {
				noteEvents = append(noteEvents, models.NoteEvent{
					MidiNoteNumber: midiNote,
					Velocity:       velocity,
					StartBeats:     currentBeat,
					DurationBeats:  chordDuration,
				})
			}

			currentBeat += chordDuration
		}
	}

	log.Printf("🎵 convertProgressionToNoteEvents: returning %d noteEvents", len(noteEvents))
	return noteEvents, nil
}

// Helper functions

func parseRootNote(chordSymbol string) (string, error) {
	if len(chordSymbol) == 0 {
		return "", fmt.Errorf("empty chord symbol")
	}

	// Extract root (first 1-2 chars: C, C#, Db, etc.)
	root := ""
	if len(chordSymbol) > 1 && (chordSymbol[1] == '#' || chordSymbol[1] == 'b') {
		root = chordSymbol[:2]
	} else {
		root = chordSymbol[:1]
	}

	// Validate root note
	validRoots := map[string]bool{
		"C": true, "C#": true, "Db": true, "D": true, "D#": true, "Eb": true,
		"E": true, "F": true, "F#": true, "Gb": true, "G": true, "G#": true,
		"Ab": true, "A": true, "A#": true, "Bb": true, "B": true,
	}

	if !validRoots[root] {
		return "", fmt.Errorf("invalid root note: %s", root)
	}

	return root, nil
}

func parseChordQuality(chordSymbol string) string {
	// Remove root note
	if len(chordSymbol) > 1 && (chordSymbol[1] == '#' || chordSymbol[1] == 'b') {
		chordSymbol = chordSymbol[2:]
	} else if len(chordSymbol) > 0 {
		chordSymbol = chordSymbol[1:]
	}

	// Check for quality markers
	if strings.HasPrefix(chordSymbol, "m") && !strings.HasPrefix(chordSymbol, "maj") && !strings.HasPrefix(chordSymbol, "min") {
		return "minor"
	}
	if strings.HasPrefix(chordSymbol, "dim") {
		return "diminished"
	}
	if strings.HasPrefix(chordSymbol, "aug") {
		return "augmented"
	}
	if strings.HasPrefix(chordSymbol, "sus2") {
		return "sus2"
	}
	if strings.HasPrefix(chordSymbol, "sus4") {
		return "sus4"
	}

	// Default to major
	return "major"
}

func parseExtensions(chordSymbol string) []string {
	extensions := []string{}

	// Remove root and quality
	if len(chordSymbol) > 1 && (chordSymbol[1] == '#' || chordSymbol[1] == 'b') {
		chordSymbol = chordSymbol[2:]
	} else if len(chordSymbol) > 0 {
		chordSymbol = chordSymbol[1:]
	}

	// Remove quality markers
	chordSymbol = strings.TrimPrefix(chordSymbol, "m")
	chordSymbol = strings.TrimPrefix(chordSymbol, "dim")
	chordSymbol = strings.TrimPrefix(chordSymbol, "aug")
	chordSymbol = strings.TrimPrefix(chordSymbol, "sus2")
	chordSymbol = strings.TrimPrefix(chordSymbol, "sus4")

	// Extract extensions
	if strings.Contains(chordSymbol, "maj7") {
		extensions = append(extensions, "maj7")
		chordSymbol = strings.ReplaceAll(chordSymbol, "maj7", "")
	}
	if strings.Contains(chordSymbol, "min7") {
		extensions = append(extensions, "min7")
		chordSymbol = strings.ReplaceAll(chordSymbol, "min7", "")
	}
	if strings.Contains(chordSymbol, "7") {
		extensions = append(extensions, "7")
		chordSymbol = strings.ReplaceAll(chordSymbol, "7", "")
	}
	if strings.Contains(chordSymbol, "9") {
		extensions = append(extensions, "9")
	}
	if strings.Contains(chordSymbol, "11") {
		extensions = append(extensions, "11")
	}
	if strings.Contains(chordSymbol, "13") {
		extensions = append(extensions, "13")
	}
	if strings.Contains(chordSymbol, "add9") {
		extensions = append(extensions, "add9")
	}
	if strings.Contains(chordSymbol, "add11") {
		extensions = append(extensions, "add11")
	}
	if strings.Contains(chordSymbol, "add13") {
		extensions = append(extensions, "add13")
	}

	return extensions
}

func buildChordIntervals(quality string, extensions []string) []int {
	var intervals []int

	// Base triad
	switch quality {
	case "major":
		intervals = []int{0, 4, 7} // Root, Major 3rd, Perfect 5th
	case "minor":
		intervals = []int{0, 3, 7} // Root, Minor 3rd, Perfect 5th
	case "diminished":
		intervals = []int{0, 3, 6} // Root, Minor 3rd, Diminished 5th
	case "augmented":
		intervals = []int{0, 4, 8} // Root, Major 3rd, Augmented 5th
	case "sus2":
		intervals = []int{0, 2, 7} // Root, Major 2nd, Perfect 5th
	case "sus4":
		intervals = []int{0, 5, 7} // Root, Perfect 4th, Perfect 5th
	default:
		intervals = []int{0, 4, 7} // Default to major
	}

	// Add extensions
	for _, ext := range extensions {
		switch ext {
		case "7", "min7":
			intervals = append(intervals, 10) // Minor 7th
		case "maj7":
			intervals = append(intervals, 11) // Major 7th
		case "9", "add9":
			intervals = append(intervals, 14) // Major 9th
		case "11", "add11":
			intervals = append(intervals, 17) // Perfect 11th
		case "13", "add13":
			intervals = append(intervals, 21) // Major 13th
		}
	}

	return intervals
}

func noteToMIDI(note string, octave int) int {
	// Note to semitone offset from C
	noteMap := map[string]int{
		"C":  0,
		"C#": 1, "Db": 1,
		"D":  2,
		"D#": 3, "Eb": 3,
		"E":  4,
		"F":  5,
		"F#": 6, "Gb": 6,
		"G":  7,
		"G#": 8, "Ab": 8,
		"A":  9,
		"A#": 10, "Bb": 10,
		"B":  11,
	}

	offset, ok := noteMap[note]
	if !ok {
		return 60 // Default to C4
	}

	// C4 = 60, so: (octave * 12) + offset
	return (octave * 12) + offset
}

func getFloat(m map[string]any, key string, defaultValue float64) (float64, bool) {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val, true
		case int:
			return float64(val), true
		case int64:
			return float64(val), true
		}
	}
	return defaultValue, false
}

func getInt(m map[string]any, key string, defaultValue int) (int, bool) {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val, true
		case int64:
			return int(val), true
		case float64:
			return int(val), true
		}
	}
	return defaultValue, false
}

func getString(m map[string]any, key string, defaultValue string) (string, bool) {
	if v, ok := m[key]; ok {
		if str, ok := v.(string); ok {
			return str, true
		}
	}
	return defaultValue, false
}

func reverseSlice(s []int) []int {
	result := make([]int, len(s))
	for i, v := range s {
		result[len(s)-1-i] = v
	}
	return result
}

