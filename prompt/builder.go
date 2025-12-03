package prompt

import (
	"fmt"
	"strings"
)

type Builder struct {
	loader *Loader
}

func NewPromptBuilder() *Builder {
	return &Builder{
		loader: NewPromptLoader(),
	}
}

// BuildPrompt builds the complete static system prompt
func (b *Builder) BuildPrompt() (string, error) {
	sections := []string{}

	// System prompt
	systemPrompt, err := b.loader.GetSystemPrompt()
	if err != nil {
		return "", err
	}
	sections = append(sections, systemPrompt)

	// User context instructions
	userContextInstructions, err := b.loader.GetUserContextInstructions()
	if err != nil {
		return "", err
	}
	sections = append(sections, fmt.Sprintf("\n\n%s\n\n", userContextInstructions))

	// Use case specific instructions (NEW)
	useCaseInstructions, err := b.loader.GetUseCaseInstructions()
	if err != nil {
		return "", err
	}
	sections = append(sections, fmt.Sprintf("\n\n%s\n\n", useCaseInstructions))

	// Chord progression specific instructions (DETAILED)
	chordProgressionInstructions, err := b.loader.GetChordProgressionInstructions()
	if err != nil {
		return "", err
	}
	sections = append(sections, fmt.Sprintf("\n\n%s\n\n", chordProgressionInstructions))

	// Rhythm and articulation instructions (CRITICAL FOR GROOVE)
	rhythmArticulationInstructions, err := b.loader.GetRhythmArticulationInstructions()
	if err != nil {
		return "", err
	}
	sections = append(sections, fmt.Sprintf("\n\n%s\n\n", rhythmArticulationInstructions))

	// MCP server instructions
	mcpInstructions, err := b.loader.GetMCPServerInstructions()
	if err != nil {
		return "", err
	}
	sections = append(sections, fmt.Sprintf("\n\n%s\n\n", mcpInstructions))

	// Musical guidelines
	musicalGuidelines, err := b.getMusicalGuidelines()
	if err != nil {
		return "", err
	}
	sections = append(sections, musicalGuidelines)

	// Anti-chromatic heuristics
	antiChromaticHeuristics, err := b.loader.GetAntiChromaticHeuristics()
	if err != nil {
		return "", err
	}
	sections = append(sections, fmt.Sprintf("\n\n%s", antiChromaticHeuristics))

	// Chord progressions
	chordProgressions, err := b.getChordProgressionsReference()
	if err != nil {
		return "", err
	}
	sections = append(sections, chordProgressions)

	// Key emotional qualities
	keyEmotionalQualities, err := b.getKeyEmotionalQualities()
	if err != nil {
		return "", err
	}
	sections = append(sections, keyEmotionalQualities)

	// Scales descriptions
	scalesDescriptions, err := b.getScalesDescriptions()
	if err != nil {
		return "", err
	}
	sections = append(sections, scalesDescriptions)

	// Theory books chapters
	theoryBooksChapters, err := b.getTheoryBooksChapters()
	if err != nil {
		return "", err
	}
	sections = append(sections, theoryBooksChapters)

	// Output format instructions
	outputFormatInstructions, err := b.loader.GetOutputFormatInstructions()
	if err != nil {
		return "", err
	}
	sections = append(sections, fmt.Sprintf("\n\n%s", outputFormatInstructions))

	return strings.Join(sections, ""), nil
}

func (b *Builder) getMusicalGuidelines() (string, error) {
	circleOfFifths := b.getCircleOfFifthsSection()
	phrasing, err := b.getPhrasingSection()
	if err != nil {
		return "", err
	}
	return circleOfFifths + phrasing, nil
}

func (b *Builder) getCircleOfFifthsSection() string {
	harmonicTheory, _ := b.loader.GetAdvancedHarmonicTheory()
	return fmt.Sprintf("\n\n%s", harmonicTheory)
}

func (b *Builder) getPhrasingSection() (string, error) {
	rhythmPhrasing, err := b.loader.GetAdvancedRhythmPhrasing()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("\n\n%s", rhythmPhrasing), nil
}

func (b *Builder) getChordProgressionsReference() (string, error) {
	progressions, err := b.loader.GetProgressions()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("\n\n## Chord Progressions Reference:\n\n%s", progressions), nil
}

func (b *Builder) getKeyEmotionalQualities() (string, error) {
	qualitiesText, err := b.loader.GetKeyEmotionalQualities()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("\n\n## Key Emotional Qualities:\n\n%s", qualitiesText), nil
}

func (b *Builder) getScalesDescriptions() (string, error) {
	scalesText, err := b.loader.GetMusicalScalesHeuristics()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("\n\n## Musical Scales Qualitative Descriptions:\n\n%s", scalesText), nil
}

func (b *Builder) getTheoryBooksChapters() (string, error) {
	chaptersText, err := b.loader.GetTheoryBooksChapters()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("\n\n## Music Theory Books - Chapter Index:\n\n%s", chaptersText), nil
}
