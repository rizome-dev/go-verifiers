package envs

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/rubrics"
	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// CodeMathEnv handles mathematical problem-solving with code/expression evaluation
type CodeMathEnv struct {
	*MultiTurnEnv
	Parser *parsers.XMLParser
}

// NewCodeMathEnv creates a new code-based math environment
func NewCodeMathEnv(config types.Config, maxTurns int) (*CodeMathEnv, error) {
	// Update system prompt for Go-based math expressions
	if config.SystemPrompt == "" {
		config.SystemPrompt = `You are a helpful assistant that solves math problems by writing mathematical expressions.

For each problem:
1. First, think through the problem step by step
2. Write mathematical expressions or step-by-step calculations to solve it
3. Provide the final answer based on your calculations

Format your response as:
<reasoning>
Explain your approach
</reasoning>
<code>
# Mathematical expressions or calculations
# e.g., 2 + 2 = 4
# e.g., sqrt(16) = 4
# e.g., sin(pi/2) = 1
</code>
<answer>
Your final answer
</answer>

The system will evaluate your mathematical expressions.`
	}

	// Create XML parser for reasoning/code/answer format
	parser, err := parsers.NewXMLParser([]interface{}{"reasoning", "code", "answer"}, "answer")
	if err != nil {
		return nil, err
	}

	env := &CodeMathEnv{
		MultiTurnEnv: NewMultiTurnEnv(config, maxTurns),
		Parser:       parser,
	}

	// Set parser and rubric
	env.SetParser(parser)
	
	// Create CodeMath rubric
	codeMathRubric, err := rubrics.NewCodeMathRubric()
	if err != nil {
		return nil, err
	}
	env.SetRubric(codeMathRubric)

	return env, nil
}

// IsCompleted checks if the problem is solved
func (e *CodeMathEnv) IsCompleted(ctx context.Context, messages []types.Message, state map[string]interface{}) bool {
	if len(messages) == 0 {
		return false
	}

	// Check last assistant message for answer
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" {
			parsed, err := e.Parser.ParseXML(messages[i].Content, true)
			if err == nil && parsed.Fields["answer"] != "" {
				return true
			}
		}
	}

	return false
}

// EnvResponse evaluates mathematical expressions and provides feedback
func (e *CodeMathEnv) EnvResponse(ctx context.Context, messages []types.Message, state map[string]interface{}) (types.Message, map[string]interface{}, error) {
	if len(messages) == 0 {
		return types.Message{}, state, fmt.Errorf("no messages to process")
	}

	// Get last assistant message
	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "assistant" {
		return types.Message{}, state, fmt.Errorf("last message must be from assistant")
	}

	// Parse for code/expressions
	parsed, err := e.Parser.ParseXML(lastMsg.Content, true)
	if err != nil {
		return types.Message{
			Role:    "user",
			Content: "Failed to parse response. Please use the correct XML format with <reasoning>, <code>, and <answer> tags.",
		}, state, nil
	}

	// Check if there's code to evaluate
	code := parsed.Fields["code"]
	if code == "" {
		return types.Message{
			Role:    "user",
			Content: "No mathematical expressions found. Please provide expressions or calculations in <code> tags.",
		}, state, nil
	}

	// Evaluate the mathematical expressions
	output, success := e.evaluateExpressions(ctx, code)
	
	// Format execution result
	var response string
	if !success {
		response = fmt.Sprintf("Evaluation error:\n%s", output)
	} else {
		response = fmt.Sprintf("Evaluation results:\n%s", output)
	}

	// Track evaluations in state
	if state["code_executions"] == nil {
		state["code_executions"] = []map[string]interface{}{}
	}
	
	executions := state["code_executions"].([]map[string]interface{})
	executions = append(executions, map[string]interface{}{
		"code":    code,
		"output":  output,
		"success": success,
	})
	state["code_executions"] = executions

	return types.Message{
		Role:    "user",
		Content: response,
	}, state, nil
}

