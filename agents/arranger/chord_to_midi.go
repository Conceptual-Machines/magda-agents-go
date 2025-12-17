package services

import (
	"fmt"
	"log"
	"strings"

	"github.com/Conceptual-Machines/magda-agents-go/models"
)

// RhythmTemplate defines timing and accent patterns for musical elements
type RhythmTemplate struct {
	Name string
	// Offsets within a bar (in beats, 0-4 for 4/4 time)
	Offsets []float64
	// Velocity multipliers for accents (1.0 = normal)
	Accents []float64
	// Duration multiplier (affects note length, 0.0-1.0)
	Articulation float64
}

// Rhythm template constants
const (
	articulationHigh    = 0.9
	articulationMedium  = 0.8
	articulationMidHigh = 0.85
	articulationShort   = 0.4
	articulationOverlap = 1.1
)

// Predefined rhythm templates (matching aideas-api)
var rhythmTemplates = map[string]RhythmTemplate{
	// Basic subdivisions
	"whole": {
		Name:         "whole",
		Offsets:      []float64{0},
		Accents:      []float64{1.0},
		Articulation: 1.0,
	},
	"half": {
		Name:         "half",
		Offsets:      []float64{0, 2},
		Accents:      []float64{1.0, 0.9},
		Articulation: 1.0,
	},
	"quarters": {
		Name:         "quarters",
		Offsets:      []float64{0, 1, 2, 3},
		Accents:      []float64{1.0, 0.8, 0.9, 0.8},
		Articulation: articulationHigh,
	},
	"8ths": {
		Name:         "8ths",
		Offsets:      []float64{0, 0.5, 1, 1.5, 2, 2.5, 3, 3.5},
		Accents:      []float64{1.0, 0.7, 0.9, 0.7, 0.95, 0.7, 0.9, 0.7},
		Articulation: articulationMidHigh,
	},
	"16ths": {
		Name:         "16ths",
		Offsets:      []float64{0, 0.25, 0.5, 0.75, 1, 1.25, 1.5, 1.75, 2, 2.25, 2.5, 2.75, 3, 3.25, 3.5, 3.75},
		Accents:      []float64{1.0, 0.6, 0.8, 0.6, 0.9, 0.6, 0.8, 0.6, 0.95, 0.6, 0.8, 0.6, 0.9, 0.6, 0.8, 0.6},
		Articulation: articulationMedium,
	},
	// Swing patterns
	"swing": {
		Name:         "swing",
		Offsets:      []float64{0, 0.67, 1, 1.67, 2, 2.67, 3, 3.67}, // Triplet feel
		Accents:      []float64{1.0, 0.7, 0.9, 0.7, 0.95, 0.7, 0.9, 0.7},
		Articulation: articulationMidHigh,
	},
	"shuffle": {
		Name:         "shuffle",
		Offsets:      []float64{0, 0.67, 1, 1.67, 2, 2.67, 3, 3.67},
		Accents:      []float64{1.0, 0.8, 0.9, 0.8, 1.0, 0.8, 0.9, 0.8},
		Articulation: articulationHigh,
	},
	// Latin patterns
	"bossa": {
		Name:         "bossa",
		Offsets:      []float64{0, 1.5, 3, 4.5, 6, 7.5}, // Characteristic bossa pattern over 2 bars
		Accents:      []float64{1.0, 0.8, 0.9, 0.8, 1.0, 0.8},
		Articulation: articulationHigh,
	},
	"samba": {
		Name:         "samba",
		Offsets:      []float64{0, 0.5, 1.5, 2, 3, 3.5},
		Accents:      []float64{1.0, 0.7, 0.9, 0.85, 0.95, 0.7},
		Articulation: articulationMedium,
	},
	"tresillo": {
		Name:         "tresillo",
		Offsets:      []float64{0, 1.5, 3}, // 3+3+2 pattern
		Accents:      []float64{1.0, 0.9, 0.95},
		Articulation: articulationHigh,
	},
	// Waltz and compound time
	"waltz": {
		Name:         "waltz",
		Offsets:      []float64{0, 1, 2}, // 3/4 time
		Accents:      []float64{1.0, 0.7, 0.75},
		Articulation: articulationHigh,
	},
	"6/8": {
		Name:         "6/8",
		Offsets:      []float64{0, 0.5, 1, 1.5, 2, 2.5},
		Accents:      []float64{1.0, 0.6, 0.7, 0.9, 0.6, 0.7},
		Articulation: articulationMidHigh,
	},
	// Syncopated patterns
	"offbeat": {
		Name:         "offbeat",
		Offsets:      []float64{0.5, 1.5, 2.5, 3.5},
		Accents:      []float64{0.9, 0.85, 0.9, 0.85},
		Articulation: articulationMidHigh,
	},
	"syncopated": {
		Name:         "syncopated",
		Offsets:      []float64{0, 0.5, 1.5, 2, 3, 3.5},
		Accents:      []float64{1.0, 0.8, 0.9, 0.85, 0.95, 0.8},
		Articulation: articulationMidHigh,
	},
	"anticipation": {
		Name:         "anticipation",
		Offsets:      []float64{0, 1, 1.75, 3, 3.75}, // Push before beats 2 and 4
		Accents:      []float64{1.0, 0.8, 0.9, 0.85, 0.9},
		Articulation: articulationMidHigh,
	},
	// Arpeggio patterns
	"broken": {
		Name:         "broken",
		Offsets:      []float64{0, 0.5, 1, 1.5},
		Accents:      []float64{1.0, 0.8, 0.85, 0.75},
		Articulation: articulationHigh,
	},
	"alberti": {
		Name:         "alberti",
		Offsets:      []float64{0, 0.25, 0.5, 0.75}, // Classical alberti bass pattern
		Accents:      []float64{1.0, 0.7, 0.85, 0.7},
		Articulation: articulationMidHigh,
	},
	"stride": {
		Name:         "stride",
		Offsets:      []float64{0, 1, 2, 3}, // Stride piano: bass-chord-bass-chord
		Accents:      []float64{1.0, 0.8, 0.9, 0.8},
		Articulation: articulationHigh,
	},
	// Special
	"staccato": {
		Name:         "staccato",
		Offsets:      []float64{0, 1, 2, 3},
		Accents:      []float64{1.0, 0.9, 0.95, 0.9},
		Articulation: articulationShort, // Short notes
	},
	"legato": {
		Name:         "legato",
		Offsets:      []float64{0, 1, 2, 3},
		Accents:      []float64{0.9, 0.85, 0.9, 0.85},
		Articulation: articulationOverlap, // Slightly overlapping
	},
}

