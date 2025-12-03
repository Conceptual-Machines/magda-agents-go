package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenAIProvider(t *testing.T) {
	provider := NewOpenAIProvider("test-api-key")
	require.NotNil(t, provider)
	assert.Equal(t, "openai", provider.Name())
	assert.NotNil(t, provider.client)
}

func TestOpenAIProvider_BuildRequestParams(t *testing.T) {
	provider := NewOpenAIProvider("test-key")

	tests := []struct {
		name    string
		request *GenerationRequest
		checks  func(t *testing.T, provider *OpenAIProvider, request *GenerationRequest)
	}{
		{
			name: "basic request with user message",
			request: &GenerationRequest{
				Model:         "gpt-5-mini",
				ReasoningMode: "medium",
				SystemPrompt:  "test system prompt",
				InputArray: []map[string]any{
					{"role": "user", "content": "test content"},
				},
			},
			checks: func(t *testing.T, provider *OpenAIProvider, request *GenerationRequest) {
				t.Helper()
				params := provider.buildRequestParams(request)
				assert.Equal(t, "gpt-5-mini", params.Model)
				assert.NotNil(t, params.Instructions.Value)
				assert.Equal(t, "test system prompt", params.Instructions.Value)
			},
		},
		{
			name: "request with developer role",
			request: &GenerationRequest{
				Model:         "gpt-5-mini",
				ReasoningMode: "low",
				SystemPrompt:  "test prompt",
				InputArray: []map[string]any{
					{"role": "developer", "content": "dev message"},
				},
			},
			checks: func(t *testing.T, provider *OpenAIProvider, request *GenerationRequest) {
				t.Helper()
				params := provider.buildRequestParams(request)
				assert.Equal(t, "gpt-5-mini", params.Model)
			},
		},
		{
			name: "request with MCP config",
			request: &GenerationRequest{
				Model:         "gpt-5-mini",
				ReasoningMode: "high",
				SystemPrompt:  "test prompt",
				InputArray: []map[string]any{
					{"role": "user", "content": "test"},
				},
				MCPConfig: &MCPConfig{
					URL:   "http://test-mcp-server",
					Label: "test-mcp",
				},
			},
			checks: func(t *testing.T, provider *OpenAIProvider, request *GenerationRequest) {
				t.Helper()
				params := provider.buildRequestParams(request)
				assert.NotEmpty(t, params.Tools)
				assert.Equal(t, 1, len(params.Tools))
			},
		},
		{
			name: "request with output schema",
			request: &GenerationRequest{
				Model:         "gpt-5-mini",
				ReasoningMode: "minimal",
				SystemPrompt:  "test prompt",
				InputArray: []map[string]any{
					{"role": "user", "content": "test"},
				},
				OutputSchema: &OutputSchema{
					Name:        "TestSchema",
					Description: "Test description",
					Schema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"test": map[string]any{"type": "string"},
						},
					},
				},
			},
			checks: func(t *testing.T, provider *OpenAIProvider, request *GenerationRequest) {
				t.Helper()
				params := provider.buildRequestParams(request)
				assert.NotNil(t, params.Text.Format)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.checks(t, provider, tt.request)
		})
	}
}

func TestOpenAIProvider_ReasoningModeMapping(t *testing.T) {
	provider := NewOpenAIProvider("test-key")

	tests := []struct {
		mode     string
		expected string
	}{
		{"minimal", "minimal"},
		{"min", "minimal"},
		{"low", "low"},
		{"medium", "medium"},
		{"med", "medium"},
		{"high", "high"},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			request := &GenerationRequest{
				Model:         "gpt-5-mini",
				ReasoningMode: tt.mode,
				SystemPrompt:  "test",
				InputArray: []map[string]any{
					{"role": "user", "content": "test"},
				},
			}
			params := provider.buildRequestParams(request)
			assert.NotNil(t, params.Reasoning.Effort)
		})
	}
}
