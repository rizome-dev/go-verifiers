package types

import (
	"context"
	"time"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SamplingArgs contains parameters for model sampling
type SamplingArgs struct {
	N                 int                    `json:"n,omitempty"`
	Temperature       float64                `json:"temperature,omitempty"`
	MaxTokens         int                    `json:"max_tokens,omitempty"`
	TopP              float64                `json:"top_p,omitempty"`
	FrequencyPenalty  float64                `json:"frequency_penalty,omitempty"`
	PresencePenalty   float64                `json:"presence_penalty,omitempty"`
	Stop              []string               `json:"stop,omitempty"`
	ExtraBody         map[string]interface{} `json:"extra_body,omitempty"`
}

// Dataset represents a collection of data items
type Dataset interface {
	Len() int
	Get(idx int) map[string]interface{}
	Shuffle(seed int64) Dataset
	Select(indices []int) Dataset
	Map(fn func(map[string]interface{}) map[string]interface{}) Dataset
}

// RewardFunc represents a function that calculates rewards
type RewardFunc func(context.Context, string, string) (float64, error)

// Rollout represents the result of an environment rollout
type Rollout struct {
	Messages []Message `json:"messages"`
	Response string    `json:"response"`
	Score    float64   `json:"score"`
}

// Config holds environment configuration
type Config struct {
	Model             string                 `json:"model"`
	SystemPrompt      string                 `json:"system_prompt,omitempty"`
	FewShot           []Message              `json:"few_shot,omitempty"`
	SamplingArgs      SamplingArgs           `json:"sampling_args"`
	MaxConcurrent     int                    `json:"max_concurrent"`
	MessageType       string                 `json:"message_type"`
	Timeout           time.Duration          `json:"timeout"`
	Extra             map[string]interface{} `json:"extra,omitempty"`
}

// Client represents an inference client interface
type Client interface {
	CreateChatCompletion(ctx context.Context, model string, messages []Message, args SamplingArgs) (string, error)
	CreateCompletion(ctx context.Context, model string, prompt string, args SamplingArgs) (string, error)
}