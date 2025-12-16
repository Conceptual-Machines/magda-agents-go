# Drummer/Percussionist Agent Specification

## Overview

A specialized agent for generating drum/percussion patterns using a text-based grid notation. The LLM outputs human-readable rhythm patterns using canonical drum names, which are then converted to MIDI notes based on user-selected drum kit mappings.

---

## 1. Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  USER REQUEST                                                        │
│  "Create a funky breakbeat with ghost notes on the snare"           │
│  + Context: @addictivedrums selected                                 │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  LLM (knows only canonical drum names)                              │
│                                                                      │
│  System prompt includes:                                             │
│  - Grid notation format                                              │
│  - Canonical drum vocabulary                                         │
│  - Musical context (genre, feel, etc.)                              │
│                                                                      │
│  Output: Grid notation with canonical names                          │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  GRID PARSER                                                         │
│                                                                      │
│  Parses text grid → structured beat data                            │
│  Validates drum names against canonical vocabulary                   │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  MIDI CONVERTER                                                      │
│                                                                      │
│  Uses user's drum kit mapping to convert:                           │
│  canonical name → MIDI note number                                  │
│                                                                      │
│  Example: "kick" → 36 (Addictive Drums)                            │
│           "kick" → 35 (Superior Drummer)                            │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  OUTPUT: add_midi actions                                           │
│                                                                      │
│  [                                                                   │
│    { action: "add_midi", track: 0, note: 36, position: 0.0, ... }, │
│    { action: "add_midi", track: 0, note: 38, position: 0.25, ... }, │
│    ...                                                               │
│  ]                                                                   │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Grid Notation Format

### 2.1 Basic Format

```
BPM: 120
TIME: 4/4
BARS: 4
RESOLUTION: 16  // 16th notes per bar

kick    |*---*---*---*---|*---*---*---*---|*---*---*---*---|*---*-*-*---*---|
snare   |----*-------*---|----*-------*---|----*-------*---|----*-------*-*-|
hi_hat  |*-*-*-*-*-*-*-*-|*-*-*-*-*-*-*-*-|*-*-*-*-*-*-*-*-|*-*-*-*-*-*-*-*-|
```

### 2.2 Symbol Definitions

| Symbol | Meaning | Velocity |
|--------|---------|----------|
| `*` | Normal hit | 100 |
| `>` | Accent (loud) | 127 |
| `o` | Ghost note (soft) | 50 |
| `O` | Open (for hi-hat) | 100 |
| `-` | Rest (no hit) | - |
| `x` | Cross-stick / rim | 100 |

### 2.3 Resolution Options

- `RESOLUTION: 8` → 8 characters per bar (8th notes)
- `RESOLUTION: 16` → 16 characters per bar (16th notes)
- `RESOLUTION: 32` → 32 characters per bar (32nd notes)

### 2.4 Extended Format (Triplets)

```
RESOLUTION: 12  // 12 = triplet feel (4 beats × 3 subdivisions)

kick    |*--*--*--*--|
snare   |---*-----*--|
hi_hat  |*-**-**-**-*|
```

---

## 3. Canonical Drum Vocabulary

### 3.1 Core Set (always available)

| Canonical Name | Description | Typical GM Note |
|----------------|-------------|-----------------|
| `kick` | Bass drum | 36 |
| `snare` | Snare drum (center) | 38 |
| `snare_rim` | Snare rim shot | 40 |
| `snare_xstick` | Cross-stick | 37 |
| `hi_hat` | Hi-hat closed | 42 |
| `hi_hat_open` | Hi-hat open | 46 |
| `hi_hat_pedal` | Hi-hat pedal | 44 |
| `tom_high` | High tom | 50 |
| `tom_mid` | Mid tom | 47 |
| `tom_low` | Low tom / floor tom | 45 |
| `crash` | Crash cymbal | 49 |
| `ride` | Ride cymbal | 51 |
| `ride_bell` | Ride bell | 53 |

