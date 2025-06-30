package envs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/rubrics"
	"github.com/rizome-dev/go-verifiers/pkg/types"
)

const (
	DefaultMaxConcurrent = 512
	DatasetMaxConcurrent = 32
)

// Environment is the base interface for all environments
type Environment interface {
	// Rollout performs a single environment rollout
	Rollout(ctx context.Context, client types.Client, model string, prompt interface{}, answer string, samplingArgs types.SamplingArgs) (*types.Rollout, error)
	
	// GetDataset returns the training dataset
	GetDataset(n int, seed int64) types.Dataset
	
	// GetEvalDataset returns the evaluation dataset
	GetEvalDataset(n int, seed int64) types.Dataset
	
	// GetRewardFuncs returns the reward functions for this environment
	GetRewardFuncs() []types.RewardFunc
	
	// GetRewardWeights returns the weights for reward functions
	GetRewardWeights() []float64
}

// BaseEnvironment provides common functionality for all environments
type BaseEnvironment struct {
	client        types.Client
	model         string
	dataset       types.Dataset
	evalDataset   types.Dataset
	systemPrompt  string
	fewShot       []types.Message
	parser        parsers.Parser
	rubric        rubrics.Rubric
	samplingArgs  types.SamplingArgs
	maxConcurrent int
	messageType   string
	logger        *slog.Logger
	mu            sync.RWMutex
}

// NewBaseEnvironment creates a new base environment
func NewBaseEnvironment(config types.Config) *BaseEnvironment {
	env := &BaseEnvironment{
		model:         config.Model,
		systemPrompt:  config.SystemPrompt,
		fewShot:       config.FewShot,
		samplingArgs:  config.SamplingArgs,
		maxConcurrent: config.MaxConcurrent,
		messageType:   config.MessageType,
		logger:        slog.Default().With("component", "environment"),
	}

	if env.maxConcurrent == 0 {
		env.maxConcurrent = DefaultMaxConcurrent
	}

	if env.messageType == "" {
		env.messageType = "chat"
	}

	// Set default sampling args
	if env.samplingArgs.N == 0 {
		env.samplingArgs.N = 1
	}
	if env.samplingArgs.ExtraBody == nil {
		env.samplingArgs.ExtraBody = map[string]interface{}{
			"skip_special_tokens":         false,
			"spaces_between_special_tokens": false,
		}
	}

	return env
}

// FormatPrompt formats a prompt with system prompt and few-shot examples
func (e *BaseEnvironment) FormatPrompt(prompt string) []types.Message {
	messages := make([]types.Message, 0)
	
	if e.systemPrompt != "" {
		messages = append(messages, types.Message{
			Role:    "system",
			Content: e.systemPrompt,
		})
	}
	
	if len(e.fewShot) > 0 {
		messages = append(messages, e.fewShot...)
	}
	
	messages = append(messages, types.Message{
		Role:    "user",
		Content: prompt,
	})
	
	return messages
}

// GetModelResponse gets a response from the model
func (e *BaseEnvironment) GetModelResponse(ctx context.Context, prompt interface{}, client types.Client, model string, samplingArgs types.SamplingArgs) (string, error) {
	switch e.messageType {
	case "chat":
		messages, ok := prompt.([]types.Message)
		if !ok {
			return "", fmt.Errorf("expected []types.Message for chat completion, got %T", prompt)
		}
		return client.CreateChatCompletion(ctx, model, messages, samplingArgs)
	case "completion":
		promptStr, ok := prompt.(string)
		if !ok {
			return "", fmt.Errorf("expected string for completion, got %T", prompt)
		}
		return client.CreateCompletion(ctx, model, promptStr, samplingArgs)
	default:
		return "", fmt.Errorf("unknown message type: %s", e.messageType)
	}
}

// GetDataset returns the training dataset
func (e *BaseEnvironment) GetDataset(n int, seed int64) types.Dataset {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if e.dataset == nil {
		return nil
	}
	
	if n > 0 && n < e.dataset.Len() {
		return e.dataset.Shuffle(seed).Select(makeRange(n))
	}
	return e.dataset
}

// GetEvalDataset returns the evaluation dataset
func (e *BaseEnvironment) GetEvalDataset(n int, seed int64) types.Dataset {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if e.evalDataset == nil {
		return nil
	}
	
	if n > 0 && n < e.evalDataset.Len() {
		return e.evalDataset.Shuffle(seed).Select(makeRange(n))
	}
	return e.evalDataset
}

// GetRewardFuncs returns the reward functions from the rubric
func (e *BaseEnvironment) GetRewardFuncs() []types.RewardFunc {
	if e.rubric != nil {
		return e.rubric.GetRewardFuncs()
	}
	return nil
}

// GetRewardWeights returns the reward weights from the rubric
func (e *BaseEnvironment) GetRewardWeights() []float64 {
	if e.rubric != nil {
		return e.rubric.GetRewardWeights()
	}
	return nil
}

// SetDataset sets the training dataset
func (e *BaseEnvironment) SetDataset(dataset types.Dataset) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.dataset = dataset
}

// SetEvalDataset sets the evaluation dataset
func (e *BaseEnvironment) SetEvalDataset(dataset types.Dataset) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.evalDataset = dataset
}

// SetParser sets the parser
func (e *BaseEnvironment) SetParser(parser parsers.Parser) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.parser = parser
}

// SetRubric sets the rubric
func (e *BaseEnvironment) SetRubric(rubric rubrics.Rubric) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rubric = rubric
}

// Helper function to create a range of indices
func makeRange(n int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	return indices
}