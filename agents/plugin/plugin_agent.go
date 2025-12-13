package plugin

import (
	"context"
	"log"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/Conceptual-Machines/magda-agents-go/config"
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
	cfg *config.Config
}

// NewPluginAgent creates a new plugin agent
func NewPluginAgent(cfg *config.Config) *PluginAgent {
	return &PluginAgent{
		cfg: cfg,
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

// GenerateAliases programmatically generates smart aliases for plugins
// Returns a map of alias -> full_name (simple string mapping)
func (a *PluginAgent) GenerateAliases(ctx context.Context, plugins []PluginInfo) (map[string]string, error) {
	if len(plugins) == 0 {
		return make(map[string]string), nil
	}

	log.Printf("🔧 Generating aliases programmatically for %d plugins", len(plugins))

	aliases := make(map[string]string)
	
	// Process all plugins
	for _, plugin := range plugins {
		pluginAliases := a.generateAliasesForPlugin(plugin)
		for _, alias := range pluginAliases {
			// Normalize to lowercase
			normalized := strings.ToLower(strings.TrimSpace(alias))
			if normalized != "" {
				// Handle conflicts: keep first mapping
				if existing, exists := aliases[normalized]; exists && existing != plugin.FullName {
					log.Printf("⚠️  Alias conflict: '%s' maps to both '%s' and '%s' (keeping first)", 
						normalized, existing, plugin.FullName)
				} else if !exists {
					aliases[normalized] = plugin.FullName
				}
			}
		}
	}

	log.Printf("✅ Generated %d aliases for %d plugins", len(aliases), len(plugins))
	return aliases, nil
}

// generateAliasesForPlugin generates all possible aliases for a single plugin
func (a *PluginAgent) generateAliasesForPlugin(plugin PluginInfo) []string {
	aliases := make(map[string]bool) // Use map to avoid duplicates
	
	// Extract base name from full name
	baseName := a.extractBaseName(plugin.FullName)
	if baseName == "" {
		return []string{}
	}
	
	// 1. Simple lowercase alias
	simple := strings.ToLower(baseName)
	aliases[simple] = true
	
	// 2. Remove spaces
	noSpaces := strings.ReplaceAll(simple, " ", "")
	if noSpaces != simple {
		aliases[noSpaces] = true
	}
	
	// 3. Extract version number and create versioned aliases
	versionAliases := a.generateVersionAliases(baseName)
	for _, v := range versionAliases {
		aliases[strings.ToLower(v)] = true
	}
	
	// 4. Split camelCase/PascalCase words
	camelAliases := a.splitCamelCase(baseName)
	for _, ca := range camelAliases {
		aliases[strings.ToLower(ca)] = true
	}
	
	// 5. Manufacturer prefix aliases
	if plugin.Manufacturer != "" {
		manufacturerAliases := a.generateManufacturerAliases(baseName, plugin.Manufacturer)
		for _, ma := range manufacturerAliases {
			aliases[strings.ToLower(ma)] = true
		}
	}
	
	// 6. Common abbreviation patterns
	abbrevAliases := a.generateAbbreviationAliases(baseName)
	for _, aa := range abbrevAliases {
		aliases[strings.ToLower(aa)] = true
	}
	
	// Convert map to slice
	result := make([]string, 0, len(aliases))
	for alias := range aliases {
		result = append(result, alias)
	}
	
	return result
}

// extractBaseName extracts the base plugin name from full name
// Examples:
//   "VST3: Serum (Xfer Records)" -> "Serum"
//   "JS: ReaEQ" -> "ReaEQ"
//   "VST: Kontakt 7 (Native Instruments)" -> "Kontakt 7"
func (a *PluginAgent) extractBaseName(fullName string) string {
	// Remove format prefix (VST3:, VST:, JS:, etc.)
	formatPrefix := regexp.MustCompile(`^(VST3|VST|AU|JS|ReaPlugs):\s*`)
	name := formatPrefix.ReplaceAllString(fullName, "")
	
	// Remove manufacturer suffix in parentheses
	manufacturerSuffix := regexp.MustCompile(`\s*\([^)]+\)\s*$`)
	name = manufacturerSuffix.ReplaceAllString(name, "")
	
	return strings.TrimSpace(name)
}

// generateVersionAliases extracts version numbers and creates aliases
// Examples:
//   "Kontakt 7" -> ["kontakt", "kontakt7", "kontakt 7"]
//   "Serum 1.2" -> ["serum", "serum1.2", "serum 1.2"]
func (a *PluginAgent) generateVersionAliases(baseName string) []string {
	aliases := []string{}
	
	// Match version patterns: "Name 7", "Name 1.2", "Name v2", etc.
	versionPattern := regexp.MustCompile(`(.+?)\s+([vV]?\d+(?:\.\d+)*)`)
	matches := versionPattern.FindStringSubmatch(baseName)
	
	if len(matches) >= 3 {
		namePart := strings.TrimSpace(matches[1])
		versionPart := matches[2]
		
		// Add name without version
		aliases = append(aliases, namePart)
		
		// Add name with version (no space)
		aliases = append(aliases, namePart+versionPart)
		
		// Add name with version (with space)
		aliases = append(aliases, namePart+" "+versionPart)
	}
	
	return aliases
}

// splitCamelCase splits camelCase/PascalCase and generates aliases
// Examples:
//   "ReaEQ" -> ["reaeq", "rea-eq", "rea eq", "eq"]
//   "ReaComp" -> ["reacomp", "rea-comp", "rea comp", "comp"]
func (a *PluginAgent) splitCamelCase(name string) []string {
	aliases := []string{}
	
	// Split on camelCase boundaries
	// Handle both: "ReaEQ" (consecutive uppercase) and "ReaComp" (normal camelCase)
	var words []string
	var currentWord strings.Builder
	
	for i, r := range name {
		// Split on uppercase if:
		// 1. Not the first character AND
		// 2. Current char is uppercase AND
		// 3. Either previous char was lowercase OR next char is lowercase (to handle "ReaEQ")
		if i > 0 && unicode.IsUpper(r) {
			prevRune := rune(name[i-1])
			nextIsLower := i+1 < len(name) && unicode.IsLower(rune(name[i+1]))
			prevIsLower := unicode.IsLower(prevRune)
			
			// Split if previous was lowercase OR if this is start of a new word (next is lowercase)
			if prevIsLower || (nextIsLower && currentWord.Len() > 0) {
				if currentWord.Len() > 0 {
					words = append(words, currentWord.String())
					currentWord.Reset()
				}
			}
		}
		currentWord.WriteRune(r)
	}
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}
	
	if len(words) <= 1 {
		return aliases
	}
	
	// Generate combinations
	// Full joined: "ReaEQ" -> "reaeq"
	aliases = append(aliases, strings.Join(words, ""))
	
	// Hyphenated: "ReaEQ" -> "rea-eq"
	aliases = append(aliases, strings.Join(words, "-"))
	
	// Spaced: "ReaEQ" -> "rea eq"
	aliases = append(aliases, strings.Join(words, " "))
	
	// Last word only (common abbreviation): "ReaEQ" -> "eq"
	if len(words) > 1 {
		aliases = append(aliases, words[len(words)-1])
	}
	
	return aliases
}

// generateManufacturerAliases creates manufacturer-prefixed aliases
// Examples:
//   baseName="Serum", manufacturer="Xfer Records" -> ["xfer serum", "xferrecords serum"]
func (a *PluginAgent) generateManufacturerAliases(baseName, manufacturer string) []string {
	aliases := []string{}
	
	// Extract key words from manufacturer (remove common words)
	manufacturerLower := strings.ToLower(manufacturer)
	manufacturerWords := strings.Fields(manufacturerLower)
	
	// Filter out common words
	commonWords := map[string]bool{
		"records": true, "inc": true, "ltd": true, "llc": true,
		"audio": true, "music": true, "technologies": true,
	}
	
	var keyWords []string
	for _, word := range manufacturerWords {
		if !commonWords[word] && len(word) > 2 {
			keyWords = append(keyWords, word)
		}
	}
	
	if len(keyWords) == 0 {
		// If no key words, use first word
		if len(manufacturerWords) > 0 {
			keyWords = []string{manufacturerWords[0]}
		}
	}
	
	// Generate aliases
	for _, keyword := range keyWords {
		aliases = append(aliases, keyword+" "+baseName)
		aliases = append(aliases, keyword+baseName)
	}
	
	return aliases
}

// generateAbbreviationAliases creates common abbreviation patterns
// Examples:
//   "ReaEQ" -> ["eq"] (if it ends with EQ)
//   "ReaComp" -> ["comp"] (if it ends with Comp)
func (a *PluginAgent) generateAbbreviationAliases(baseName string) []string {
	aliases := []string{}
	baseLower := strings.ToLower(baseName)
	
	// Common suffix patterns
	suffixPatterns := map[string]string{
		"eq":      "eq",
		"comp":    "comp",
		"compressor": "comp",
		"verb":    "verb",
		"reverb":  "verb",
		"delay":   "delay",
		"limiter": "limit",
		"gate":    "gate",
		"filter":  "filter",
		"synth":   "synth",
		"synthesizer": "synth",
	}
	
	for suffix, abbrev := range suffixPatterns {
		if strings.HasSuffix(baseLower, suffix) {
			aliases = append(aliases, abbrev)
			break
		}
	}
	
	return aliases
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

