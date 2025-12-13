# Streaming Status - Real-Time Parsing Infrastructure

## Current State

### ✅ What Still Exists

#### 1. Grammar-School Streaming (`grammar-school-go/gs/engine.go`)

**Available Methods:**
- `Engine.Stream(ctx, code string) <-chan error` - Returns a channel of errors as methods execute
- `Engine.interpretStream(ctx, callChain)` - Executes methods incrementally from a parsed call chain

**Current Implementation:**
```go
func (e *Engine) Stream(ctx context.Context, code string) <-chan error {
    errors := make(chan error, 1)
    go func() {
        defer close(errors)
        // ⚠️ Parses ENTIRE DSL code first
        callChain, err := e.parser.Parse(code)
        if err != nil {
            errors <- fmt.Errorf("parse error: %w", err)
            return
        }
        // Then executes methods incrementally
        if err := e.interpretStream(ctx, callChain); err != nil {
            errors <- err
        }
    }()
    return errors
}
```

**Status:** ✅ Infrastructure exists, but **not true real-time parsing**
- Parses entire DSL code before streaming execution
- Methods execute incrementally after full parse
- Would need incremental parser for true character-by-character parsing

#### 2. MAGDA Agents Streaming (`magda-agents-go/agents/daw/daw_agent.go`)

**Available Methods:**
- `DawAgent.GenerateActionsStream(ctx, question, state, callback)` - Streaming action generation
- `StreamActionCallback` interface for receiving actions as they're parsed

**Current Implementation:**
```go
func (a *DawAgent) GenerateActionsStream(...) {
    // ⚠️ Calls NON-STREAMING provider
    resp, err := a.provider.Generate(ctx, request)  // Not GenerateStream!
    
    // Then parses complete DSL
    allActions, err := a.parseActionsIncremental(resp.RawOutput, state)
    
    // Calls callback for each action (but all at once, not incrementally)
    for _, action := range allActions {
        _ = callback(action)
    }
}
```

**Status:** ✅ Infrastructure exists, but **not currently using streaming**
- Method exists but calls non-streaming `provider.Generate()`
- Would need to call `provider.GenerateStream()` to enable
- Would need incremental DSL parsing to parse as text arrives

#### 3. LLM Provider Streaming (`magda-agents-go/llm/`)

**Available Interfaces:**
- `Provider.GenerateStream(ctx, request, callback)` interface defined in `provider.go`
- `GeminiProvider.GenerateStream()` - Fully implemented for Gemini
- `StreamEvent` and `StreamCallback` types defined

**Status:** ✅ Infrastructure exists
- Gemini provider has full streaming implementation
- OpenAI provider would need streaming implementation
- Interface is ready for use

### ❌ What's Missing for True Real-Time Streaming

#### 1. Incremental DSL Parser

**Current:** `LarkParser.Parse()` parses entire DSL string at once
**Needed:** Incremental parser that can:
- Parse DSL as characters arrive
- Emit method calls as soon as they're complete
- Handle partial/incomplete statements gracefully

**Example:**
```go
// Current (batch)
parser.Parse(`track(name="A").track(name="B")`)  // Parses all at once

// Needed (incremental)
parser.ParseIncremental(`track(name="A")`, callback)  // Emits Track call immediately
parser.ParseIncremental(`.track(name="B")`, callback)  // Emits next Track call
```

#### 2. OpenAI Provider Streaming

**Current:** `OpenAIProvider.Generate()` - Non-streaming only
**Needed:** `OpenAIProvider.GenerateStream()` - Stream from OpenAI Responses API

**Implementation would need:**
- Stream from OpenAI `/v1/responses` endpoint
- Extract DSL from CFG tool calls as they arrive
- Parse DSL incrementally as text streams in

#### 3. Integration in DawAgent

**Current:** `GenerateActionsStream()` calls non-streaming provider
**Needed:** 
- Call `provider.GenerateStream()` instead
- Use incremental parser in callback
- Emit actions as they're parsed (not all at once)

## Re-Enabling Streaming

### Option 1: Quick Re-Enable (Partial Streaming)

