package coordination

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/Conceptual-Machines/magda-agents-go/agents/daw"
	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
)

// expandedKeywordsJSON contains the expanded keywords as embedded JSON
const expandedKeywordsJSON = `{
  "daw": [
    "track",
    "clip",
    "fx",
    "volume",
    "pan",
    "mute",
    "solo",
    "reaper",
    "daw",
    "create",
    "delete",
    "move",
    "select",
    "color",
    "rename",
    "add",
    "remove",
    "enable",
    "disable",
    "instrument",
    "plugin",
    "effect",
    "compressor",
    "reverb",
    "eq",
    "mix",
    "master",
    "bus",
    "return",
    "layer",
    "channel",
    "pista",
    "piste",
    "spur",
    "sp_track",
    "track layer",
    "music track",
    "audio track",
    "segment",
    "snippet",
    "clipa",
    "extrait",
    "ausschnitt",
    "frammento",
    "clipe",
    "クリップ",
    "kurippu",
    "effects",
    "processing",
    "efectos",
    "effets",
    "effekte",
    "effetti",
    "efeitos",
    "エフェクト",
    "efekuto",
    "loudness",
    "amplitude",
    "volumen",
    "lautstärke",
    "ボリューム",
    "borūmu",
    "panning",
    "stereo balance",
    "panoramización",
    "panoramique",
    "panorama",
    "panoramica",
    "パン",
    "silence",
    "cut",
    "silenciar",
    "couper",
    "stumm",
    "silenziare",
    "mudo",
    "ミュート",
    "myūto",
    "isolate",
    "one track",
    "ソロ",
    "soro",
    "digital audio workstation",
    "リーパー",
    "rīpā",
    "production software",
    "stazione audio digitale",
    "estação de áudio digital",
    "デジタルオーディオワークステーション",
    "dejitaru ōdio wākusuteeshon",
    "generate",
    "produce",
    "crear",
    "créer",
    "erstellen",
    "creare",
    "criar",
    "作成する",
    "sakusei suru",
    "erase",
    "eliminar",
    "supprimer",
    "löschen",
    "cancellare",
    "remover",
    "削除する",
    "sakujo suru",
    "shift",
    "drag",
    "mover",
    "déplacer",
    "bewegen",
    "spostare",
    "動かす",
    "ugokasu",
    "choose",
    "highlight",
    "seleccionar",
    "sélectionner",
    "auswählen",
    "selezionare",
    "selecionar",
    "選択する",
    "sentaku suru",
    "hue",
    "shade",
    "couleur",
    "farbe",
    "colore",
    "cor",
    "iro",
    "relabel",
    "change name",
    "renombrar",
    "renommer",
    "umbenennen",
    "rinominare",
    "renomear",
    "名前を変更する",
    "namae o henkō suru",
    "include",
    "insert",
    "agregar",
    "ajouter",
    "hinzufügen",
    "aggiungere",
    "adicionar",
    "追加する",
    "tsuika suru",
    "extract",
    "quitar",
    "retirer",
    "entfernen",
    "rimuovere",
    "取り除く",
    "torinozoku",
    "activate",
    "turn on",
    "habilitar",
    "activer",
    "aktivieren",
    "abilitare",
    "ativar",
    "有効にする",
    "yūkō ni suru",
    "deactivate",
    "turn off",
    "deshabilitar",
    "désactiver",
    "deaktivieren",
    "disabilitare",
    "desativar",
    "無効にする",
    "mukō ni suru",
    "tool",
    "device",
    "instrumento",
    "strumento",
    "楽器",
    "gakki",
    "extension",
    "add-on",
    "プラグイン",
    "puraguin",
    "result",
    "efecto",
    "effet",
    "effekt",
    "effetto",
    "efeito",
    "dynamic range compressor",
    "compression",
    "compresor",
    "compresseur",
    "kompressor",
    "compressore",
    "コンプレッサー",
    "konpuressā",
    "reverberation",
    "echo",
    "réverbération",
    "hall",
    "verb",
    "riverbero",
    "reverberação",
    "リバーブ",
    "ribābu",
    "equalization",
    "tone control",
    "ecualización",
    "égalisation",
    "equalizer",
    "equalizzazione",
    "equalização",
    "イコライザー",
    "ikoraizā",
    "blend",
    "combine",
    "mezclar",
    "mélanger",
    "mischen",
    "mescolare",
    "misturar",
    "ミックス",
    "mikkusu",
    "finalize",
    "masterizar",
    "masteriser",
    "mastering",
    "masterizzare",
    "マスタリング",
    "masutaringu",
    "channel strip",
    "signal route",
    "バス",
    "basu",
    "route",
    "forward",
    "enviar",
    "envoyer",
    "senden",
    "inviare",
    "送信する",
    "sōshin suru",
    "feedback",
    "retrace",
    "retorno",
    "retour",
    "rückkehr",
    "ritorno",
    "リターン",
    "ritān"
  ],
  "arranger": [
    "chord",
    "progression",
    "melody",
    "note",
    "notes",
    "i",
    "vi",
    "iv",
    "v",
    "ii",
    "iii",
    "vii",
    "roman",
    "scale",
    "harmony",
    "sequence",
    "pattern",
    "major",
    "minor",
    "diminished",
    "augmented",
    "triad",
    "seventh",
    "ninth",
    "arpeggio",
    "bassline",
    "riff",
    "hook",
    "groove",
    "lick",
    "phrase",
    "motif",
    "ostinato",
    "fill",
    "break",
    "c",
    "d",
    "e",
    "f",
    "g",
    "a",
    "b",
    "sharp",
    "flat",
    "natural",
    "pentatonic",
    "dorian",
    "mixolydian",
    "sus2",
    "sus4",
    "add9",
    "voicing",
    "acorde",
    "accord",
    "akkord",
    "accordo",
    "コード",
    "kōdo",
    "development",
    "progresión",
    "progresso",
    "進行",
    "shinkō",
    "tune",
    "theme",
    "melodía",
    "mélodie",
    "melodie",
    "melodia",
    "メロディ",
    "merodi",
    "pitch",
    "tone",
    "nota",
    "音符",
    "onpu",
    "pitches",
    "tones",
    "notas",
    "noten",
    "one",
    "tonic",
    "six",
    "submediant",
    "four",
    "subdominant",
    "five",
    "dominant",
    "two",
    "supertonic",
    "three",
    "mediant",
    "seven",
    "subtonic",
    "numeral",
    "notation",
    "romano",
    "numéral",
    "römisch",
    "ローマ数字",
    "rōma suūji",
    "range",
    "spectrum",
    "escala",
    "échelle",
    "skala",
    "scala",
    "スケール",
    "sukēru",
    "concord",
    "unity",
    "armonía",
    "harmonie",
    "armonia",
    "harmonia",
    "和声",
    "wasei",
    "order",
    "series",
    "secuencia",
    "séquence",
    "folge",
    "sequenza",
    "sequência",
    "配列",
    "hairetsu",
    "design",
    "arrangement",
    "patrón",
    "modèle",
    "muster",
    "schema",
    "padrão",
    "パターン",
    "patān",
    "happy",
    "bright",
    "mayor",
    "majeur",
    "dur",
    "maggiore",
    "maior",
    "メジャー",
    "mejā",
    "sad",
    "dark",
    "menor",
    "mineur",
    "moll",
    "minore",
    "マイナー",
    "mainā",
    "reduced",
    "lowered",
    "disminuido",
    "diminué",
    "vermindert",
    "diminuito",
    "diminuído",
    "減少",
    "genshō",
    "increased",
    "expanded",
    "aumentado",
    "augmenté",
    "erhöht",
    "aumentato",
    "増加",
    "zōka",
    "three-note chord",
    "threefold",
    "triada",
    "triade",
    "tríade",
    "トライアド",
    "toraiado",
    "7th chord",
    "dominant seventh",
    "séptima",
    "septième",
    "siebte",
    "settima",
    "sétima",
    "セブンス",
    "sebunsu",
    "9th chord",
    "ninth interval",
    "novena",
    "neuvième",
    "neunte",
    "nona",
    "ナインス",
    "nainsu",
    "broken chord",
    "arpeggiated chord",
    "arpegio",
    "arpège",
    "アルペジオ",
    "arpejio",
    "bass part",
    "low line",
    "línea de bajo",
    "ligne de basse",
    "basslinie",
    "linea di basso",
    "linha de baixo",
    "ベースライン",
    "bēsura",
    "repeated phrase",
    "リフ",
    "rifu",
    "catchphrase",
    "catchy part",
    "gancho",
    "crochet",
    "gancio",
    "フック",
    "hukku",
    "rhythm",
    "feel",
    "rythme",
    "グルーヴ",
    "gurūvu",
    "short solo",
    "リック",
    "rikku",
    "segment",
    "expression",
    "frase",
    "フレーズ",
    "furēzu",
    "idea",
    "motivo",
    "motiv",
    "モチーフ",
    "motīfu",
    "repeated pattern",
    "loop",
    "オスティナート",
    "osutināto",
    "decoration",
    "embellishment",
    "relleno",
    "remplissage",
    "füller",
    "enchimento",
    "フィル",
    "firu",
    "pause",
    "interruption",
    "descanso",
    "interruzione",
    "quebra",
    "ブレイク",
    "bureiku",
    "c note",
    "c major",
    "do",
    "dó",
    "d note",
    "d major",
    "re",
    "ré",
    "e note",
    "e major",
    "mi",
    "f note",
    "f major",
    "fa",
    "ファ",
    "g note",
    "g major",
    "sol",
    "so",
    "a note",
    "a major",
    "la",
    "ra",
    "b note",
    "b major",
    "si",
    "shi",
    "raised",
    "crossed",
    "sostenido",
    "dièse",
    "kreuz",
    "diesis",
    "sustenido",
    "シャープ",
    "shāpu",
    "bemol",
    "bémol",
    "bemolle",
    "フラット",
    "furatto",
    "unmodified",
    "plain",
    "naturel",
    "natürlich",
    "naturale",
    "ナチュラル",
    "nachuraru",
    "five-note scale",
    "five tones",
    "pentatónico",
    "pentatonique",
    "pentatonisch",
    "pentatonico",
    "pentatônico",
    "ペンタトニック",
    "pentatonikku",
    "mode",
    "dorian scale",
    "dórico",
    "dorien",
    "dorisch",
    "dorico",
    "ドリアン",
    "mixolydian scale",
    "mixolidio",
    "mixolydien",
    "mixolydisch",
    "mixolídio",
    "ミクソリディアン",
    "mikusoridian",
    "suspended second",
    "add2",
    "サス2",
    "sasu2",
    "suspended fourth",
    "add4",
    "サス4",
    "sasu4",
    "added ninth",
    "9th added",
    "アッド9",
    "addo9"
  ]
}`