### 3.2 Extended Set (optional)

| Canonical Name | Description |
|----------------|-------------|
| `china` | China cymbal |
| `splash` | Splash cymbal |
| `cowbell` | Cowbell |
| `tambourine` | Tambourine |
| `clap` | Hand clap |
| `snap` | Finger snap |
| `shaker` | Shaker |
| `conga_high` | High conga |
| `conga_low` | Low conga |
| `bongo_high` | High bongo |
| `bongo_low` | Low bongo |

---

## 4. Drum Kit Mappings

### 4.1 Preset Mappings (shipped with app)

**Addictive Drums 2** (`addictive_drums_v2`):
```json
{
  "id": "addictive_drums_v2",
  "name": "Addictive Drums 2",
  "notes": {
    "kick": 36,
    "snare": 38,
    "snare_rim": 40,
    "snare_xstick": 37,
    "hi_hat": 42,
    "hi_hat_open": 46,
    "hi_hat_pedal": 44,
    "tom_high": 50,
    "tom_mid": 47,
    "tom_low": 45,
    "crash": 49,
    "ride": 51,
    "ride_bell": 53
  }
}
```

**Superior Drummer 3** (`superior_drummer_v3`):
```json
{
  "id": "superior_drummer_v3",
  "name": "Superior Drummer 3",
  "notes": {
    "kick": 36,
    "snare": 38,
    ...
  }
}
```

**General MIDI** (`general_midi`):
```json
{
  "id": "general_midi",
  "name": "General MIDI Drums",
  "notes": { ... }
}
```

### 4.2 Custom Mappings

Users can create custom mappings via:
1. API endpoint (stored server-side)
2. Preferences UI in Reaper extension
3. JSON file import

### 4.3 Mapping Resolution

Priority order:
1. User's selected kit (via `@mention` or preferences)
2. User's default kit (from preferences)
3. General MIDI fallback

---

## 5. Parser Implementation

### 5.1 Input

```
BPM: 95
TIME: 4/4
BARS: 2
RESOLUTION: 16

kick    |*---*---*-*-*---|*---*---*-*-*---|
snare   |----*--o----*---|----*--o----*-*-|
hi_hat  |*-*-*-*-*-*-*-*-|*-*-*-*-*-*-*-*-|
```

### 5.2 Parsed Output (intermediate)

```go
type GridPattern struct {
    BPM        int
    TimeSignature string  // "4/4"
    Bars       int
    Resolution int
    Tracks     []DrumTrack
}

type DrumTrack struct {
    CanonicalName string       // "kick", "snare", etc.
    Hits          []DrumHit
}

type DrumHit struct {
    Position  float64  // In beats (0.0, 0.25, 0.5, ...)
    Velocity  int      // 0-127
    Symbol    string   // Original symbol for debugging
}
```

### 5.3 Final Output (MIDI actions)

```json
[
  {"action": "add_midi", "track": 0, "note": 36, "position": 0.0, "length": 0.1, "velocity": 100},
  {"action": "add_midi", "track": 0, "note": 36, "position": 0.5, "length": 0.1, "velocity": 100},
  {"action": "add_midi", "track": 0, "note": 38, "position": 0.25, "length": 0.1, "velocity": 100},
  {"action": "add_midi", "track": 0, "note": 38, "position": 0.4375, "length": 0.1, "velocity": 50},
  ...
]
```

---

## 6. LLM Prompt Engineering

### 6.1 System Prompt (excerpt)

