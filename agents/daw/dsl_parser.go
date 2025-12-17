package daw

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

const (
	// TestInstrumentName is used in tests and examples
	TestInstrumentName = "Serum"
	// BooleanTrue is the string representation of true
	BooleanTrue = "true"
	// MaxDSLPreviewLength is the maximum length for DSL preview in logs
	MaxDSLPreviewLength = 200
)

// DSLParser parses MAGDA DSL code and translates it to REAPER API actions
type DSLParser struct {
	trackCounter int            // Track index counter for implicit track references
	state        map[string]any // Current REAPER state for track resolution
}

// NewDSLParser creates a new DSL parser
func NewDSLParser() *DSLParser {
	return &DSLParser{
		trackCounter: 0,
		state:        nil,
	}
}

// SetState sets the current REAPER state for track resolution
func (p *DSLParser) SetState(state map[string]any) {
	p.state = state
}

// ParseDSL parses DSL code and returns REAPER API actions
// Example: track(instrument="Serum").newClip(bar=3, length_bars=4)
// Returns: [{"action": "create_track", "instrument": "Serum"}, {"action": "create_clip_at_bar", "track": 0, "bar": 3, "length_bars": 4}]
//
//nolint:gocyclo // Complex parsing logic is necessary for DSL translation
func (p *DSLParser) ParseDSL(dslCode string) ([]map[string]any, error) {
	dslCode = strings.TrimSpace(dslCode)
	if dslCode == "" {
		return nil, fmt.Errorf("empty DSL code")
	}

	var actions []map[string]any
	currentTrackIndex := -1

	// Split by method chains (e.g., track().newClip().addMidi())
	// Simple regex-based parser for now - can be enhanced with proper AST later
	parts := p.splitMethodChains(dslCode)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse track() call - could be creation or reference
		if strings.HasPrefix(part, "track(") {
			// Check if this is a track reference (track(id), track(1), or track(selected=true))
			params := p.extractParams(part)
			if idStr, hasID := params["id"]; hasID {
				// track(id=1) - reference existing track
				if trackNum, err := strconv.Atoi(idStr); err == nil {
					currentTrackIndex = trackNum - 1 // Convert 1-based to 0-based
					// No action needed - just set the track context for chaining
					continue
				}
			} else if selectedStr, hasSelected := params["selected"]; hasSelected {
				// track(selected=true) - reference currently selected track
				// NOTE: Currently returns first selected track only (REAPER supports multiple selections)
				if selectedStr == "true" || selectedStr == "True" {
					selectedIndex := p.getSelectedTrackIndex()
					if selectedIndex >= 0 {
						currentTrackIndex = selectedIndex
						// No action needed - just set the track context for chaining
						continue
					}
					// If no selected track found, fall through to error or creation
					return nil, fmt.Errorf("no selected track found in state")
				}
			} else if len(params) == 0 {
				// Check if it's just track(1) - a bare number
				// Extract content between parentheses
				start := strings.Index(part, "(")
				end := strings.LastIndex(part, ")")
				if start >= 0 && end > start {
					content := strings.TrimSpace(part[start+1 : end])
					if trackNum, err := strconv.Atoi(content); err == nil {
						// track(1) - reference existing track
						currentTrackIndex = trackNum - 1 // Convert 1-based to 0-based
						// No action needed - just set the track context for chaining
						continue
					}
				}
			}

			// If we get here, it's a track creation call
			trackAction, trackIndex, err := p.parseTrackCall(part)
			if err != nil {
				return nil, fmt.Errorf("failed to parse track call: %w", err)
			}
			actions = append(actions, trackAction)
			currentTrackIndex = trackIndex
		} else if strings.HasPrefix(part, ".newClip(") {
			// Parse .newClip() call
			// Use currentTrackIndex from track() or track(id) context, or fallback to selected track
			trackIndex := currentTrackIndex
			if trackIndex < 0 {
				// No track context - use selected track from state as fallback
				trackIndex = p.getSelectedTrackIndex()
			}
			clipAction, err := p.parseClipCall(part, trackIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to parse clip call: %w", err)
			}
			actions = append(actions, clipAction)
		} else if strings.HasPrefix(part, ".addFX(") || strings.HasPrefix(part, ".addInstrument(") {
			// Parse FX/instrument call
			fxAction, err := p.parseFXCall(part, currentTrackIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to parse FX call: %w", err)
			}
			actions = append(actions, fxAction)
		} else if strings.HasPrefix(part, ".setVolume(") {
			// Parse volume call
			volumeAction, err := p.parseVolumeCall(part, currentTrackIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to parse volume call: %w", err)
			}
			actions = append(actions, volumeAction)
		} else if strings.HasPrefix(part, ".setPan(") {
			// Parse pan call
			panAction, err := p.parsePanCall(part, currentTrackIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to parse pan call: %w", err)
			}
			actions = append(actions, panAction)
		} else if strings.HasPrefix(part, ".setMute(") {
			// Parse mute call
			muteAction, err := p.parseMuteCall(part, currentTrackIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to parse mute call: %w", err)
			}
			actions = append(actions, muteAction)
		} else if strings.HasPrefix(part, ".setSolo(") {
			// Parse solo call
			soloAction, err := p.parseSoloCall(part, currentTrackIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to parse solo call: %w", err)
			}
			actions = append(actions, soloAction)
		} else if strings.HasPrefix(part, ".setName(") {
			// Parse name call
			nameAction, err := p.parseNameCall(part, currentTrackIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to parse name call: %w", err)
			}
			actions = append(actions, nameAction)
		} else if strings.HasPrefix(part, ".addAutomation(") {
			// Parse automation call
			automationAction, err := p.parseAutomationCall(part, currentTrackIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to parse automation call: %w", err)
			}
			actions = append(actions, automationAction)
		}
	}

	if len(actions) == 0 {
		return nil, fmt.Errorf("no actions found in DSL code")
	}

	log.Printf("✅ DSL Parser: Translated %d actions from DSL", len(actions))
	return actions, nil
}

