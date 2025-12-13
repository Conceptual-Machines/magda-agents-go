# Test Prompts for DAW/Arranger Orchestrator Integration

## Basic Tests

### Test 1: Simple Chord Progression
**Prompt:**
```
create a new track called "Piano" and add a C Am F G chord progression
```

**Expected Actions:**
1. `create_track` - name: "Piano", index: 0
2. `create_clip_at_bar` or `add_midi` with chord progression notes

**Expected Notes:**
- C major chord (C, E, G) - pitches 60, 64, 67
- A minor chord (A, C, E) - pitches 57, 60, 64
- F major chord (F, A, C) - pitches 65, 69, 72
- G major chord (G, B, D) - pitches 67, 71, 74

---

### Test 2: Simple Arpeggio
**Prompt:**
```
create a track with Serum and add an E minor arpeggio
```

**Expected Actions:**
1. `create_track` - name includes instrument, index: 0
2. `add_plugin` or `create_track` with instrument: "Serum"
3. `add_midi` with arpeggio notes

**Expected Notes:**
- E minor arpeggio (E, G, B) - pitches 64, 67, 71 (assuming octave 4)
- Sequential timing (arpeggio pattern)

---

### Test 3: Track with Clip and Chord Progression
**Prompt:**
```
create a track called piano and add a 4 bar clip starting at bar 1 with an E minor arpeggio
```

**Expected Actions:**
1. `create_track` - name: "piano", index: 0
2. `create_clip_at_bar` - bar: 1, length_bars: 4, track: 0
3. `add_midi` - E minor arpeggio notes

**Expected Notes:**
- E minor arpeggio (E, G, B) - pitches 52, 55, 59 (E4, G4, B4)
- Or (E, G, B) - pitches 64, 67, 71 (E5, G5, B5)
- Sequential pattern over 4 bars

---

## Advanced Tests

### Test 4: Multiple Tracks with Different Content
**Prompt:**
```
create a bass track with a C minor arpeggio and a piano track with a C Eb G Bb progression
```

**Expected Actions:**
1. `create_track` - name: "bass"
2. `add_midi` - C minor arpeggio (C, Eb, G)
3. `create_track` - name: "piano"
4. `add_midi` - Cm7 chord progression

**Expected Notes:**
- Bass: C minor arpeggio (C, Eb, G) - pitches 36, 39, 43 (lower octave)
- Piano: Cm7 chord (C, Eb, G, Bb) - pitches 60, 63, 67, 70

---

### Test 5: Complex Chord Progression with Specific Bar Placement
**Prompt:**
```
create a track called keys, add a 4 bar clip at bar 1, and fill it with an Am Dm G C progression
```

**Expected Actions:**
1. `create_track` - name: "keys"
2. `create_clip_at_bar` - bar: 1, length_bars: 4
3. `add_midi` - Am Dm G C progression

**Expected Notes:**
- Each chord 1 bar long (total 4 bars)
- Am: A, C, E (pitches 57, 60, 64)
- Dm: D, F, A (pitches 62, 65, 69)
- G: G, B, D (pitches 67, 71, 74)
- C: C, E, G (pitches 60, 64, 67)

---

### Test 6: Arpeggio with Specific Parameters
**Prompt:**
```
add a 2-bar upward arpeggio in G major
```

**Expected Actions:**
1. `add_midi` - G major arpeggio with upward direction, 2 bars length

**Expected Notes:**
- G major arpeggio (G, B, D) - pitches 67, 71, 74
- Upward direction
- 2 bars total length

---

## Edge Cases

### Test 7: Chord Inversions
**Prompt:**
```
create a track and add a Cmaj7/E chord
```

**Expected Actions:**
1. `create_track`
2. `add_midi` - Cmaj7 with E as bass note

**Expected Notes:**
- Bass note: E (pitch 64)
- Chord tones: C, E, G, B (pitches 60, 64, 67, 71)
- E should be emphasized or placed in bass register

---

### Test 8: Multiple Repetitions
**Prompt:**
```
add a C major chord repeated 4 times, each lasting 1 beat
```

**Expected Actions:**
1. `add_midi` - C major chord with repeat: 4, length: 1

