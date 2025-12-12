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

1. **Smart Detection**: Detect which agents are needed BEFORE launching (avoid wasting API calls)
2. **Parallel Execution**: Only launch needed agents simultaneously to minimize latency
3. **Result Merging**: Parser intelligently merges DAW actions with Arranger musical content
4. **Placeholder Resolution**: DAW agent can generate placeholders that Arranger fills
5. **Error Handling**: If one agent fails, the other can still succeed (partial results)
6. **Streaming Support**: Both agents can stream results incrementally (only if needed)

## Architecture Options

### Option 1: Orchestrator Pattern (Recommended)

```
┌─────────────────────────────────────────────────────────────┐
│                    Orchestrator Service                     │
│  1. Detect which agents are needed (based on question)      │
│  2. Launch only needed agents in parallel                    │
│  3. Collect results from active agents                       │
│  4. Merge results via Parser                                 │
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
    dawAgent      *daw.DawAgent
    arrangerAgent *arranger.ArrangerAgent
    parser        *MergedParser
}

func (o *Orchestrator) GenerateActions(ctx context.Context, question string, state map[string]any) (*MergedResult, error) {
    // Step 1: Detect which agents are needed (BEFORE launching)
    needsDAW, needsArranger := o.DetectAgentsNeeded(question)
    
    if !needsDAW && !needsArranger {
        // Default to DAW if no detection (backward compatibility)
        needsDAW = true
    }
    
    log.Printf("🔍 Agent detection: DAW=%v, Arranger=%v", needsDAW, needsArranger)
    
    // Step 2: Launch only needed agents in parallel
    var wg sync.WaitGroup
    var dawResult *daw.DawResult
    var arrangerResult *arranger.GenerationResult
    var dawErr, arrangerErr error
    
    if needsDAW {
        wg.Add(1)
        go func() {
            defer wg.Done()
            result, err := o.dawAgent.GenerateActions(ctx, question, state)
            if err != nil {
                dawErr = fmt.Errorf("daw agent: %w", err)
                return
            }
            dawResult = result
        }()
    }
    
    if needsArranger {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Arranger agent needs different input format
            inputArray := o.buildArrangerInput(question)
            result, err := o.arrangerAgent.Generate(ctx, "gpt-5.1", inputArray, "none", "dsl")
            if err != nil {
                arrangerErr = fmt.Errorf("arranger agent: %w", err)
                return
            }
            arrangerResult = result
        }()
    }
    
    // Wait for all active agents to complete
    wg.Wait()
    
    // Step 3: Handle errors (partial results OK)
    if dawErr != nil && arrangerErr != nil {
        return nil, fmt.Errorf("both agents failed: %v, %v", dawErr, arrangerErr)
    }
    if dawErr != nil && needsDAW {
        log.Printf("⚠️ DAW agent failed: %v", dawErr)
        // Continue with Arranger if available
    }
    if arrangerErr != nil && needsArranger {
        log.Printf("⚠️ Arranger agent failed: %v", arrangerErr)
        // Continue with DAW if available
    }
    
    // Step 4: Merge results via parser
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

## Detection Logic (CRITICAL: Run BEFORE launching agents)

The orchestrator **must detect which agents are needed BEFORE launching them** to avoid wasting API calls.

### ⚠️ Problem with Simple Keyword Matching

Simple keyword matching is **brittle** and will fail on many musical terms:

**Works:**
- ✅ "add a clip with an arpeggio in e minor" → detects "minor" → Arranger needed
- ✅ "add I VI IV progression" → detects Roman numerals → Arranger needed

**Fails:**
- ❌ "add a clip with a bassline" → no keywords → Arranger NOT detected (WRONG!)
- ❌ "add a clip with a riff" → no keywords → Arranger NOT detected (WRONG!)
- ❌ "add a clip with a hook" → no keywords → Arranger NOT detected (WRONG!)
- ❌ "add a clip with a groove" → no keywords → Arranger NOT detected (WRONG!)

### Solution Options

#### Option 1: Traditional ML Classifier

Train a lightweight classifier model (e.g., fine-tuned BERT, RoBERTa, or even a simple logistic regression):

```go
type MLClassifier struct {
    model *onnxruntime.Session // or TensorFlow Lite, etc.
}

