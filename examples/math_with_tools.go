package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rizome-dev/go-verifiers/pkg/envs"
	"github.com/rizome-dev/go-verifiers/pkg/inference"
	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/rubrics"
	"github.com/rizome-dev/go-verifiers/pkg/tools"
	"github.com/rizome-dev/go-verifiers/pkg/types"
	"github.com/rizome-dev/go-verifiers/pkg/utils"
)

func main() {
	// Example 1: Math problem with XML formatting
	fmt.Println("=== Example 1: Math Problem with XML Formatting ===")
	runMathExample()

	fmt.Println("\n=== Example 2: Tool-based Calculation ===")
	runToolExample()

	fmt.Println("\n=== Example 3: Batch Processing ===")
	runBatchExample()

	fmt.Println("\n=== Example 4: Multi-turn Dialog ===")
	runMultiTurnExample()
}

func runMathExample() {
	// Create configuration
	config := types.Config{
		Model: "gpt-4",
		SystemPrompt: `You are a math tutor. For each problem:
1. First explain your thinking process in a <think> tag
2. Then provide the final answer in an <answer> tag

Example format:
<think>
To solve this problem, I need to...
</think>
<answer>
42
</answer>`,
		MessageType:   "chat",
		MaxConcurrent: 10,
		SamplingArgs: types.SamplingArgs{
			Temperature: 0.7,
			MaxTokens:   500,
		},
	}

	// Create single-turn environment
	env := envs.NewSingleTurnEnv(config)

	// Set XML parser
	xmlParser, err := parsers.NewXMLParser([]interface{}{"think", "answer"}, "answer")
	if err != nil {
		log.Fatalf("Failed to create XML parser: %v", err)
	}
	env.SetParser(xmlParser)

	// Set math rubric
	mathRubric, err := rubrics.NewMathRubric()
	if err != nil {
		log.Fatalf("Failed to create math rubric: %v", err)
	}
	env.SetRubric(mathRubric)

	// Create HTTP client
	client := inference.NewHTTPClient("http://localhost:8000/v1", "your-api-key")

	// Create a math problem
	prompt := env.FormatPrompt("What is the sum of the first 10 positive integers? Show your work.")
	answer := "55"

	// Perform rollout
	ctx := context.Background()
	rollout, err := env.Rollout(ctx, client, config.Model, prompt, answer, config.SamplingArgs)
	if err != nil {
		log.Printf("Rollout failed: %v", err)
		return
	}

	fmt.Printf("Response:\n%s\n", rollout.Response)
	fmt.Printf("Score: %.2f\n", rollout.Score)
}

func runToolExample() {
	// Create calculator tool
	calc := tools.NewCalculator()

	// Create tool environment configuration
	config := types.Config{
		Model:        "gpt-4",
		MessageType:  "chat",
		SamplingArgs: types.SamplingArgs{
			Temperature: 0.7,
			MaxTokens:   500,
		},
	}

	// Create tool environment
	toolEnv, err := envs.NewToolEnv(config, []tools.Tool{calc}, 5)
	if err != nil {
		log.Fatalf("Failed to create tool environment: %v", err)
	}

	// Create HTTP client
	client := inference.NewHTTPClient("http://localhost:8000/v1", "your-api-key")

	// Create a problem that requires calculation
	messages := []types.Message{
		{
			Role:    "user",
			Content: "Calculate the value of sin(π/2) + cos(0) + sqrt(16)",
		},
	}

	answer := "6" // sin(π/2) = 1, cos(0) = 1, sqrt(16) = 4, total = 6

	// Perform rollout
	ctx := context.Background()
	rollout, err := toolEnv.Rollout(ctx, client, config.Model, messages, answer, config.SamplingArgs)
	if err != nil {
		log.Printf("Tool rollout failed: %v", err)
		return
	}

	fmt.Printf("Conversation:\n")
	for _, msg := range rollout.Messages {
		fmt.Printf("[%s]: %s\n\n", msg.Role, msg.Content)
	}
	fmt.Printf("Final Score: %.2f\n", rollout.Score)
}

