package prompt

import (
	"strings"

	"github.com/Conceptual-Machines/magda-agents-go/pkg/embedded"
)

type Loader struct{}

func NewPromptLoader() *Loader {
	return &Loader{}
}

// GetSystemPrompt loads the main system prompt
func (l *Loader) GetSystemPrompt() (string, error) {
	return strings.TrimSpace(string(embedded.SystemPromptTxt)), nil
}

// GetOutputFormatInstructions loads output format instructions
func (l *Loader) GetOutputFormatInstructions() (string, error) {
	return strings.TrimSpace(string(embedded.OutputFormatInstructionsTxt)), nil
}

// GetUserContextInstructions loads user context instructions
func (l *Loader) GetUserContextInstructions() (string, error) {
	return strings.TrimSpace(string(embedded.UserContextInstructionsTxt)), nil
}

// GetMCPServerInstructions loads MCP server instructions
func (l *Loader) GetMCPServerInstructions() (string, error) {
	return strings.TrimSpace(string(embedded.MCPServerInstructionsTxt)), nil
}

// GetMusicalScalesHeuristics loads musical scales heuristics CSV
func (l *Loader) GetMusicalScalesHeuristics() (string, error) {
	return strings.TrimSpace(string(embedded.MusicalScalesHeuristicsCsv)), nil
}

// GetKeyEmotionalQualities loads key emotional qualities CSV
func (l *Loader) GetKeyEmotionalQualities() (string, error) {
	return strings.TrimSpace(string(embedded.KeyEmotionalQualitiesCsv)), nil
}

// GetProgressions loads chord progressions as text
func (l *Loader) GetProgressions() (string, error) {
	return strings.TrimSpace(string(embedded.ProgressionsJSON)), nil
}

// GetAdvancedHarmonicTheory loads advanced harmonic theory
func (l *Loader) GetAdvancedHarmonicTheory() (string, error) {
	return strings.TrimSpace(string(embedded.AdvancedHarmonicTheoryTxt)), nil
}

// GetAdvancedRhythmPhrasing loads advanced rhythm and phrasing
func (l *Loader) GetAdvancedRhythmPhrasing() (string, error) {
	return strings.TrimSpace(string(embedded.AdvancedRhythmPhrasingTxt)), nil
}

// GetAntiChromaticHeuristics loads anti-chromatic heuristics
func (l *Loader) GetAntiChromaticHeuristics() (string, error) {
	return strings.TrimSpace(string(embedded.AntiChromaticHeuristicsTxt)), nil
}

// GetTheoryBooksChapters loads theory books chapters index
func (l *Loader) GetTheoryBooksChapters() (string, error) {
	return strings.TrimSpace(string(embedded.TheoryBooksChaptersTxt)), nil
}

// GetUseCaseInstructions loads use case specific instructions
func (l *Loader) GetUseCaseInstructions() (string, error) {
	return strings.TrimSpace(string(embedded.UseCaseInstructionsTxt)), nil
}

// GetChordProgressionInstructions loads detailed chord progression guidelines
func (l *Loader) GetChordProgressionInstructions() (string, error) {
	return strings.TrimSpace(string(embedded.ChordProgressionInstructionsTxt)), nil
}

// GetRhythmArticulationInstructions loads rhythm and groove guidelines
func (l *Loader) GetRhythmArticulationInstructions() (string, error) {
	return strings.TrimSpace(string(embedded.RhythmArticulationInstructionsTxt)), nil
}
