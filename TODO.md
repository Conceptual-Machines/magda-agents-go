# TODO - Remaining Implementation Tasks

## ✅ Recently Completed

### 1. Unified Actions Implementation
**Status**: ✅ Completed
- ✅ Unified `set_track` method handles: name, volume_db, pan, mute, solo, selected
- ✅ Unified `set_clip` method handles: name, color, selected
- ✅ All legacy individual methods removed
- ✅ Grammar updated to support method chaining (`chain*` instead of `chain?`)
- ✅ Filtered collections support for all unified actions
- ✅ `AddFx` now handles filtered collections
- ✅ All integration tests passing

### 2. Grammar and Parser Improvements
**Status**: ✅ Completed
- ✅ Fixed grammar to support multiple method chains (`track().new_clip().set_track()`)
- ✅ Fixed parser to handle `>=` and `<=` operators when split by Lark parser
- ✅ Removed boolean literal handling (grammar now enforces proper predicates)
- ✅ Updated prompt to use `track.index >= 0` instead of `true` for matching all tracks

## High Priority Features

### 3. Clip Length Modification
**Status**: ✅ Completed
**Action**: `set_clip(length=...)` (unified method)
- **REAPER API**: `SetMediaItemLength(MediaItem *item, double length, bool adjust_take_length)` (already implemented in C++)
- **Use Cases**: 
  - "extend clip to 8 bars"
  - "make clip 2 seconds long"
- **DSL**: `filter(clips, clip.length < 2.0).set_clip(length=4.0)`
- **Implementation**: Added `length` property to unified `set_clip` method
- **Files Modified**:
  - ✅ `magda-agents-go/agents/daw/dsl_parser_functional.go` - Added `length` handling to `SetClip()` method
  - ✅ `magda-agents-go/agents/daw/dsl_parser_functional.go` - Updated grammar to include `length` in `clip_property_param`
  - ✅ `magda-reaper/src/magda_actions.cpp` - Already supported in `SetClipProperties()`

### 3.5. Clip Position Unification
**Status**: Not Implemented (TODO for tomorrow)
**Action**: Add `position` property to unified `set_clip` method
- **Current State**: Position is set via separate `.move_clip(position=...)` method
- **Proposed**: Unify into `set_clip(position=...)` similar to how `length` was unified
- **Use Cases**: 
  - "move all clips shorter than 2 seconds to position 4.0"
  - "move clip 0 to bar 8"
- **DSL**: `filter(clips, clip.length < 2.0).set_clip(position=4.0)` or `set_clip(clip=0, position=8.0)`
- **C++ Backend**: Already supports position in `SetClipProperties()` when clip is identified by index
- **Files to Modify**:
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Add `position` handling to `SetClip()` method
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Update grammar to include `position` in `clip_property_param`
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Consider deprecating/removing `MoveClip()` method
  - `magda-reaper/src/magda_actions.cpp` - Already supported, but may need refinement for position-based identification

### 4. Track Color Support
**Status**: Not Implemented
**Action**: Add `color` property to unified `set_track` method
- **REAPER API**: `GetSetMediaTrackInfo` with "I_CUSTOMCOLOR"
- **Use Cases**: 
  - "color all muted tracks red"
  - "set track color to blue"
- **DSL**: `filter(tracks, track.muted == true).set_track(color="#ff0000")` or `set_track(color="red")`
- **Files to Modify**:
  - `magda-reaper/include/magda_actions.h` - Add color handling to `SetTrackProperties()`
  - `magda-reaper/src/magda_actions.cpp` - Implement color in `SetTrackProperties()`
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Add `color` handling to `SetTrack()` method
  - `magda-agents-go/agents/daw/dsl_parser_functional.go` - Update grammar to include `color` in `track_property_param`

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

## ✅ Test Fixes Completed

### 11. TestMagdaCreateTrackWithSerum
**Status**: ✅ Fixed
- Updated test to check for `instrument` in `create_track` action instead of separate `add_instrument` action

## Documentation

### 12. Update Documentation
**Status**: Partially Needed
- ✅ Unified actions are implemented and working
- ⚠️ Update `IMPLEMENTATION_STATUS.md` to reflect unified actions (remove references to individual methods)
- ⚠️ Document unified `set_track` and `set_clip` methods with all supported properties
- ⚠️ Add examples for method chaining in documentation

## Summary

**✅ Recently Completed**:
- Unified `set_track` and `set_clip` actions with full filtered collection support
- Grammar fixes for method chaining and predicate parsing
- All integration tests passing

**High Priority (Next Steps)**:
- Track color support (add to `set_track`)
- FX management (remove, enable, disable, set params)

**Medium Priority**:
- Track reordering (`move_track`)
- Track folder/grouping (`set_track_folder`)
- Complete MIDI operations (notes array parsing)
- Complete map() and for_each() function reference execution


