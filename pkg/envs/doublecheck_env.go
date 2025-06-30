package envs

import (
	"context"
	"fmt"
	"strings"

	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/rubrics"
	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// DoubleCheckEnv implements a double-checking mechanism for answers
type DoubleCheckEnv struct {
	*MultiTurnEnv
}

// NewDoubleCheckEnv creates a new double-check environment
func NewDoubleCheckEnv(config types.Config) (*DoubleCheckEnv, error) {
	// Create parser for think/answer format
	parser, err := parsers.NewXMLParser([]interface{}{"think", "answer"}, "answer")
	if err != nil {
		return nil, err
	}

	env := &DoubleCheckEnv{
		MultiTurnEnv: NewMultiTurnEnv(config, 2), // Max 2 turns: answer + verification
	}

	// Set parser and rubric
	env.SetParser(parser)
	
	mathRubric, err := rubrics.NewMathRubric()
	if err != nil {
		return nil, err
	}
	env.SetRubric(mathRubric)

	return env, nil
}

// IsCompleted checks if double-checking is done
func (e *DoubleCheckEnv) IsCompleted(ctx context.Context, messages []types.Message, state map[string]interface{}) bool {
	// Complete after the double-check question has been asked
	if askedDoubleCheck, ok := state["asked_double_check"].(bool); ok && askedDoubleCheck {
		return true
	}

	// Also complete if we have a final answer after being asked "Are you sure?"
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" && strings.Contains(strings.ToLower(messages[i].Content), "are you sure") {
			// Check if there's an assistant response after this
			if i < len(messages)-1 && messages[i+1].Role == "assistant" {
				return true
			}
		}
	}

	return false
}

// EnvResponse provides the double-check prompt
func (e *DoubleCheckEnv) EnvResponse(ctx context.Context, messages []types.Message, state map[string]interface{}) (types.Message, map[string]interface{}, error) {
	if len(messages) == 0 {
		return types.Message{}, state, fmt.Errorf("no messages to process")
	}

	// Check if we've already asked the double-check question
	if askedDoubleCheck, ok := state["asked_double_check"].(bool); ok && askedDoubleCheck {
		return types.Message{}, state, fmt.Errorf("already asked double-check question")
	}

	// Get last assistant message
	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "assistant" {
		return types.Message{}, state, fmt.Errorf("last message must be from assistant")
	}

	// Parse the response to check if there's an answer
	if parser, ok := e.parser.(*parsers.XMLParser); ok {
		parsed, err := parser.ParseXML(lastMsg.Content, true)
		if err != nil || parsed.Fields["answer"] == "" {
			return types.Message{
				Role:    "user",
				Content: "Please provide your answer in the correct format with <think> and <answer> tags.",
			}, state, nil
		}
	}

	// Mark that we're asking the double-check question
	state["asked_double_check"] = true

	// Ask the double-check question
	return types.Message{
		Role:    "user",
		Content: "Are you sure? Double-check your answer.",
	}, state, nil
}

// Rollout performs the double-check environment rollout
func (e *DoubleCheckEnv) Rollout(ctx context.Context, client types.Client, model string, prompt interface{}, answer string, samplingArgs types.SamplingArgs) (*types.Rollout, error) {
	rollout, err := BaseMultiTurnRollout(ctx, e, client, model, prompt, answer, samplingArgs, e.MaxTurns)
	if err != nil {
		return nil, err
	}

	// Score based on the final answer (after double-checking)
	if e.parser != nil && len(rollout.Messages) > 0 {
		// Find the last assistant message
		var finalResponse string
		for i := len(rollout.Messages) - 1; i >= 0; i-- {
			if rollout.Messages[i].Role == "assistant" {
				finalResponse = rollout.Messages[i].Content
				break
			}
		}

		if finalResponse != "" {
			parsed, err := e.parser.Parse(ctx, finalResponse)
			if err != nil {
				return rollout, nil
			}

			if e.rubric != nil {
				score, err := e.rubric.ComputeReward(ctx, parsed, answer)
				if err != nil {
					return rollout, nil
				}
				rollout.Score = score
			}
		}
	}

	return rollout, nil
}