// splitMethodChains splits DSL code into method calls
// Example: "track(instrument=\"Serum\").newClip(bar=3)" -> ["track(instrument=\"Serum\")", ".newClip(bar=3)"]
func (p *DSLParser) splitMethodChains(dslCode string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	inString := false
	escape := false

	for i, char := range dslCode {
		if escape {
			current.WriteRune(char)
			escape = false
			continue
		}

		switch char {
		case '\\':
			escape = true
			current.WriteRune(char)
		case '"':
			inString = !inString
			current.WriteRune(char)
		case '(':
			if !inString {
				depth++
			}
			current.WriteRune(char)
		case ')':
			if !inString {
				depth--
				if depth == 0 {
					current.WriteRune(char)
					parts = append(parts, current.String())
					current.Reset()
					// Skip whitespace and dots after closing paren
					for i+1 < len(dslCode) && (dslCode[i+1] == '.' || dslCode[i+1] == ' ' || dslCode[i+1] == '\n') {
						i++
					}
					continue
				}
			}
			current.WriteRune(char)
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parseTrackCall parses track(instrument="Serum", name="Bass")
func (p *DSLParser) parseTrackCall(call string) (map[string]any, int, error) {
	action := map[string]any{
		"action": "create_track",
	}

	// Extract parameters from track(...)
	params := p.extractParams(call)
	if instrument, ok := params["instrument"]; ok {
		action["instrument"] = instrument
	}
	if name, ok := params["name"]; ok {
		action["name"] = name
	}
	if indexStr, ok := params["index"]; ok {
		if index, err := strconv.Atoi(indexStr); err == nil {
			action["index"] = index
			p.trackCounter = index + 1
		}
	} else {
		action["index"] = p.trackCounter
		p.trackCounter++
	}

	return action, action["index"].(int), nil
}

// parseClipCall parses .newClip(bar=3, length_bars=4) or .newClip(start=1.5, length=2.0)
// trackIndex should already be resolved (0-based) before calling this
func (p *DSLParser) parseClipCall(call string, trackIndex int) (map[string]any, error) {
	if trackIndex < 0 {
		// Try fallback to selected track one more time
		trackIndex = p.getSelectedTrackIndex()
		if trackIndex < 0 {
			return nil, fmt.Errorf("no track context for clip call and no selected track found")
		}
	}

	params := p.extractParams(call)
	action := map[string]any{
		"action": "create_clip",
		"track":  trackIndex,
	}

	if bar, ok := params["bar"]; ok {
		// Use create_clip_at_bar
		action["action"] = "create_clip_at_bar"
		if barInt, err := strconv.Atoi(bar); err == nil {
			action["bar"] = barInt
		}
		if lengthBars, ok := params["length_bars"]; ok {
			if lengthInt, err := strconv.Atoi(lengthBars); err == nil {
				action["length_bars"] = lengthInt
			}
		} else {
			action["length_bars"] = 4 // Default
		}
	} else if start, ok := params["start"]; ok {
		// Use create_clip with time-based positioning
		if startFloat, err := strconv.ParseFloat(start, 64); err == nil {
			action["position"] = startFloat
		}
		if length, ok := params["length"]; ok {
			if lengthFloat, err := strconv.ParseFloat(length, 64); err == nil {
				action["length"] = lengthFloat
			}
		} else {
			action["length"] = 4.0 // Default
		}
	} else if position, ok := params["position"]; ok {
		// Alias for start
		if posFloat, err := strconv.ParseFloat(position, 64); err == nil {
			action["position"] = posFloat
		}
		if length, ok := params["length"]; ok {
			if lengthFloat, err := strconv.ParseFloat(length, 64); err == nil {
				action["length"] = lengthFloat
			}
		} else {
			action["length"] = 4.0 // Default
		}
	} else {
		return nil, fmt.Errorf("clip call must specify bar or start/position")
	}

	return action, nil
}

// NOTE: MIDI parsing removed - add_midi is handled by ARRANGER agent, not DAW agent

// parseFXCall parses .addFX(fxname="ReaEQ") or .addInstrument(instrument="Serum")
func (p *DSLParser) parseFXCall(call string, trackIndex int) (map[string]any, error) {
	if trackIndex < 0 {
		return nil, fmt.Errorf("no track context for FX call")
	}

	params := p.extractParams(call)
	action := map[string]any{
		"action": "add_track_fx",
		"track":  trackIndex,
	}

	if fxname, ok := params["fxname"]; ok {
		action["fxname"] = fxname
	} else if instrument, ok := params["instrument"]; ok {
		action["action"] = "add_instrument"
		action["fxname"] = instrument
	} else {
		return nil, fmt.Errorf("FX call must specify fxname or instrument")
	}

	return action, nil
}

// parseVolumeCall parses .setVolume(volume_db=-3.0)
func (p *DSLParser) parseVolumeCall(call string, trackIndex int) (map[string]any, error) {
	if trackIndex < 0 {
		return nil, fmt.Errorf("no track context for volume call")
	}

	params := p.extractParams(call)
	action := map[string]any{
		"action": "set_track",
		"track":  trackIndex,
	}

	if volume, ok := params["volume_db"]; ok {
		if volFloat, err := strconv.ParseFloat(volume, 64); err == nil {
			action["volume_db"] = volFloat
		}
	} else {
		return nil, fmt.Errorf("volume call must specify volume_db")
	}

	return action, nil
}

// parsePanCall parses .setPan(pan=0.5)
func (p *DSLParser) parsePanCall(call string, trackIndex int) (map[string]any, error) {
	if trackIndex < 0 {
		return nil, fmt.Errorf("no track context for pan call")
	}

	params := p.extractParams(call)
	action := map[string]any{
		"action": "set_track",
		"track":  trackIndex,
	}

	if pan, ok := params["pan"]; ok {
		if panFloat, err := strconv.ParseFloat(pan, 64); err == nil {
			action["pan"] = panFloat
		}
	} else {
		return nil, fmt.Errorf("pan call must specify pan")
	}

	return action, nil
}

// parseMuteCall parses .setMute(mute=true)
func (p *DSLParser) parseMuteCall(call string, trackIndex int) (map[string]any, error) {
	if trackIndex < 0 {
		return nil, fmt.Errorf("no track context for mute call")
	}

	params := p.extractParams(call)
	action := map[string]any{
		"action": "set_track",
		"track":  trackIndex,
	}

	if mute, ok := params["mute"]; ok {
		action["mute"] = mute == BooleanTrue
	} else {
		return nil, fmt.Errorf("mute call must specify mute")
	}

	return action, nil
}

// parseSoloCall parses .setSolo(solo=true)
func (p *DSLParser) parseSoloCall(call string, trackIndex int) (map[string]any, error) {
	if trackIndex < 0 {
		return nil, fmt.Errorf("no track context for solo call")
	}

	params := p.extractParams(call)
	action := map[string]any{
		"action": "set_track",
		"track":  trackIndex,
	}

	if solo, ok := params["solo"]; ok {
		action["solo"] = solo == BooleanTrue
	} else {
		return nil, fmt.Errorf("solo call must specify solo")
	}

	return action, nil
}

// parseNameCall parses .setName(name="Bass")
func (p *DSLParser) parseNameCall(call string, trackIndex int) (map[string]any, error) {
	if trackIndex < 0 {
		return nil, fmt.Errorf("no track context for name call")
	}

	params := p.extractParams(call)
	action := map[string]any{
		"action": "set_track",
		"track":  trackIndex,
	}

	if name, ok := params["name"]; ok {
		action["name"] = name
	} else {
		return nil, fmt.Errorf("name call must specify name")
	}

	return action, nil
}

// extractParams extracts key=value parameters from a function call
// Example: track(instrument="Serum", name="Bass") -> {"instrument": "Serum", "name": "Bass"}
//
//nolint:gocyclo // Complex parsing logic is necessary for parameter extraction
func (p *DSLParser) extractParams(call string) map[string]string {
	params := make(map[string]string)

	// Find the content between parentheses
	start := strings.Index(call, "(")
	end := strings.LastIndex(call, ")")
	if start < 0 || end < 0 || end <= start {
		return params
	}

	content := call[start+1 : end]
	content = strings.TrimSpace(content)
	if content == "" {
		return params
	}

	// Simple parameter parsing - split by comma, respecting strings
	var currentKey strings.Builder
	var currentValue strings.Builder
	inString := false
	escape := false
	expectingValue := false
	currentParamKey := ""

	for _, char := range content {
		if escape {
			if inString {
				currentValue.WriteRune(char)
			}
			escape = false
			continue
		}

		switch char {
		case '\\':
			escape = true
			if inString {
				currentValue.WriteRune(char)
			}
		case '"':
			inString = !inString
			if !inString {
				// Ending string value
				if currentParamKey != "" {
					params[currentParamKey] = currentValue.String()
					currentParamKey = ""
					currentValue.Reset()
					expectingValue = false
				}
			}
		case '=':
			if !inString {
				currentParamKey = strings.TrimSpace(currentKey.String())
				currentKey.Reset()
				expectingValue = true
			} else {
				currentValue.WriteRune(char)
			}
		case ',':
			if !inString {
				if currentParamKey != "" && currentValue.Len() > 0 {
					// Non-string value
					valueStr := strings.TrimSpace(currentValue.String())
					params[currentParamKey] = valueStr
					currentParamKey = ""
					currentValue.Reset()
					currentKey.Reset()
					expectingValue = false
				}
			} else {
				currentValue.WriteRune(char)
			}
		default:
			if expectingValue {
				currentValue.WriteRune(char)
			} else {
				currentKey.WriteRune(char)
			}
		}
	}

	// Handle last parameter
	if currentParamKey != "" {
		valueStr := strings.TrimSpace(currentValue.String())
		if valueStr != "" {
			params[currentParamKey] = valueStr
		}
	}

	return params
}

// parseAutomationCall parses automation with either curve-based or point-based syntax
// Curve-based: .addAutomation(param="volume", curve="fade_in", start=0, end=4)
// Point-based: .addAutomation(param="volume", points=[{time=0, value=-60}, {time=4, value=0}])
//
//nolint:gocyclo // Complex parsing logic is necessary for automation parsing
func (p *DSLParser) parseAutomationCall(call string, trackIndex int) (map[string]any, error) {
	if trackIndex < 0 {
		return nil, fmt.Errorf("no track context for automation call")
	}

	action := map[string]any{
		"action": "add_automation",
		"track":  trackIndex,
	}

	// Extract the full content between parentheses
	start := strings.Index(call, "(")
	end := strings.LastIndex(call, ")")
	if start < 0 || end < 0 || end <= start {
		return nil, fmt.Errorf("invalid automation call syntax")
	}
	content := call[start+1 : end]

	// Parse param parameter (required)
	paramMatch := extractStringParam(content, "param")
	if paramMatch != "" {
		action["param"] = paramMatch
	} else {
		return nil, fmt.Errorf("automation call must specify param")
	}

	// Check for curve-based syntax (preferred)
	curveMatch := extractStringParam(content, "curve")
	if curveMatch != "" {
		// Curve-based automation
		action["curve"] = curveMatch

		// Parse timing parameters
		if startVal := extractNumberParam(content, "start"); startVal >= 0 || strings.Contains(content, "start=0") {
			action["start"] = startVal
		}
		if endVal := extractNumberParam(content, "end"); endVal >= 0 {
			action["end"] = endVal
		}
		if startBar := extractNumberParam(content, "start_bar"); startBar >= 0 {
			action["start_bar"] = startBar
		}
		if endBar := extractNumberParam(content, "end_bar"); endBar >= 0 {
			action["end_bar"] = endBar
		}

		// Parse value range (for ramp, exp curves)
		if fromVal := extractNumberParamStr(content, "from"); fromVal != "" {
			if f, err := strconv.ParseFloat(fromVal, 64); err == nil {
				action["from"] = f
			}
		}
		if toVal := extractNumberParamStr(content, "to"); toVal != "" {
			if f, err := strconv.ParseFloat(toVal, 64); err == nil {
				action["to"] = f
			}
		}

		// Parse oscillator parameters (for sine, saw, square)
		if freqVal := extractNumberParam(content, "freq"); freqVal >= 0 {
			action["freq"] = freqVal
		}
		if ampVal := extractNumberParamStr(content, "amplitude"); ampVal != "" {
			if f, err := strconv.ParseFloat(ampVal, 64); err == nil {
				action["amplitude"] = f
			}
		}
		if phaseVal := extractNumberParamStr(content, "phase"); phaseVal != "" {
			if f, err := strconv.ParseFloat(phaseVal, 64); err == nil {
				action["phase"] = f
			}
		}

		return action, nil
	}

	// Fall back to point-based syntax
	// Parse shape parameter (optional)
	shapeMatch := extractNumberParam(content, "shape")
	if shapeMatch >= 0 {
		action["shape"] = int(shapeMatch)
	}

	// Parse points array
	pointsStart := strings.Index(content, "points=[")
	if pointsStart < 0 {
		return nil, fmt.Errorf("automation call must specify either 'curve' or 'points'")
	}
	pointsStart += len("points=[")

	// Find matching closing bracket
	depth := 1
	pointsEnd := pointsStart
	for i := pointsStart; i < len(content); i++ {
		if content[i] == '[' {
			depth++
		} else if content[i] == ']' {
			depth--
			if depth == 0 {
				pointsEnd = i
				break
			}
		}
	}

	pointsContent := content[pointsStart:pointsEnd]
	points, err := p.parseAutomationPoints(pointsContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse automation points: %w", err)
	}
	action["points"] = points

	return action, nil
}

// parseAutomationPoints parses [{time=0, value=-60}, {time=4, value=0}]
func (p *DSLParser) parseAutomationPoints(content string) ([]map[string]any, error) {
	var points []map[string]any

	// Split by }, { pattern - each point is enclosed in {}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("empty points array")
	}

	// Find each point object
	depth := 0
	pointStart := -1
	for i, char := range content {
		if char == '{' {
			if depth == 0 {
				pointStart = i + 1
			}
			depth++
		} else if char == '}' {
			depth--
			if depth == 0 && pointStart >= 0 {
				pointContent := content[pointStart:i]
				point, err := p.parseAutomationPoint(pointContent)
				if err != nil {
					return nil, err
				}
				points = append(points, point)
				pointStart = -1
			}
		}
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("no points found in automation array")
	}

	return points, nil
}

// parseAutomationPoint parses time=0, value=-60 or bar=1, value=0
func (p *DSLParser) parseAutomationPoint(content string) (map[string]any, error) {
	point := make(map[string]any)

	// Parse time or bar
	timeVal := extractNumberParam(content, "time")
	barVal := extractNumberParam(content, "bar")
	if timeVal >= 0 || (timeVal == 0 && strings.Contains(content, "time=")) {
		point["time"] = timeVal
	} else if barVal >= 0 || (barVal == 0 && strings.Contains(content, "bar=")) {
		point["bar"] = barVal
	} else {
		return nil, fmt.Errorf("automation point must specify time or bar")
	}

	// Parse value (required)
	valueStr := extractNumberParamStr(content, "value")
	if valueStr == "" {
		return nil, fmt.Errorf("automation point must specify value")
	}
	valueFloat, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid value in automation point: %s", valueStr)
	}
	point["value"] = valueFloat

	return point, nil
}

// extractStringParam extracts a string parameter like param="volume" from content
func extractStringParam(content, paramName string) string {
	// Look for param="value"
	pattern := paramName + "=\""
	idx := strings.Index(content, pattern)
	if idx < 0 {
		return ""
	}
	start := idx + len(pattern)
	end := strings.Index(content[start:], "\"")
	if end < 0 {
		return ""
	}
	return content[start : start+end]
}

// extractNumberParam extracts a numeric parameter like shape=5 from content
func extractNumberParam(content, paramName string) float64 {
	str := extractNumberParamStr(content, paramName)
	if str == "" {
		return -1
	}
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return -1
	}
	return val
}

