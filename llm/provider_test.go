package llm

import (
	"context"
	"testing"

	"github.com/Conceptual-Machines/magda-agents-go/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProvider is a test implementation of the Provider interface
type MockProvider struct {
	name               string
	generateFunc       func(ctx context.Context, request *GenerationRequest) (*GenerationResponse, error)
	generateStreamFunc func(ctx context.Context, request *GenerationRequest, callback StreamCallback) (*GenerationResponse, error)
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Generate(ctx context.Context, request *GenerationRequest) (*GenerationResponse, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, request)
	}
	return &GenerationResponse{}, nil
}

func (m *MockProvider) GenerateStream(
	ctx context.Context, request *GenerationRequest, callback StreamCallback,
) (*GenerationResponse, error) {
	if m.generateStreamFunc != nil {
		return m.generateStreamFunc(ctx, request, callback)
	}
	return &GenerationResponse{}, nil
}

func TestProviderInterface(t *testing.T) {
	mock := &MockProvider{
		name: "mock",
	}

	assert.Equal(t, "mock", mock.Name())
}

func TestGenerationRequest(t *testing.T) {
	req := &GenerationRequest{
		Model:         "test-model",
		ReasoningMode: "medium",
		SystemPrompt:  "test prompt",
		InputArray: []map[string]any{
			{"role": "user", "content": "test"},
		},
		OutputSchema: &OutputSchema{
			Name:        "TestSchema",
			Description: "Test schema",
			Schema: map[string]any{
				"type": "object",
			},
		},
	}

	assert.Equal(t, "test-model", req.Model)
	assert.Equal(t, "medium", req.ReasoningMode)
	assert.NotNil(t, req.OutputSchema)
}

func TestGenerationResponse(t *testing.T) {
	resp := &GenerationResponse{
		MCPUsed:  true,
		MCPCalls: 3,
		MCPTools: []string{"search_compositions", "search_scales"},
	}

	assert.True(t, resp.MCPUsed)
	assert.Equal(t, 3, resp.MCPCalls)
	assert.Len(t, resp.MCPTools, 2)
}

func TestMockProviderGenerate(t *testing.T) {
	callCount := 0
	mock := &MockProvider{
		name: "test",
		generateFunc: func(_ context.Context, request *GenerationRequest) (*GenerationResponse, error) {
			callCount++
			require.Equal(t, "test-model", request.Model)
			return &GenerationResponse{
				OutputParsed: struct {
					Choices []models.MusicalChoice `json:"choices"`
				}{
					Choices: []models.MusicalChoice{
						{Description: "Test choice"},
					},
				},
			}, nil
		},
	}

	req := &GenerationRequest{
		Model: "test-model",
	}

	resp, err := mock.Generate(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 1, callCount)
	assert.Len(t, resp.OutputParsed.Choices, 1)
}

func TestStreamCallback(t *testing.T) {
	callCount := 0
	callback := func(event StreamEvent) error {
		callCount++
		assert.NotEmpty(t, event.Type)
		return nil
	}

	err := callback(StreamEvent{Type: "test", Message: "test message"})
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}
