# Keyword Expansion Script

This script uses an LLM to expand keyword lists with synonyms, translations, and variations.

## Usage

```bash
export OPENAI_API_KEY=your-api-key
go run agents/coordination/expand_keywords.go > expanded_keywords.json
```

## What it does

1. Takes base keyword lists (DAW operations + Arranger content)
2. Uses GPT-4o-mini to generate:
   - Synonyms and variations
   - Translations (Spanish, French, German, Italian, Portuguese, Japanese)
   - Slang and informal terms
   - Related terms
3. Outputs expanded JSON with deduplicated keywords

## Output Format

```json
{
  "daw": [
    "track", "pista", "piste", "spur", "traccia",
    "reverb", "reverberation", "echo", "réverbération",
    ...
  ],
  "arranger": [
    "chord", "acorde", "accord", "akkord", "accordo",
    "progression", "progresión", "progression", "progressione",
    ...
  ]
}
```

## Integration

After generating expanded keywords:

1. Review the output for quality
2. Update `orchestrator.go` with expanded keywords
3. Or load from JSON file at runtime (future enhancement)

## Cost

- ~$0.01-0.02 per run (GPT-4o-mini, ~500-1000 tokens)
- Run periodically (monthly/quarterly) to catch new terms

