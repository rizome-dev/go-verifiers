package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Tool represents a callable tool interface
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
	Schema() ToolSchema
}

// ToolSchema describes a tool's interface
type ToolSchema struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Args        map[string]ArgumentSchema    `json:"args"`
	Returns     string                       `json:"returns"`
	Examples    []string                     `json:"examples"`
}

// ArgumentSchema describes a single argument
type ArgumentSchema struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Required    bool        `json:"required"`
}

// ToolCall represents a JSON tool call
type ToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// BaseTool provides common tool functionality
type BaseTool struct {
	name        string
	description string
	schema      ToolSchema
	executor    func(context.Context, map[string]interface{}) (interface{}, error)
}

// NewBaseTool creates a new base tool
func NewBaseTool(name, description string, executor func(context.Context, map[string]interface{}) (interface{}, error)) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		executor:    executor,
		schema: ToolSchema{
			Name:        name,
			Description: description,
			Args:        make(map[string]ArgumentSchema),
			Returns:     "string",
			Examples:    []string{},
		},
	}
}

// Name returns the tool name
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns the tool description
func (t *BaseTool) Description() string {
	return t.description
}

// Execute runs the tool
func (t *BaseTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return t.executor(ctx, args)
}

// Schema returns the tool schema
func (t *BaseTool) Schema() ToolSchema {
	return t.schema
}

// SetSchema updates the tool schema
func (t *BaseTool) SetSchema(schema ToolSchema) {
	t.schema = schema
}

// FormatToolDescriptions formats tool schemas into a readable description
func FormatToolDescriptions(tools []Tool) string {
	var descriptions []string
	
	for _, tool := range tools {
		schema := tool.Schema()
		desc := []string{fmt.Sprintf("%s: %s", schema.Name, schema.Description)}
		
		if len(schema.Args) > 0 {
			desc = append(desc, "\nArguments:")
			for argName, argInfo := range schema.Args {
				defaultStr := ""
				if argInfo.Default != nil {
					defaultStr = fmt.Sprintf(" (default: %v)", argInfo.Default)
				}
				required := ""
				if argInfo.Required {
					required = " [required]"
				}
				desc = append(desc, fmt.Sprintf("  - %s: %s%s%s", 
					argName, argInfo.Description, defaultStr, required))
			}
		}
		
		if len(schema.Examples) > 0 {
			desc = append(desc, "\nExamples:")
			for _, example := range schema.Examples {
				desc = append(desc, fmt.Sprintf("  %s", example))
			}
		}
		
		if schema.Returns != "" {
			desc = append(desc, fmt.Sprintf("\nReturns: %s", schema.Returns))
		}
		
		descriptions = append(descriptions, strings.Join(desc, "\n"))
	}
	
	return strings.Join(descriptions, "\n\n")
}

// ParseToolCall parses a JSON tool call
func ParseToolCall(jsonStr string) (*ToolCall, error) {
	var call ToolCall
	if err := json.Unmarshal([]byte(jsonStr), &call); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	
	if call.Name == "" {
		return nil, fmt.Errorf("tool call must specify 'name'")
	}
	
	if call.Args == nil {
		call.Args = make(map[string]interface{})
	}
	
	return &call, nil
}

// ExecuteTool executes a tool by name with the given arguments
func ExecuteTool(ctx context.Context, tools map[string]Tool, toolCall *ToolCall, maxChars int) string {
	tool, exists := tools[toolCall.Name]
	if !exists {
		availableTools := make([]string, 0, len(tools))
		for name := range tools {
			availableTools = append(availableTools, name)
		}
		return fmt.Sprintf("Error: Unknown tool '%s'. Available tools: %s", 
			toolCall.Name, strings.Join(availableTools, ", "))
	}
	
	// Execute the tool
	result, err := tool.Execute(ctx, toolCall.Args)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	
	// Convert result to string
	resultStr := ""
	switch v := result.(type) {
	case string:
		resultStr = v
	case error:
		resultStr = fmt.Sprintf("Error: %v", v)
	default:
		// Try to marshal as JSON
		if jsonBytes, err := json.Marshal(result); err == nil {
			resultStr = string(jsonBytes)
		} else {
			resultStr = fmt.Sprintf("%v", result)
		}
	}
	
	// Truncate if needed
	if maxChars > 0 && len(resultStr) > maxChars {
		resultStr = resultStr[:maxChars] + "..."
	}
	
	return resultStr
}

// ValidateArgs validates tool arguments against the schema
func ValidateArgs(schema ToolSchema, args map[string]interface{}) error {
	// Check required arguments
	for argName, argSchema := range schema.Args {
		if argSchema.Required {
			if _, exists := args[argName]; !exists {
				return fmt.Errorf("missing required argument: %s", argName)
			}
		}
	}
	
	// Check argument types (basic validation)
	for argName, argValue := range args {
		argSchema, exists := schema.Args[argName]
		if !exists {
			continue // Allow extra arguments for flexibility
		}
		
		// Basic type checking
		valueType := reflect.TypeOf(argValue)
		switch argSchema.Type {
		case "string":
			if valueType.Kind() != reflect.String {
				return fmt.Errorf("argument %s must be a string", argName)
			}
		case "int", "integer":
			switch valueType.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Float32, reflect.Float64:
				// Allow numeric types
			default:
				return fmt.Errorf("argument %s must be a number", argName)
			}
		case "float", "number":
			switch valueType.Kind() {
			case reflect.Float32, reflect.Float64, reflect.Int, reflect.Int8, 
				reflect.Int16, reflect.Int32, reflect.Int64:
				// Allow numeric types
			default:
				return fmt.Errorf("argument %s must be a number", argName)
			}
		case "bool", "boolean":
			if valueType.Kind() != reflect.Bool {
				return fmt.Errorf("argument %s must be a boolean", argName)
			}
		}
	}
	
	return nil
}