// Orchestrator coordinates multiple agents (DAW + Arranger) running in parallel
type Orchestrator struct {
	dawAgent           *daw.DawAgent
	arrangerAgent      ArrangerAgent // Will be set when we integrate
	llmProvider        llm.Provider
	dawKeywords        []string
	arrangerKeywords   []string
	keywordsLoaded    bool
	keywordsLoadMutex  sync.Mutex
}

// ArrangerAgent interface for the arranger agent (to be implemented/integrated)
type ArrangerAgent interface {
	Generate(ctx context.Context, model string, inputArray []map[string]any, reasoningMode, outputFormat string) (*ArrangerGenerationResult, error)
}

// ArrangerGenerationResult matches the arranger agent's GenerationResult
type ArrangerGenerationResult struct {
	OutputParsed struct {
		Choices []MusicalChoice `json:"choices"`
	} `json:"output_parsed"`
	Usage    any      `json:"usage"`
	MCPUsed  bool     `json:"mcpUsed,omitempty"`
	MCPCalls int      `json:"mcpCalls,omitempty"`
	MCPTools []string `json:"mcpTools,omitempty"`
}

// ArrangerResult represents the output from the arranger agent
type ArrangerResult struct {
	Choices []MusicalChoice `json:"choices"`
	Usage   any             `json:"usage"`
}

