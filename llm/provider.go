package llm

import (
	"context"

	"github.com/Conceptual-Machines/magda-agents-go/models"
)

// Provider defines the interface for LLM providers (OpenAI, Gemini, etc.)
// All providers MUST support structured output (JSON Schema) for reliable response parsing
type Provider interface {
	// Generate creates a musical composition using the LLM with structured output
	// The provider MUST enforce the OutputSchema to ensure valid JSON responses
	Generate(ctx context.Context, request *GenerationRequest) (*GenerationResponse, error)

	// GenerateStream creates a musical composition with streaming updates and structured output
	GenerateStream(ctx context.Context, request *GenerationRequest, callback StreamCallback) (*GenerationResponse, error)

	// Name returns the provider name (e.g., "openai", "gemini")
	Name() string
}

// GenerationRequest contains all parameters needed for generation
type GenerationRequest struct {
	Model         string
	InputArray    []map[string]any
	ReasoningMode string
	SystemPrompt  string
	MCPConfig     *MCPConfig
	// Structured output schema - REQUIRED for reliable JSON parsing
	OutputSchema *OutputSchema
	// CFG Grammar for DSL output (alternative to JSON Schema)
	CFGGrammar *CFGConfig
}

// CFGConfig contains context-free grammar configuration
type CFGConfig struct {
	ToolName    string // Name of the tool that will receive the DSL output
	Description string // Description of what the tool does
	Grammar     string // Lark grammar definition
	Syntax      string // "lark" or "regex" (default: "lark")
}

// OutputSchema defines the expected JSON output structure
type OutputSchema struct {
	Name        string
	Description string
	Schema      map[string]any // JSON Schema object
}

// MCPConfig contains MCP server configuration
type MCPConfig struct {
	URL   string
	Label string
}

// GenerationResponse contains the result from the LLM
type GenerationResponse struct {
	OutputParsed struct {
		Choices []models.MusicalChoice `json:"choices"`
	} `json:"output_parsed"`
	RawOutput string   `json:"-"` // Raw JSON text output (for custom parsing)
	Usage     any      `json:"usage"`
	MCPUsed   bool     `json:"mcpUsed,omitempty"`
	MCPCalls  int      `json:"mcpCalls,omitempty"`
	MCPTools  []string `json:"mcpTools,omitempty"`
}

// StreamCallback is called for each streaming event
type StreamCallback func(event StreamEvent) error

// StreamEvent represents a server-sent event during streaming
type StreamEvent struct {
	Type    string                 `json:"type"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}
