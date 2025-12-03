package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
)

// PluginInfo represents a REAPER plugin
type PluginInfo struct {
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	Format       string `json:"format"`
	Manufacturer string `json:"manufacturer"`
	IsInstrument bool   `json:"is_instrument"`
	Ident        string `json:"ident"`
}

// PluginAlias represents an alias mapping
type PluginAlias struct {
	Alias      string  `json:"alias"`
	FullName   string  `json:"full_name"`
	Format     string  `json:"format"`
	Confidence float64 `json:"confidence,omitempty"`
}

// Preferences defines user preferences for plugin format selection
type Preferences struct {
	FormatOrder []string `json:"format_order"` // e.g., ["VST3", "VST", "AU", "JS"]
	PreferNewer bool     `json:"prefer_newer"` // Prefer newer versions when available
}

// DefaultPreferences returns default plugin preferences (VST3 > VST > AU > JS)
func DefaultPreferences() Preferences {
	return Preferences{
		FormatOrder: []string{"VST3", "VST3i", "VST", "VSTi", "AU", "AUi", "JS", "ReaPlugs"},
		PreferNewer: true,
	}
}

// PluginAgent handles plugin-related operations
type PluginAgent struct {
	llmProvider llm.Provider
	cfg         *config.Config
}

// NewPluginAgent creates a new plugin agent
func NewPluginAgent(cfg *config.Config) *PluginAgent {
	provider := llm.NewOpenAIProvider(cfg.OpenAIAPIKey)
	return &PluginAgent{
		llmProvider: provider,
		cfg:         cfg,
	}
}

// GenerateAliasesRequest is the request for generating plugin aliases
type GenerateAliasesRequest struct {
	Plugins []PluginInfo `json:"plugins" binding:"required"`
}

// GenerateAliasesResponse is the response with generated aliases
type GenerateAliasesResponse struct {
	Aliases map[string]PluginAlias `json:"aliases"` // alias -> PluginAlias
}

// GenerateAliases uses LLM to generate smart aliases for plugins
func (a *PluginAgent) GenerateAliases(ctx context.Context, plugins []PluginInfo) (map[string]PluginAlias, error) {
	if len(plugins) == 0 {
		return make(map[string]PluginAlias), nil
	}

	log.Printf("🤖 Generating aliases for %d plugins", len(plugins))

	// Build prompt for LLM
	prompt := a.buildAliasPrompt(plugins)

	// Call LLM
	request := &llm.GenerationRequest{
		Model:         "gpt-5.1",
		InputArray:    []map[string]interface{}{{"role": "user", "content": prompt}},
		ReasoningMode: "none",
		SystemPrompt:  "You are a helpful assistant that generates smart aliases for REAPER plugins. Return only valid JSON.",
		OutputSchema: &llm.OutputSchema{
			Name:        "PluginAliases",
			Description: "Map of aliases to plugin full names",
			Schema:      a.getAliasSchemaMap(),
		},
	}

	resp, err := a.llmProvider.Generate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse response
	aliases, err := a.parseAliasResponse(resp.RawOutput, plugins)
	if err != nil {
		return nil, fmt.Errorf("failed to parse alias response: %w", err)
	}

	log.Printf("✅ Generated %d aliases", len(aliases))
	return aliases, nil
}

// DeduplicatePlugins removes duplicate plugins based on user preferences
// Returns a map of plugin name -> best plugin (based on format preferences)
func (a *PluginAgent) DeduplicatePlugins(plugins []PluginInfo, prefs Preferences) map[string]PluginInfo {
	if len(prefs.FormatOrder) == 0 {
		prefs = DefaultPreferences()
	}

	// Group plugins by name (case-insensitive)
	pluginGroups := make(map[string][]PluginInfo)
	for _, plugin := range plugins {
		key := strings.ToLower(plugin.Name)
		pluginGroups[key] = append(pluginGroups[key], plugin)
	}

	// For each group, select the best plugin based on preferences
	result := make(map[string]PluginInfo)
	for name, group := range pluginGroups {
		best := a.selectBestPlugin(group, prefs)
		if best != nil {
			result[name] = *best
		}
	}

	return result
}

