package envs

import (
	"context"
	"fmt"

	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// SingleTurnEnv implements single-turn interactions (chat or completion)
type SingleTurnEnv struct {
	*BaseEnvironment
}

// NewSingleTurnEnv creates a new single-turn environment
func NewSingleTurnEnv(config types.Config) *SingleTurnEnv {
	return &SingleTurnEnv{
		BaseEnvironment: NewBaseEnvironment(config),
	}
}

// Rollout performs a single-turn rollout
func (e *SingleTurnEnv) Rollout(ctx context.Context, client types.Client, model string, prompt interface{}, answer string, samplingArgs types.SamplingArgs) (*types.Rollout, error) {
	// Get model response
	response, err := e.GetModelResponse(ctx, prompt, client, model, samplingArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to get model response: %w", err)
	}

	// Parse the response
	parsed := response
	if e.parser != nil {
		parsed, err = e.parser.Parse(ctx, response)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	// Compute reward
	score := 0.0
	if e.rubric != nil {
		score, err = e.rubric.ComputeReward(ctx, parsed, answer)
		if err != nil {
			return nil, fmt.Errorf("failed to compute reward: %w", err)
		}
	}

	// Create rollout result
	rollout := &types.Rollout{
		Response: response,
		Score:    score,
	}

	// Add messages if chat mode
	if e.messageType == "chat" {
		messages, ok := prompt.([]types.Message)
		if ok {
			rollout.Messages = append(messages, types.Message{
				Role:    "assistant",
				Content: response,
			})
		}
	}

	return rollout, nil
}

// SingleTurnCompletionEnv is a convenience type for completion-mode single turn
type SingleTurnCompletionEnv struct {
	*SingleTurnEnv
}

// NewSingleTurnCompletionEnv creates a completion-mode single turn environment
func NewSingleTurnCompletionEnv(config types.Config) *SingleTurnCompletionEnv {
	config.MessageType = "completion"
	return &SingleTurnCompletionEnv{
		SingleTurnEnv: NewSingleTurnEnv(config),
	}
}

// SingleTurnChatEnv is a convenience type for chat-mode single turn
type SingleTurnChatEnv struct {
	*SingleTurnEnv
}

// NewSingleTurnChatEnv creates a chat-mode single turn environment
func NewSingleTurnChatEnv(config types.Config) *SingleTurnChatEnv {
	config.MessageType = "chat"
	return &SingleTurnChatEnv{
		SingleTurnEnv: NewSingleTurnEnv(config),
	}
}