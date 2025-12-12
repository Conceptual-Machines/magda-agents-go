# ML Classifier Training Data Strategy

## Training Data Requirements

### Data Format

For a binary classification task (needsDAW, needsArranger), we need:

```json
{
  "question": "add reverb to track 1",
  "needsDAW": true,
  "needsArranger": false,
  "confidence": 1.0
}
```

### Data Categories

#### 1. DAW-Only Requests (needsDAW=true, needsArranger=false)
- Track operations: "create track", "delete track 3", "rename track to Bass"
- Clip operations: "add clip at bar 5", "delete clip", "move clip to position 10"
- FX operations: "add reverb to track 1", "remove EQ", "enable compressor"
- Mixing: "set volume to -3dB", "pan left", "mute track 2"
- Selection: "select all tracks", "deselect clips"
- Color/rename: "color track red", "rename clip to Verse"

**Examples:**
- "add reverb to track 1"
- "set volume to -3dB on track 2"
- "create a new track called Drums"
- "delete clip at bar 5"
- "mute all tracks except track 1"

#### 2. Arranger-Only Requests (needsDAW=false, needsArranger=true)
- Chord progressions: "I VI IV V progression"
- Melodies: "create a melody in C major"
- Scales: "play a pentatonic scale"
- Musical patterns: "arpeggio in E minor", "bassline in A"

**Examples:**
- "create I VI IV progression"
- "generate a melody in C major"
- "arpeggio in E minor"
- "bassline pattern"
- "chord sequence: Am F C G"

#### 3. Both Agents Needed (needsDAW=true, needsArranger=true)
- Musical content + DAW operations: "add I VI IV progression to piano track"
- Complex requests: "create a track with a bassline", "add a clip with a melody"

**Examples:**
- "add I VI IV progression to piano track at bar 9"
- "create a track with a bassline"
- "add a clip with an arpeggio in e minor"
- "make a track with a riff"
- "add a hook to track 3"

#### 4. Edge Cases / Ambiguous
- Unclear intent: "make it sound better" (needs context)
- Creative descriptions: "add a vibe", "make it groovy"
- Mixed terminology: "add some notes to the track"

## Synthetic Data Generation Strategies

### Strategy 1: Template-Based Generation

Use templates with variable substitution:

```go
type Template struct {
    Template string
    NeedsDAW bool
    NeedsArranger bool
    Variables []string
}

templates := []Template{
    {
        Template: "add {fx} to track {number}",
        NeedsDAW: true,
        NeedsArranger: false,
        Variables: []string{"reverb", "EQ", "compressor", "delay"},
    },
    {
        Template: "create {progression} progression",
        NeedsDAW: false,
        NeedsArranger: true,
        Variables: []string{"I VI IV V", "ii V I", "I IV V"},
    },
    {
        Template: "add {progression} progression to {instrument} track",
        NeedsDAW: true,
        NeedsArranger: true,
        Variables: []string{"I VI IV", "ii V I"},
    },
}
```

**Pros:**
- Fast to generate (thousands of examples)
- Covers common patterns
- Easy to control quality

**Cons:**
- May not capture natural language variation
- Limited creativity/edge cases
- May overfit to templates

### Strategy 2: LLM-Generated Synthetic Data

Use LLM to generate diverse training examples:

```go
func GenerateSyntheticData(ctx context.Context, category string, count int) ([]TrainingExample, error) {
    prompt := fmt.Sprintf(`Generate %d diverse examples of music production requests that are:
- Category: %s
- NeedsDAW: %v
- NeedsArranger: %v
- Natural, varied language
- Different phrasings and styles

Return as JSON array:
[{"question": "...", "needsDAW": true/false, "needsArranger": true/false}, ...]`, 
        count, category, needsDAW, needsArranger)
    
    // Call LLM
    examples, err := llm.GenerateExamples(ctx, prompt)
    return examples, err
}
```

**Pros:**
- Natural language variation
- Captures edge cases
- Diverse phrasings
- Can generate creative examples

**Cons:**
- Costs tokens
- May generate incorrect labels (needs validation)
- Slower than templates

### Strategy 3: Hybrid: Templates + LLM Augmentation

1. Generate base examples from templates (fast, high coverage)
2. Use LLM to paraphrase/vary them (adds naturalness)

```go
func GenerateHybridData(ctx context.Context) []TrainingExample {
    // Step 1: Template generation (1000 examples)
    baseExamples := generateFromTemplates(1000)
    
    // Step 2: LLM augmentation (paraphrase 20% of examples)
    augmentedExamples := make([]TrainingExample, 0)
    for i := 0; i < len(baseExamples)/5; i++ {
        paraphrased := llm.Paraphrase(ctx, baseExamples[i].Question)
        augmentedExamples = append(augmentedExamples, TrainingExample{
            Question: paraphrased,
            NeedsDAW: baseExamples[i].NeedsDAW,
            NeedsArranger: baseExamples[i].NeedsArranger,
        })
    }
    
    return append(baseExamples, augmentedExamples...)
}
```

**Pros:**
- Fast (templates) + Natural (LLM)
- Cost-effective (LLM only for subset)
- Good coverage + variation

**Cons:**
- More complex pipeline
- Still needs validation

### Strategy 4: Real Data Collection + Synthetic Augmentation

1. Collect real user requests (from logs, beta testing)
2. Label them manually or with LLM
3. Augment with synthetic data for underrepresented categories

