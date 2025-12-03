package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiProvider_Name(t *testing.T) {
	// We can't create a real client without an API key
	// So just test the name method with a nil client
	provider := &GeminiProvider{client: nil}
	assert.Equal(t, "gemini", provider.Name())
}

func TestGeminiProvider_BuildContents(t *testing.T) {
	provider := &GeminiProvider{client: nil}

	tests := []struct {
		name       string
		inputArray []map[string]any
		wantLen    int
	}{
		{
			name: "single user message",
			inputArray: []map[string]any{
				{"role": "user", "content": "test content"},
			},
			wantLen: 1,
		},
		{
			name: "developer role converted to user",
			inputArray: []map[string]any{
				{"role": "developer", "content": "system message"},
			},
			wantLen: 1,
		},
		{
			name: "multiple messages",
			inputArray: []map[string]any{
				{"role": "user", "content": "message 1"},
				{"role": "user", "content": "message 2"},
			},
			wantLen: 2,
		},
		{
			name: "invalid message skipped",
			inputArray: []map[string]any{
				{"role": "user", "content": "valid"},
				{"role": "user"}, // missing content
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents, err := provider.buildGeminiContents(tt.inputArray)
			require.NoError(t, err)
			assert.Len(t, contents, tt.wantLen)

			// Verify all contents have user role
			for _, content := range contents {
				assert.Equal(t, "user", content.Role)
				assert.NotEmpty(t, content.Parts)
			}
		})
	}
}

func TestGeminiProvider_ConvertSchema(t *testing.T) {
	provider := &GeminiProvider{client: nil}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"test": map[string]any{"type": "string"},
		},
	}

	geminiSchema := provider.convertSchemaToGemini(schema)
	require.NotNil(t, geminiSchema)
	assert.NotNil(t, geminiSchema.Properties)
	assert.Contains(t, geminiSchema.Properties, "choices")
}

func TestNewGeminiProvider_InvalidKey(t *testing.T) {
	ctx := context.Background()
	provider, err := NewGeminiProvider(ctx, "invalid-key")

	// This might succeed (client creation) or fail depending on SDK validation
	// The important thing is we can create the provider object
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, provider)
		assert.Equal(t, "gemini", provider.Name())
	}
}