**Expected Notes:**
- C major (C, E, G) - pitches 60, 64, 67
- 4 repetitions, each 1 beat long
- Total: 12 notes (3 notes × 4 repetitions)
- Timing: 0-1, 1-2, 2-3, 3-4 beats

---

### Test 9: Mixed DAW and Arranger Operations
**Prompt:**
```
create 3 tracks called drums, bass, and piano, set the bass volume to -6dB, and add an E minor arpeggio to the bass track
```

**Expected Actions:**
1. `create_track` - name: "drums"
2. `create_track` - name: "bass"
3. `create_track` - name: "piano"
4. `set_track` - track: 1, volume_db: -6.0 (bass track)
5. `add_midi` - E minor arpeggio on bass track

---

## Current Test Execution

To test, use:
```bash
# Run standalone orchestrator test
cd /Users/lucaromagnoli/Dropbox/Code/Projects/magda-agents-go
go run cmd/test-orchestrator-arranger/main.go
```

Or test via REAPER:
1. Open REAPER
2. Open MAGDA chat
3. Enter one of the test prompts above
4. Check console for:
   - `"🔄 Merging X DAW actions with Y arranger actions"`
   - `"✅ Injected N notes into add_midi action"`
5. Verify MIDI notes appear in the clip

---

## Expected Console Output (API Server)

```
2025/12/13 XX:XX:XX ✅ CFG tool structure detected, text format set to CFG mode
2025/12/13 XX:XX:XX 🚨 CRITICAL: About to call OpenAI API with params.Model='gpt-5.1'
2025/12/13 XX:XX:XX 🔧 CFG GRAMMAR CONFIGURED: arranger_dsl (syntax: lark)
2025/12/13 XX:XX:XX ✅✅✅ Found DSL code in raw JSON input field: track(name="piano")...
2025/12/13 XX:XX:XX ✅ Functional DSL Parser: Translated X actions from DSL
2025/12/13 XX:XX:XX 🔄 Merging X DAW actions with Y arranger actions
2025/12/13 XX:XX:XX 🎵 Converting arranger action: type=chord, chord=Em
2025/12/13 XX:XX:XX ✅ Converted to N NoteEvents (starting at beat 0.00)
2025/12/13 XX:XX:XX 📊 Total NoteEvents from arranger: N
2025/12/13 XX:XX:XX ✅ Injected N notes into add_midi action
2025/12/13 XX:XX:XX ✅ MAGDA Chat: GenerateActions succeeded
```

---

## Troubleshooting

### Issue: Empty Clip
**Possible Causes:**
1. Orchestrator not running (check API server logs)
2. `add_midi` action missing from response
3. REAPER extension not processing `add_midi` correctly

**Check:**
```bash
# API server logs
docker-compose logs aideas-api --tail 50 | grep -E "(MAGDA|arranger|Merging)"

# REAPER console
# Look for "MAGDA: AddMIDI called: track_index=X"
```

### Issue: Wrong Notes
**Possible Causes:**
1. Chord symbol parsing incorrect
2. Octave calculation wrong
3. Note timing incorrect (beats vs PPQ)

**Check:**
- API server logs for "Converting arranger action"
- REAPER console for "Inserting note" logs
- Verify pitch values match expected MIDI note numbers

---

## Quick Reference: MIDI Note Numbers

| Note | Octave 3 | Octave 4 | Octave 5 |
|------|----------|----------|----------|
| C    | 48       | 60       | 72       |
| C#   | 49       | 61       | 73       |
| D    | 50       | 62       | 74       |
| Eb   | 51       | 63       | 75       |
| E    | 52       | 64       | 76       |
| F    | 53       | 65       | 77       |
| F#   | 54       | 66       | 78       |
| G    | 55       | 67       | 79       |
| Ab   | 56       | 68       | 80       |
| A    | 57       | 69       | 81       |
| Bb   | 58       | 70       | 82       |
| B    | 59       | 71       | 83       |

**E minor chord (Octave 4):** E4=64, G4=67, B4=71
**E minor chord (Octave 3):** E3=52, G3=55, B3=59
**C major chord (Octave 4):** C4=60, E4=64, G4=67
