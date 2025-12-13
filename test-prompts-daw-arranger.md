# Test Prompts for DAW/Arranger Integration

These prompts test the orchestrator's ability to coordinate both DAW and Arranger agents.

## Basic Integration Tests

### 1. Track with Arpeggio
```
create a track called piano and add a 4 bar clip starting at bar 1 with an E minor arpeggio
```

**Expected:**
- DAW: `create_track` (name="piano"), `create_clip_at_bar` (bar=1, length_bars=4)
- Arranger: `arpeggio("Em", length=4)`
- Result: `add_midi` action with E minor arpeggio notes (E4, G4, B4)

### 2. Track with Chord Progression
```
create a new track with Serum instrument and add a C Am F G chord progression
```

**Expected:**
- DAW: `create_track` (instrument="Serum"), `create_clip_at_bar` (or similar)
- Arranger: `progression(chords=["C", "Am", "F", "G"], length=4)`
- Result: `add_midi` action with chord progression notes

### 3. Multiple Tracks with Different Musical Content
```
create a bass track with a C major chord and a lead track with a D minor arpeggio
```

**Expected:**
- DAW: Two `create_track` actions, two `create_clip` actions
- Arranger: `chord("C", length=1)` and `arpeggio("Dm", length=2)`
- Result: Two `add_midi` actions, one for each track

## Advanced Tests

### 4. Arpeggio with Specific Parameters
```
create a track with piano and add a 2 bar clip at bar 3 with an A minor arpeggio going up, 4 repetitions
```

**Expected:**
- DAW: `create_track`, `create_clip_at_bar` (bar=3, length_bars=2)
- Arranger: `arpeggio("Am", length=2, repeat=4, direction="up")`
- Result: `add_midi` with arpeggiated notes

### 5. Chord Progression with Inversions
```
create a track called strings and add a chord progression: C major, A minor with F bass, F major, G major
```

**Expected:**
- DAW: `create_track` (name="strings"), `create_clip`
- Arranger: `progression(chords=["C", "Am/F", "F", "G"], length=4)`
- Result: `add_midi` with progression including inversion

### 6. Track with Instrument and Musical Content
```
create a new track with Serum instrument, name it "Lead Synth", and add an E minor arpeggio for 4 bars starting at bar 1
```

**Expected:**
- DAW: `create_track` (instrument="Serum", name="Lead Synth"), `create_clip_at_bar` (bar=1, length_bars=4)
- Arranger: `arpeggio("Em", length=4)`
- Result: `add_midi` with arpeggio notes

## Edge Cases

### 7. No Clip Specified (should create one)
```
create a track with piano and add a C major chord
```

**Expected:**
- DAW: `create_track`, `create_clip` (default position/length)
- Arranger: `chord("C", length=1)`
- Result: `add_midi` with chord notes

### 8. Multiple Musical Elements
```
create a track called "Harmony" and add a C Am F G progression followed by an E minor arpeggio
```

**Expected:**
- DAW: `create_track`, `create_clip` (long enough for both)
- Arranger: `progression(chords=["C", "Am", "F", "G"], length=4)` and `arpeggio("Em", length=2)`
- Result: `add_midi` with both progression and arpeggio notes (sequential)

## Quick Test Commands

### Test via curl (assuming API running on localhost:8080)
```bash
# Test 1: Arpeggio
curl -X POST http://localhost:8080/api/v1/magda/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "question": "create a track called piano and add a 4 bar clip starting at bar 1 with an E minor arpeggio",
    "state": {
      "project": {"name": "", "length": 0.0},
      "play_state": {"playing": false, "paused": false, "recording": false, "position": 0.0, "cursor": 0.0},
      "time_selection": {"start": 0.0, "end": 0.0},
      "tracks": []
    }
  }'

# Test 2: Chord Progression
curl -X POST http://localhost:8080/api/v1/magda/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "question": "create a new track with Serum instrument and add a C Am F G chord progression",
    "state": {
      "project": {"name": "", "length": 0.0},
      "play_state": {"playing": false, "paused": false, "recording": false, "position": 0.0, "cursor": 0.0},
      "time_selection": {"start": 0.0, "end": 0.0},
      "tracks": []
    }
  }'
```

## Validation Checklist

For each test, verify:
- [ ] Both DAW and Arranger agents are detected (check logs for "Agent detection: DAW=true, Arranger=true")
- [ ] DAW actions include track and clip creation
- [ ] Arranger actions include musical content (arpeggio/chord/progression)
- [ ] Final result includes `add_midi` action with `notes` array
- [ ] Notes array contains valid MIDI note objects with `pitch`, `velocity`, `start`, `length`
- [ ] Notes are correctly positioned (start times, lengths)
- [ ] Chord/arpeggio notes match expected musical intervals

