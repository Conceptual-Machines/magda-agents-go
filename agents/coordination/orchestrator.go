package coordination

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	arranger "github.com/Conceptual-Machines/magda-agents-go/agents/arranger"
	"github.com/Conceptual-Machines/magda-agents-go/agents/daw"
	"github.com/Conceptual-Machines/magda-agents-go/agents/drummer"
	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/Conceptual-Machines/magda-agents-go/models"
)

// Orchestrator coordinates multiple agents (DAW + Arranger + Drummer) running in parallel
type Orchestrator struct {
	dawAgent      *daw.DawAgent
	arrangerAgent ArrangerAgent // Will be set when we integrate
	drummerAgent  *drummer.DrummerAgent
	llmProvider   llm.Provider
}

// ArrangerAgent interface for the arranger agent
// Uses the actual arranger agent's ArrangerResult type
type ArrangerAgent interface {
	GenerateActions(ctx context.Context, question string) (*arranger.ArrangerResult, error)
}

// ArrangerResult represents the output from the arranger agent (internal format)
type ArrangerResult struct {
	Actions []map[string]any `json:"actions"` // Parsed DSL actions
	Usage   any              `json:"usage"`
}

// MusicalChoice represents a musical composition choice
type MusicalChoice struct {
	Description string      `json:"description"`
	Notes       []NoteEvent `json:"notes"`
}

// NoteEvent represents a MIDI note event
type NoteEvent struct {
	MIDINoteNumber int     `json:"midiNoteNumber"`
	Velocity       int     `json:"velocity"`
	StartBeats     float64 `json:"startBeats"`
	LengthBeats    float64 `json:"lengthBeats"`
}

// OrchestratorResult combines results from all agents
type OrchestratorResult struct {
	Actions []map[string]any `json:"actions"`
	Usage   any              `json:"usage"`
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(cfg *config.Config) *Orchestrator {
	dawAgent := daw.NewDawAgent(cfg)
	llmProvider := llm.NewOpenAIProvider(cfg.OpenAIAPIKey)

	// Initialize arranger agent (basic, no MCP for now)
	arrangerAgent := arranger.NewBasicArrangerAgent(cfg)

	// Initialize drummer agent
	drummerAgent := drummer.NewDrummerAgent(cfg)

	o := &Orchestrator{
		dawAgent:      dawAgent,
		arrangerAgent: arrangerAgent,
		drummerAgent:  drummerAgent,
		llmProvider:   llmProvider,
	}

	return o
}

// GenerateActions coordinates parallel agent execution and merges results
func (o *Orchestrator) GenerateActions(ctx context.Context, question string, state map[string]any) (*OrchestratorResult, error) {
	// Step 1: Detect which agents are needed
	detectionStart := time.Now()
	needsDAW, needsArranger, needsDrummer, err := o.DetectAgentsNeeded(ctx, question)
	detectionDuration := time.Since(detectionStart)
	if err != nil {
		log.Printf("⏱️ Agent detection failed in %v", detectionDuration)
		// DetectAgentsNeeded already handles LLM validation when no keywords are found
		// If it returns an error, the request is out of scope
		return nil, err
	}

	log.Printf("🔍 Agent detection: DAW=%v, Arranger=%v, Drummer=%v (took %v)", needsDAW, needsArranger, needsDrummer, detectionDuration)

	// Step 1.5: Auto-enable DAW if arranger or drummer is needed but no tracks exist
	// This ensures track creation happens before musical content is added
	if (needsArranger || needsDrummer) && !needsDAW {
		trackCount := getTrackCount(state)
		if trackCount == 0 {
			log.Printf("🔧 Auto-enabling DAW agent: Musical agent needs a track but none exist")
			needsDAW = true
		}
	}

	// Step 2: Launch only needed agents in parallel
	var wg sync.WaitGroup
	var dawResult *daw.DawResult
	var arrangerResult *ArrangerResult
	var drummerResult *drummer.DrummerResult
	var dawErr error
	var dawDuration, arrangerDuration, drummerDuration time.Duration

	if needsDAW {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			result, err := o.dawAgent.GenerateActions(ctx, question, state)
			dawDuration = time.Since(start)
			if err != nil {
				dawErr = fmt.Errorf("daw agent: %w", err)
				log.Printf("⏱️ DAW agent failed in %v", dawDuration)
				return
			}
			log.Printf("⏱️ DAW agent completed in %v", dawDuration)
			dawResult = result
		}()
	}

	if needsArranger && o.arrangerAgent != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			// Call arranger agent with question
			result, err := o.arrangerAgent.GenerateActions(ctx, question)
			arrangerDuration = time.Since(start)
			if err != nil {
				log.Printf("⚠️ Arranger agent failed in %v: %v", arrangerDuration, err)
				return
			}
			log.Printf("⏱️ Arranger agent completed in %v", arrangerDuration)
			// Use arranger result directly
			arrangerResult = &ArrangerResult{
				Actions: result.Actions,
				Usage:   result.Usage,
			}
		}()
	}

	if needsDrummer && o.drummerAgent != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			// Build input array from question
			inputArray := []map[string]any{
				{
					"role":    "user",
					"content": question,
				},
			}
			result, err := o.drummerAgent.Generate(ctx, "gpt-5.1", inputArray)
			drummerDuration = time.Since(start)
			if err != nil {
				log.Printf("⚠️ Drummer agent failed in %v: %v", drummerDuration, err)
				return
			}
			log.Printf("⏱️ Drummer agent completed in %v", drummerDuration)
			drummerResult = result
		}()
	}

	// Wait for all active agents to complete
	wg.Wait()

	// Log timing summary
	log.Printf("⏱️ Agent timing summary: DAW=%v, Arranger=%v, Drummer=%v", dawDuration, arrangerDuration, drummerDuration)

	// Step 3: Handle errors
	// DAW is the gatekeeper - if it fails, fail the entire request
	// This prevents garbage results from Arranger/Drummer being returned for out-of-scope requests
	if dawErr != nil {
		return nil, fmt.Errorf("DAW agent failed: %w", dawErr)
	}
	// For non-DAW agents, partial failures are OK (their results just won't be included)

	// Step 4: Merge results
	return o.mergeResults(dawResult, arrangerResult, drummerResult)
}