// MusicalChoice represents a musical composition choice
type MusicalChoice struct {
	Description string      `json:"description"`
	Notes       []NoteEvent `json:"notes"`
}

// NoteEvent represents a MIDI note event
type NoteEvent struct {
	MIDINoteNumber int     `json:"midiNoteNumber"`
	Velocity       int     `json:"velocity"`
	StartBeats     float64 `json:"startBeats"`
	LengthBeats    float64 `json:"lengthBeats"`
}

// OrchestratorResult combines results from all agents
type OrchestratorResult struct {
	Actions []map[string]any `json:"actions"`
	Usage   any              `json:"usage"`
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(cfg *config.Config) *Orchestrator {
	dawAgent := daw.NewDawAgent(cfg)
	llmProvider := llm.NewOpenAIProvider(cfg.OpenAIAPIKey)

	o := &Orchestrator{
		dawAgent:    dawAgent,
		llmProvider: llmProvider,
		// arrangerAgent will be set when we integrate
	}

	// Load expanded keywords (lazy load on first use if file not found)
	o.loadKeywords()

	return o
}

// GenerateActions coordinates parallel agent execution and merges results
func (o *Orchestrator) GenerateActions(ctx context.Context, question string, state map[string]any) (*OrchestratorResult, error) {
	// Step 1: Detect which agents are needed
	needsDAW, needsArranger, err := o.DetectAgentsNeeded(ctx, question)
	if err != nil {
		log.Printf("⚠️ Detection error, defaulting to DAW: %v", err)
		needsDAW = true
		needsArranger = false
	}

	log.Printf("🔍 Agent detection: DAW=%v, Arranger=%v", needsDAW, needsArranger)

	// DetectAgentsNeeded already handles LLM validation when no keywords are found
	// If it returns an error, the request is out of scope
	if err != nil {
		return nil, err
	}

	// Step 2: Launch only needed agents in parallel
	var wg sync.WaitGroup
	var dawResult *daw.DawResult
	var arrangerResult *ArrangerResult
	var dawErr, arrangerErr error

	if needsDAW {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := o.dawAgent.GenerateActions(ctx, question, state)
			if err != nil {
				dawErr = fmt.Errorf("daw agent: %w", err)
				return
			}
			dawResult = result
		}()
	}

	if needsArranger && o.arrangerAgent != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Build arranger input from question
			inputArray := o.buildArrangerInput(question)
			result, err := o.arrangerAgent.Generate(ctx, "gpt-5.1", inputArray, "none", "dsl")
			if err != nil {
				arrangerErr = fmt.Errorf("arranger agent: %w", err)
				return
			}
			// Convert arranger result to our format
			arrangerResult = &ArrangerResult{
				Choices: result.OutputParsed.Choices,
				Usage:   result.Usage,
			}
		}()
	}

	// Wait for all active agents to complete
	wg.Wait()

	// Step 3: Handle errors (partial results OK)
	if dawErr != nil && arrangerErr != nil {
		return nil, fmt.Errorf("both agents failed: %v, %v", dawErr, arrangerErr)
	}
	if dawErr != nil && needsDAW {
		log.Printf("⚠️ DAW agent failed: %v", dawErr)
		// Continue with Arranger if available
	}
	if arrangerErr != nil && needsArranger {
		log.Printf("⚠️ Arranger agent failed: %v", arrangerErr)
		// Continue with DAW if available
	}

	// Step 4: Merge results
	return o.mergeResults(dawResult, arrangerResult)
}

