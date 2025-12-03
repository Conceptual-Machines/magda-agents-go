package config

// Config contains configuration for MAGDA agents
type Config struct {
	OpenAIAPIKey string // OpenAI API key for LLM provider
	GeminiAPIKey string // Google Gemini API key (optional)
	MCPServerURL string // MCP server URL (optional)
}