// extractNumberParamStr extracts a numeric parameter as string
func extractNumberParamStr(content, paramName string) string {
	// Look for param=value (number, possibly negative)
	pattern := paramName + "="
	idx := strings.Index(content, pattern)
	if idx < 0 {
		return ""
	}
	start := idx + len(pattern)

	// Find end of number (space, comma, or end)
	end := start
	for end < len(content) {
		c := content[end]
		if c == ',' || c == ' ' || c == '}' || c == ']' {
			break
		}
		end++
	}

	return strings.TrimSpace(content[start:end])
}

// getSelectedTrackIndex returns the index of the currently selected track from state
// Returns -1 if no selected track is found
// NOTE: REAPER supports multiple selected tracks, but we currently only return the first one.
// TODO: Handle multiple selected tracks in the future (e.g., return array or apply to all)
func (p *DSLParser) getSelectedTrackIndex() int {
	if p.state == nil {
		return -1
	}

	// Navigate to tracks array - state is wrapped as {"state": {...}}
	stateMap, ok := p.state["state"].(map[string]any)
	if !ok {
		return -1
	}

	tracks, ok := stateMap["tracks"].([]any)
	if !ok {
		return -1
	}

	// Find first selected track
	for i, track := range tracks {
		trackMap, ok := track.(map[string]any)
		if !ok {
			continue
		}
		if selected, ok := trackMap["selected"].(bool); ok && selected {
			return i
		}
	}

	return -1
}
