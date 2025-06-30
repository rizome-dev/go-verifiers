package envs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/prompts"
	"github.com/rizome-dev/go-verifiers/pkg/rubrics"
	"github.com/rizome-dev/go-verifiers/pkg/tools"
	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// SmolaToolEnv implements SmolaAgents-style tool environment
type SmolaToolEnv struct {
	*MultiTurnEnv
	Tools           map[string]tools.Tool
	ToolSchemas     []tools.ToolSchema
	Parser          *parsers.SmolaParser
	EnvParser       *parsers.XMLParser
	ExcludeFewShot  bool
}

// NewSmolaToolEnv creates a new Smola tool environment
func NewSmolaToolEnv(config types.Config, toolList []tools.Tool, maxTurns int) (*SmolaToolEnv, error) {
	// Create parsers - Smola uses different field structure
	parser, err := parsers.NewSmolaParser([]interface{}{"think", "tool", "answer"})
	if err != nil {
		return nil, err
	}
	
	envParser, err := parsers.NewXMLParser([]interface{}{"result"}, "result")
	if err != nil {
		return nil, err
	}
	
	// Build tool map and schemas
	toolMap := make(map[string]tools.Tool)
	schemas := make([]tools.ToolSchema, 0, len(toolList))
	
	for _, tool := range toolList {
		toolMap[tool.Name()] = tool
		schemas = append(schemas, tool.Schema())
	}
	
	// Format system prompt with tool descriptions
	if config.SystemPrompt == "" {
		config.SystemPrompt = prompts.DefaultSmolaPromptTemplate
	}
	
	toolDescriptions := tools.FormatToolDescriptions(toolList)
	config.SystemPrompt = strings.ReplaceAll(config.SystemPrompt, "%s", toolDescriptions)
	
	env := &SmolaToolEnv{
		MultiTurnEnv:   NewMultiTurnEnv(config, maxTurns),
		Tools:          toolMap,
		ToolSchemas:    schemas,
		Parser:         parser,
		EnvParser:      envParser,
		ExcludeFewShot: false,
	}
	
	// Set parser and rubric
	env.SetParser(parser)
	
	// Create Smola tool rubric
	smolaRubric, err := rubrics.NewSmolaToolRubric(toolList, parser, envParser)
	if err != nil {
		return nil, err
	}
	env.SetRubric(smolaRubric)
	
	return env, nil
}

// IsCompleted checks if the task is completed
func (e *SmolaToolEnv) IsCompleted(ctx context.Context, messages []types.Message, state map[string]interface{}) bool {
	if len(messages) == 0 {
		return false
	}
	
	// Count tool usage steps (excluding few-shot)
	toolSteps := 0
	startCounting := false
	
	for _, msg := range messages {
		// Start counting after few-shot examples
		if !startCounting && msg.Role == "user" && !e.isFewShotMessage(msg) {
			startCounting = true
		}
		
		if startCounting && msg.Role == "assistant" {
			parsed, err := e.Parser.ParseSmola(msg.Content, true)
			if err == nil {
				// Check if this is a tool call
				if parsed.Fields["tool"] != "" {
					toolSteps++
				}
				// Check if we have an answer
				if parsed.Fields["answer"] != "" {
					return true
				}
			}
		}
	}
	
	// Track tool steps in state
	state["tool_steps"] = toolSteps
	
	return false
}

// EnvResponse generates environment response to tool calls
func (e *SmolaToolEnv) EnvResponse(ctx context.Context, messages []types.Message, state map[string]interface{}) (types.Message, map[string]interface{}, error) {
	if len(messages) == 0 {
		return types.Message{}, state, fmt.Errorf("no messages to process")
	}
	
	// Get last assistant message
	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "assistant" {
		return types.Message{}, state, fmt.Errorf("last message must be from assistant")
	}
	
	// Parse for tool call
	parsed, err := e.Parser.ParseSmola(lastMsg.Content, true)
	if err != nil {
		return types.Message{
			Role:    "user",
			Content: e.formatError("Failed to parse response. Please use the correct XML format."),
		}, state, nil
	}
	
	// Check if there's a tool call
	toolJSON := parsed.Fields["tool"]
	if toolJSON == "" {
		return types.Message{
			Role:    "user", 
			Content: e.formatError("No tool call found. Use <tool>{json}</tool> to call a tool."),
		}, state, nil
	}
	
	// Execute tool call
	result := e.callTool(ctx, toolJSON, 1024)
	
	// Track tool execution
	if state["tool_executions"] == nil {
		state["tool_executions"] = []rubrics.ToolExecution{}
	}
	
	executions := state["tool_executions"].([]rubrics.ToolExecution)
	
	// Parse tool call to track execution
	var toolCall map[string]interface{}
	success := true
	toolName := "unknown"
	
	if err := json.Unmarshal([]byte(toolJSON), &toolCall); err == nil {
		if name, ok := toolCall["name"].(string); ok {
			toolName = name
		}
		if strings.HasPrefix(result, "Error:") {
			success = false
		}
	} else {
		success = false
	}
	
	executions = append(executions, rubrics.ToolExecution{
		ToolName: toolName,
		Args:     toolCall["args"].(map[string]interface{}),
		Result:   result,
		Success:  success,
	})
	state["tool_executions"] = executions
	
	// Format result as XML
	response := fmt.Sprintf("<result>\n%s\n</result>", result)
	
	return types.Message{
		Role:    "user",
		Content: response,
	}, state, nil
}

// callTool executes a tool based on JSON command
func (e *SmolaToolEnv) callTool(ctx context.Context, toolJSON string, maxChars int) string {
	// Parse tool call
	toolCall, err := tools.ParseToolCall(toolJSON)
	if err != nil {
		return fmt.Sprintf("Error: %v. Please format your tool call as '{\"name\": \"tool_name\", \"args\": {\"arg1\": \"value1\"}}'", err)
	}
	
	// Execute tool
	return tools.ExecuteTool(ctx, e.Tools, toolCall, maxChars)
}

// formatError formats an error message as XML
func (e *SmolaToolEnv) formatError(msg string) string {
	return fmt.Sprintf("<result>\n%s\n</result>", msg)
}

// isFewShotMessage checks if a message is part of few-shot examples
func (e *SmolaToolEnv) isFewShotMessage(msg types.Message) bool {
	if !e.ExcludeFewShot {
		return false
	}
	
	// Check if this message matches any few-shot example
	// This is a simplified check - in practice, we'd compare with actual few-shot examples
	return false
}

// Rollout performs the Smola tool environment rollout
func (e *SmolaToolEnv) Rollout(ctx context.Context, client types.Client, model string, prompt interface{}, answer string, samplingArgs types.SamplingArgs) (*types.Rollout, error) {
	rollout, err := BaseMultiTurnRollout(ctx, e, client, model, prompt, answer, samplingArgs, e.MaxTurns)
	if err != nil {
		return nil, err
	}
	
	// Enhanced scoring with execution trace
	if smolaRubric, ok := e.rubric.(*rubrics.SmolaToolRubric); ok {
		// Extract execution trace from state
		var trace []rubrics.ToolExecution
		// This is simplified - in practice we'd track actual executions from state
		
		score, err := smolaRubric.ComputeRewardWithTrace(ctx, rollout.Response, answer, trace)
		if err == nil {
			rollout.Score = score
		}
	}
	
	return rollout, nil
}