package embedded

import (
	_ "embed"
)

// Embed all prompt data files
//
//go:embed data/core_data/system_prompt.txt
var SystemPromptTxt []byte

//go:embed data/core_data/output_format_instructions.txt
var OutputFormatInstructionsTxt []byte

//go:embed data/core_data/user_context_instructions.txt
var UserContextInstructionsTxt []byte

//go:embed data/core_data/mcp_server_instructions.txt
var MCPServerInstructionsTxt []byte

//go:embed data/core_data/musical_scales_heuristics.csv
var MusicalScalesHeuristicsCsv []byte

//go:embed data/core_data/key_emotional_qualities.csv
var KeyEmotionalQualitiesCsv []byte

//go:embed data/core_data/progressions.json
var ProgressionsJSON []byte

//go:embed data/core_data/advanced_harmonic_theory.txt
var AdvancedHarmonicTheoryTxt []byte

//go:embed data/core_data/advanced_rhythm_phrasing.txt
var AdvancedRhythmPhrasingTxt []byte

//go:embed data/core_data/anti_chromatic_heuristics.txt
var AntiChromaticHeuristicsTxt []byte

//go:embed data/core_data/theory_books_chapters.txt
var TheoryBooksChaptersTxt []byte

//go:embed data/core_data/use_case_instructions.txt
var UseCaseInstructionsTxt []byte

//go:embed data/core_data/chord_progression_instructions.txt
var ChordProgressionInstructionsTxt []byte

//go:embed data/core_data/rhythm_articulation_instructions.txt
var RhythmArticulationInstructionsTxt []byte

//go:embed data/prompts/harmonic_planner_prompt.txt
var HarmonicPlannerPromptTxt []byte

//go:embed data/prompts/rhythmic_placement_prompt.txt
var RhythmicPlacementPromptTxt []byte