// DetectAgentsNeeded uses hybrid keywords + LLM to detect which agents are needed
func (o *Orchestrator) DetectAgentsNeeded(ctx context.Context, question string) (needsDAW bool, needsArranger bool, err error) {
	// Fast path: Enhanced keyword matching (<1ms)
	needsDAW, needsArranger = o.detectAgentsNeededKeywords(question)

	// If keywords found, return immediately (no validation needed)
	if needsDAW || needsArranger {
		// If only one detected but question seems musical, double-check with LLM
		if (needsDAW && !needsArranger) && o.looksMusical(question) {
			llmDAW, llmArranger, err := o.detectAgentsNeededLLM(ctx, question)
			if err == nil {
				needsDAW = llmDAW
				needsArranger = llmArranger
			}
		}
		return needsDAW, needsArranger, nil
	}

	// If no keywords found, use LLM to validate scope
	llmDAW, llmArranger, err := o.detectAgentsNeededLLM(ctx, question)
	if err != nil {
		return false, false, fmt.Errorf("LLM classification failed: %w", err)
	}
	
	needsDAW = llmDAW
	needsArranger = llmArranger
	
	// Runtime (orchestrator) checks: if LLM returns both false, the request is out of scope
	if !needsDAW && !needsArranger {
		return false, false, fmt.Errorf("request is out of scope: no agents can handle this request")
	}
	
	return needsDAW, needsArranger, nil
}

