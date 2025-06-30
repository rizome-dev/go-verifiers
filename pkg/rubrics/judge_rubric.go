package rubrics

import (
	"context"
	"fmt"
	"strings"

	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// JudgeRubric uses an LLM to judge response correctness
type JudgeRubric struct {
	*BaseRubric
	judgeClient types.Client
	judgeModel  string
	systemPrompt string
}

// NewJudgeRubric creates a new LLM-based judge rubric
func NewJudgeRubric(judgeClient types.Client, judgeModel string) *JudgeRubric {
	if judgeModel == "" {
		judgeModel = "gpt-4-turbo-preview"
	}

	rubric := &JudgeRubric{
		BaseRubric:   NewBaseRubric(),
		judgeClient:  judgeClient,
		judgeModel:   judgeModel,
		systemPrompt: defaultJudgeSystemPrompt,
	}

	// Replace the default exact match with judge evaluation
	judgeFunc := func(ctx context.Context, parsed, groundTruth string) (float64, error) {
		return rubric.judge(ctx, parsed, groundTruth)
	}

	rubric.rewardFuncs = []types.RewardFunc{judgeFunc}
	rubric.rewardWeights = []float64{1.0}

	return rubric
}

// SetSystemPrompt updates the judge system prompt
func (r *JudgeRubric) SetSystemPrompt(prompt string) {
	r.systemPrompt = prompt
}

// judge uses the LLM to evaluate correctness
func (r *JudgeRubric) judge(ctx context.Context, modelResponse, groundTruth string) (float64, error) {
	// Format the judge prompt
	userPrompt := fmt.Sprintf(`Please evaluate if the model's response is correct.

Ground Truth Answer: %s

Model Response: %s

Is the model's response correct? Reply with only "Yes" or "No".`, groundTruth, modelResponse)

	// Create messages for the judge
	messages := []types.Message{
		{
			Role:    "system",
			Content: r.systemPrompt,
		},
		{
			Role:    "user",
			Content: userPrompt,
		},
	}

	// Call the judge model
	samplingArgs := types.SamplingArgs{
		Temperature: 0.0, // Deterministic judgment
		MaxTokens:   10,  // Only need "Yes" or "No"
	}

	response, err := r.judgeClient.CreateChatCompletion(ctx, r.judgeModel, messages, samplingArgs)
	if err != nil {
		return 0.0, fmt.Errorf("judge evaluation failed: %w", err)
	}

	// Parse the judgment
	response = strings.TrimSpace(strings.ToLower(response))
	
	if strings.Contains(response, "yes") {
		return 1.0, nil
	} else if strings.Contains(response, "no") {
		return 0.0, nil
	}

	// If unclear, default to incorrect
	return 0.0, nil
}

// JudgeWithReasoning provides detailed judgment with reasoning
func (r *JudgeRubric) JudgeWithReasoning(ctx context.Context, modelResponse, groundTruth string) (float64, string, error) {
	// Format the judge prompt for detailed evaluation
	userPrompt := fmt.Sprintf(`Please evaluate if the model's response is correct.

Ground Truth Answer: %s

Model Response: %s

Provide your evaluation in the following format:
<reasoning>
Explain why the response is correct or incorrect
</reasoning>
<judgment>
Yes or No
</judgment>`, groundTruth, modelResponse)

	// Create messages for the judge
	messages := []types.Message{
		{
			Role:    "system",
			Content: r.systemPrompt,
		},
		{
			Role:    "user",
			Content: userPrompt,
		},
	}

	// Call the judge model
	samplingArgs := types.SamplingArgs{
		Temperature: 0.0,
		MaxTokens:   200,
	}

	response, err := r.judgeClient.CreateChatCompletion(ctx, r.judgeModel, messages, samplingArgs)
	if err != nil {
		return 0.0, "", fmt.Errorf("judge evaluation failed: %w", err)
	}

	// Extract reasoning and judgment
	reasoning := ""
	judgment := ""

	// Simple extraction
	if strings.Contains(response, "<reasoning>") && strings.Contains(response, "</reasoning>") {
		start := strings.Index(response, "<reasoning>") + 11
		end := strings.Index(response, "</reasoning>")
		if start < end {
			reasoning = strings.TrimSpace(response[start:end])
		}
	}

	if strings.Contains(response, "<judgment>") && strings.Contains(response, "</judgment>") {
		start := strings.Index(response, "<judgment>") + 10
		end := strings.Index(response, "</judgment>")
		if start < end {
			judgment = strings.TrimSpace(response[start:end])
		}
	}

	// Determine score
	score := 0.0
	if strings.Contains(strings.ToLower(judgment), "yes") {
		score = 1.0
	}

	return score, reasoning, nil
}

// defaultJudgeSystemPrompt is the default prompt for the judge
const defaultJudgeSystemPrompt = `You are a fair and accurate judge that evaluates whether model responses are correct.

Consider the following when making judgments:
1. Mathematical equivalence (e.g., 0.5 = 1/2 = 50%)
2. Semantic equivalence (same meaning, different wording)
3. Acceptable variations in formatting or presentation
4. Partial credit is not given - responses are either correct or incorrect

Be strict but fair in your evaluations.`