// selectBestPlugin selects the best plugin from a group based on preferences
func (a *PluginAgent) selectBestPlugin(plugins []PluginInfo, prefs Preferences) *PluginInfo {
	if len(plugins) == 0 {
		return nil
	}
	if len(plugins) == 1 {
		return &plugins[0]
	}

	// Create format priority map
	formatPriority := make(map[string]int)
	for i, format := range prefs.FormatOrder {
		formatPriority[format] = i
		// Also handle instrument variants (VST3i -> VST3)
		if strings.HasSuffix(format, "i") {
			baseFormat := format[:len(format)-1]
			if _, exists := formatPriority[baseFormat]; !exists {
				formatPriority[baseFormat] = i
			}
		}
	}

	// Sort plugins by priority
	sort.Slice(plugins, func(i, j int) bool {
		pi := a.getFormatPriority(plugins[i].Format, formatPriority)
		pj := a.getFormatPriority(plugins[j].Format, formatPriority)
		if pi != pj {
			return pi < pj // Lower priority number = higher preference
		}
		// If same priority, prefer instrument if available
		if plugins[i].IsInstrument != plugins[j].IsInstrument {
			return plugins[i].IsInstrument
		}
		// If still tied, prefer shorter ident (newer versions often have longer paths)
		return len(plugins[i].Ident) < len(plugins[j].Ident)
	})

	return &plugins[0]
}

// getFormatPriority returns the priority of a format (lower = higher preference)
func (a *PluginAgent) getFormatPriority(format string, priorityMap map[string]int) int {
	// Check exact match first
	if priority, exists := priorityMap[format]; exists {
		return priority
	}
	// Check base format (remove 'i' suffix)
	if strings.HasSuffix(format, "i") {
		baseFormat := format[:len(format)-1]
		if priority, exists := priorityMap[baseFormat]; exists {
			return priority
		}
	}
	// Default to lowest priority (highest number)
	return 999
}

// buildAliasPrompt builds the LLM prompt for generating aliases
func (a *PluginAgent) buildAliasPrompt(plugins []PluginInfo) string {
	var builder strings.Builder
	builder.WriteString("Analyze these REAPER plugins and generate smart aliases for each one.\n\n")
	builder.WriteString("For each plugin, create multiple aliases:\n")
	builder.WriteString("- Short name (Serum -> \"serum\")\n")
	builder.WriteString("- Manufacturer prefix (Xfer Records -> \"xfer serum\")\n")
	builder.WriteString("- Common abbreviations (ReaEQ -> \"reaeq\", \"rea-eq\", \"eq\")\n")
	builder.WriteString("- Version numbers (Kontakt 7 -> \"kontakt\", \"kontakt7\")\n")
	builder.WriteString("- Category names if applicable (synth, eq, compressor)\n\n")
	builder.WriteString("Return a JSON object mapping aliases to full plugin names.\n\n")
	builder.WriteString("Plugins:\n")

	// Limit to first 100 plugins to avoid token limits
	maxPlugins := 100
	if len(plugins) > maxPlugins {
		log.Printf("⚠️  Limiting plugin list to %d plugins for alias generation", maxPlugins)
		plugins = plugins[:maxPlugins]
	}

	for i, plugin := range plugins {
		builder.WriteString(fmt.Sprintf("%d. %s (Format: %s, Manufacturer: %s, Instrument: %v)\n",
			i+1, plugin.FullName, plugin.Format, plugin.Manufacturer, plugin.IsInstrument))
	}

	return builder.String()
}

// getAliasSchemaMap returns the JSON schema map for alias generation
func (a *PluginAgent) getAliasSchemaMap() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"aliases": map[string]any{
				"type": "object",
				"additionalProperties": map[string]any{
					"type": "string",
				},
				"description": "Map of alias -> full plugin name",
			},
		},
		"required": []string{"aliases"},
	}
}

// parseAliasResponse parses the LLM response and creates alias mappings
func (a *PluginAgent) parseAliasResponse(response string, plugins []PluginInfo) (map[string]PluginAlias, error) {
	// Create a map of full_name -> PluginInfo for lookup
	pluginMap := make(map[string]*PluginInfo)
	for i := range plugins {
		pluginMap[plugins[i].FullName] = &plugins[i]
	}

	// Parse JSON response
	var result struct {
		Aliases map[string]string `json:"aliases"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to PluginAlias map
	aliases := make(map[string]PluginAlias)
	for alias, fullName := range result.Aliases {
		// Normalize alias to lowercase
		normalizedAlias := strings.ToLower(strings.TrimSpace(alias))
		if normalizedAlias == "" {
			continue
		}

		// Find matching plugin
		var plugin *PluginInfo
		if p, exists := pluginMap[fullName]; exists {
			plugin = p
		} else {
			// Try fuzzy match (case-insensitive)
			for i := range plugins {
				if strings.EqualFold(plugins[i].FullName, fullName) {
					plugin = &plugins[i]
					break
				}
			}
		}

		if plugin != nil {
			aliases[normalizedAlias] = PluginAlias{
				Alias:    normalizedAlias,
				FullName: plugin.FullName,
				Format:   plugin.Format,
			}
		}
	}

	return aliases, nil
}