// loadKeywords loads expanded keywords from embedded JSON (with fallback to hardcoded)
func (o *Orchestrator) loadKeywords() {
	o.keywordsLoadMutex.Lock()
	defer o.keywordsLoadMutex.Unlock()

	if o.keywordsLoaded {
		return
	}

	var keywords struct {
		DAW      []string `json:"daw"`
		Arranger []string `json:"arranger"`
	}

	if err := json.Unmarshal([]byte(expandedKeywordsJSON), &keywords); err != nil {
		log.Printf("⚠️ Failed to parse embedded expanded_keywords.json: %v, using hardcoded keywords", err)
		o.loadDefaultKeywords()
		o.keywordsLoaded = true
		return
	}

	o.dawKeywords = keywords.DAW
	o.arrangerKeywords = keywords.Arranger
	o.keywordsLoaded = true
	log.Printf("✅ Loaded %d DAW keywords and %d Arranger keywords from embedded data",
		len(o.dawKeywords), len(o.arrangerKeywords))
}

// loadDefaultKeywords sets fallback hardcoded keywords
func (o *Orchestrator) loadDefaultKeywords() {
	o.dawKeywords = []string{
		"track", "clip", "fx", "volume", "pan", "mute", "solo",
		"reaper", "daw", "instrument", "plugin", "effect",
		"compressor", "reverb", "eq", "mix", "master", "bus", "return",
		"create", "delete", "move", "select", "color", "rename",
		"add", "remove", "enable", "disable", "set",
	}

	o.arrangerKeywords = []string{
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
	}
}

// detectAgentsNeededKeywords does keyword matching without defaulting to DAW
// This allows the orchestrator to validate scope when no keywords are found
func (o *Orchestrator) detectAgentsNeededKeywords(question string) (needsDAW bool, needsArranger bool) {
	// Ensure keywords are loaded
	if !o.keywordsLoaded {
		o.loadKeywords()
	}

	questionLower := strings.ToLower(question)

	// Filter out single-character keywords to avoid false positives (e.g., "a" matching in "bake me a cake")
	dawKeywordsFiltered := o.filterSingleCharKeywords(o.dawKeywords)
	arrangerKeywordsFiltered := o.filterSingleCharKeywords(o.arrangerKeywords)

	// Check for DAW operations (independent check)
	needsDAW = containsAny(questionLower, dawKeywordsFiltered)
	
	// Check for musical content (independent check - can be true alongside DAW)
	needsArranger = containsAny(questionLower, arrangerKeywordsFiltered)
	
	// Both can be true! Example: "add a chord progression to track 1"
	// - "add", "track" → needsDAW = true
	// - "chord", "progression" → needsArranger = true

	return needsDAW, needsArranger
}

// filterSingleCharKeywords removes single-character keywords to avoid false positives
func (o *Orchestrator) filterSingleCharKeywords(keywords []string) []string {
	filtered := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		// Only include keywords with 2+ characters
		if len(strings.TrimSpace(kw)) > 1 {
			filtered = append(filtered, kw)
		}
	}
	return filtered
}

