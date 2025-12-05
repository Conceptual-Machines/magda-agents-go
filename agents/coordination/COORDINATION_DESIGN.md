# Multi-Agent Coordination Design: Arranger + DAW Agents

## Overview

When users request musical content (chord progressions, melodies), we need to coordinate two agents:

1. **Arranger Agent**: Understands musical DSL (chord notation like "Gmaj/E", Roman numerals "I VI IV") → generates `NoteEvent` arrays
2. **DAW Agent**: Understands REAPER API → generates REAPER actions with placeholders for musical content

The coordination layer injects arranger output into DAW action placeholders.

## Architecture

```
User Request: "add I VI IV progression to piano track at bar 9"
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 1: DAW Agent (First Pass)                              │
│ - Detects musical content in request                        │
│ - Generates structural actions:                             │
│   • create_clip_at_bar(track=0, bar=9)                      │
│   • add_midi(notes=<PLACEHOLDER_MUSICAL_CONTENT>)           │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 2: Arranger Agent                                      │
│ - Extracts musical DSL: "I VI IV progression"               │
│ - Generates NoteEvent array:                                │
│   [                                                          │
│     {midiNoteNumber: 60, velocity: 100, startBeats: 0, ...},│
│     {midiNoteNumber: 64, velocity: 100, startBeats: 0, ...},│
│     ...                                                      │
│   ]                                                          │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 3: Coordination Layer (Injection)                      │
│ - Matches placeholder markers                               │
│ - Replaces <PLACEHOLDER_MUSICAL_CONTENT> with NoteEvent[]   │
│ - Returns final actions array                               │
└─────────────────────────────────────────────────────────────┘
```

## Placeholder Design

### Option 1: Special Marker Object (Recommended)
```json
{
  "action": "add_midi",
  "track": 0,
  "notes": {
    "_placeholder": true,
    "_type": "musical_content",
    "_description": "I VI IV progression",
    "_key": "C:maj",
    "_context": "chord_progression"
  }
}
```

**Pros:**
- Clear, structured marker
- Preserves context for arranger agent
- Easy to detect and replace

**Cons:**
- Slightly more complex JSON structure

### Option 2: Null with Metadata
```json
{
  "action": "add_midi",
  "track": 0,
  "notes": null,
  "_notes_placeholder": {
    "type": "musical_content",
    "description": "I VI IV progression"
  }
}
```

### Option 3: String Marker (Simplest)
```json
{
  "action": "add_midi",
  "track": 0,
  "notes": "<PLACEHOLDER:I VI IV progression:C:maj>"
}
```

**Pros:**
- Very simple
- Human-readable

**Cons:**
- Parsing required
- Less structured

## Recommendation: Option 1 (Special Marker Object)

## Implementation Plan

### 1. Update DAW Agent Parser

**File**: `agents/daw/dsl_parser_functional.go` or `dsl_parser.go`

When parsing `.add_midi()` calls, if notes are not provided, create placeholder:

```go
func (r *ReaperDSL) AddMidi(args gs.Args) error {
    // ...
    if notes, ok := args["notes"]; !ok || isEmpty(notes) {
        // Create placeholder
        action["notes"] = map[string]interface{}{
            "_placeholder": true,
            "_type": "musical_content",
            "_description": extractMusicalDescription(), // From context
            "_key": detectKey(), // Optional: from state or default
        }
    }
    // ...
}
```

### 2. Create Coordination Service

**File**: `agents/coordination/orchestrator.go`

