package rubrics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/tools"
	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// SmolaToolRubric evaluates SmolaAgents-style tool usage
type SmolaToolRubric struct {
	*MultiMetricRubric
	tools        []tools.Tool
	parser       *parsers.SmolaParser
	envParser    *parsers.XMLParser
	includeUsage bool
}

// NewSmolaToolRubric creates a new Smola tool rubric
func NewSmolaToolRubric(toolList []tools.Tool, parser *parsers.SmolaParser, envParser *parsers.XMLParser) (*SmolaToolRubric, error) {
	rubric := &SmolaToolRubric{
		MultiMetricRubric: NewMultiMetricRubric(),
		tools:            toolList,
		parser:           parser,
		envParser:        envParser,
		includeUsage:     true,
	}

	// Add correct answer reward function
	correctAnswerFunc := func(ctx context.Context, parsed, groundTruth string) (float64, error) {
		// Extract answer from parsed response
		if rubric.parser != nil {
			parsedSmola, err := rubric.parser.ParseSmola(parsed, true)
			if err == nil {
				// Check for answer in any of the parser's fields
				fields := rubric.parser.GetFields()
				if len(fields) > 0 {
					lastField := fields[len(fields)-1]
					if answer, ok := parsedSmola.Fields[lastField]; ok && answer != "" {
						parsed = answer
					}
				}
			}
		}

		// Simple exact match
		parsed = strings.TrimSpace(parsed)
		groundTruth = strings.TrimSpace(groundTruth)
		
		if parsed == groundTruth {
			return 1.0, nil
		}
		return 0.0, nil
	}

	// Add format reward function
	formatFunc := func(ctx context.Context, response, groundTruth string) (float64, error) {
		return rubric.parser.FollowsFormat(response), nil
	}

	// Add metrics with weights
	rubric.AddMetric("correct_answer", correctAnswerFunc, 0.7)
	rubric.AddMetric("format", formatFunc, 0.3)

	// Add individual tool usage metrics if requested
	if rubric.includeUsage {
		for _, tool := range toolList {
			toolName := tool.Name()
			toolUsageFunc := rubric.createToolUsageFunc(toolName)
			rubric.AddMetric(fmt.Sprintf("%s_usage", toolName), toolUsageFunc, 0.1)
		}
	}

	return rubric, nil
}

// SetIncludeUsage sets whether to include individual tool usage metrics
func (r *SmolaToolRubric) SetIncludeUsage(include bool) {
	r.includeUsage = include
}

// createToolUsageFunc creates a reward function for specific tool usage
func (r *SmolaToolRubric) createToolUsageFunc(toolName string) types.RewardFunc {
	return func(ctx context.Context, response, groundTruth string) (float64, error) {
		// Count successful uses of this specific tool
		toolCalls := r.extractToolCalls(response)
		
		successCount := 0
		totalCount := 0
		
		for _, toolJSON := range toolCalls {
			var toolCall map[string]interface{}
			if err := json.Unmarshal([]byte(toolJSON), &toolCall); err != nil {
				continue
			}
			
			if name, ok := toolCall["name"].(string); ok && name == toolName {
				totalCount++
				// Check if this tool call appears to be successful
				// (In practice, we'd check execution results)
				if toolCall["args"] != nil {
					successCount++
				}
			}
		}
		
		if totalCount > 0 {
			return float64(successCount) / float64(totalCount), nil
		}
		
		// No usage of this tool
		return 0.0, nil
	}
}

// extractToolCalls extracts tool JSON from Smola-formatted response
func (r *SmolaToolRubric) extractToolCalls(response string) []string {
	var toolCalls []string
	
	// Parse with Smola parser
	parsed, err := r.parser.ParseSmola(response, true)
	if err != nil {
		return toolCalls
	}
	
	// Look for tool fields
	for field, content := range parsed.Fields {
		if field == "tool" && content != "" {
			toolCalls = append(toolCalls, content)
		}
	}
	
	// Also try to extract from raw response in case of multiple calls
	parts := strings.Split(response, "<tool>")
	for i := 1; i < len(parts); i++ {
		if endIdx := strings.Index(parts[i], "</tool>"); endIdx > 0 {
			toolJSON := strings.TrimSpace(parts[i][:endIdx])
			if toolJSON != "" && !contains(toolCalls, toolJSON) {
				toolCalls = append(toolCalls, toolJSON)
			}
		}
	}
	
	return toolCalls
}

// ComputeRewardWithTrace computes reward with execution trace
func (r *SmolaToolRubric) ComputeRewardWithTrace(ctx context.Context, parsed string, groundTruth string, trace []ToolExecution) (float64, error) {
	// Base reward computation
	baseReward, err := r.ComputeReward(ctx, parsed, groundTruth)
	if err != nil {
		return 0.0, err
	}
	
	// If we have execution trace, adjust tool usage scores
	if len(trace) > 0 && r.includeUsage {
		// Count successful executions per tool
		toolSuccess := make(map[string]float64)
		toolTotal := make(map[string]float64)
		
		for _, exec := range trace {
			toolTotal[exec.ToolName]++
			if exec.Success {
				toolSuccess[exec.ToolName]++
			}
		}
		
		// Update tool usage metrics based on actual execution
		for toolName, total := range toolTotal {
			if total > 0 {
				successRate := toolSuccess[toolName] / total
				// This would update the specific tool metric
				// In practice, we'd need a way to update individual metrics
				_ = successRate
			}
		}
	}
	
	return baseReward, nil
}

// ToolExecution represents a tool execution in the trace
type ToolExecution struct {
	ToolName string
	Args     map[string]interface{}
	Result   string
	Success  bool
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}