func (c *MLClassifier) DetectAgentsNeeded(question string) (needsDAW bool, needsArranger bool, error) {
    // Tokenize and encode question
    features := c.tokenize(question)
    
    // Run inference (very fast: 1-5ms)
    predictions := c.model.Predict(features)
    
    return predictions.NeedsDAW > 0.5, predictions.NeedsArranger > 0.5, nil
}
```

**Pros**: 
- ⚡ Very fast (~1-5ms inference)
- 💰 No API costs (runs locally)
- 🔒 Deterministic, no network calls
- 📦 Can be embedded in binary

**Cons**: 
- 📚 Needs training data (collect real user requests)
- 🔄 Needs retraining for new patterns
- 🛠️ Harder to maintain (model versioning, updates)
- 📉 May struggle with edge cases not in training data
- 🏗️ Infrastructure overhead (model serving, versioning)

**Best for**: High-volume, predictable patterns, cost-sensitive

#### Option 2: LLM-Based Classification

Use a small, fast LLM call to classify intent:

```go
func (o *Orchestrator) DetectAgentsNeeded(ctx context.Context, question string) (needsDAW bool, needsArranger bool, error) {
    // Use a small, fast model (e.g., GPT-4o-mini) for classification
    classificationPrompt := fmt.Sprintf(`Classify this music production request. Return JSON:
{
  "needsDAW": true/false,  // REAPER operations: tracks, clips, FX, volume, etc.
  "needsArranger": true/false  // Musical content: chords, melodies, notes, arpeggios, basslines, riffs, etc.
}

Request: "%s"`, question)
    
    result, err := o.classifierLLM.Classify(ctx, classificationPrompt)
    if err != nil {
        // Fallback to keyword matching
        return o.detectAgentsNeededFallback(question)
    }
    
    return result.NeedsDAW, result.NeedsArranger, nil
}
```

**Pros**: 
- 🧠 Handles edge cases, understands context
- 🔧 Easy to update (just change prompt)
- 📈 Extensible (can add new agents easily)
- 🎯 High accuracy on edge cases

**Cons**: 
- ⏱️ Adds ~50-100ms latency (API call)
- 💰 Costs tokens (~100-200 tokens per request)
- 🌐 Requires network/API availability

**Best for**: Low-medium volume, high accuracy needs, rapid iteration

#### Option 3: Enhanced Keyword Matching (Quick Fix)

Expand keyword list significantly:

```go
arrangerKeywords := []string{
    // Existing
    "chord", "progression", "melody", "note", "notes",
    "I", "VI", "IV", "V", "ii", "iii", "vii",
    "roman", "scale", "harmony", "sequence", "pattern",
    "major", "minor", "diminished", "augmented",
    "triad", "seventh", "ninth",
    // ADD MORE:
    "arpeggio", "bassline", "riff", "hook", "groove", "lick",
    "phrase", "motif", "ostinato", "fill", "break",
    "C", "D", "E", "F", "G", "A", "B", // Note names
    "sharp", "flat", "natural", // Accidentals
    "pentatonic", "dorian", "mixolydian", // Scales
    "sus2", "sus4", "add9", // Chord extensions
}
```

**Pros**: 
- ⚡ Fast, no API calls
- 💰 No costs

**Cons**: 
- 🐛 Still brittle, needs constant maintenance
- ❌ False positives/negatives
- 📝 Manual keyword list maintenance

**Best for**: MVP, very simple use cases

#### Option 4: Hybrid Keywords + LLM (Simpler Hybrid)

Combine enhanced keywords (fast path) with LLM fallback:

Combine ML classifier (fast path) with LLM fallback (edge cases):

1. **Fast path**: Enhanced keyword matching (handles 90% of cases)
2. **Fallback**: LLM classification for ambiguous cases

```go
func (o *Orchestrator) DetectAgentsNeeded(ctx context.Context, question string) (needsDAW bool, needsArranger bool, error) {
    // Fast path: ML classifier (1-5ms)
    mlDAW, mlArranger, confidence := o.mlClassifier.Predict(question)
    
    // If high confidence, use ML result
    if confidence > 0.9 {
        return mlDAW, mlArranger, nil
    }
    
    // Low confidence or edge case: fallback to LLM (50-100ms)
    llmDAW, llmArranger, err := o.detectAgentsNeededLLM(ctx, question)
    if err != nil {
        // If LLM fails, use ML result anyway
        return mlDAW, mlArranger, nil
    }
    
    // Use LLM result for low-confidence cases
    return llmDAW, llmArranger, nil
}
```

**Pros**: 
- ⚡ Fast for 90% of cases (ML: 1-5ms)
- 🎯 Accurate for edge cases (LLM fallback)
- 💰 Cost-effective (LLM only for ~10% of requests)
- 📈 Improves over time (can retrain ML on LLM results)

**Cons**: 
- 🏗️ More complex (needs ML infrastructure)
- 📚 Needs initial training data
- 🔄 Needs retraining pipeline

**Best for**: Production at scale, cost optimization, high accuracy needs

#### Option 5: Hybrid Keywords + LLM (Simpler Hybrid - Recommended for MVP)

Simpler version without ML infrastructure:

**Important**: This detection happens **BEFORE** any agent is launched, so we only make API calls for agents that are actually needed.

## Streaming Support

For streaming, detect agents first, then launch only needed ones:

```go
func (o *Orchestrator) GenerateActionsStream(
    ctx context.Context,
    question string,
    state map[string]any,
    callback func(action map[string]any) error,
) error {
    // Step 1: Detect which agents are needed
    needsDAW, needsArranger := o.DetectAgentsNeeded(question)
    if !needsDAW && !needsArranger {
        needsDAW = true // Default
    }
    
    // Step 2: Launch only needed agents with streaming callbacks
    var wg sync.WaitGroup
    var arrangerChoices []models.MusicalChoice
    
    dawCallback := func(action map[string]any) error {
        // Check if action has placeholder
        if hasPlaceholder(action) {
            // Queue for later merge (wait for Arranger)
            return nil
        }
        // Emit immediately if no placeholder
        return callback(action)
    }
    
    arrangerCallback := func(choice models.MusicalChoice) error {
        // Store for placeholder resolution
        arrangerChoices = append(arrangerChoices, choice)
        return nil
    }
    
    // Launch only needed agents in parallel
    if needsDAW {
        wg.Add(1)
        go func() {
            defer wg.Done()
            o.dawAgent.GenerateActionsStream(ctx, question, state, dawCallback)
        }()
    }
    
    if needsArranger {
        wg.Add(1)
        go func() {
            defer wg.Done()
            inputArray := o.buildArrangerInput(question)
            o.arrangerAgent.GenerateStream(ctx, "gpt-5.1", inputArray, arrangerCallback)
        }()
    }
    
    wg.Wait()
    
    // Step 3: Resolve placeholders and emit merged actions
    if needsDAW && needsArranger {
        return o.resolvePlaceholders(callback, arrangerChoices)
    }
    
    return nil
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

