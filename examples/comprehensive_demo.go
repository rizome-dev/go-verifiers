package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rizome-dev/go-verifiers/pkg/envs"
	"github.com/rizome-dev/go-verifiers/pkg/inference"
	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/prompts"
	"github.com/rizome-dev/go-verifiers/pkg/rubrics"
	"github.com/rizome-dev/go-verifiers/pkg/tools"
	"github.com/rizome-dev/go-verifiers/pkg/types"
	"github.com/rizome-dev/go-verifiers/pkg/utils"
)

// This example demonstrates all major features of go-verifiers
func main() {
	fmt.Println("=== Go Verifiers Comprehensive Demo ===\n")

	// Create HTTP client for inference
	client := inference.NewHTTPClient("http://localhost:8000/v1", "your-api-key")
	ctx := context.Background()

	// 1. SingleTurn Environment
	fmt.Println("1. SingleTurn Environment - Basic Q&A")
	demoSingleTurn(ctx, client)

	// 2. MultiTurn Dialog Environment
	fmt.Println("\n2. MultiTurn Dialog Environment")
	demoMultiTurn(ctx, client)

	// 3. Tool Environment
	fmt.Println("\n3. Tool Environment with Calculator")
	demoToolEnv(ctx, client)

	// 4. CodeMath Environment
	fmt.Println("\n4. CodeMath Environment - Expression Evaluation")
	demoCodeMath(ctx, client)

	// 5. DoubleCheck Environment
	fmt.Println("\n5. DoubleCheck Environment")
	demoDoubleCheck(ctx, client)

	// 6. Environment Group
	fmt.Println("\n6. Environment Group - Multiple Tasks")
	demoEnvGroup(ctx, client)

	// 7. Judge Rubric
	fmt.Println("\n7. Judge Rubric - LLM Evaluation")
	demoJudgeRubric(ctx, client)

	// 8. Concurrent Processing
	fmt.Println("\n8. Concurrent Batch Processing")
	demoConcurrentProcessing(ctx, client)
}

