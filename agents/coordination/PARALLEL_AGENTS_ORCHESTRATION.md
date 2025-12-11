# Parallel Agents Orchestration: DAW + Arranger Agents

## Overview

The MAGDA system needs to coordinate multiple agents that generate different DSL outputs:
- **DAW Agent**: Generates MAGDA DSL (REAPER actions like `create_track`, `set_clip`, etc.)
- **Arranger Agent**: Generates Music DSL (NoteEvent arrays like `choice("...", [note(...), ...])`)

These agents should run **in parallel** for better latency, and their results need to be **joined** by the parser before execution.

## Current Architecture

### Single Agent Flow (Current)
```
User Request → DAW Agent → MAGDA DSL → Parser → REAPER Actions
```

### Proposed Multi-Agent Flow
```
User Request → [DAW Agent (parallel) + Arranger Agent (parallel)] → Parser → Joined Actions
```

## Design Goals

1. **Parallel Execution**: Both agents run simultaneously to minimize latency
2. **Result Merging**: Parser intelligently merges DAW actions with Arranger musical content
3. **Placeholder Resolution**: DAW agent can generate placeholders that Arranger fills
4. **Error Handling**: If one agent fails, the other can still succeed (partial results)
5. **Streaming Support**: Both agents can stream results incrementally

## Architecture Options

### Option 1: Orchestrator Pattern (Recommended)

```
┌─────────────────────────────────────────────────────────────┐
│                    Orchestrator Service                     │
│  - Detects which agents are needed                          │
│  - Launches agents in parallel (goroutines/channels)        │
│  - Collects results from both agents                        │
│  - Merges results via Parser                                 │
└─────────────────────────────────────────────────────────────┘
         │                              │
         ├──────────────┬──────────────┤
         │              │              │
    ┌────▼────┐    ┌────▼────┐    ┌────▼────┐
    │  DAW    │    │ Arranger │    │  Parser │
    │  Agent  │    │  Agent   │    │ (Merge) │
    └─────────┘    └─────────┘    └─────────┘
```

**Implementation**:
```go
type Orchestrator struct {
    dawAgent     *daw.DawAgent
    arrangerAgent *arranger.ArrangerAgent
    parser       *MergedParser
}

func (o *Orchestrator) GenerateActions(ctx context.Context, question string, state map[string]any) (*MergedResult, error) {
    // Launch both agents in parallel
    dawChan := make(chan *daw.DawResult)
    arrangerChan := make(chan *arranger.GenerationResult)
    errChan := make(chan error, 2)
    
    // DAW Agent (parallel)
    go func() {
        result, err := o.dawAgent.GenerateActions(ctx, question, state)
        if err != nil {
            errChan <- fmt.Errorf("daw agent: %w", err)
            return
        }
        dawChan <- result
    }()
    
    // Arranger Agent (parallel)
    go func() {
        result, err := o.arrangerAgent.Generate(ctx, model, inputArray, reasoningMode, outputFormat)
        if err != nil {
            errChan <- fmt.Errorf("arranger agent: %w", err)
            return
        }
        arrangerChan <- result
    }()
    
    // Collect results (with timeout)
    var dawResult *daw.DawResult
    var arrangerResult *arranger.GenerationResult
    
    for i := 0; i < 2; i++ {
        select {
        case result := <-dawChan:
            dawResult = result
        case result := <-arrangerChan:
            arrangerResult = result
        case err := <-errChan:
            // Log error but continue (partial results OK)
            log.Printf("⚠️ Agent error: %v", err)
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
    
    // Merge results via parser
    return o.parser.MergeResults(dawResult, arrangerResult)
}
```

### Option 2: Parser-Level Merging

The parser receives both DSL outputs and merges them:

```go
type MergedParser struct {
    dawParser     *daw.FunctionalDSLParser
    arrangerParser *arranger.MusicalDSLParser
}

func (p *MergedParser) MergeResults(dawResult *daw.DawResult, arrangerResult *arranger.GenerationResult) (*MergedResult, error) {
    // Parse DAW DSL
    dawActions, err := p.dawParser.ParseDSL(dawResult.DSL)
    if err != nil {
        return nil, fmt.Errorf("daw parse error: %w", err)
    }
    
    // Parse Arranger DSL
    musicalChoices, err := p.arrangerParser.ParseDSL(arrangerResult.DSL)
    if err != nil {
        return nil, fmt.Errorf("arranger parse error: %w", err)
    }
    
    // Merge: Find placeholders in DAW actions and inject Arranger content
    mergedActions := p.injectMusicalContent(dawActions, musicalChoices)
    
    return &MergedResult{
        Actions: mergedActions,
        Usage:   mergeUsage(dawResult.Usage, arrangerResult.Usage),
    }, nil
}
```

### Option 3: Placeholder-Based Coordination

DAW agent generates actions with placeholders, Arranger fills them:

**DAW Agent Output**:
```dsl
track(instrument="Piano").new_clip(bar=9, length_bars=4).add_midi(notes=<MUSICAL_CONTENT_PLACEHOLDER>)
```

**Arranger Agent Output**:
```dsl
choice("I VI IV progression", [
    note(midi=60, velocity=100, start=0.0, duration=2.0),
    note(midi=64, velocity=100, start=0.0, duration=2.0),
    ...
])
```