// looksMusical checks if question contains musical terms that might not be in keywords
func (o *Orchestrator) looksMusical(question string) bool {
	musicalTerms := []string{
		"arpeggio", "bassline", "riff", "hook", "groove", "lick",
		"phrase", "motif", "vibe", "groovy", "punchy", "warm",
		"musical", "composition", "arrangement",
	}
	questionLower := strings.ToLower(question)
	return containsAny(questionLower, musicalTerms)
}

// detectAgentsNeededLLM uses LLM to classify the request (fallback when keywords detect nothing)
// Returns both false if the request is out of scope (e.g., "bake me a cake")
// Only returns error for LLM failures (API errors, parsing errors), NOT for out-of-scope requests
func (o *Orchestrator) detectAgentsNeededLLM(ctx context.Context, question string) (needsDAW bool, needsArranger bool, err error) {
	prompt := fmt.Sprintf(`Classify this music production request. Return JSON:
{
  "needsDAW": true/false,  // REAPER operations: tracks, clips, FX, volume, pan, mute, solo, etc.
  "needsArranger": true/false  // Musical content: chords, melodies, notes, arpeggios, basslines, riffs, etc.
}

If the request is completely out of scope (e.g., "bake me a cake", "send an email", "what's the weather"), return both false.

Request: "%s"`, question)

	// Use a small, fast model for classification
	request := &llm.GenerationRequest{
		Model:         "gpt-4.1-mini", // Fast and cheap for classification
		InputArray:    []map[string]any{{"role": "user", "content": prompt}},
		ReasoningMode: "none",
		OutputSchema: &llm.OutputSchema{
			Name:        "AgentClassification",
			Description: "Classification of which agents are needed",
			Schema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"needsDAW": map[string]any{
						"type": "boolean",
					},
					"needsArranger": map[string]any{
						"type": "boolean",
					},
				},
				"required": []string{"needsDAW", "needsArranger"},
			},
		},
	}

	resp, err := o.llmProvider.Generate(ctx, request)
	if err != nil {
		return false, false, fmt.Errorf("LLM classification failed: %w", err)
	}

	// Parse response from RawOutput (JSON Schema returns structured JSON)
	// For now, parse from RawOutput or use a simple heuristic
	// TODO: Properly parse JSON Schema response
	result := struct {
		NeedsDAW      bool `json:"needsDAW"`
		NeedsArranger bool `json:"needsArranger"`
	}{
		NeedsDAW:      false, // No default - let LLM decide
		NeedsArranger: false, // No default - let LLM decide
	}

	// Try to parse from RawOutput if available
	if resp.RawOutput != "" {
		// Parse JSON from RawOutput
		if err := json.Unmarshal([]byte(resp.RawOutput), &result); err != nil {
			log.Printf("⚠️ Failed to parse LLM classification JSON: %v, raw: %s", err, resp.RawOutput)
			// If parsing fails, return error (don't fallback to keywords - we're here because keywords found nothing)
			return false, false, fmt.Errorf("failed to parse LLM classification: %w", err)
		}
	}

	// Return LLM's decision - if both are false, caller will treat as out of scope
	return result.NeedsDAW, result.NeedsArranger, nil
}

// mergeResults combines DAW and Arranger results
func (o *Orchestrator) mergeResults(dawResult *daw.DawResult, arrangerResult *ArrangerResult) (*OrchestratorResult, error) {
	result := &OrchestratorResult{
		Actions: []map[string]any{},
	}

	// Add DAW actions
	if dawResult != nil {
		result.Actions = append(result.Actions, dawResult.Actions...)
		result.Usage = dawResult.Usage // TODO: merge usage from both agents
	}

	// TODO: Inject Arranger musical content into DAW actions (placeholder resolution)
	// For now, just return DAW actions
	// Phase 2 will implement placeholder resolution

	return result, nil
}

// buildArrangerInput converts question to arranger agent input format
func (o *Orchestrator) buildArrangerInput(question string) []map[string]any {
	// Simple conversion - arranger agent expects array of message maps
	return []map[string]any{
		{
			"role":    "user",
			"content": question,
		},
	}
}

// containsAny checks if text contains any of the keywords (case-insensitive)
func containsAny(text string, keywords []string) bool {
	textLower := strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}


