package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rizome-dev/go-verifiers/pkg/envs"
	"github.com/rizome-dev/go-verifiers/pkg/inference"
	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/rubrics"
	"github.com/rizome-dev/go-verifiers/pkg/types"
)

func main() {
	// Create configuration
	config := types.Config{
		Model:         "gpt-4",
		SystemPrompt:  "You are a helpful assistant.",
		MessageType:   "chat",
		MaxConcurrent: 10,
		SamplingArgs: types.SamplingArgs{
			Temperature: 0.7,
			MaxTokens:   150,
		},
	}

	// Create single-turn environment
	env := envs.NewSingleTurnEnv(config)

	// Set parser and rubric
	env.SetParser(parsers.NewBaseParser())
	env.SetRubric(rubrics.NewBaseRubric())

	// Create HTTP client
	client := inference.NewHTTPClient("http://localhost:8000/v1", "your-api-key")

	// Example prompt
	prompt := env.FormatPrompt("What is 2 + 2?")
	answer := "4"

	// Perform rollout
	ctx := context.Background()
	rollout, err := env.Rollout(ctx, client, config.Model, prompt, answer, config.SamplingArgs)
	if err != nil {
		log.Fatalf("Rollout failed: %v", err)
	}

	// Display results
	fmt.Printf("Response: %s\n", rollout.Response)
	fmt.Printf("Score: %.2f\n", rollout.Score)

	// Example with multi-turn environment
	fmt.Println("\n--- Multi-turn Example ---")
	
	// Create a dialog multi-turn environment
	dialogEnv := envs.NewDialogMultiTurnEnv(config, 5, "DONE")
	dialogEnv.SetParser(parsers.NewLastLineParser())
	dialogEnv.SetRubric(rubrics.NewBaseRubric())

	// Multi-turn prompt
	mtPrompt := []types.Message{
		{Role: "user", Content: "Let's play 20 questions. Think of an animal and I'll guess it. Say 'DONE' when I guess correctly."},
	}

	mtRollout, err := dialogEnv.Rollout(ctx, client, config.Model, mtPrompt, "elephant", config.SamplingArgs)
	if err != nil {
		log.Fatalf("Multi-turn rollout failed: %v", err)
	}

	fmt.Printf("Multi-turn conversation had %d messages\n", len(mtRollout.Messages))
	fmt.Printf("Final score: %.2f\n", mtRollout.Score)
}

// Example of creating a custom rubric
func createCustomRubric() rubrics.Rubric {
	rubric := rubrics.NewMultiMetricRubric()

	// Add exact match metric
	exactMatch := func(ctx context.Context, parsed, groundTruth string) (float64, error) {
		if parsed == groundTruth {
			return 1.0, nil
		}
		return 0.0, nil
	}

	// Add partial match metric
	partialMatch := func(ctx context.Context, parsed, groundTruth string) (float64, error) {
		// Simple character overlap ratio
		matches := 0
		for i := 0; i < len(parsed) && i < len(groundTruth); i++ {
			if parsed[i] == groundTruth[i] {
				matches++
			}
		}
		maxLen := max(len(parsed), len(groundTruth))
		if maxLen == 0 {
			return 1.0, nil
		}
		return float64(matches) / float64(maxLen), nil
	}

	rubric.AddMetric("exact_match", exactMatch, 0.7)
	rubric.AddMetric("partial_match", partialMatch, 0.3)

	return rubric
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}