func demoSingleTurn(ctx context.Context, client types.Client) {
	config := types.Config{
		Model:        "gpt-4",
		SystemPrompt: prompts.SimplePrompt,
		MessageType:  "chat",
		SamplingArgs: types.SamplingArgs{
			Temperature: 0.7,
			MaxTokens:   200,
		},
	}

	env := envs.NewSingleTurnEnv(config)
	env.SetParser(parsers.NewBaseParser())
	env.SetRubric(rubrics.NewBaseRubric())

	prompt := env.FormatPrompt("What is the capital of France?")
	answer := "Paris"

	rollout, err := env.Rollout(ctx, client, config.Model, prompt, answer, config.SamplingArgs)
	if err != nil {
		log.Printf("SingleTurn rollout failed: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", truncate(rollout.Response, 100))
	fmt.Printf("Score: %.2f\n", rollout.Score)
}

func demoMultiTurn(ctx context.Context, client types.Client) {
	config := types.Config{
		Model:        "gpt-4",
		SystemPrompt: "You are playing 20 questions. I'm thinking of an animal. Ask yes/no questions to guess it.",
		MessageType:  "chat",
	}

	env := envs.NewDialogMultiTurnEnv(config, 5, "CORRECT")
	env.SetParser(parsers.NewBaseParser())
	env.SetRubric(rubrics.NewBaseRubric())

	messages := []types.Message{
		{Role: "user", Content: "I'm thinking of an animal. Start guessing! Say 'CORRECT' when you guess it."},
	}

	rollout, err := env.Rollout(ctx, client, config.Model, messages, "elephant", config.SamplingArgs)
	if err != nil {
		log.Printf("MultiTurn rollout failed: %v", err)
		return
	}

	fmt.Printf("Conversation length: %d messages\n", len(rollout.Messages))
	fmt.Printf("Final score: %.2f\n", rollout.Score)
}

func demoToolEnv(ctx context.Context, client types.Client) {
	// Create calculator tool
	calc := tools.NewCalculator()

	config := types.Config{
		Model:       "gpt-4",
		MessageType: "chat",
		SamplingArgs: types.SamplingArgs{
			Temperature: 0.3,
			MaxTokens:   300,
		},
	}

	toolEnv, err := envs.NewToolEnv(config, []tools.Tool{calc}, 3)
	if err != nil {
		log.Printf("Failed to create tool env: %v", err)
		return
	}

	messages := []types.Message{
		{Role: "user", Content: "What is sqrt(144) + sin(pi/2)?"},
	}

	rollout, err := toolEnv.Rollout(ctx, client, config.Model, messages, "13", config.SamplingArgs)
	if err != nil {
		log.Printf("Tool rollout failed: %v", err)
		return
	}

	fmt.Printf("Used tools: %v\n", extractToolUsage(rollout.Messages))
	fmt.Printf("Score: %.2f\n", rollout.Score)
}

func demoCodeMath(ctx context.Context, client types.Client) {
	config := types.Config{
		Model:       "gpt-4",
		MessageType: "chat",
		SamplingArgs: types.SamplingArgs{
			Temperature: 0.3,
		},
	}

	env, err := envs.NewCodeMathEnv(config, 3)
	if err != nil {
		log.Printf("Failed to create CodeMath env: %v", err)
		return
	}

	prompt := env.FormatPrompt("Calculate the sum of the first 10 squares")
	answer := "385" // 1² + 2² + ... + 10² = 385

	rollout, err := env.Rollout(ctx, client, config.Model, prompt, answer, config.SamplingArgs)
	if err != nil {
		log.Printf("CodeMath rollout failed: %v", err)
		return
	}

	fmt.Printf("Mathematical evaluation used: %v\n", containsCode(rollout.Messages))
	fmt.Printf("Score: %.2f\n", rollout.Score)
}

func demoDoubleCheck(ctx context.Context, client types.Client) {
	config := types.Config{
		Model:        "gpt-4",
		SystemPrompt: prompts.SimplePrompt,
		MessageType:  "chat",
	}

	env, err := envs.NewDoubleCheckEnv(config)
	if err != nil {
		log.Printf("Failed to create DoubleCheck env: %v", err)
		return
	}

	prompt := env.FormatPrompt("What is 15% of 80?")
	answer := "12"

	rollout, err := env.Rollout(ctx, client, config.Model, prompt, answer, config.SamplingArgs)
	if err != nil {
		log.Printf("DoubleCheck rollout failed: %v", err)
		return
	}

	fmt.Printf("Double-checked: %v\n", containsDoubleCheck(rollout.Messages))
	fmt.Printf("Score: %.2f\n", rollout.Score)
}

func demoEnvGroup(ctx context.Context, client types.Client) {
	// Create multiple environments
	mathEnv := envs.NewSingleTurnEnv(types.Config{
		Model:        "gpt-4",
		SystemPrompt: "Solve math problems. Provide only the numeric answer.",
		MessageType:  "chat",
	})
	mathEnv.SetRubric(rubrics.NewBaseRubric())

	triviaEnv := envs.NewSingleTurnEnv(types.Config{
		Model:        "gpt-4",
		SystemPrompt: "Answer trivia questions concisely.",
		MessageType:  "chat",
	})
	triviaEnv.SetRubric(rubrics.NewBaseRubric())

	// Create environment group
	envMap := map[string]envs.Environment{
		"math":   mathEnv,
		"trivia": triviaEnv,
	}

	group := envs.NewEnvGroup(types.Config{Model: "gpt-4"}, envMap)

	// Test math task
	mathPrompt := mathEnv.FormatPrompt("What is 25 * 4?")
	rollout, err := group.Rollout(ctx, client, "gpt-4", mathPrompt, "math:100", types.SamplingArgs{})
	if err != nil {
		log.Printf("EnvGroup math failed: %v", err)
		return
	}
	fmt.Printf("Math task score: %.2f\n", rollout.Score)

	// Test trivia task
	triviaPrompt := triviaEnv.FormatPrompt("Who wrote Romeo and Juliet?")
	rollout, err = group.Rollout(ctx, client, "gpt-4", triviaPrompt, "trivia:Shakespeare", types.SamplingArgs{})
	if err != nil {
		log.Printf("EnvGroup trivia failed: %v", err)
		return
	}
	fmt.Printf("Trivia task score: %.2f\n", rollout.Score)
}

func demoJudgeRubric(ctx context.Context, client types.Client) {
	// Create judge rubric
	judge := rubrics.NewJudgeRubric(client, "gpt-4")

	response := "The capital of France is Paris, which is located on the Seine River."
	groundTruth := "Paris"

	score, reasoning, err := judge.JudgeWithReasoning(ctx, response, groundTruth)
	if err != nil {
		log.Printf("Judge evaluation failed: %v", err)
		return
	}

	fmt.Printf("Judge score: %.2f\n", score)
	fmt.Printf("Reasoning: %s\n", truncate(reasoning, 100))
}

func demoConcurrentProcessing(ctx context.Context, client types.Client) {
	// Create dataset
	problems := []struct{ Question, Answer string }{
		{"What is 2 + 2?", "4"},
		{"What is 5 * 6?", "30"},
		{"What is 10 - 3?", "7"},
		{"What is 20 / 4?", "5"},
		{"What is 3³?", "27"},
	}

	dataset := types.DatasetUtils{}.LoadFromQuestionAnswer(problems)

	// Create environment
	config := types.Config{
		Model:        "gpt-4",
		SystemPrompt: "Solve the math problem. Give only the numeric answer.",
		MessageType:  "chat",
	}
	env := envs.NewSingleTurnEnv(config)
	env.SetRubric(rubrics.NewBaseRubric())

	// Process concurrently
	processor := utils.NewBatchProcessor[map[string]interface{}, float64](3, 10*time.Second)
	
	items := make([]map[string]interface{}, dataset.Len())
	for i := 0; i < dataset.Len(); i++ {
		items[i] = dataset.Get(i)
	}

	start := time.Now()
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
			fmt.Printf("\rProgress: %d/%d", completed, total)
		},
	)

	elapsed := time.Since(start)
	fmt.Printf("\nProcessed %d items in %.2fs\n", len(results), elapsed.Seconds())

	totalScore := 0.0
	for _, result := range results {
		if result.Error == nil {
			totalScore += result.Result
		}
	}
	fmt.Printf("Average score: %.2f\n", totalScore/float64(len(results)))
}

// Helper functions
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func extractToolUsage(messages []types.Message) []string {
	var tools []string
	for _, msg := range messages {
		if strings.Contains(msg.Content, "<tool>") {
			// Extract tool name from JSON
			start := strings.Index(msg.Content, `"name":"`) + 8
			if start > 7 {
				end := strings.Index(msg.Content[start:], `"`)
				if end > 0 {
					tools = append(tools, msg.Content[start:start+end])
				}
			}
		}
	}
	return tools
}

func containsCode(messages []types.Message) bool {
	for _, msg := range messages {
		if strings.Contains(msg.Content, "<code>") {
			return true
		}
	}
	return false
}

func containsDoubleCheck(messages []types.Message) bool {
	for _, msg := range messages {
		if strings.Contains(strings.ToLower(msg.Content), "are you sure") {
			return true
		}
	}
	return false
}