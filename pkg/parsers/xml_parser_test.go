package parsers

import (
	"context"
	"testing"
)

func TestXMLParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		fields      []interface{}
		answerField string
		input       string
		expected    string
		wantErr     bool
	}{
		{
			name:        "simple think and answer",
			fields:      []interface{}{"think", "answer"},
			answerField: "answer",
			input: `<think>
Let me calculate 2 + 2.
2 + 2 = 4
</think>
<answer>
4
</answer>`,
			expected: "4",
			wantErr:  false,
		},
		{
			name:        "answer with alternatives",
			fields:      []interface{}{"reasoning", []string{"solution", "answer"}},
			answerField: "answer",
			input: `<reasoning>
First, I need to solve this step by step.
</reasoning>
<answer>
42
</answer>`,
			expected: "42",
			wantErr:  false,
		},
		{
			name:        "missing answer field",
			fields:      []interface{}{"think", "answer"},
			answerField: "answer",
			input: `<think>
Some thinking but no answer tag
</think>`,
			expected: "",
			wantErr:  false,
		},
		{
			name:        "nested content",
			fields:      []interface{}{"code", "output"},
			answerField: "output",
			input: `<code>
def add(a, b):
    return a + b
</code>
<output>
Result: 10
</output>`,
			expected: "Result: 10",
			wantErr:  false,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewXMLParser(tt.fields, tt.answerField)
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			got, err := parser.Parse(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("Parse() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestXMLParser_Format(t *testing.T) {
	parser, err := NewXMLParser([]interface{}{"think", "answer"}, "answer")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	values := map[string]string{
		"think":  "Let me solve this step by step.",
		"answer": "42",
	}

	formatted, err := parser.Format(values)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	expected := `<think>
Let me solve this step by step.
</think>
<answer>
42
</answer>`

	if formatted != expected {
		t.Errorf("Format() = %v, want %v", formatted, expected)
	}
}

func TestXMLParser_GetFormatStr(t *testing.T) {
	tests := []struct {
		name     string
		fields   []interface{}
		expected string
	}{
		{
			name:   "simple fields",
			fields: []interface{}{"think", "answer"},
			expected: `<think>
...
</think>
<answer>
...
</answer>`,
		},
		{
			name:   "fields with alternatives",
			fields: []interface{}{"reasoning", []string{"code", "solution", "answer"}},
			expected: `<reasoning>
...
</reasoning>
<[ code | solution | answer ]>
...
</[ code | solution | answer ]>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewXMLParser(tt.fields, "answer")
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			got := parser.GetFormatStr()
			if got != tt.expected {
				t.Errorf("GetFormatStr() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestXMLParser_ParseXML(t *testing.T) {
	parser, err := NewXMLParser([]interface{}{"think", []string{"tool", "answer"}}, "answer")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	input := `<think>
I need to use a tool to calculate this.
</think>
<tool>
{"name": "calculate", "args": {"expression": "2 + 2"}}
</tool>`

	parsed, err := parser.ParseXML(input, true)
	if err != nil {
		t.Fatalf("ParseXML() error = %v", err)
	}

	if parsed.Fields["think"] != "I need to use a tool to calculate this." {
		t.Errorf("Expected think field to be parsed correctly")
	}

	if parsed.Fields["tool"] != `{"name": "calculate", "args": {"expression": "2 + 2"}}` {
		t.Errorf("Expected tool field to be parsed correctly")
	}

	if parsed.Fields["answer"] != "" {
		t.Errorf("Expected answer field to be empty")
	}
}

func TestXMLParser_Alternatives(t *testing.T) {
	parser, err := NewXMLParser([]interface{}{[]string{"solution", "answer", "result"}}, "answer")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Test that any alternative works
	inputs := []struct {
		xml      string
		expected string
	}{
		{
			xml:      "<solution>42</solution>",
			expected: "",
		},
		{
			xml:      "<answer>42</answer>",
			expected: "42",
		},
		{
			xml:      "<result>42</result>",
			expected: "",
		},
	}

	ctx := context.Background()
	for _, input := range inputs {
		got, err := parser.Parse(ctx, input.xml)
		if err != nil {
			t.Errorf("Parse() error = %v", err)
		}
		if got != input.expected {
			t.Errorf("Parse(%s) = %v, want %v", input.xml, got, input.expected)
		}
	}
}