```
You are a drummer/percussionist AI assistant. When asked to create drum patterns,
output them in GRID NOTATION format.

GRID NOTATION FORMAT:
- Header: BPM, TIME (time signature), BARS, RESOLUTION
- Each line: drum_name |pattern|pattern|...|
- Symbols: * (hit), - (rest), > (accent), o (ghost), O (open hi-hat)

CANONICAL DRUM NAMES (use exactly these):
kick, snare, snare_rim, snare_xstick, hi_hat, hi_hat_open, hi_hat_pedal,
tom_high, tom_mid, tom_low, crash, ride, ride_bell

EXAMPLE OUTPUT:
BPM: 120
TIME: 4/4
BARS: 2
RESOLUTION: 16

kick    |*---*---*---*---|*---*---*---*---|
snare   |----*-------*---|----*-------*---|
hi_hat  |*-*-*-*-*-*-*-*-|*-*-*-*-*-*-*-*-|

IMPORTANT:
- Always include header (BPM, TIME, BARS, RESOLUTION)
- Use exactly the canonical drum names
- Each bar must have exactly RESOLUTION characters
- Align patterns vertically for readability
```

### 6.2 CFG Grammar Constraint (optional)

For strict enforcement, use grammar-school-go to constrain output:

```
drum_pattern: header "\n\n" drum_lines

header: bpm_line "\n" time_line "\n" bars_line "\n" resolution_line

bpm_line: "BPM: " NUMBER
time_line: "TIME: " time_sig
bars_line: "BARS: " NUMBER
resolution_line: "RESOLUTION: " NUMBER

time_sig: NUMBER "/" NUMBER

drum_lines: drum_line ("\n" drum_line)*

drum_line: drum_name WHITESPACE "|" bar ("|" bar)* "|"

drum_name: "kick" | "snare" | "snare_rim" | "snare_xstick" 
         | "hi_hat" | "hi_hat_open" | "hi_hat_pedal"
         | "tom_high" | "tom_mid" | "tom_low"
         | "crash" | "ride" | "ride_bell"

bar: hit+

hit: "*" | "-" | ">" | "o" | "O" | "x"
```

---

## 7. API Endpoints

### 7.1 Drum Pattern Generation

```
POST /api/v1/drums/generate

Body:
{
  "prompt": "funky breakbeat with ghost notes",
  "bpm": 95,
  "bars": 4,
  "time_signature": "4/4",
  "mapping_id": "addictive_drums_v2"  // optional, uses default if omitted
}

Response:
{
  "grid": "BPM: 95\nTIME: 4/4\n...",
  "actions": [
    {"action": "add_midi", "track": 0, "note": 36, ...},
    ...
  ],
  "mapping_used": "addictive_drums_v2"
}
```

### 7.2 Validate/Parse Grid (utility)

```
POST /api/v1/drums/parse

Body:
{
  "grid": "BPM: 120\n...",
  "mapping_id": "addictive_drums_v2"
}

Response:
{
  "valid": true,
  "actions": [...],
  "errors": []  // or list of parsing errors
}
```

---

## 8. Integration with Arranger Agent

The drummer agent can be invoked from the main arranger when:
1. User explicitly asks for drums (`@addictivedrums create a beat`)
2. User asks for a full arrangement including drums
3. Arranger decides drums are needed for the request

**Handoff**:
```go
// In arranger agent
if requestInvolvesDrums(prompt) {
    drumActions := drummerAgent.Generate(drumPrompt, mapping)
    allActions = append(allActions, drumActions...)
}
```

---

## 9. Files to Create

```
agents/drummer/
  drummer_agent.go         // Main agent
  grid_parser.go           // Parse grid notation
  grid_parser_test.go
  midi_converter.go        // Convert to MIDI actions
  midi_converter_test.go
  mappings.go              // Drum kit mapping types
  mappings_presets.go      // Built-in mappings (AD2, SD3, GM)
  DRUMMER_AGENT_SPEC.md    // This file
```

---

## 10. Future Extensions

- **Humanization**: Add slight timing/velocity variations
- **Groove templates**: Swing, shuffle, push/pull feel
- **Fill generation**: Automatic fills at phrase boundaries
- **Pattern variations**: Generate variations on a theme
- **Multi-kit support**: Layer multiple drum plugins
- **Import from audio**: Analyze audio → generate grid (stretch goal)