// GetRhythmTemplate returns a rhythm template by name
func GetRhythmTemplate(name string) (RhythmTemplate, bool) {
	tmpl, ok := rhythmTemplates[name]
	return tmpl, ok
}

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
// Handles: arpeggios, chords, progressions, single notes
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
	case "note":
		return convertSingleNoteToNoteEvents(action, startBeat)
	default:
		return nil, fmt.Errorf("unknown action type: %s", actionType)
	}
}

// convertSingleNoteToNoteEvents converts a single note action to a NoteEvent
// Example: note(pitch="E1", duration=4) -> single E1 note for 4 beats
func convertSingleNoteToNoteEvents(action map[string]any, startBeat float64) ([]models.NoteEvent, error) {
	pitch, ok := action["pitch"].(string)
	if !ok {
		return nil, fmt.Errorf("note missing pitch field")
	}

	duration, _ := getFloat(action, "duration", 4.0) // Default: 4 beats (1 bar)
	velocity, _ := getInt(action, "velocity", 100)

	// Check for explicit start time in the action
	if explicitStart, ok := getFloat(action, "start", 0); ok && explicitStart != 0 {
		startBeat = explicitStart
	}

	// Convert note name (e.g., "E1", "C4", "F#3") to MIDI note number
	midiNote, err := NoteNameToMIDI(pitch)
	if err != nil {
		return nil, fmt.Errorf("invalid pitch %q: %w", pitch, err)
	}

	log.Printf("🎵 Single note: %s -> MIDI %d, duration=%.1f, velocity=%d, start=%.1f",
		pitch, midiNote, duration, velocity, startBeat)

	return []models.NoteEvent{
		{
			MidiNoteNumber: midiNote,
			Velocity:       velocity,
			StartBeats:     startBeat,
			DurationBeats:  duration,
		},
	}, nil
}

