# TODO - Remaining Implementation Tasks

## Critical Issues (Blocking)

### 1. Compound Actions - LLM Not Generating Multiple Actions for Clips
**Status**: Partially Working
**Issue**: LLM generates selection but not rename/color actions for clips
- ✅ Tracks: `filter(tracks, ...).set_selected(); filter(tracks, ...).set_name()` works
- ❌ Clips: `filter(clips, ...).set_selected(); filter(clips, ...).set_clip_name()` - LLM only generates selection
- **Tests Failing**: 
  - `TestMagdaCompoundActions/select_and_rename_clips`
  - `TestMagdaCompoundActions/filter_and_color_clips`
- **Next Steps**: 
  - Investigate why LLM skips second action for clips
  - May need more explicit prompt guidance or examples
  - Consider if there's a grammar limitation

### 2. SetName Not Handling Standalone Filter() Calls
**Status**: Bug
**Issue**: `filter(tracks, track.name == "X").set_name(name="Y")` fails with "no track context"
- **Error**: `method SetName error: no track context for name call`
- **Test Failing**: `TestMagdaRenameTracksWithFilter`
- **Root Cause**: `SetName` checks for `current_filtered` but may not be set correctly when filter is chained directly
- **Fix Needed**: Ensure `SetName` correctly handles filtered collections from chained filter calls

## High Priority Features

### 3. Clip Length Modification
**Status**: Not Implemented
**Action**: `set_clip_length` / `extend_clip`
- **REAPER API**: `SetMediaItemLength(MediaItem *item, double length, bool adjust_take_length)`
- **Use Cases**: 
  - "extend clip to 8 bars"
  - "make clip 2 seconds long"
- **DSL**: `filter(clips, clip.length < 2.0).set_clip_length(length=4.0)`
- **Files to Modify**:
  - `magda-reaper/include/magda_actions.h` - Add `SetClipLength()` declaration
  - `magda-reaper/src/magda_actions.cpp` - Implement `SetClipLength()`
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Add `SetClipLength()` method
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Update grammar

### 4. Track Color Support
**Status**: Not Implemented
**Action**: `set_track_color`
- **REAPER API**: `GetSetMediaTrackInfo` with "I_CUSTOMCOLOR"
- **Use Cases**: 
  - "color all muted tracks red"
  - "set track color to blue"
- **DSL**: `filter(tracks, track.muted == true).set_track_color(color="#ff0000")`
- **Files to Modify**:
  - `magda-reaper/include/magda_actions.h` - Add `SetTrackColor()` declaration
  - `magda-reaper/src/magda_actions.cpp` - Implement `SetTrackColor()`
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Add `SetTrackColor()` method
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Update grammar

### 5. FX Management - Remove/Enable/Disable
**Status**: Not Implemented
**Actions**: `remove_fx`, `enable_fx`, `disable_fx`, `set_fx_param`
- **REAPER APIs**: 
  - `TrackFX_Delete(MediaTrack *track, int fx_index)`
  - `TrackFX_SetEnabled(MediaTrack *track, int fx_index, bool enabled)`
  - `TrackFX_SetParam(MediaTrack *track, int fx_index, int param_index, double value)`
- **Use Cases**: 
  - "remove all EQ from track"
  - "disable all reverb"
  - "set reverb mix to 50%"
- **DSL**: 
  - `filter(fx_chain, fx.name == "ReaEQ").remove_fx()`
  - `filter(fx_chain, fx.name == "ReaVerb").set_fx_enabled(enabled=false)`
- **Files to Modify**:
  - `magda-reaper/include/magda_actions.h` - Add FX management methods
  - `magda-reaper/src/magda_actions.cpp` - Implement FX management
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Add FX methods
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Update grammar

## Medium Priority Features

### 6. Track Reordering
**Status**: Not Implemented
**Action**: `move_track` / `reorder_track`
- **REAPER API**: `ReorderSelectedTracks(int from_index, int to_index)` or track GUID manipulation
- **Use Cases**: 
  - "move track 3 to position 1"
  - "reorder tracks by name"
- **DSL**: `track(id=3).move_track(to_index=1)`
- **Files to Modify**:
  - `magda-reaper/include/magda_actions.h` - Add `MoveTrack()` declaration
  - `magda-reaper/src/magda_actions.cpp` - Implement `MoveTrack()`
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Add `MoveTrack()` method

### 7. Track Folder/Grouping
**Status**: Not Implemented
**Action**: `set_track_folder` / `group_tracks`
- **REAPER API**: Track folder flags via `GetSetMediaTrackInfo`
- **Use Cases**: 
  - "create folder track"
  - "group tracks 1-3"
- **DSL**: `filter(tracks, track.index in [0, 1, 2]).set_track_folder(folder=true)`
- **Files to Modify**:
  - `magda-reaper/include/magda_actions.h` - Add `SetTrackFolder()` declaration
  - `magda-reaper/src/magda_actions.cpp` - Implement `SetTrackFolder()`
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Add `SetTrackFolder()` method

### 8. MIDI Operations - Full Implementation
**Status**: Partially Implemented
**Issue**: `.add_midi()` notes parsing is placeholder
- **Current**: Basic structure exists but notes array parsing needs work
- **Needed**: Parse MIDI note arrays like `notes=[{pitch:60, velocity:100, start:0.0, length:1.0}, ...]`
- **Files to Modify**:
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Complete `AddMidi()` implementation

### 9. Map() Function Reference Execution
**Status**: Partially Implemented
**Issue**: `map()` currently just passes through items
- **Current**: Placeholder that returns items unchanged
- **Needed**: Actually call function references (e.g., `@get_name`)
- **Files to Modify**:
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Implement function registry and execution

### 10. ForEach() Function Reference Execution
**Status**: Partially Implemented
**Issue**: `for_each()` with function references returns error
- **Current**: Method calls work, function references don't
- **Needed**: Function registry and execution for `@function_name` references
- **Files to Modify**:
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Complete function reference support

## Test Fixes Needed

### 11. TestMagdaCreateTrackWithSerum
**Status**: Test Expectation Issue
**Issue**: Test expects `add_instrument` action but DSL correctly combines into `create_track` with `instrument` field
- **Fix**: Update test expectations to match actual behavior
- **File**: `aideas-api/internal/api/handlers/magda_integration_test.go`

## Documentation

### 12. Update Documentation
**Status**: Needed
- Update `IMPLEMENTATION_STATUS.md` with compound actions status
- Document new clip methods (set_clip_name, set_clip_color, move_clip)
- Add examples for compound actions in documentation

## Summary

**Critical (Blocking)**:
- Compound actions for clips (LLM not generating both actions)
- SetName handling filtered collections from chained filter calls

**High Priority**:
- Clip length modification
- Track color support
- FX management (remove, enable, disable, set params)

**Medium Priority**:
- Track reordering
- Track folder/grouping
- Complete MIDI operations
- Complete map() and for_each() function reference execution

**Test Fixes**:
- Update TestMagdaCreateTrackWithSerum expectations