```go
package coordination

import (
    "github.com/Conceptual-Machines/magda-agents-go/models"
    magdadaw "github.com/Conceptual-Machines/magda-agents-go/agents/daw"
    magdaarranger "github.com/Conceptual-Machines/magda-agents-go/agents/arranger"
)

type Orchestrator struct {
    dawAgent *magdadaw.DawAgent
    arrangerAgent *magdaarranger.ArrangerAgent
}

// ProcessRequest coordinates between DAW and Arranger agents
func (o *Orchestrator) ProcessRequest(
    ctx context.Context,
    question string,
    state map[string]interface{},
) (*DawResult, error) {
    // Step 1: DAW agent generates actions with placeholders
    dawResult, err := o.dawAgent.GenerateActions(ctx, question, state)
    if err != nil {
        return nil, err
    }

    // Step 2: Find placeholders in actions
    placeholders := findPlaceholders(dawResult.Actions)
    if len(placeholders) == 0 {
        // No musical content, return as-is
        return dawResult, nil
    }

    // Step 3: Generate musical content for each placeholder
    for _, placeholder := range placeholders {
        // Extract musical description from placeholder
        musicalDesc := placeholder["_description"].(string)
        
        // Call arranger agent
        arrangerResult, err := o.arrangerAgent.Generate(ctx, musicalDesc)
        if err != nil {
            return nil, err
        }

        // Step 4: Inject NoteEvent array into placeholder location
        notes := arrangerResult.OutputParsed.Choices[0].Notes
        injectNotes(dawResult.Actions, placeholder, notes)
    }

    return dawResult, nil
}
```

### 3. Update Arranger Agent Integration

**File**: `agents/arranger/arranger_agent.go`

The arranger agent already generates `MusicalChoice` with `Notes []NoteEvent`. We just need to ensure it can handle:
- Roman numeral progressions: "I VI IV"
- Chord symbols: "Gmaj/E", "Am7"
- Musical descriptions: "add a I VI IV progression"

### 4. Update API Handler

**File**: `aideas-api/internal/api/handlers/magda.go`

```go
func (h *MagdaHandler) Chat(c *gin.Context) {
    // ...
    
    // Use orchestrator instead of direct DAW agent
    orchestrator := coordination.NewOrchestrator(
        h.magdaService,  // DAW agent
        h.arrangerService, // Arranger agent
    )
    
    result, err := orchestrator.ProcessRequest(
        c.Request.Context(),
        req.Question,
        req.State,
    )
    // ...
}
```

## Example Flow

### Input
```json
{
  "question": "add I VI IV progression to piano track at bar 9",
  "state": {...}
}
```

### Step 1: DAW Agent Output
```json
{
  "actions": [
    {
      "action": "create_clip_at_bar",
      "track": 0,
      "bar": 9,
      "length_bars": 4
    },
    {
      "action": "add_midi",
      "track": 0,
      "notes": {
        "_placeholder": true,
        "_type": "musical_content",
        "_description": "I VI IV progression",
        "_key": "C:maj"
      }
    }
  ]
}
```

### Step 2: Arranger Agent Output
```json
{
  "choices": [{
    "description": "I VI IV progression in C major",
    "notes": [
      {
        "midiNoteNumber": 60,
        "velocity": 100,
        "startBeats": 0.0,
        "durationBeats": 4.0
      },
      {
        "midiNoteNumber": 64,
        "velocity": 100,
        "startBeats": 0.0,
        "durationBeats": 4.0
      },
      {
        "midiNoteNumber": 67,
        "velocity": 100,
        "startBeats": 0.0,
        "durationBeats": 4.0
      },
      // ... VI chord notes
      // ... IV chord notes
    ]
  }]
}
```

### Step 3: Final Injected Actions
```json
{
  "actions": [
    {
      "action": "create_clip_at_bar",
      "track": 0,
      "bar": 9,
      "length_bars": 4
    },
    {
      "action": "add_midi",
      "track": 0,
      "notes": [
        {
          "midiNoteNumber": 60,
          "velocity": 100,
          "startBeats": 0.0,
          "durationBeats": 4.0
        },
        // ... rest of notes
      ]
    }
  ]
}
```

## Key Detection

The key can be:
1. **Explicit**: User says "I VI IV in C major"
2. **From State**: Extract from REAPER project key signature
3. **Default**: C major if not specified

## Timing Context

Notes need timing relative to:
- Bar position (e.g., bar 9)
- Clip start position
- Beat positions within the bar

The arranger agent should generate `startBeats` relative to bar start (0-based within the bar), and the coordination layer may need to adjust based on the clip's bar position.

## Next Steps

1. ✅ Design placeholder mechanism
2. ⏳ Implement placeholder detection in DAW agent
3. ⏳ Create coordination orchestrator
4. ⏳ Update API handler to use orchestrator
5. ⏳ Test with example: "add I VI IV progression to piano track at bar 9"