// StreamActionCallback is called for each action found during streaming
type StreamActionCallback func(action map[string]any) error

// GenerateActionsStream coordinates agents and emits actions progressively via callback.
// This allows the UI to execute actions (create track, create clip) as they arrive,
// masking latency. MIDI notes are buffered until the clip is created, then emitted.
func (o *Orchestrator) GenerateActionsStream(
	ctx context.Context,
	question string,
	state map[string]any,
	callback StreamActionCallback,
) (*OrchestratorResult, error) {
	// Step 1: Detect which agents are needed
	detectionStart := time.Now()
	needsDAW, needsArranger, needsDrummer, err := o.DetectAgentsNeeded(ctx, question)
	detectionDuration := time.Since(detectionStart)
	if err != nil {
		log.Printf("⏱️ [Stream] Agent detection failed in %v", detectionDuration)
		// DetectAgentsNeeded already handles LLM validation when no keywords are found
		// If it returns an error, the request is out of scope
		return nil, err
	}

	log.Printf("🔍 [Stream] Agent detection: DAW=%v, Arranger=%v, Drummer=%v (took %v)", needsDAW, needsArranger, needsDrummer, detectionDuration)

	// Step 1.5: Auto-enable DAW if arranger or drummer is needed but no tracks exist
	if (needsArranger || needsDrummer) && !needsDAW {
		trackCount := getTrackCount(state)
		if trackCount == 0 {
			log.Printf("🔧 [Stream] Auto-enabling DAW agent: Musical agent needs a track but none exist")
			needsDAW = true
		}
	}

	// Track state for dependency resolution
	var (
		mu               sync.Mutex
		pendingNotes     []models.NoteEvent
		clipCreated      bool
		targetTrackIdx   int = 0
		allActions       []map[string]any
		dawComplete      bool
		arrangerComplete bool
		drummerComplete  bool
	)

	// Helper to emit action via callback and track it
	emitAction := func(action map[string]any) error {
		mu.Lock()
		allActions = append(allActions, action)
		mu.Unlock()
		if callback != nil {
			return callback(action)
		}
		return nil
	}

	// Helper to check if we can emit add_midi (needs clip and notes, and all agents done)
	tryEmitMidi := func() error {
		mu.Lock()
		defer mu.Unlock()

		if clipCreated && len(pendingNotes) > 0 && dawComplete && arrangerComplete && drummerComplete {
			// Convert NoteEvents to map format
			notesArray := make([]map[string]any, len(pendingNotes))
			for i, note := range pendingNotes {
				notesArray[i] = map[string]any{
					"pitch":    note.MidiNoteNumber,
					"velocity": note.Velocity,
					"start":    note.StartBeats,
					"length":   note.DurationBeats,
				}
			}

			midiAction := map[string]any{
				"action": "add_midi",
				"track":  targetTrackIdx,
				"notes":  notesArray,
			}

			log.Printf("🎵 [Stream] Emitting add_midi with %d notes to track %d", len(pendingNotes), targetTrackIdx)
			allActions = append(allActions, midiAction)
			pendingNotes = nil // Clear buffer

			if callback != nil {
				// Unlock before callback to avoid deadlock
				mu.Unlock()
				err := callback(midiAction)
				mu.Lock()
				return err
			}
		}
		return nil
	}

	// Step 2: Launch agents
	var wg sync.WaitGroup
	var dawErr error

	if needsDAW {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			defer func() {
				mu.Lock()
				dawComplete = true
				mu.Unlock()
				log.Printf("⏱️ [Stream] DAW agent completed in %v", time.Since(start))
				_ = tryEmitMidi()
			}()

			// Use streaming DAW agent
			dawCallback := func(action map[string]any) error {
				actionType, _ := action["action"].(string)
				log.Printf("🎬 [Stream] DAW action: %s", actionType)

				// Track clip creation for dependency resolution
				if actionType == "create_clip_at_bar" || actionType == "new_clip" {
					mu.Lock()
					clipCreated = true
					if trackIdx, ok := action["track"].(int); ok {
						targetTrackIdx = trackIdx
					}
					mu.Unlock()
					log.Printf("📋 [Stream] Clip created on track %d", targetTrackIdx)
				}

				// Track the track index from create_track
				if actionType == "create_track" {
					if idx, ok := action["index"].(int); ok {
						mu.Lock()
						targetTrackIdx = idx
						mu.Unlock()
					}
				}

				// Emit immediately (create_track, create_clip, etc.)
				return emitAction(action)
			}

			_, err := o.dawAgent.GenerateActionsStream(ctx, question, state, dawCallback)
			if err != nil {
				dawErr = fmt.Errorf("daw agent stream: %w", err)
				log.Printf("❌ [Stream] DAW agent error: %v", err)
			}
		}()
	} else {
		mu.Lock()
		dawComplete = true
		mu.Unlock()
	}

	if needsArranger && o.arrangerAgent != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			defer func() {
				mu.Lock()
				arrangerComplete = true
				mu.Unlock()
				log.Printf("⏱️ [Stream] Arranger agent completed in %v", time.Since(start))
				_ = tryEmitMidi()
			}()

			result, err := o.arrangerAgent.GenerateActions(ctx, question)
			if err != nil {
				log.Printf("⚠️ [Stream] Arranger agent error: %v", err)
				return
			}

			// Convert arranger actions to NoteEvents and buffer them
			currentBeat := 0.0
			for _, action := range result.Actions {
				noteEvents, err := arranger.ConvertArrangerActionToNoteEvents(action, currentBeat)
				if err != nil {
					log.Printf("⚠️ [Stream] Failed to convert arranger action: %v", err)
					continue
				}

				mu.Lock()
				pendingNotes = append(pendingNotes, noteEvents...)
				mu.Unlock()

				log.Printf("📦 [Stream] Buffered %d notes (total: %d)", len(noteEvents), len(pendingNotes))

				// Update beat position
				if length, ok := getFloat(action, "length"); ok {
					if repeat, ok := getInt(action, "repeat"); ok && repeat > 0 {
						currentBeat += length * float64(repeat)
					} else {
						currentBeat += length
					}
				}
			}
		}()
	} else {
		mu.Lock()
		arrangerComplete = true
		mu.Unlock()
	}

	if needsDrummer && o.drummerAgent != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			defer func() {
				mu.Lock()
				drummerComplete = true
				mu.Unlock()
				log.Printf("⏱️ [Stream] Drummer agent completed in %v", time.Since(start))
				_ = tryEmitMidi()
			}()

			// Build input array from question
			inputArray := []map[string]any{
				{
					"role":    "user",
					"content": question,
				},
			}
			result, err := o.drummerAgent.Generate(ctx, "gpt-5.1", inputArray)
			if err != nil {
				log.Printf("⚠️ [Stream] Drummer agent error: %v", err)
				return
			}

			// Emit drummer actions directly (they're already in action format)
			for _, action := range result.Actions {
				log.Printf("🥁 [Stream] Emitting drummer action: %v", action["type"])
				if emitErr := emitAction(action); emitErr != nil {
					log.Printf("⚠️ [Stream] Failed to emit drummer action: %v", emitErr)
				}
			}
		}()
	} else {
		mu.Lock()
		drummerComplete = true
		mu.Unlock()
	}

	// Wait for all agents
	wg.Wait()

	// Final check - emit any remaining MIDI
	_ = tryEmitMidi()

	// DAW is the gatekeeper - if it fails, fail the entire request
	// This prevents garbage results from Arranger/Drummer being returned for out-of-scope requests
	if dawErr != nil {
		return nil, fmt.Errorf("DAW agent failed: %w", dawErr)
	}

	// For non-DAW agents, partial failures are OK (their results just won't be included)

	// Return all collected actions
	mu.Lock()
	result := &OrchestratorResult{
		Actions: allActions,
	}
	mu.Unlock()

	log.Printf("✅ [Stream] Complete: %d total actions emitted", len(result.Actions))
	return result, nil
}

