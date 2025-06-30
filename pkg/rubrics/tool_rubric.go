package rubrics

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/tools"
)

// ToolRubric evaluates tool-using environments
type ToolRubric struct {
	*MultiMetricRubric
	tools     []tools.Tool
	parser    *parsers.XMLParser
	envParser *parsers.XMLParser
}

// NewToolRubric creates a new tool rubric
func NewToolRubric(toolList []tools.Tool, parser *parsers.XMLParser, envParser *parsers.XMLParser) (*ToolRubric, error) {
	rubric := &ToolRubric{
		MultiMetricRubric: NewMultiMetricRubric(),
		tools:            toolList,
		parser:           parser,
		envParser:        envParser,
	}

	// Add correct answer reward function
	correctAnswerFunc := func(ctx context.Context, parsed, groundTruth string) (float64, error) {
		// Extract answer from XML if needed
		if rubric.parser != nil {
			parsedXML, err := rubric.parser.ParseXML(parsed, true)
			if err == nil && parsedXML.Fields["answer"] != "" {
				parsed = parsedXML.Fields["answer"]
			}
		}

		// Simple exact match for now
		parsed = strings.TrimSpace(parsed)
		groundTruth = strings.TrimSpace(groundTruth)
		
		if parsed == groundTruth {
			return 1.0, nil
		}
		return 0.0, nil
	}

	// Add format reward function
	formatFunc := func(ctx context.Context, response, groundTruth string) (float64, error) {
		return rubric.evaluateFormat(response)
	}

	// Add tool usage reward function
	toolUsageFunc := func(ctx context.Context, response, groundTruth string) (float64, error) {
		return rubric.evaluateToolUsage(response)
	}

	// Add metrics with weights
	rubric.AddMetric("correct_answer", correctAnswerFunc, 0.6)
	rubric.AddMetric("format", formatFunc, 0.2)
	rubric.AddMetric("tool_usage", toolUsageFunc, 0.2)

	return rubric, nil
}

// evaluateFormat checks if the response follows the expected XML format
func (r *ToolRubric) evaluateFormat(response string) (float64, error) {
	// Split response into messages if it contains multiple
	messages := strings.Split(response, "\n---\n")
	if len(messages) == 0 {
		messages = []string{response}
	}

	totalScore := 0.0
	for _, msg := range messages {
		score := 0.0
		
		// Check for think tags
		if strings.Contains(msg, "<think>") && strings.Contains(msg, "</think>") {
			score += 0.3
		}
		
		// Check for either tool or answer tags
		hasToolTag := strings.Contains(msg, "<tool>") && strings.Contains(msg, "</tool>")
		hasAnswerTag := strings.Contains(msg, "<answer>") && strings.Contains(msg, "</answer>")
		
		if hasToolTag || hasAnswerTag {
			score += 0.4
		}
		
		// Parse and validate structure
		parsed, err := r.parser.ParseXML(msg, true)
		if err == nil {
			// Valid XML structure
			score += 0.3
			
			// Bonus for having content in fields
			if parsed.Fields["think"] != "" {
				score += 0.1
			}
			if parsed.Fields["tool"] != "" || parsed.Fields["answer"] != "" {
				score += 0.1
			}
		}
		
		// Cap at 1.0
		if score > 1.0 {
			score = 1.0
		}
		
		totalScore += score
	}
	
	if len(messages) > 0 {
		return totalScore / float64(len(messages)), nil
	}
	return 0.0, nil
}

// evaluateToolUsage checks if tools are used correctly
func (r *ToolRubric) evaluateToolUsage(response string) (float64, error) {
	// Extract all tool calls from the response
	toolCalls := r.extractToolCalls(response)
	
	if len(toolCalls) == 0 {
		// No tool usage - might be okay for some problems
		return 0.5, nil
	}
	
	validCalls := 0
	for _, toolJSON := range toolCalls {
		// Try to parse the tool call
		var toolCall map[string]interface{}
		if err := json.Unmarshal([]byte(toolJSON), &toolCall); err != nil {
			continue
		}
		
		// Check if it has required fields
		if name, ok := toolCall["name"].(string); ok && name != "" {
			// Check if tool exists
			toolExists := false
			for _, tool := range r.tools {
				if tool.Name() == name {
					toolExists = true
					break
				}
			}
			
			if toolExists && toolCall["args"] != nil {
				validCalls++
			}
		}
	}
	
	// Calculate score based on valid tool usage
	if validCalls > 0 {
		return 1.0, nil
	}
	return 0.0, nil
}

// extractToolCalls extracts all tool JSON calls from the response
func (r *ToolRubric) extractToolCalls(response string) []string {
	var toolCalls []string
	
	// Simple extraction of content between <tool> tags
	parts := strings.Split(response, "<tool>")
	for i := 1; i < len(parts); i++ {
		if endIdx := strings.Index(parts[i], "</tool>"); endIdx > 0 {
			toolJSON := strings.TrimSpace(parts[i][:endIdx])
			if toolJSON != "" {
				toolCalls = append(toolCalls, toolJSON)
			}
		}
	}
	
	return toolCalls
}