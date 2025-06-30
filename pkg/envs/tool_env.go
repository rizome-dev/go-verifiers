package envs

import (
	"context"
	"fmt"
	"strings"

	"github.com/rizome-dev/go-verifiers/pkg/parsers"
	"github.com/rizome-dev/go-verifiers/pkg/rubrics"
	"github.com/rizome-dev/go-verifiers/pkg/tools"
	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// ToolEnv implements a multi-turn environment with tool support
type ToolEnv struct {
	*MultiTurnEnv
	Tools       map[string]tools.Tool
	ToolSchemas []tools.ToolSchema
	Parser      *parsers.XMLParser
	EnvParser   *parsers.XMLParser
}

// NewToolEnv creates a new tool environment
func NewToolEnv(config types.Config, toolList []tools.Tool, maxTurns int) (*ToolEnv, error) {
	// Create parsers
	parser, err := parsers.NewXMLParser([]interface{}{"think", []string{"tool", "answer"}}, "answer")
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
		config.SystemPrompt = DefaultToolSystemPrompt
	}
	
	toolDescriptions := tools.FormatToolDescriptions(toolList)
	config.SystemPrompt = strings.ReplaceAll(config.SystemPrompt, "{tool_descriptions}", toolDescriptions)
	
	env := &ToolEnv{
		MultiTurnEnv: NewMultiTurnEnv(config, maxTurns),
		Tools:        toolMap,
		ToolSchemas:  schemas,
		Parser:       parser,
		EnvParser:    envParser,
	}
	
	// Set parser and rubric
	env.SetParser(parser)
	
	// Create tool rubric
	toolRubric, err := rubrics.NewToolRubric(toolList, parser, envParser)
	if err != nil {
		return nil, err
	}
	env.SetRubric(toolRubric)
	
	return env, nil
}

// IsCompleted checks if the task is completed
func (e *ToolEnv) IsCompleted(ctx context.Context, messages []types.Message, state map[string]interface{}) bool {
	// Check if we have an answer
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

// EnvResponse generates environment response to tool calls
func (e *ToolEnv) EnvResponse(ctx context.Context, messages []types.Message, state map[string]interface{}) (types.Message, map[string]interface{}, error) {
	if len(messages) == 0 {
		return types.Message{}, state, fmt.Errorf("no messages to process")
	}
	
	// Get last assistant message
	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "assistant" {
		return types.Message{}, state, fmt.Errorf("last message must be from assistant")
	}
	
	// Parse for tool call
	parsed, err := e.Parser.ParseXML(lastMsg.Content, true)
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
	
	// Format result as XML
	response := fmt.Sprintf("<result>\n%s\n</result>", result)
	
	return types.Message{
		Role:    "user",
		Content: response,
	}, state, nil
}

// callTool executes a tool based on JSON command
func (e *ToolEnv) callTool(ctx context.Context, toolJSON string, maxChars int) string {
	// Parse tool call
	toolCall, err := tools.ParseToolCall(toolJSON)
	if err != nil {
		return fmt.Sprintf("Error: %v. Please format your tool call as '{\"name\": \"tool_name\", \"args\": {\"arg1\": \"value1\"}}'", err)
	}
	
	// Execute tool
	return tools.ExecuteTool(ctx, e.Tools, toolCall, maxChars)
}

// formatError formats an error message as XML
func (e *ToolEnv) formatError(msg string) string {
	return fmt.Sprintf("<result>\n%s\n</result>", msg)
}

// Rollout performs the tool environment rollout
func (e *ToolEnv) Rollout(ctx context.Context, client types.Client, model string, prompt interface{}, answer string, samplingArgs types.SamplingArgs) (*types.Rollout, error) {
	return BaseMultiTurnRollout(ctx, e, client, model, prompt, answer, samplingArgs, e.MaxTurns)
}

// DefaultToolSystemPrompt is the default system prompt for tool environments
const DefaultToolSystemPrompt = `You are a helpful assistant with access to tools. You can use tools by wrapping your tool calls in XML tags.

Available tools:
{tool_descriptions}

To use a tool, format your request as:
<think>
...your reasoning...
</think>
<tool>
{"name": "tool_name", "args": {"arg1": "value1", "arg2": "value2"}}
</tool>

After receiving the tool result, you can either call another tool or provide your final answer:
<think>
...your reasoning...
</think>
<answer>
...your final answer...
</answer>

Always think before using tools or providing answers.`