**Parser Merges**:
```json
{
  "action": "create_track",
  "instrument": "Piano"
},
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
    {"midiNoteNumber": 60, "velocity": 100, "startBeats": 0.0, "lengthBeats": 2.0},
    ...
  ]
}
```

## Detection Logic

The orchestrator needs to detect which agents are needed:

```go
func (o *Orchestrator) DetectAgentsNeeded(question string) (needsDAW bool, needsArranger bool) {
    // DAW keywords: track, clip, fx, volume, pan, mute, solo, etc.
    dawKeywords := []string{"track", "clip", "fx", "volume", "pan", "mute", "solo", "reaper", "daw"}
    
    // Arranger keywords: chord, progression, melody, note, roman numeral, etc.
    arrangerKeywords := []string{"chord", "progression", "melody", "note", "I", "VI", "IV", "V", "roman", "scale"}
    
    questionLower := strings.ToLower(question)
    
    needsDAW = containsAny(questionLower, dawKeywords)
    needsArranger = containsAny(questionLower, arrangerKeywords)
    
    // Default: if no keywords, assume DAW (backward compatibility)
    if !needsDAW && !needsArranger {
        needsDAW = true
    }
    
    return needsDAW, needsArranger
}
```

## Streaming Support

For streaming, both agents can stream incrementally:

```go
func (o *Orchestrator) GenerateActionsStream(
    ctx context.Context,
    question string,
    state map[string]any,
    callback func(action map[string]any) error,
) error {
    // Launch both agents with streaming callbacks
    dawCallback := func(action map[string]any) error {
        // Check if action has placeholder
        if hasPlaceholder(action) {
            // Queue for later merge
            return nil
        }
        return callback(action)
    }
    
    arrangerCallback := func(choice models.MusicalChoice) error {
        // Store for placeholder resolution
        return nil
    }
    
    // Run in parallel with goroutines
    var wg sync.WaitGroup
    wg.Add(2)
    
    go func() {
        defer wg.Done()
        o.dawAgent.GenerateActionsStream(ctx, question, state, dawCallback)
    }()
    
    go func() {
        defer wg.Done()
        o.arrangerAgent.GenerateStream(ctx, model, inputArray, arrangerCallback)
    }()
    
    wg.Wait()
    
    // Resolve placeholders and emit merged actions
    return o.resolvePlaceholders(callback)
}
```

## Error Handling Strategy

1. **Partial Success**: If DAW succeeds but Arranger fails, return DAW actions (musical content optional)
2. **Timeout**: Each agent has its own timeout; if one times out, continue with the other
3. **Retry**: Retry failed agents individually (don't retry both if one succeeds)

## Implementation Plan

### Phase 1: Basic Orchestration
1. Create `Orchestrator` service in `magda-agents-go/agents/coordination/`
2. Implement parallel execution with goroutines
3. Basic result merging (concatenate actions)
4. Update API handler to use orchestrator

### Phase 2: Placeholder Resolution
1. DAW agent generates placeholders for musical content
2. Parser detects placeholders in actions
3. Inject Arranger output into placeholders

### Phase 3: Streaming Support
1. Streaming callbacks for both agents
2. Incremental placeholder resolution
3. Real-time action emission

### Phase 4: Advanced Features
1. Agent detection heuristics
2. Error recovery and partial results
3. Performance optimization (caching, batching)

## Files to Create/Modify

### New Files
- `magda-agents-go/agents/coordination/orchestrator.go` - Main orchestrator service
- `magda-agents-go/agents/coordination/merger.go` - Result merging logic
- `magda-agents-go/agents/coordination/detector.go` - Agent detection heuristics

### Modified Files
- `aideas-api/internal/api/handlers/magda.go` - Use orchestrator instead of direct DAW agent
- `magda-agents-go/agents/daw/daw_agent.go` - Support placeholder generation
- `magda-agents-go/agents/daw/dsl_parser_functional.go` - Support placeholder resolution

## Questions to Resolve

1. **Placeholder Format**: How should DAW agent mark placeholders? (`<MUSICAL_CONTENT>`, `{{arranger}}`, etc.)
2. **Priority**: If both agents generate conflicting actions, which takes precedence?
3. **State Sharing**: Should Arranger agent receive REAPER state? (probably not needed)
4. **Caching**: Should we cache Arranger results for similar requests?
5. **Fallback**: If Arranger fails, should DAW agent try to generate basic MIDI?

## Example Use Cases

### Use Case 1: "Add I VI IV progression to piano track at bar 9"
- **DAW Agent**: `track(instrument="Piano").new_clip(bar=9).add_midi(notes=<PLACEHOLDER>)`
- **Arranger Agent**: `choice("I VI IV", [note(60,100,0,2), ...])`
- **Merged**: Piano track created, clip at bar 9, MIDI notes injected

### Use Case 2: "Create a track and add reverb"
- **DAW Agent**: `track().add_fx(fxname="ReaVerb")`
- **Arranger Agent**: (not needed, skipped)
- **Merged**: Just DAW actions

### Use Case 3: "Generate a chord progression"
- **DAW Agent**: (not needed, but could create basic structure)
- **Arranger Agent**: `choice("I VI IV V", [...])`
- **Merged**: Musical content only (or DAW creates default track structure)

