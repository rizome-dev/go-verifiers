package parsers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// SmolaParser parses XML-formatted responses with tool support
type SmolaParser struct {
	fields []XMLField
}

// ParsedSmola represents the result of Smola parsing
type ParsedSmola struct {
	Fields   map[string]string      // Map of field name to content
	ToolJSON map[string]interface{} // Parsed JSON if tool field exists
}

// NewSmolaParser creates a new Smola parser with field definitions
func NewSmolaParser(fields []interface{}) (*SmolaParser, error) {
	parser := &SmolaParser{
		fields: make([]XMLField, 0),
	}

	seen := make(map[string]bool)
	for _, field := range fields {
		var xmlField XMLField

		switch f := field.(type) {
		case string:
			xmlField.Canonical = f
			xmlField.Alternatives = []string{f}
		case []string:
			if len(f) == 0 {
				return nil, fmt.Errorf("field array cannot be empty")
			}
			xmlField.Canonical = f[0]
			xmlField.Alternatives = f
		default:
			return nil, fmt.Errorf("each field must be a string or array of strings")
		}

		if seen[xmlField.Canonical] {
			return nil, fmt.Errorf("duplicate field name: %s", xmlField.Canonical)
		}
		seen[xmlField.Canonical] = true
		parser.fields = append(parser.fields, xmlField)
	}

	return parser, nil
}

// Parse extracts the final field content (usually answer or result)
func (p *SmolaParser) Parse(ctx context.Context, response string) (string, error) {
	parsed, err := p.ParseSmola(response, true)
	if err != nil {
		return "", err
	}

	// Return the last field's content if it exists
	if len(p.fields) > 0 {
		lastField := p.fields[len(p.fields)-1]
		for _, alt := range lastField.Alternatives {
			if val, ok := parsed.Fields[alt]; ok && val != "" {
				return val, nil
			}
		}
	}

	return "", nil
}

// ParseSmola parses XML and returns structured data with tool support
func (p *SmolaParser) ParseSmola(text string, strip bool) (*ParsedSmola, error) {
	result := &ParsedSmola{
		Fields:   make(map[string]string),
		ToolJSON: make(map[string]interface{}),
	}

	for _, field := range p.fields {
		// Check each alternative tag name
		for _, alt := range field.Alternatives {
			// Create regex pattern for the tag
			pattern := fmt.Sprintf(`<%s>\s*(.*?)\s*</%s>`, alt, alt)
			re, err := regexp.Compile("(?s)" + pattern) // (?s) makes . match newlines
			if err != nil {
				return nil, fmt.Errorf("failed to compile regex: %w", err)
			}

			matches := re.FindStringSubmatch(text)
			if len(matches) > 1 {
				content := matches[1]
				if strip {
					content = strings.TrimSpace(content)
				}
				result.Fields[alt] = content

				// If this is a tool field, try to parse as JSON
				if alt == "tool" && content != "" {
					var toolData map[string]interface{}
					if err := json.Unmarshal([]byte(content), &toolData); err == nil {
						result.ToolJSON = toolData
					}
				}
			}
		}
	}

	return result, nil
}

// ParseWithTracking returns parsed content with metadata
func (p *SmolaParser) ParseWithTracking(ctx context.Context, response string) (string, map[string]interface{}, error) {
	parsed, err := p.ParseSmola(response, true)
	if err != nil {
		return "", nil, err
	}

	answer := ""
	// Get the last field's content
	if len(p.fields) > 0 {
		lastField := p.fields[len(p.fields)-1]
		for _, alt := range lastField.Alternatives {
			if val, ok := parsed.Fields[alt]; ok && val != "" {
				answer = val
				break
			}
		}
	}

	metadata := map[string]interface{}{
		"parser_type":  "smola",
		"fields_found": len(parsed.Fields),
		"has_tool":     len(parsed.ToolJSON) > 0,
		"all_fields":   parsed.Fields,
	}

	if len(parsed.ToolJSON) > 0 {
		metadata["tool_json"] = parsed.ToolJSON
	}

	return answer, metadata, nil
}

