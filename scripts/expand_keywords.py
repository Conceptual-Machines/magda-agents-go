#!/usr/bin/env python3
"""
Simple keyword expansion script using OpenAI SDK directly.
Expands DAW and Arranger keywords with synonyms, translations, and slang.
"""

import json
import os
import re
from pathlib import Path
from typing import List

from openai import OpenAI
from dotenv import load_dotenv


def load_env_file():
    """Load .env file from project root."""
    # Find project root (look for go.mod or .env)
    current_dir = Path.cwd()
    for parent in [current_dir] + list(current_dir.parents):
        env_path = parent / ".env"
        if env_path.exists():
            load_dotenv(env_path)
            print(f"✅ Loaded .env file from {env_path}")
            return
    print("⚠️ No .env file found, using environment variables only")


def expand_keywords(client: OpenAI, keywords: List[str], category: str) -> List[str]:
    """Expand keywords using OpenAI API."""
    prompt = f"""Generate comprehensive keyword variations for music production terms.

Category: {category}
Base keywords: {keywords}

For each keyword, provide:
1. Synonyms and variations (e.g., "reverb" → "reverberation", "echo", "verb")
2. Translations in common languages:
   - Spanish (español)
   - French (français)
   - German (deutsch)
   - Italian (italiano)
   - Portuguese (português)
   - Japanese (日本語) - romanized
3. Slang and informal terms
4. Related terms in the same domain

Return as JSON array of strings (all lowercase, no duplicates):
["keyword1", "keyword2", "synonym1", "translation1", ...]

Example for "reverb":
["reverb", "reverberation", "echo", "delay", "pista", "réverbération", "hall", "verb"]

Generate variations for ALL keywords. Return ONLY a JSON array, no markdown, no explanation, just the array."""

    try:
        response = client.chat.completions.create(
            model="gpt-4o-mini",  # Fast and cheap
            messages=[
                {"role": "user", "content": prompt}
            ],
            temperature=0.7,
        )

        content = response.choices[0].message.content.strip()
        
        # Remove markdown code blocks if present
        content = re.sub(r'```json\s*', '', content)
        content = re.sub(r'```\s*', '', content)
        content = content.strip()
        
        # Try to parse as JSON array
        try:
            keywords_list = json.loads(content)
            if isinstance(keywords_list, list):
                # Normalize: lowercase, trim, remove empty
                normalized = []
                for kw in keywords_list:
                    if isinstance(kw, str):
                        kw = kw.lower().strip()
                        if kw and len(kw) > 1:
                            normalized.append(kw)
                return normalized
        except json.JSONDecodeError:
            # Fallback: extract quoted strings
            keywords_list = re.findall(r'"([^"]+)"', content)
            return [kw.lower().strip() for kw in keywords_list if kw.strip() and len(kw.strip()) > 1]
        
        return []
        
    except Exception as e:
        print(f"❌ Error expanding keywords: {e}")
        return []


def deduplicate(keywords: List[str]) -> List[str]:
    """Remove duplicates while preserving order."""
    seen = set()
    result = []
    for kw in keywords:
        kw_lower = kw.lower().strip()
        if kw_lower and kw_lower not in seen:
            seen.add(kw_lower)
            result.append(kw_lower)
    return result


def main():
    # Load environment variables
    load_env_file()
    
    # Get API key
    api_key = os.getenv("OPENAI_API_KEY") or os.getenv("OPENAI_KEY")
    if not api_key:
        print("❌ OPENAI_API_KEY environment variable not set.")
        print("Set it in .env file or as environment variable.")
        return
    
    client = OpenAI(api_key=api_key)
    
    # Base keywords
    daw_keywords = [
        "track", "clip", "fx", "volume", "pan", "mute", "solo",
        "reaper", "daw", "create", "delete", "move", "select",
        "color", "rename", "add", "remove", "enable", "disable",
        "instrument", "plugin", "effect", "compressor", "reverb", "eq",
        "mix", "master", "bus", "send", "return",
    ]
    
    arranger_keywords = [
        "chord", "progression", "melody", "note", "notes",
        "I", "VI", "IV", "V", "ii", "iii", "vii",
        "roman", "scale", "harmony", "sequence", "pattern",
        "major", "minor", "diminished", "augmented",
        "triad", "seventh", "ninth",
        "arpeggio", "bassline", "riff", "hook", "groove", "lick",
        "phrase", "motif", "ostinato", "fill", "break",
        "C", "D", "E", "F", "G", "A", "B",
        "sharp", "flat", "natural",
        "pentatonic", "dorian", "mixolydian",
        "sus2", "sus4", "add9",
    ]
    
    print(f"🔍 Expanding DAW keywords ({len(daw_keywords)} base keywords)...")
    expanded_daw = expand_keywords(client, daw_keywords, "DAW operations and REAPER-specific terms")
    
    print(f"🔍 Expanding Arranger keywords ({len(arranger_keywords)} base keywords)...")
    expanded_arranger = expand_keywords(client, arranger_keywords, "musical content and music theory terms")
    
    # Combine base + expanded (deduplicate)
    all_daw = deduplicate(daw_keywords + expanded_daw)
    all_arranger = deduplicate(arranger_keywords + expanded_arranger)
    
    result = {
        "daw": all_daw,
        "arranger": all_arranger
    }
    
    # Output as JSON
    output = json.dumps(result, indent=2, ensure_ascii=False)
    print("\n" + output)
    
    # Also write to file
    output_file = "expanded_keywords.json"
    with open(output_file, "w", encoding="utf-8") as f:
        f.write(output)
    print(f"\n💾 Also saved to {output_file}")
    
    print(f"\n✅ Expanded to {len(all_daw)} DAW keywords (from {len(daw_keywords)}) "
          f"and {len(all_arranger)} Arranger keywords (from {len(arranger_keywords)})")


if __name__ == "__main__":
    main()