// DetectAgentsNeeded uses LLM to detect which musical agents are needed
// DAW agent is ALWAYS used (handles all REAPER operations: tracks, clips, FX, etc.)
// Arranger and Drummer are optional based on musical content requested
func (o *Orchestrator) DetectAgentsNeeded(ctx context.Context, question string) (needsDAW, needsArranger, needsDrummer bool, err error) {
	// Use LLM to classify if Arranger or Drummer are needed
	_, needsArranger, needsDrummer, llmErr := o.detectAgentsNeededLLM(ctx, question)
	if llmErr != nil {
		return false, false, false, fmt.Errorf("LLM classification failed: %w", llmErr)
	}

	// DAW is always needed - it handles all REAPER operations
	needsDAW = true

	return needsDAW, needsArranger, needsDrummer, nil
}

// detectAgentsNeededLLM uses LLM to classify which musical agents are needed
// DAW agent is always used (handled by caller), this only classifies Arranger and Drummer
// Returns needsArranger=false, needsDrummer=false if request is out of scope
func (o *Orchestrator) detectAgentsNeededLLM(ctx context.Context, question string) (needsDAW, needsArranger, needsDrummer bool, err error) {
	prompt := fmt.Sprintf(`You are a router for a music production AI system. Classify requests to determine which specialized agents are needed.

THE SYSTEM HAS 3 AGENTS:
1. DAW AGENT (always runs): Handles REAPER operations - tracks, clips, FX, volume, pan, mute, solo, routing. Does NOT generate musical content.
2. ARRANGER AGENT: Generates melodic/harmonic MIDI content - chords, arpeggios, melodies, basslines, chord progressions. Creates actual notes with pitches.
3. DRUMMER AGENT: Generates drum/percussion patterns - kick, snare, hi-hat, toms, cymbals. Creates rhythmic patterns on a grid.

YOUR TASK: Decide if ARRANGER and/or DRUMMER are needed (DAW always runs).

EXAMPLES:
- "create a track called Drums" → {"needsArranger": false, "needsDrummer": false} (just naming a track, no content)
- "add reverb to the bass" → {"needsArranger": false, "needsDrummer": false} (FX operation)
- "mute track 2" → {"needsArranger": false, "needsDrummer": false} (track control)
- "add a breakbeat pattern" → {"needsArranger": false, "needsDrummer": true} (generating drums)
- "create a funk groove with ghost notes" → {"needsArranger": false, "needsDrummer": true} (drum pattern)
- "add a chord progression in C major" → {"needsArranger": true, "needsDrummer": false} (harmonic content)
- "create an arpeggio" → {"needsArranger": true, "needsDrummer": false} (melodic content)
- "add sustained E1" → {"needsArranger": true, "needsDrummer": false} (single note = melodic content)
- "add note C4" → {"needsArranger": true, "needsDrummer": false} (single note = melodic content)
- "bass note at bar 2" → {"needsArranger": true, "needsDrummer": false} (single note = melodic content)
- "create a hip hop beat with kicks and snares" → {"needsArranger": false, "needsDrummer": true} (drum pattern)

REQUEST: "%s"

Return JSON: {"needsArranger": bool, "needsDrummer": bool}`, question)

	// Use a small, fast model for classification
	request := &llm.GenerationRequest{
		Model:         "gpt-4.1-mini", // Fast and cheap for classification
		InputArray:    []map[string]any{{"role": "user", "content": prompt}},
		ReasoningMode: "none",
		OutputSchema: &llm.OutputSchema{
			Name:        "MusicalAgentClassification",
			Description: "Classification of which musical agents (Arranger/Drummer) are needed",
			Schema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"needsArranger": map[string]any{
						"type": "boolean",
					},
					"needsDrummer": map[string]any{
						"type": "boolean",
					},
				},
				"required": []string{"needsArranger", "needsDrummer"},
			},
		},
	}

	resp, llmErr := o.llmProvider.Generate(ctx, request)
	if llmErr != nil {
		return false, false, false, fmt.Errorf("LLM classification failed: %w", llmErr)
	}

	// Parse response from RawOutput (JSON Schema returns structured JSON)
	result := struct {
		NeedsArranger bool `json:"needsArranger"`
		NeedsDrummer  bool `json:"needsDrummer"`
	}{
		NeedsArranger: false,
		NeedsDrummer:  false,
	}

	// Try to parse from RawOutput if available
	if resp.RawOutput != "" {
		if parseErr := json.Unmarshal([]byte(resp.RawOutput), &result); parseErr != nil {
			log.Printf("⚠️ Failed to parse LLM classification JSON: %v, raw: %s", parseErr, resp.RawOutput)
			return false, false, false, fmt.Errorf("failed to parse LLM classification: %w", parseErr)
		}
	}

	// DAW is always true (handled by caller), return Arranger and Drummer classification
	return true, result.NeedsArranger, result.NeedsDrummer, nil
}

