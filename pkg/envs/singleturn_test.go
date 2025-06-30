package envs

import (
	"context"
	"testing"

	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/rubrics"
	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// MockClient implements types.Client for testing
type MockClient struct {
	Response string
	Error    error
}

func (m *MockClient) CreateChatCompletion(ctx context.Context, model string, messages []types.Message, args types.SamplingArgs) (string, error) {
	if m.Error != nil {
		return "", m.Error
	}
	return m.Response, nil
}

func (m *MockClient) CreateCompletion(ctx context.Context, model string, prompt string, args types.SamplingArgs) (string, error) {
	if m.Error != nil {
		return "", m.Error
	}
	return m.Response, nil
}

func TestSingleTurnEnv_Rollout(t *testing.T) {
	config := types.Config{
		Model:        "test-model",
		SystemPrompt: "You are a test assistant.",
		MessageType:  "chat",
		SamplingArgs: types.SamplingArgs{
			Temperature: 0.7,
		},
	}

	env := NewSingleTurnEnv(config)
	env.SetParser(parsers.NewBaseParser())
	env.SetRubric(rubrics.NewBaseRubric())

	mockClient := &MockClient{
		Response: "4",
	}

	ctx := context.Background()
	prompt := env.FormatPrompt("What is 2 + 2?")
	answer := "4"

	rollout, err := env.Rollout(ctx, mockClient, config.Model, prompt, answer, config.SamplingArgs)
	if err != nil {
		t.Fatalf("Rollout failed: %v", err)
	}

	if rollout.Response != "4" {
		t.Errorf("Expected response '4', got '%s'", rollout.Response)
	}

	if rollout.Score != 1.0 {
		t.Errorf("Expected score 1.0, got %.2f", rollout.Score)
	}

	if len(rollout.Messages) != 3 { // system, user, assistant
		t.Errorf("Expected 3 messages, got %d", len(rollout.Messages))
	}
}

func TestSingleTurnEnv_CompletionMode(t *testing.T) {
	config := types.Config{
		Model:       "test-model",
		MessageType: "completion",
	}

	env := NewSingleTurnCompletionEnv(config)
	env.SetParser(parsers.NewBaseParser())
	env.SetRubric(rubrics.NewBaseRubric())

	mockClient := &MockClient{
		Response: "The answer is 4",
	}

	ctx := context.Background()
	prompt := "What is 2 + 2? The answer is"
	answer := "The answer is 4"

	rollout, err := env.Rollout(ctx, mockClient, config.Model, prompt, answer, config.SamplingArgs)
	if err != nil {
		t.Fatalf("Rollout failed: %v", err)
	}

	if rollout.Response != "The answer is 4" {
		t.Errorf("Expected response 'The answer is 4', got '%s'", rollout.Response)
	}

	if rollout.Score != 1.0 {
		t.Errorf("Expected score 1.0, got %.2f", rollout.Score)
	}
}

func TestBaseEnvironment_FormatPrompt(t *testing.T) {
	config := types.Config{
		SystemPrompt: "System message",
		FewShot: []types.Message{
			{Role: "user", Content: "Example input"},
			{Role: "assistant", Content: "Example output"},
		},
	}

	env := NewBaseEnvironment(config)
	messages := env.FormatPrompt("Test prompt")

	if len(messages) != 4 { // system + 2 few-shot + user
		t.Fatalf("Expected 4 messages, got %d", len(messages))
	}

	if messages[0].Role != "system" || messages[0].Content != "System message" {
		t.Errorf("Incorrect system message")
	}

	if messages[3].Role != "user" || messages[3].Content != "Test prompt" {
		t.Errorf("Incorrect user message")
	}
}