// GetFormatStr returns a string describing the expected format
func (p *SmolaParser) GetFormatStr() string {
	var parts []string
	for _, field := range p.fields {
		if len(field.Alternatives) > 1 {
			options := strings.Join(field.Alternatives, " | ")
			parts = append(parts, fmt.Sprintf("<[ %s ]>\n...\n</[ %s ]>", options, options))
		} else {
			parts = append(parts, fmt.Sprintf("<%s>\n...\n</%s>", field.Canonical, field.Canonical))
		}
	}
	return strings.Join(parts, "\n")
}

// Format creates an XML string from provided values
func (p *SmolaParser) Format(values map[string]interface{}) (string, error) {
	var parts []string
	
	for _, field := range p.fields {
		var value string
		found := false

		// Check canonical name first
		if val, ok := values[field.Canonical]; ok {
			found = true
			switch v := val.(type) {
			case string:
				value = v
			case map[string]interface{}, []interface{}:
				// For tool fields, marshal to JSON
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					return "", fmt.Errorf("failed to marshal %s to JSON: %w", field.Canonical, err)
				}
				value = string(jsonBytes)
			default:
				value = fmt.Sprintf("%v", v)
			}
		} else {
			// Check alternatives
			for _, alt := range field.Alternatives {
				if val, ok := values[alt]; ok {
					found = true
					switch v := val.(type) {
					case string:
						value = v
					case map[string]interface{}, []interface{}:
						jsonBytes, err := json.Marshal(v)
						if err != nil {
							return "", fmt.Errorf("failed to marshal %s to JSON: %w", alt, err)
						}
						value = string(jsonBytes)
					default:
						value = fmt.Sprintf("%v", v)
					}
					break
				}
			}
		}

		if !found {
			return "", fmt.Errorf("missing value for field '%s' (allowed: %v)", 
				field.Canonical, field.Alternatives)
		}

		// Use canonical name for formatting
		parts = append(parts, fmt.Sprintf("<%s>\n%s\n</%s>", 
			field.Canonical, value, field.Canonical))
	}

	return strings.Join(parts, "\n"), nil
}

// GetFields returns the canonical field names in order
func (p *SmolaParser) GetFields() []string {
	fields := make([]string, len(p.fields))
	for i, field := range p.fields {
		fields[i] = field.Canonical
	}
	return fields
}

// FollowsFormat checks if the message follows expected Smola format
func (p *SmolaParser) FollowsFormat(text string) float64 {
	parsed, _ := p.ParseSmola(text, true)
	parsedNoStrip, _ := p.ParseSmola(text, false)
	
	if parsed == nil || parsedNoStrip == nil {
		return 0.0
	}

	score := 0.0
	expectedFieldCount := len(p.fields)
	presentFieldSets := make(map[int]bool)
	hasCorrectSpacing := true

	// Check which fields are present
	for i, field := range p.fields {
		fieldSetPresent := false
		for _, alt := range field.Alternatives {
			if val, ok := parsed.Fields[alt]; ok && val != "" {
				fieldSetPresent = true
				
				// Check spacing
				if valNoStrip, ok := parsedNoStrip.Fields[alt]; !ok || valNoStrip == "" {
					hasCorrectSpacing = false
				}
			}
		}
		if fieldSetPresent {
			presentFieldSets[i] = true
		}
	}

	// Calculate score components
	if len(presentFieldSets) > 0 {
		fieldSetRatio := float64(len(presentFieldSets)) / float64(expectedFieldCount)
		score += 0.4 * fieldSetRatio
	}

	if hasCorrectSpacing {
		score += 0.2
	}

	// Check if starts with first field
	trimmed := strings.TrimSpace(text)
	if len(p.fields) > 0 {
		for _, alt := range p.fields[0].Alternatives {
			if strings.HasPrefix(trimmed, fmt.Sprintf("<%s>", alt)) {
				score += 0.2
				break
			}
		}
	}

	// Check if ends with last field
	if len(p.fields) > 0 {
		lastField := p.fields[len(p.fields)-1]
		for _, alt := range lastField.Alternatives {
			if strings.HasSuffix(trimmed, fmt.Sprintf("</%s>", alt)) {
				score += 0.2
				break
			}
		}
	}

	return score
}