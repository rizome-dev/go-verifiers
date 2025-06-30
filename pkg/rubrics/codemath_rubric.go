package rubrics

import (
	"context"
	"strings"

	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/types"
	"github.com/rizome-dev/go-verifiers/pkg/utils"
)

// CodeMathRubric evaluates code-based mathematical solutions
type CodeMathRubric struct {
	*MathRubric
}

// NewCodeMathRubric creates a new code-math rubric
func NewCodeMathRubric() (*CodeMathRubric, error) {
	// Create base math rubric
	mathRubric, err := NewMathRubric()
	if err != nil {
		return nil, err
	}

	rubric := &CodeMathRubric{
		MathRubric: mathRubric,
	}

	// Override parser to support code field
	codeParser, err := parsers.NewXMLParser([]interface{}{"reasoning", "code", "answer"}, "answer")
	if err != nil {
		return nil, err
	}
	rubric.parser = codeParser

	// Add code execution reward function
	codeExecutionFunc := func(ctx context.Context, response, groundTruth string) (float64, error) {
		return rubric.evaluateCodeExecution(response)
	}

	// Update metrics - replace format metric with code execution
	rubric.metrics = make(map[string]types.RewardFunc)
	rubric.rewardFuncs = nil
	rubric.rewardWeights = nil

	// Re-add correct answer function
	correctAnswerFunc := func(ctx context.Context, parsed, groundTruth string) (float64, error) {
		// Use the XML parser to extract answer
		parsedXML, err := codeParser.ParseXML(parsed, true)
		if err == nil && parsedXML.Fields["answer"] != "" {
			parsed = parsedXML.Fields["answer"]
		}

		// Compare answers using math comparison
		if utils.CompareMathAnswers(parsed, groundTruth) {
			return 1.0, nil
		}
		return 0.0, nil
	}

	// Add metrics with weights
	rubric.AddMetric("correct_answer", correctAnswerFunc, 0.7)
	rubric.AddMetric("code_execution", codeExecutionFunc, 0.3)

	return rubric, nil
}

// evaluateCodeExecution checks if code executions were successful
func (r *CodeMathRubric) evaluateCodeExecution(response string) (float64, error) {
	// Parse response to check for code blocks
	parsed, err := r.parser.ParseXML(response, true)
	if err != nil {
		return 0.0, nil
	}

	// Check if code field exists and has content
	code := parsed.Fields["code"]
	if code == "" {
		return 0.0, nil
	}

	// Look for error indicators in the response
	// This is a simplified check - in practice, we'd need access to execution results
	lowerResponse := strings.ToLower(response)
	
	// Common error patterns
	errorIndicators := []string{
		"error:",
		"traceback",
		"exception",
		"syntaxerror",
		"nameerror",
		"typeerror",
		"valueerror",
		"zerodivisionerror",
		"code execution error",
	}

	hasError := false
	for _, indicator := range errorIndicators {
		if strings.Contains(lowerResponse, indicator) {
			hasError = true
			break
		}
	}

	// Check for successful execution indicators
	successIndicators := []string{
		"code output:",
		"result:",
		"output:",
	}

	hasSuccess := false
	for _, indicator := range successIndicators {
		if strings.Contains(lowerResponse, indicator) && !hasError {
			hasSuccess = true
			break
		}
	}

	// Score based on execution success
	if hasSuccess && !hasError {
		return 1.0, nil
	} else if hasError {
		return 0.0, nil
	}

	// If we can't determine, give partial credit for having code
	return 0.5, nil
}

// ComputeRewardWithState computes reward with access to execution state
func (r *CodeMathRubric) ComputeRewardWithState(ctx context.Context, parsed string, groundTruth string, state map[string]interface{}) (float64, error) {
	// Get base score
	baseScore, err := r.ComputeReward(ctx, parsed, groundTruth)
	if err != nil {
		return 0.0, err
	}

	// If we have code execution history in state, use it for more accurate scoring
	if executions, ok := state["code_executions"].([]map[string]interface{}); ok && len(executions) > 0 {
		successCount := 0
		totalCount := len(executions)

		for _, exec := range executions {
			if success, ok := exec["success"].(bool); ok && success {
				successCount++
			}
		}

		// Replace code execution score with actual execution results
		executionScore := float64(successCount) / float64(totalCount)
		
		// Recalculate weighted score
		// Assuming weights: correct_answer=0.7, code_execution=0.3
		answerScore := baseScore / (0.7 + 0.3) * 0.7 // Extract answer component
		return answerScore + executionScore*0.3, nil
	}

	return baseScore, nil
}