// mergeResults combines DAW, Arranger, and Drummer results
func (o *Orchestrator) mergeResults(dawResult *daw.DawResult, arrangerResult *ArrangerResult, drummerResult *drummer.DrummerResult) (*OrchestratorResult, error) {
	result := &OrchestratorResult{
		Actions: []map[string]any{},
	}

	// If we only have arranger results (no DAW), convert arranger actions to NoteEvents
	// and create a simple DAW action structure
	if arrangerResult != nil && len(arrangerResult.Actions) > 0 && (dawResult == nil || len(dawResult.Actions) == 0) {
		// Convert arranger actions to NoteEvents
		allNoteEvents := []models.NoteEvent{}
		currentBeat := 0.0

		for _, action := range arrangerResult.Actions {
			noteEvents, err := arranger.ConvertArrangerActionToNoteEvents(action, currentBeat)
			if err != nil {
				log.Printf("⚠️ Failed to convert arranger action to NoteEvents: %v", err)
				continue
			}

			allNoteEvents = append(allNoteEvents, noteEvents...)

			// Update currentBeat for next action (sum of lengths)
			if length, ok := getFloat(action, "length"); ok {
				if repeat, ok := getInt(action, "repeat"); ok {
					currentBeat += length * float64(repeat)
				} else {
					currentBeat += length
				}
			}
		}

		// Create a DAW action to add MIDI notes
		if len(allNoteEvents) > 0 {
			// Convert models.NoteEvent to map format expected by DAW
			notesArray := make([]map[string]any, len(allNoteEvents))
			for i, note := range allNoteEvents {
				notesArray[i] = map[string]any{
					"pitch":    note.MidiNoteNumber,
					"velocity": note.Velocity,
					"start":    note.StartBeats,
					"length":   note.DurationBeats,
				}
			}

			// Create add_midi action
			midiAction := map[string]any{
				"action": "add_midi",
				"notes":  notesArray,
			}
			result.Actions = append(result.Actions, midiAction)
		}
	}

	// Add DAW actions
	if dawResult != nil {
		// If we have both DAW and arranger results, inject arranger NoteEvents into DAW actions
		if arrangerResult != nil && len(arrangerResult.Actions) > 0 {
			log.Printf("🔄 Merging %d DAW actions with %d arranger actions", len(dawResult.Actions), len(arrangerResult.Actions))

			// Convert all arranger actions to NoteEvents
			allNoteEvents := []models.NoteEvent{}
			currentBeat := 0.0

			for _, action := range arrangerResult.Actions {
				log.Printf("🎵 Converting arranger action: type=%v, chord=%v", action["type"], action["chord"])
				noteEvents, err := arranger.ConvertArrangerActionToNoteEvents(action, currentBeat)
				if err != nil {
					log.Printf("⚠️ Failed to convert arranger action to NoteEvents: %v", err)
					continue
				}

				log.Printf("✅ Converted to %d NoteEvents (starting at beat %.2f)", len(noteEvents), currentBeat)
				allNoteEvents = append(allNoteEvents, noteEvents...)

				// Update currentBeat for next action
				if length, ok := getFloat(action, "length"); ok {
					if repeat, ok := getInt(action, "repeat"); ok {
						currentBeat += length * float64(repeat)
					} else {
						currentBeat += length
					}
				}
			}

			log.Printf("📊 Total NoteEvents from arranger: %d", len(allNoteEvents))

			// Find add_midi actions and inject NoteEvents, or create one if needed
			hasMidiAction := false
			for _, action := range dawResult.Actions {
				actionType, ok := action["action"].(string)
				if !ok {
					result.Actions = append(result.Actions, action)
					continue
				}

				if actionType == "add_midi" {
					hasMidiAction = true
					// Convert models.NoteEvent to map format expected by DAW
					notesArray := make([]map[string]any, len(allNoteEvents))
					for i, note := range allNoteEvents {
						notesArray[i] = map[string]any{
							"pitch":    note.MidiNoteNumber,
							"velocity": note.Velocity,
							"start":    note.StartBeats,
							"length":   note.DurationBeats,
						}
					}
					action["notes"] = notesArray
					log.Printf("✅ Injected %d notes into add_midi action", len(notesArray))
				}
				result.Actions = append(result.Actions, action)
			}

			// If no add_midi action exists but we have NoteEvents, create one
			if !hasMidiAction && len(allNoteEvents) > 0 {
				// Find the last track index from DAW actions
				lastTrackIndex := -1
				for _, action := range dawResult.Actions {
					if track, ok := action["track"].(int); ok {
						lastTrackIndex = track
					} else if track, ok := action["index"].(int); ok {
						lastTrackIndex = track
					}
				}

				// Convert NoteEvents to map format
				notesArray := make([]map[string]any, len(allNoteEvents))
				for i, note := range allNoteEvents {
					notesArray[i] = map[string]any{
						"pitch":    note.MidiNoteNumber,
						"velocity": note.Velocity,
						"start":    note.StartBeats,
						"length":   note.DurationBeats,
					}
				}

				midiAction := map[string]any{
					"action": "add_midi",
					"notes":  notesArray,
				}
				if lastTrackIndex >= 0 {
					midiAction["track"] = lastTrackIndex
				}

				result.Actions = append(result.Actions, midiAction)
				log.Printf("✅ Created new add_midi action with %d notes (track=%d)", len(notesArray), lastTrackIndex)
			}
		} else {
			// No arranger results, just add DAW actions as-is
			result.Actions = append(result.Actions, dawResult.Actions...)
		}
		result.Usage = dawResult.Usage // TODO: merge usage from all agents
	}

	// Add drummer results (drum patterns)
	if drummerResult != nil && len(drummerResult.Actions) > 0 {
		log.Printf("🥁 Adding %d drummer actions", len(drummerResult.Actions))
		result.Actions = append(result.Actions, drummerResult.Actions...)
	}

	return result, nil
}

// Helper functions for type conversion
func getFloat(m map[string]any, key string) (float64, bool) {
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
	return 0, false
}

func getInt(m map[string]any, key string) (int, bool) {
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
	return 0, false
}

// getTrackCount extracts the number of tracks from the REAPER state
func getTrackCount(state map[string]any) int {
	if state == nil {
		return 0
	}
	if tracks, ok := state["tracks"]; ok {
		if trackArr, ok := tracks.([]any); ok {
			return len(trackArr)
		}
		// Handle typed slice (e.g., from JSON unmarshaling)
		if trackArr, ok := tracks.([]map[string]any); ok {
			return len(trackArr)
		}
	}
	return 0
}
