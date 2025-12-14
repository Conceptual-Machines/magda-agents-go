package coordination

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	arranger "github.com/Conceptual-Machines/magda-agents-go/agents/arranger"
	"github.com/Conceptual-Machines/magda-agents-go/agents/daw"
	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/Conceptual-Machines/magda-agents-go/models"
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
	dawAgent          *daw.DawAgent
	arrangerAgent     ArrangerAgent // Will be set when we integrate
	llmProvider       llm.Provider
	dawKeywords       []string
	arrangerKeywords  []string
	keywordsLoaded    bool
	keywordsLoadMutex sync.Mutex
}

// ArrangerAgent interface for the arranger agent
// Uses the actual arranger agent's ArrangerResult type
type ArrangerAgent interface {
	GenerateActions(ctx context.Context, question string) (*arranger.ArrangerResult, error)
}

// ArrangerResult represents the output from the arranger agent (internal format)
type ArrangerResult struct {
	Actions []map[string]any `json:"actions"` // Parsed DSL actions
	Usage   any              `json:"usage"`
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

	// Initialize arranger agent (basic, no MCP for now)
	arrangerAgent := arranger.NewBasicArrangerAgent(cfg)

	o := &Orchestrator{
		dawAgent:      dawAgent,
		arrangerAgent: arrangerAgent,
		llmProvider:   llmProvider,
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

	// Step 1.5: Auto-enable DAW if arranger is needed but no tracks exist
	// This ensures track creation happens before musical content is added
	if needsArranger && !needsDAW {
		trackCount := getTrackCount(state)
		if trackCount == 0 {
			log.Printf("🔧 Auto-enabling DAW agent: Arranger needs a track but none exist")
			needsDAW = true
		}
	}

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
			// Call arranger agent with question
			result, err := o.arrangerAgent.GenerateActions(ctx, question)
			if err != nil {
				arrangerErr = fmt.Errorf("arranger agent: %w", err)
				return
			}
			// Use arranger result directly
			arrangerResult = &ArrangerResult{
				Actions: result.Actions,
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

// StreamActionCallback is called for each action found during streaming
type StreamActionCallback func(action map[string]any) error

// GenerateActionsStream coordinates agents and emits actions progressively via callback.
// This allows the UI to execute actions (create track, create clip) as they arrive,
// masking latency. MIDI notes are buffered until the clip is created, then emitted.
func (o *Orchestrator) GenerateActionsStream(
	ctx context.Context,
	question string,
	state map[string]any,
	callback StreamActionCallback,
) (*OrchestratorResult, error) {
	// Step 1: Detect which agents are needed
	needsDAW, needsArranger, err := o.DetectAgentsNeeded(ctx, question)
	if err != nil {
		log.Printf("⚠️ Detection error, defaulting to DAW: %v", err)
		needsDAW = true
		needsArranger = false
	}

	log.Printf("🔍 [Stream] Agent detection: DAW=%v, Arranger=%v", needsDAW, needsArranger)

	// Step 1.5: Auto-enable DAW if arranger is needed but no tracks exist
	if needsArranger && !needsDAW {
		trackCount := getTrackCount(state)
		if trackCount == 0 {
			log.Printf("🔧 [Stream] Auto-enabling DAW agent: Arranger needs a track but none exist")
			needsDAW = true
		}
	}

	// Track state for dependency resolution
	var (
		mu               sync.Mutex
		pendingNotes     []models.NoteEvent
		clipCreated      bool
		targetTrackIdx   int = 0
		allActions       []map[string]any
		dawComplete      bool
		arrangerComplete bool
	)

	// Helper to emit action via callback and track it
	emitAction := func(action map[string]any) error {
		mu.Lock()
		allActions = append(allActions, action)
		mu.Unlock()
		if callback != nil {
			return callback(action)
		}
		return nil
	}

	// Helper to check if we can emit add_midi (needs both clip and notes)
	tryEmitMidi := func() error {
		mu.Lock()
		defer mu.Unlock()

		if clipCreated && len(pendingNotes) > 0 && dawComplete && arrangerComplete {
			// Convert NoteEvents to map format
			notesArray := make([]map[string]any, len(pendingNotes))
			for i, note := range pendingNotes {
				notesArray[i] = map[string]any{
					"pitch":    note.MidiNoteNumber,
					"velocity": note.Velocity,
					"start":    note.StartBeats,
					"length":   note.DurationBeats,
				}
			}

			midiAction := map[string]any{
				"action": "add_midi",
				"track":  targetTrackIdx,
				"notes":  notesArray,
			}

			log.Printf("🎵 [Stream] Emitting add_midi with %d notes to track %d", len(pendingNotes), targetTrackIdx)
			allActions = append(allActions, midiAction)
			pendingNotes = nil // Clear buffer

			if callback != nil {
				// Unlock before callback to avoid deadlock
				mu.Unlock()
				err := callback(midiAction)
				mu.Lock()
				return err
			}
		}
		return nil
	}

	// Step 2: Launch agents
	var wg sync.WaitGroup
	var dawErr, arrangerErr error

	if needsDAW {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				mu.Lock()
				dawComplete = true
				mu.Unlock()
				_ = tryEmitMidi()
			}()

			// Use streaming DAW agent
			dawCallback := func(action map[string]any) error {
				actionType, _ := action["action"].(string)
				log.Printf("🎬 [Stream] DAW action: %s", actionType)

				// Track clip creation for dependency resolution
				if actionType == "create_clip_at_bar" || actionType == "new_clip" {
					mu.Lock()
					clipCreated = true
					if trackIdx, ok := action["track"].(int); ok {
						targetTrackIdx = trackIdx
					}
					mu.Unlock()
					log.Printf("📋 [Stream] Clip created on track %d", targetTrackIdx)
				}

				// Track the track index from create_track
				if actionType == "create_track" {
					if idx, ok := action["index"].(int); ok {
						mu.Lock()
						targetTrackIdx = idx
						mu.Unlock()
					}
				}

				// Emit immediately (create_track, create_clip, etc.)
				return emitAction(action)
			}

			_, err := o.dawAgent.GenerateActionsStream(ctx, question, state, dawCallback)
			if err != nil {
				dawErr = fmt.Errorf("daw agent stream: %w", err)
				log.Printf("❌ [Stream] DAW agent error: %v", err)
			}
		}()
	} else {
		mu.Lock()
		dawComplete = true
		mu.Unlock()
	}

	if needsArranger && o.arrangerAgent != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				mu.Lock()
				arrangerComplete = true
				mu.Unlock()
				_ = tryEmitMidi()
			}()

			result, err := o.arrangerAgent.GenerateActions(ctx, question)
			if err != nil {
				arrangerErr = fmt.Errorf("arranger agent: %w", err)
				log.Printf("❌ [Stream] Arranger agent error: %v", err)
				return
			}

			// Convert arranger actions to NoteEvents and buffer them
			currentBeat := 0.0
			for _, action := range result.Actions {
				noteEvents, err := arranger.ConvertArrangerActionToNoteEvents(action, currentBeat)
				if err != nil {
					log.Printf("⚠️ [Stream] Failed to convert arranger action: %v", err)
					continue
				}

				mu.Lock()
				pendingNotes = append(pendingNotes, noteEvents...)
				mu.Unlock()

				log.Printf("📦 [Stream] Buffered %d notes (total: %d)", len(noteEvents), len(pendingNotes))

				// Update beat position
				if length, ok := getFloat(action, "length"); ok {
					if repeat, ok := getInt(action, "repeat"); ok && repeat > 0 {
						currentBeat += length * float64(repeat)
					} else {
						currentBeat += length
					}
				}
			}
		}()
	} else {
		mu.Lock()
		arrangerComplete = true
		mu.Unlock()
	}

	// Wait for all agents
	wg.Wait()

	// Final check - emit any remaining MIDI
	_ = tryEmitMidi()

	// Handle errors
	if dawErr != nil && arrangerErr != nil {
		return nil, fmt.Errorf("both agents failed: %v, %v", dawErr, arrangerErr)
	}

	// Return all collected actions
	mu.Lock()
	result := &OrchestratorResult{
		Actions: allActions,
	}
	mu.Unlock()

	log.Printf("✅ [Stream] Complete: %d total actions emitted", len(result.Actions))
	return result, nil
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

	// If we only have arranger results (no DAW), convert arranger actions to NoteEvents
	// and create a simple DAW action structure
	if arrangerResult != nil && len(arrangerResult.Actions) > 0 && (dawResult == nil || len(dawResult.Actions) == 0) {
		// Convert arranger actions to NoteEvents
		allNoteEvents := []models.NoteEvent{}
		currentBeat := 0.0

		for _, action := range arrangerResult.Actions {
			noteEvents, err := arranger.ConvertArrangerActionToNoteEvents(action, currentBeat)
			if err != nil {
				log.Printf("⚠️ Failed to convert arranger action to NoteEvents: %v", err)
				continue
			}

			allNoteEvents = append(allNoteEvents, noteEvents...)

			// Update currentBeat for next action (sum of lengths)
			if length, ok := getFloat(action, "length"); ok {
				if repeat, ok := getInt(action, "repeat"); ok {
					currentBeat += length * float64(repeat)
				} else {
					currentBeat += length
				}
			}
		}

		// Create a DAW action to add MIDI notes
		if len(allNoteEvents) > 0 {
			// Convert models.NoteEvent to map format expected by DAW
			notesArray := make([]map[string]any, len(allNoteEvents))
			for i, note := range allNoteEvents {
				notesArray[i] = map[string]any{
					"pitch":    note.MidiNoteNumber,
					"velocity": note.Velocity,
					"start":    note.StartBeats,
					"length":   note.DurationBeats,
				}
			}

			// Create add_midi action
			midiAction := map[string]any{
				"action": "add_midi",
				"notes":  notesArray,
			}
			result.Actions = append(result.Actions, midiAction)
		}
	}

	// Add DAW actions
	if dawResult != nil {
		// If we have both DAW and arranger results, inject arranger NoteEvents into DAW actions
		if arrangerResult != nil && len(arrangerResult.Actions) > 0 {
			log.Printf("🔄 Merging %d DAW actions with %d arranger actions", len(dawResult.Actions), len(arrangerResult.Actions))

			// Convert all arranger actions to NoteEvents
			allNoteEvents := []models.NoteEvent{}
			currentBeat := 0.0

			for _, action := range arrangerResult.Actions {
				log.Printf("🎵 Converting arranger action: type=%v, chord=%v", action["type"], action["chord"])
				noteEvents, err := arranger.ConvertArrangerActionToNoteEvents(action, currentBeat)
				if err != nil {
					log.Printf("⚠️ Failed to convert arranger action to NoteEvents: %v", err)
					continue
				}

				log.Printf("✅ Converted to %d NoteEvents (starting at beat %.2f)", len(noteEvents), currentBeat)
				allNoteEvents = append(allNoteEvents, noteEvents...)

				// Update currentBeat for next action
				if length, ok := getFloat(action, "length"); ok {
					if repeat, ok := getInt(action, "repeat"); ok {
						currentBeat += length * float64(repeat)
					} else {
						currentBeat += length
					}
				}
			}

			log.Printf("📊 Total NoteEvents from arranger: %d", len(allNoteEvents))

			// Find add_midi actions and inject NoteEvents, or create one if needed
			hasMidiAction := false
			for _, action := range dawResult.Actions {
				actionType, ok := action["action"].(string)
				if !ok {
					result.Actions = append(result.Actions, action)
					continue
				}

				if actionType == "add_midi" {
					hasMidiAction = true
					// Convert models.NoteEvent to map format expected by DAW
					notesArray := make([]map[string]any, len(allNoteEvents))
					for i, note := range allNoteEvents {
						notesArray[i] = map[string]any{
							"pitch":    note.MidiNoteNumber,
							"velocity": note.Velocity,
							"start":    note.StartBeats,
							"length":   note.DurationBeats,
						}
					}
					action["notes"] = notesArray
					log.Printf("✅ Injected %d notes into add_midi action", len(notesArray))
				}
				result.Actions = append(result.Actions, action)
			}

			// If no add_midi action exists but we have NoteEvents, create one
			if !hasMidiAction && len(allNoteEvents) > 0 {
				// Find the last track index from DAW actions
				lastTrackIndex := -1
				for _, action := range dawResult.Actions {
					if track, ok := action["track"].(int); ok {
						lastTrackIndex = track
					} else if track, ok := action["index"].(int); ok {
						lastTrackIndex = track
					}
				}

				// Convert NoteEvents to map format
				notesArray := make([]map[string]any, len(allNoteEvents))
				for i, note := range allNoteEvents {
					notesArray[i] = map[string]any{
						"pitch":    note.MidiNoteNumber,
						"velocity": note.Velocity,
						"start":    note.StartBeats,
						"length":   note.DurationBeats,
					}
				}

				midiAction := map[string]any{
					"action": "add_midi",
					"notes":  notesArray,
				}
				if lastTrackIndex >= 0 {
					midiAction["track"] = lastTrackIndex
				}

				result.Actions = append(result.Actions, midiAction)
				log.Printf("✅ Created new add_midi action with %d notes (track=%d)", len(notesArray), lastTrackIndex)
			}
		} else {
			// No arranger results, just add DAW actions as-is
			result.Actions = append(result.Actions, dawResult.Actions...)
		}
		result.Usage = dawResult.Usage // TODO: merge usage from both agents
	}

	return result, nil
}

// Helper functions for type conversion
func getFloat(m map[string]any, key string) (float64, bool) {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val, true
		case int:
			return float64(val), true
		case int64:
			return float64(val), true
		}
	}
	return 0, false
}

func getInt(m map[string]any, key string) (int, bool) {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val, true
		case int64:
			return int(val), true
		case float64:
			return int(val), true
		}
	}
	return 0, false
}

// getTrackCount extracts the number of tracks from the REAPER state
func getTrackCount(state map[string]any) int {
	if state == nil {
		return 0
	}
	if tracks, ok := state["tracks"]; ok {
		if trackArr, ok := tracks.([]any); ok {
			return len(trackArr)
		}
		// Handle typed slice (e.g., from JSON unmarshaling)
		if trackArr, ok := tracks.([]map[string]any); ok {
			return len(trackArr)
		}
	}
	return 0
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
