package tools

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
)

// Calculator implements a mathematical expression evaluator
type Calculator struct {
	*BaseTool
}

// NewCalculator creates a new calculator tool
func NewCalculator() *Calculator {
	calc := &Calculator{
		BaseTool: NewBaseTool(
			"calculate",
			"Evaluate mathematical expressions. Supports basic arithmetic, trigonometry, logarithms, and more.",
			nil, // Set below
		),
	}
	
	// Set the executor
	calc.executor = calc.execute
	
	// Define schema
	calc.schema = ToolSchema{
		Name:        "calculate",
		Description: calc.description,
		Args: map[string]ArgumentSchema{
			"expression": {
				Type:        "string",
				Description: "Mathematical expression to evaluate",
				Required:    true,
			},
		},
		Returns: "The result of the mathematical expression as a number",
		Examples: []string{
			`{"name": "calculate", "args": {"expression": "2 + 2"}}`,
			`{"name": "calculate", "args": {"expression": "sqrt(16) + log(100)"}}`,
			`{"name": "calculate", "args": {"expression": "sin(pi/2) * cos(0)"}}`,
		},
	}
	
	return calc
}

// execute evaluates a mathematical expression
func (c *Calculator) execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	exprInterface, ok := args["expression"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'expression'")
	}
	
	expr, ok := exprInterface.(string)
	if !ok {
		return nil, fmt.Errorf("expression must be a string")
	}
	
	// Preprocess the expression to handle common mathematical functions
	processed := preprocessExpression(expr)
	
	// Create expression evaluator
	expression, err := govaluate.NewEvaluableExpression(processed)
	if err != nil {
		// Try simpler evaluation for basic expressions
		result, evalErr := evaluateSimple(expr)
		if evalErr != nil {
			return nil, fmt.Errorf("invalid expression: %v", err)
		}
		return result, nil
	}
	
	// Define mathematical functions and constants
	parameters := map[string]interface{}{
		"pi":   math.Pi,
		"e":    math.E,
		"sqrt": sqrt,
		"sin":  sin,
		"cos":  cos,
		"tan":  tan,
		"log":  log,
		"ln":   ln,
		"exp":  exp,
		"pow":  pow,
		"abs":  abs,
		"ceil": ceil,
		"floor": floor,
		"round": round,
	}
	
	// Evaluate the expression
	result, err := expression.Evaluate(parameters)
	if err != nil {
		return nil, fmt.Errorf("evaluation error: %v", err)
	}
	
	// Format the result
	switch v := result.(type) {
	case float64:
		// Format to remove unnecessary decimal places
		if v == float64(int64(v)) {
			return int64(v), nil
		}
		return v, nil
	case int64:
		return v, nil
	default:
		return fmt.Sprintf("%v", result), nil
	}
}

// Mathematical function wrappers for govaluate
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

// preprocessExpression handles common mathematical notation
func preprocessExpression(expr string) string {
	// Replace common mathematical constants
	expr = strings.ReplaceAll(expr, "π", "pi")
	
	// Handle implicit multiplication (e.g., 2pi -> 2*pi)
	// This is a simple implementation and may need refinement
	expr = strings.ReplaceAll(expr, "2pi", "2*pi")
	expr = strings.ReplaceAll(expr, "2e", "2*e")
	
	return expr
}

// evaluateSimple handles basic arithmetic for fallback
func evaluateSimple(expr string) (interface{}, error) {
	// Remove spaces
	expr = strings.ReplaceAll(expr, " ", "")
	
	// Try to parse as a simple number
	if val, err := strconv.ParseFloat(expr, 64); err == nil {
		if val == float64(int64(val)) {
			return int64(val), nil
		}
		return val, nil
	}
	
	// For more complex expressions, return an error to use the main evaluator
	return nil, fmt.Errorf("expression too complex for simple evaluation")
}