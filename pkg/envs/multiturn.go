package envs

import (
	"context"
	"fmt"
	"strings"

	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// MultiTurnEnv implements multi-turn interactions
type MultiTurnEnv struct {
	*BaseEnvironment
	MaxTurns int
}

// MultiTurnEnvironment extends Environment with multi-turn specific methods
type MultiTurnEnvironment interface {
	Environment
	IsCompleted(ctx context.Context, messages []types.Message, state map[string]interface{}) bool
	EnvResponse(ctx context.Context, messages []types.Message, state map[string]interface{}) (types.Message, map[string]interface{}, error)
}

// NewMultiTurnEnv creates a new multi-turn environment
func NewMultiTurnEnv(config types.Config, maxTurns int) *MultiTurnEnv {
	if maxTurns <= 0 {
		maxTurns = 10
	}
	return &MultiTurnEnv{
		BaseEnvironment: NewBaseEnvironment(config),
		MaxTurns:        maxTurns,
	}
}

// BaseMultiTurnRollout implements the common rollout logic for multi-turn environments
func BaseMultiTurnRollout(ctx context.Context, env MultiTurnEnvironment, client types.Client, model string, prompt interface{}, answer string, samplingArgs types.SamplingArgs, maxTurns int) (*types.Rollout, error) {
	// Ensure prompt is a message list
	messages, ok := prompt.([]types.Message)
	if !ok {
		return nil, fmt.Errorf("multi-turn environment requires []types.Message prompt, got %T", prompt)
	}

	// Make a copy of messages to avoid modifying the original
	workingMessages := make([]types.Message, len(messages))
	copy(workingMessages, messages)

	// Initialize state
	state := map[string]interface{}{
		"answer": answer,
	}

	// Track completion messages
	completion := make([]types.Message, 0)
	turn := 0

	if maxTurns <= 0 {
		maxTurns = 10
	}

	// Run the multi-turn conversation
	for turn < maxTurns {
		// Check if already completed
		if env.IsCompleted(ctx, workingMessages, state) {
			break
		}

		// Get model response
		response, err := client.CreateChatCompletion(ctx, model, workingMessages, samplingArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to get model response at turn %d: %w", turn, err)
		}

		// Check for errors in response
		hasError := strings.HasPrefix(response, "[ERROR]")

		// Add assistant message
		assistantMsg := types.Message{
			Role:    "assistant",
			Content: response,
		}
		workingMessages = append(workingMessages, assistantMsg)
		completion = append(completion, assistantMsg)
		turn++

		// Check completion conditions
		if env.IsCompleted(ctx, workingMessages, state) || turn >= maxTurns || hasError {
			break
		}

		// Get environment response
		envMsg, newState, err := env.EnvResponse(ctx, workingMessages, state)
		if err != nil {
			return nil, fmt.Errorf("failed to get environment response at turn %d: %w", turn, err)
		}
		state = newState

		// Add environment message
		workingMessages = append(workingMessages, envMsg)
		completion = append(completion, envMsg)
	}

	// Extract final response for scoring
	finalResponse := ""
	if len(completion) > 0 {
		// Find last assistant message
		for i := len(completion) - 1; i >= 0; i-- {
			if completion[i].Role == "assistant" {
				finalResponse = completion[i].Content
				break
			}
		}
	}

	// For now, return basic rollout without parsing/scoring
	// The concrete implementation should handle parsing and scoring

	// Create rollout result
	rollout := &types.Rollout{
		Messages: workingMessages,
		Response: finalResponse,
		Score:    0.0, // Concrete implementations should handle scoring
	}

	return rollout, nil
}

// Example implementation of a simple dialog multi-turn environment
type DialogMultiTurnEnv struct {
	*MultiTurnEnv
	CompletionKeyword string
}

// NewDialogMultiTurnEnv creates a dialog-based multi-turn environment
func NewDialogMultiTurnEnv(config types.Config, maxTurns int, completionKeyword string) *DialogMultiTurnEnv {
	if completionKeyword == "" {
		completionKeyword = "DONE"
	}
	return &DialogMultiTurnEnv{
		MultiTurnEnv:      NewMultiTurnEnv(config, maxTurns),
		CompletionKeyword: completionKeyword,
	}
}

// IsCompleted checks if the dialog is completed
func (e *DialogMultiTurnEnv) IsCompleted(ctx context.Context, messages []types.Message, state map[string]interface{}) bool {
	if len(messages) == 0 {
		return false
	}
	
	// Check if last message contains completion keyword
	lastMsg := messages[len(messages)-1]
	return strings.Contains(lastMsg.Content, e.CompletionKeyword)
}

// EnvResponse generates a simple acknowledgment
func (e *DialogMultiTurnEnv) EnvResponse(ctx context.Context, messages []types.Message, state map[string]interface{}) (types.Message, map[string]interface{}, error) {
	// Simple acknowledgment
	msg := types.Message{
		Role:    "user",
		Content: "Please continue or say '" + e.CompletionKeyword + "' when finished.",
	}
	return msg, state, nil
}

// Rollout performs the multi-turn rollout
func (e *DialogMultiTurnEnv) Rollout(ctx context.Context, client types.Client, model string, prompt interface{}, answer string, samplingArgs types.SamplingArgs) (*types.Rollout, error) {
	rollout, err := BaseMultiTurnRollout(ctx, e, client, model, prompt, answer, samplingArgs, e.MaxTurns)
	if err != nil {
		return nil, err
	}

	// Apply parsing and scoring
	if e.parser != nil && rollout.Response != "" {
		parsed, err := e.parser.Parse(ctx, rollout.Response)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if e.rubric != nil {
			score, err := e.rubric.ComputeReward(ctx, parsed, answer)
			if err != nil {
				return nil, fmt.Errorf("failed to compute reward: %w", err)
			}
			rollout.Score = score
		}
	}

	return rollout, nil
}