```go
func BuildTrainingDataset() {
    // Real data (gold standard)
    realData := collectFromLogs() // ~500 examples
    
    // Synthetic augmentation for balance
    syntheticDAW := generateSynthetic("DAW-only", 1000)
    syntheticArranger := generateSynthetic("Arranger-only", 1000)
    syntheticBoth := generateSynthetic("Both", 500)
    
    // Combine and balance
    dataset := balanceDataset(realData, syntheticDAW, syntheticArranger, syntheticBoth)
    
    return dataset
}
```

**Pros:**
- Real data = high quality
- Synthetic = coverage
- Best of both worlds

**Cons:**
- Needs real data collection
- Manual labeling effort

## Recommended Approach

### Phase 1: Synthetic Data Generation (MVP)

**Start with Strategy 3 (Hybrid: Templates + LLM Augmentation):**

1. **Template Generation** (80% of dataset):
   - 2000 DAW-only examples
   - 1000 Arranger-only examples
   - 1000 Both examples
   - Total: 4000 examples

2. **LLM Augmentation** (20% of dataset):
   - Paraphrase 800 examples (200 from each category)
   - Adds natural language variation

3. **Validation**:
   - Use LLM to validate labels
   - Manual spot-check 100 examples
   - Remove obvious errors

**Estimated Cost**: ~$5-10 for LLM augmentation (4000 examples × 20% × ~$0.001/example)

### Phase 2: Real Data Collection (Production)

**Migrate to Strategy 4 (Real + Synthetic):**

1. Collect real user requests from production logs
2. Label with LLM (faster than manual)
3. Use synthetic data to fill gaps (rare categories, edge cases)
4. Continuously retrain as new patterns emerge

## Data Quality Metrics

### Training Set Balance
- DAW-only: 40%
- Arranger-only: 30%
- Both: 25%
- Edge cases: 5%

### Validation Metrics
- Accuracy: >95% on validation set
- Precision/Recall: >90% for each class
- F1-score: >0.90

### Edge Case Coverage
- Musical terms: arpeggio, bassline, riff, hook, groove, lick, phrase, motif
- Creative descriptions: "vibe", "groovy", "punchy", "warm"
- Ambiguous requests: "make it better", "add some music"

## Implementation Example

```go
package coordination

import (
    "context"
    "encoding/json"
    "fmt"
    "math/rand"
)

type TrainingExample struct {
    Question      string `json:"question"`
    NeedsDAW      bool   `json:"needsDAW"`
    NeedsArranger bool   `json:"needsArranger"`
}

type DataGenerator struct {
    llm LLMProvider
}

func (g *DataGenerator) GenerateSyntheticDataset(ctx context.Context, size int) ([]TrainingExample, error) {
    // Template-based generation (80%)
    templateExamples := g.generateFromTemplates(size * 4 / 5)
    
    // LLM augmentation (20%)
    augmentedExamples, err := g.augmentWithLLM(ctx, templateExamples[:size/5])
    if err != nil {
        return nil, err
    }
    
    // Combine and shuffle
    allExamples := append(templateExamples, augmentedExamples...)
    rand.Shuffle(len(allExamples), func(i, j int) {
        allExamples[i], allExamples[j] = allExamples[j], allExamples[i]
    })
    
    return allExamples, nil
}

func (g *DataGenerator) generateFromTemplates(count int) []TrainingExample {
    templates := []struct {
        template      string
        needsDAW      bool
        needsArranger bool
        variables     map[string][]string
    }{
        {
            template:      "add {fx} to track {n}",
            needsDAW:      true,
            needsArranger: false,
            variables: map[string][]string{
                "fx": {"reverb", "EQ", "compressor", "delay", "chorus"},
                "n":  {"1", "2", "3", "track 1", "track 2"},
            },
        },
        {
            template:      "create {progression} progression",
            needsDAW:      false,
            needsArranger: true,
            variables: map[string][]string{
                "progression": {"I VI IV V", "ii V I", "I IV V", "vi IV I V"},
            },
        },
        {
            template:      "add {progression} progression to {instrument} track",
            needsDAW:      true,
            needsArranger: true,
            variables: map[string][]string{
                "progression": {"I VI IV", "ii V I"},
                "instrument":  {"piano", "guitar", "bass", "synth"},
            },
        },
        // ... more templates
    }
    
    examples := make([]TrainingExample, 0, count)
    for i := 0; i < count; i++ {
        t := templates[rand.Intn(len(templates))]
        example := g.fillTemplate(t.template, t.variables)
        examples = append(examples, TrainingExample{
            Question:      example,
            NeedsDAW:      t.needsDAW,
            NeedsArranger: t.needsArranger,
        })
    }
    
    return examples
}

func (g *DataGenerator) augmentWithLLM(ctx context.Context, examples []TrainingExample) ([]TrainingExample, error) {
    // Use LLM to paraphrase examples
    // Implementation depends on LLM provider
    return examples, nil
}
```

## Validation Strategy

### Automated Validation
1. **LLM Validation**: Use LLM to check if labels are correct
2. **Consistency Check**: Ensure similar questions have same labels
3. **Edge Case Detection**: Flag ambiguous examples for manual review

### Manual Validation
1. **Spot Check**: Review 5-10% of examples
2. **Edge Case Review**: All ambiguous examples
3. **Category Balance**: Ensure balanced distribution

## Conclusion

**For MVP**: Start with **synthetic data (Strategy 3)** - fast, cheap, good coverage
**For Production**: Migrate to **real + synthetic (Strategy 4)** - higher quality, real patterns

**Key Insight**: Synthetic data is great for bootstrapping, but real data is essential for production accuracy. Use synthetic to fill gaps and augment real data.