// NoteNameToMIDI converts a note name like "E1", "C4", "F#3", "Bb2" to MIDI note number
// Format: <note><accidental?><octave> where:
//   - note: A-G (case insensitive)
//   - accidental: # (sharp) or b (flat), optional
//   - octave: -1 to 9 (C4 = 60 = middle C)
func NoteNameToMIDI(noteName string) (int, error) {
	if len(noteName) < 2 {
		return 0, fmt.Errorf("note name too short: %s", noteName)
	}

	// Parse note letter (A-G)
	noteChar := strings.ToUpper(string(noteName[0]))
	if noteChar < "A" || noteChar > "G" {
		return 0, fmt.Errorf("invalid note letter: %s", noteChar)
	}

	// Note semitone offsets from C
	noteOffsets := map[string]int{
		"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11,
	}
	semitone := noteOffsets[noteChar]

	// Check for accidental (# or b)
	idx := 1
	if idx < len(noteName) {
		if noteName[idx] == '#' {
			semitone++
			idx++
		} else if noteName[idx] == 'b' {
			semitone--
			idx++
		}
	}

	// Parse octave (can be negative like -1)
	if idx >= len(noteName) {
		return 0, fmt.Errorf("missing octave in note name: %s", noteName)
	}

	octaveStr := noteName[idx:]
	var octave int
	_, err := fmt.Sscanf(octaveStr, "%d", &octave)
	if err != nil {
		return 0, fmt.Errorf("invalid octave in note name %s: %w", noteName, err)
	}

	// MIDI calculation: (octave + 1) * 12 + semitone
	// This gives C-1 = 0, C0 = 12, C4 = 60
	midiNote := (octave+1)*12 + semitone

	// Clamp to valid MIDI range
	if midiNote < 0 {
		midiNote = 0
	}
	if midiNote > 127 {
		midiNote = 127
	}

	return midiNote, nil
}