// evaluateExpressions evaluates mathematical expressions line by line
func (e *CodeMathEnv) evaluateExpressions(ctx context.Context, code string) (string, bool) {
	lines := strings.Split(code, "\n")
	var results []string
	success := true

	// Mathematical functions and constants
	parameters := map[string]interface{}{
		"pi":    math.Pi,
		"e":     math.E,
		"sqrt":  sqrt,
		"sin":   sin,
		"cos":   cos,
		"tan":   tan,
		"log":   log,
		"ln":    ln,
		"exp":   exp,
		"pow":   pow,
		"abs":   abs,
		"ceil":  ceil,
		"floor": floor,
		"round": round,
		"max":   max,
		"min":   min,
	}

	// Variables to store results
	variables := make(map[string]interface{})
	for k, v := range parameters {
		variables[k] = v
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Handle variable assignments (e.g., "x = 5")
		if strings.Contains(line, "=") && !strings.Contains(line, "==") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimSpace(parts[0])
				expr := strings.TrimSpace(parts[1])

				// Evaluate the expression
				result, err := evaluateExpression(expr, variables)
				if err != nil {
					results = append(results, fmt.Sprintf("Error in '%s': %v", line, err))
					success = false
					continue
				}

				// Store the variable
				variables[varName] = result
				results = append(results, fmt.Sprintf("%s = %v", varName, formatResult(result)))
				continue
			}
		}

		// Evaluate standalone expressions
		result, err := evaluateExpression(line, variables)
		if err != nil {
			results = append(results, fmt.Sprintf("Error in '%s': %v", line, err))
			success = false
			continue
		}

		results = append(results, fmt.Sprintf("%s = %v", line, formatResult(result)))
	}

	return strings.Join(results, "\n"), success
}

// evaluateExpression evaluates a single mathematical expression
func evaluateExpression(expr string, variables map[string]interface{}) (interface{}, error) {
	// Preprocess the expression
	expr = preprocessExpression(expr)

	// Create and evaluate expression
	expression, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		return nil, err
	}

	result, err := expression.Evaluate(variables)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// preprocessExpression handles common mathematical notation
func preprocessExpression(expr string) string {
	// Replace common mathematical notation
	expr = strings.ReplaceAll(expr, "π", "pi")
	expr = strings.ReplaceAll(expr, "×", "*")
	expr = strings.ReplaceAll(expr, "÷", "/")
	expr = strings.ReplaceAll(expr, "²", "^2")
	expr = strings.ReplaceAll(expr, "³", "^3")
	
	// Handle implicit multiplication (e.g., 2pi -> 2*pi)
	// Simple cases only
	expr = strings.ReplaceAll(expr, "2pi", "2*pi")
	expr = strings.ReplaceAll(expr, "2e", "2*e")
	
	return expr
}

// formatResult formats a result for display
func formatResult(result interface{}) string {
	switch v := result.(type) {
	case float64:
		// Format nicely, removing unnecessary decimals
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", result)
	}
}

// Mathematical function wrappers (reuse from calculator.go)
func sqrt(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sqrt requires exactly 1 argument")
	}
	val, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Sqrt(val), nil
}

func sin(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sin requires exactly 1 argument")
	}
	val, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Sin(val), nil
}

func cos(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("cos requires exactly 1 argument")
	}
	val, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Cos(val), nil
}

func tan(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("tan requires exactly 1 argument")
	}
	val, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Tan(val), nil
}

func log(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("log requires exactly 1 argument")
	}
	val, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Log10(val), nil
}

func ln(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ln requires exactly 1 argument")
	}
	val, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Log(val), nil
}

func exp(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("exp requires exactly 1 argument")
	}
	val, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Exp(val), nil
}

func pow(args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("pow requires exactly 2 arguments")
	}
	base, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	exponent, err := toFloat64(args[1])
	if err != nil {
		return nil, err
	}
	return math.Pow(base, exponent), nil
}

func abs(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("abs requires exactly 1 argument")
	}
	val, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Abs(val), nil
}

func ceil(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ceil requires exactly 1 argument")
	}
	val, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Ceil(val), nil
}

func floor(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("floor requires exactly 1 argument")
	}
	val, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Floor(val), nil
}

func round(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("round requires exactly 1 argument")
	}
	val, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	return math.Round(val), nil
}

func max(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("max requires at least 1 argument")
	}
	maxVal, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(args); i++ {
		val, err := toFloat64(args[i])
		if err != nil {
			return nil, err
		}
		if val > maxVal {
			maxVal = val
		}
	}
	return maxVal, nil
}

func min(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("min requires at least 1 argument")
	}
	minVal, err := toFloat64(args[0])
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(args); i++ {
		val, err := toFloat64(args[i])
		if err != nil {
			return nil, err
		}
		if val < minVal {
			minVal = val
		}
	}
	return minVal, nil
}

// toFloat64 converts an interface to float64
func toFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

// Rollout performs the code-math environment rollout
func (e *CodeMathEnv) Rollout(ctx context.Context, client types.Client, model string, prompt interface{}, answer string, samplingArgs types.SamplingArgs) (*types.Rollout, error) {
	return BaseMultiTurnRollout(ctx, e, client, model, prompt, answer, samplingArgs, e.MaxTurns)
}