func runBatchExample() {
	// Create a dataset of math problems
	dataset := types.DatasetUtils{}.LoadFromQuestionAnswer([]struct{ Question, Answer string }{
		{"What is 2 + 2?", "4"},
		{"What is 5 * 6?", "30"},
		{"What is 10 - 3?", "7"},
		{"What is 20 / 4?", "5"},
	})

	// Create processor for concurrent evaluation
	processor := utils.NewBatchProcessor[map[string]interface{}, float64](3, 10*time.Second)

	// Create environment and client
	config := types.Config{
		Model:        "gpt-4",
		SystemPrompt: "You are a helpful math assistant. Provide only the numeric answer.",
		MessageType:  "chat",
	}
	env := envs.NewSingleTurnEnv(config)
	env.SetRubric(rubrics.NewBaseRubric())
	client := inference.NewHTTPClient("http://localhost:8000/v1", "your-api-key")

	// Process all problems concurrently
	ctx := context.Background()
	items := make([]map[string]interface{}, dataset.Len())
	for i := 0; i < dataset.Len(); i++ {
		items[i] = dataset.Get(i)
	}

	results := processor.ProcessWithProgress(ctx, items, 
		func(ctx context.Context, item map[string]interface{}) (float64, error) {
			question := item["question"].(string)
			answer := item["answer"].(string)
			
			prompt := env.FormatPrompt(question)
			rollout, err := env.Rollout(ctx, client, config.Model, prompt, answer, config.SamplingArgs)
			if err != nil {
				return 0.0, err
			}
			
			return rollout.Score, nil
		},
		func(completed, total int) {
			fmt.Printf("Progress: %d/%d\n", completed, total)
		},
	)

	// Display results
	totalScore := 0.0
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("Item %d: Error - %v\n", result.Index, result.Error)
		} else {
			fmt.Printf("Item %d: Score = %.2f\n", result.Index, result.Result)
			totalScore += result.Result
		}
	}
	
	avgScore := totalScore / float64(len(results))
	fmt.Printf("\nAverage Score: %.2f\n", avgScore)
}

func runMultiTurnExample() {
	// Create a think parser for step-by-step reasoning
	thinkParser := parsers.NewThinkParser()

	// Create configuration
	config := types.Config{
		Model: "gpt-4",
		SystemPrompt: `You are a step-by-step problem solver. For each step:
<think>
Explain your reasoning
</think>
Your conclusion or next question`,
		MessageType: "chat",
		SamplingArgs: types.SamplingArgs{
			Temperature: 0.7,
			MaxTokens:   300,
		},
	}

	// Create dialog environment
	dialogEnv := envs.NewDialogMultiTurnEnv(config, 5, "FINAL ANSWER")
	dialogEnv.SetParser(thinkParser)
	
	// Create custom rubric that checks for think tags
	rubric := rubrics.NewMultiMetricRubric()
	
	// Add think format checker
	thinkFormatFunc := func(ctx context.Context, response, groundTruth string) (float64, error) {
		if thinkParser.FollowsFormat(response) {
			return 1.0, nil
		}
		return 0.0, nil
	}
	
	// Add answer checker
	answerFunc := func(ctx context.Context, response, groundTruth string) (float64, error) {
		parsed, _ := thinkParser.Parse(ctx, response)
		if parsed == groundTruth {
			return 1.0, nil
		}
		return 0.0, nil
	}
	
	rubric.AddMetric("think_format", thinkFormatFunc, 0.3)
	rubric.AddMetric("correct_answer", answerFunc, 0.7)
	dialogEnv.SetRubric(rubric)

	// Create HTTP client
	client := inference.NewHTTPClient("http://localhost:8000/v1", "your-api-key")

	// Start conversation
	messages := []types.Message{
		{
			Role:    "user",
			Content: "Let's solve this step by step: If a train travels 60 mph for 2 hours, then 80 mph for 3 hours, what's the total distance? When you have the final answer, say 'FINAL ANSWER: [your answer]'",
		},
	}

	answer := "FINAL ANSWER: 360 miles"

	// Perform rollout
	ctx := context.Background()
	rollout, err := dialogEnv.Rollout(ctx, client, config.Model, messages, answer, config.SamplingArgs)
	if err != nil {
		log.Printf("Multi-turn rollout failed: %v", err)
		return
	}

	fmt.Printf("Multi-turn Conversation:\n")
	for i, msg := range rollout.Messages {
		fmt.Printf("[Turn %d - %s]:\n%s\n\n", i+1, msg.Role, msg.Content)
	}
	fmt.Printf("Final Score: %.2f\n", rollout.Score)
}