// convertArpeggioToNoteEvents converts an arpeggio action to sequential NoteEvents
func convertArpeggioToNoteEvents(action map[string]any, startBeat float64) ([]models.NoteEvent, error) {
	chordSymbol, ok := action["chord"].(string)
	if !ok {
		return nil, fmt.Errorf("arpeggio missing chord field")
	}

	length, _ := getFloat(action, "length", 4.0) // Default: 1 bar (4 beats)
	repeat, _ := getInt(action, "repeat", 0)     // 0 means auto-calculate to fill the bar
	velocity, _ := getInt(action, "velocity", 100)
	octave, _ := getInt(action, "octave", 4)
	direction, _ := getString(action, "direction", "up")
	rhythmTemplate, _ := getString(action, "rhythm", "")

	// Check for rhythm template first (overrides note_duration)
	if rhythmTemplate != "" {
		if _, ok := GetRhythmTemplate(rhythmTemplate); ok {
			// Use rhythm template for arpeggio timing
			log.Printf("🎵 Using rhythm template: %s", rhythmTemplate)
		} else {
			log.Printf("⚠️ Unknown rhythm template: %s, falling back to note_duration", rhythmTemplate)
			rhythmTemplate = "" // Clear invalid template
		}
	}

	// Check for explicit note_duration (e.g., 0.25 for 16th notes)
	// Default to 16th notes (0.25 beats) if not specified - this fills 1 bar nicely
	explicitNoteDuration, hasNoteDuration := getFloat(action, "note_duration", 0)
	var noteDuration float64
	if rhythmTemplate == "" { // Only use note_duration if no rhythm template
		if hasNoteDuration && explicitNoteDuration > 0 {
			noteDuration = explicitNoteDuration
			log.Printf("🎵 Using explicit note_duration: %.4f beats", noteDuration)
		} else {
			// Default to 16th notes (0.25 beats) for arpeggios
			noteDuration = 0.25
			log.Printf("🎵 Using default note_duration: 0.25 beats (16th notes)")
		}
	}

	// Get chord notes
	chordNotes, err := ChordToMIDI(chordSymbol, octave)
	if err != nil {
		return nil, err
	}

	// Check for rhythm template - if present, use it for timing
	if rhythmTemplate != "" {
		if tmpl, ok := GetRhythmTemplate(rhythmTemplate); ok {
			// Apply direction to create arpeggio sequence
			arpeggioNotes := chordNotes
			if direction == "down" {
				arpeggioNotes = reverseSlice(chordNotes)
			} else if direction == "updown" {
				// Create up-down pattern: up then reverse (skip last to avoid duplicate)
				up := make([]int, len(chordNotes))
				copy(up, chordNotes)
				down := reverseSlice(chordNotes[1:]) // Skip first to avoid duplicate
				arpeggioNotes = append(up, down...)
			}
			return applyRhythmTemplateToArpeggio(arpeggioNotes, velocity, startBeat, length, repeat, tmpl), nil
		}
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

	noteCount := len(chordNotes)

	// Calculate how many times to repeat to fill the bar
	// If repeat is 0 (auto), calculate based on length and note_duration
	actualRepeat := repeat
	if actualRepeat == 0 {
		totalNotes := int(length / noteDuration)
		actualRepeat = (totalNotes + noteCount - 1) / noteCount // Ceiling division
		if actualRepeat < 1 {
			actualRepeat = 1
		}
		log.Printf("🎵 Auto-calculated repeat=%d to fill %.1f beats with %d notes at %.2f beats each",
			actualRepeat, length, noteCount, noteDuration)
	}

	var noteEvents []models.NoteEvent
	currentBeat := startBeat
	endBeat := startBeat + length

	for r := 0; r < actualRepeat; r++ {
		for _, midiNote := range chordNotes {
			// Don't exceed the clip length
			if currentBeat >= endBeat {
				break
			}
			// Trim last note if it would exceed
			actualDuration := noteDuration
			if currentBeat+noteDuration > endBeat {
				actualDuration = endBeat - currentBeat
			}
			noteEvents = append(noteEvents, models.NoteEvent{
				MidiNoteNumber: midiNote,
				Velocity:       velocity,
				StartBeats:     currentBeat,
				DurationBeats:  actualDuration,
			})
			currentBeat += noteDuration
		}
		if currentBeat >= endBeat {
			break
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
	rhythmTemplate, _ := getString(action, "rhythm", "")

	// Get chord notes
	chordNotes, err := ChordToMIDI(chordSymbol, octave)
	if err != nil {
		return nil, err
	}

	// Check for rhythm template
	if rhythmTemplate != "" {
		if tmpl, ok := GetRhythmTemplate(rhythmTemplate); ok {
			return applyRhythmTemplateToChord(chordNotes, velocity, startBeat, length, repeat, tmpl), nil
		} else {
			log.Printf("⚠️ Unknown rhythm template: %s, using default chord behavior", rhythmTemplate)
		}
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

// applyRhythmTemplateToChord applies a rhythm template to chord notes
// This creates multiple chord hits at different beats based on the template
func applyRhythmTemplateToChord(chordNotes []int, velocity int, startBeat, length float64, repeat int, tmpl RhythmTemplate) []models.NoteEvent {
	var noteEvents []models.NoteEvent

	for r := 0; r < repeat; r++ {
		cycleStart := startBeat + (float64(r) * length)

		// Apply template offsets within each cycle
		for i, offset := range tmpl.Offsets {
			// Normalize offset to fit within the length
			beatPos := cycleStart + (offset * (length / 4.0)) // Assuming 4 beats = template cycle

			// Skip if beyond the cycle length
			if beatPos >= cycleStart+length {
				break
			}

			// Apply accent to velocity
			accent := velocity
			if i < len(tmpl.Accents) {
				accent = int(float64(velocity) * tmpl.Accents[i])
			}

			// Calculate note duration based on articulation
			noteDuration := (length / float64(len(tmpl.Offsets))) * tmpl.Articulation
			// Ensure note doesn't extend beyond next hit or cycle end
			if i+1 < len(tmpl.Offsets) {
				nextOffset := tmpl.Offsets[i+1] * (length / 4.0)
				maxDuration := nextOffset - offset*(length/4.0)
				if noteDuration > maxDuration {
					noteDuration = maxDuration
				}
			} else {
				maxDuration := length - (offset * (length / 4.0))
				if noteDuration > maxDuration {
					noteDuration = maxDuration
				}
			}

			// Create chord notes at this rhythm position
			for _, midiNote := range chordNotes {
				noteEvents = append(noteEvents, models.NoteEvent{
					MidiNoteNumber: midiNote,
					Velocity:       accent,
					StartBeats:     beatPos,
					DurationBeats:  noteDuration,
				})
			}
		}
	}

	return noteEvents
}

// applyRhythmTemplateToArpeggio applies a rhythm template to arpeggio notes
// This spaces out arpeggio notes according to the template timing
func applyRhythmTemplateToArpeggio(arpeggioNotes []int, velocity int, startBeat, length float64, repeat int, tmpl RhythmTemplate) []models.NoteEvent {
	var noteEvents []models.NoteEvent

	for r := 0; r < repeat; r++ {
		cycleStart := startBeat + (float64(r) * length)
		noteIndex := 0

		// Apply template offsets within each cycle
		for i, offset := range tmpl.Offsets {
			// Normalize offset to fit within the length
			beatPos := cycleStart + (offset * (length / 4.0)) // Assuming 4 beats = template cycle

			// Skip if beyond the cycle length
			if beatPos >= cycleStart+length {
				break
			}

			// Cycle through arpeggio notes
			if noteIndex >= len(arpeggioNotes) {
				noteIndex = 0
			}

			// Apply accent to velocity
			accent := velocity
			if i < len(tmpl.Accents) {
				accent = int(float64(velocity) * tmpl.Accents[i])
			}

			// Calculate note duration based on articulation
			noteDuration := (length / float64(len(tmpl.Offsets))) * tmpl.Articulation
			// Ensure note doesn't extend beyond next hit or cycle end
			if i+1 < len(tmpl.Offsets) {
				nextOffset := tmpl.Offsets[i+1] * (length / 4.0)
				maxDuration := nextOffset - offset*(length/4.0)
				if noteDuration > maxDuration {
					noteDuration = maxDuration
				}
			} else {
				maxDuration := length - (offset * (length / 4.0))
				if noteDuration > maxDuration {
					noteDuration = maxDuration
				}
			}

			// Create note at this rhythm position
			noteEvents = append(noteEvents, models.NoteEvent{
				MidiNoteNumber: arpeggioNotes[noteIndex],
				Velocity:       accent,
				StartBeats:     beatPos,
				DurationBeats:  noteDuration,
			})

			noteIndex++
		}
	}

	return noteEvents
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

	// Remove root note first
	if len(chordSymbol) > 1 && (chordSymbol[1] == '#' || chordSymbol[1] == 'b') {
		chordSymbol = chordSymbol[2:]
	} else if len(chordSymbol) > 0 {
		chordSymbol = chordSymbol[1:]
	}

	// Extract extensions BEFORE removing quality markers
	// This prevents "maj7" from being corrupted to "aj7" by TrimPrefix("m")
	if strings.Contains(chordSymbol, "maj7") {
		extensions = append(extensions, "maj7")
		chordSymbol = strings.ReplaceAll(chordSymbol, "maj7", "")
	}
	if strings.Contains(chordSymbol, "min7") {
		extensions = append(extensions, "min7")
		chordSymbol = strings.ReplaceAll(chordSymbol, "min7", "")
	}

	// Now remove quality markers (after extracting maj7/min7)
	chordSymbol = strings.TrimPrefix(chordSymbol, "m")
	chordSymbol = strings.TrimPrefix(chordSymbol, "dim")
	chordSymbol = strings.TrimPrefix(chordSymbol, "aug")
	chordSymbol = strings.TrimPrefix(chordSymbol, "sus2")
	chordSymbol = strings.TrimPrefix(chordSymbol, "sus4")

	// Extract remaining extensions
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
		"B": 11,
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