**Steps:**
1. Implement `OpenAIProvider.GenerateStream()` to stream from Responses API
2. Update `DawAgent.GenerateActionsStream()` to call `provider.GenerateStream()`
3. Parse complete DSL after stream completes (current approach)

**Result:** 
- ✅ LLM response streams in real-time
- ✅ Actions parsed and emitted after complete DSL received
- ❌ Not true incremental parsing (still waits for complete DSL)

### Option 2: Full Real-Time Streaming (True Incremental)

**Steps:**
1. Implement incremental DSL parser in `grammar-school-go`
   - Parse as characters arrive
   - Emit method calls when statements complete
   - Handle partial statements gracefully
2. Implement `OpenAIProvider.GenerateStream()` with incremental parsing
3. Update `DawAgent.GenerateActionsStream()` to use incremental parser
4. Emit actions as they're parsed (real-time)

**Result:**
- ✅ LLM response streams in real-time
- ✅ DSL parsed incrementally as text arrives
- ✅ Actions emitted immediately as they're parsed
- ✅ True real-time experience

## Code Locations

### Streaming Infrastructure (Still Exists)

1. **Grammar-School Engine:**
   - `grammar-school-go/gs/engine.go:129` - `Stream()` method
   - `grammar-school-go/gs/engine.go:171` - `interpretStream()` method

2. **MAGDA Agents:**
   - `magda-agents-go/agents/daw/daw_agent.go:271` - `GenerateActionsStream()` method
   - `magda-agents-go/agents/daw/daw_agent.go:266` - `StreamActionCallback` type

3. **LLM Provider:**
   - `magda-agents-go/llm/provider.go:66` - `StreamCallback` interface
   - `magda-agents-go/llm/provider.go:70` - `StreamEvent` type
   - `magda-agents-go/llm/gemini_provider.go:109` - `GenerateStream()` implementation

### What Was Removed

1. **OpenAI Provider Streaming:**
   - `OpenAIProvider.GenerateStream()` was removed
   - Only `Generate()` (non-streaming) remains

2. **Direct Usage:**
   - `DawAgent.GenerateActionsStream()` no longer calls streaming provider
   - All code paths use non-streaming `Generate()`

## Recommendations

### For Quick Re-Enable (Option 1)

**Effort:** Low (1-2 days)
**Files to Modify:**
- `magda-agents-go/llm/openai_provider.go` - Add `GenerateStream()` method
- `magda-agents-go/agents/daw/daw_agent.go` - Update to call `GenerateStream()`

**Benefits:**
- Real-time LLM response streaming
- Better UX (user sees progress)
- Minimal code changes

### For Full Real-Time (Option 2)

**Effort:** High (1-2 weeks)
**Files to Modify:**
- `grammar-school-go/gs/lark_parser.go` - Add incremental parsing
- `grammar-school-go/gs/engine.go` - Update `Stream()` for incremental parsing
- `magda-agents-go/llm/openai_provider.go` - Add streaming with incremental parsing
- `magda-agents-go/agents/daw/daw_agent.go` - Integrate incremental parsing

**Benefits:**
- True real-time action execution
- Actions appear as soon as they're parsed
- Best user experience

## Current Usage

**Status:** Streaming infrastructure exists but is **NOT currently used**

**Active Code Path:**
- `DawAgent.GenerateActions()` → `provider.Generate()` → Parse complete DSL → Return all actions

**Available but Unused:**
- `DawAgent.GenerateActionsStream()` - Exists but calls non-streaming provider
- `Engine.Stream()` - Exists but requires complete DSL first
- `GeminiProvider.GenerateStream()` - Fully implemented but not used for MAGDA

## Conclusion

✅ **Yes, streaming infrastructure still exists** and can be re-enabled

**For Quick Re-Enable:**
- Infrastructure is 80% ready
- Need to implement `OpenAIProvider.GenerateStream()`
- Need to update `DawAgent` to use streaming provider

**For Full Real-Time:**
- Need incremental DSL parser
- More significant refactoring required
- Better long-term solution

---

**Created:** 2025-12-10
**Status:** Infrastructure exists, not currently active


