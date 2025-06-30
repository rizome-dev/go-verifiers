package rubrics

import (
	"context"

	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/utils"
)

// MathRubric evaluates mathematical responses
type MathRubric struct {
	*MultiMetricRubric
	parser *parsers.XMLParser
}

// NewMathRubric creates a new math rubric
func NewMathRubric() (*MathRubric, error) {
	// Create XML parser for think/answer format
	parser, err := parsers.NewXMLParser([]interface{}{"think", "answer"}, "answer")
	if err != nil {
		return nil, err
	}

	rubric := &MathRubric{
		MultiMetricRubric: NewMultiMetricRubric(),
		parser:            parser,
	}

	// Add correct answer reward function
	correctAnswerFunc := func(ctx context.Context, parsed, groundTruth string) (float64, error) {
		// Use the XML parser to extract answer
		parsedXML, err := parser.ParseXML(parsed, true)
		if err == nil && parsedXML.Fields["answer"] != "" {
			parsed = parsedXML.Fields["answer"]
		}

		// Compare answers using math comparison
		if utils.CompareMathAnswers(parsed, groundTruth) {
			return 1.0, nil
		}
		return 0.0, nil
	}

	// Add format reward function
	formatFunc := func(ctx context.Context, response, groundTruth string) (float64, error) {
		parsedXML, err := parser.ParseXML(response, true)
		if err != nil {
			return 0.0, nil
		}

		score := 0.0
		
		// Check if both think and answer tags are present
		if parsedXML.Fields["think"] != "" {
			score += 0.4
		}
		if parsedXML.Fields["answer"] != "" {
			score += 0.4
		}

		// Check if follows proper XML structure
		if parser.HasField("think") && parser.HasField("answer") {
			// Basic structure check
			if len(parsedXML.Fields) >= 2 {
				score += 0.2
			}
		}

		return score, nil
	}

	// Add metrics with weights
	rubric.AddMetric("correct_answer", correctAnswerFunc, 0.8)
	rubric.AddMetric("format", formatFunc, 0.2)

	return rubric, nil
}

// GetParser returns the XML parser used by this rubric
func (r *MathRubric) GetParser() *parsers.XMLParser {
	return r.parser
}

// ComputeReward computes the weighted reward for math problems
func (r *MathRubric) ComputeReward(ctx context.Context, parsed string, groundTruth string) (float64, error) {
	// Extract boxed answer from ground truth if present
	groundTruth = utils.ExtractBoxedAnswer(groundTruth)
	
	return r.MultiMetricRubric.ComputeReward(ctx, parsed, groundTruth)
}