package parsers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// XMLField represents a field definition in XML parsing
type XMLField struct {
	Canonical    string   // The canonical name (used for formatting)
	Alternatives []string // All allowed tag names (including canonical)
}

// XMLParser parses XML-formatted responses
type XMLParser struct {
	fields      []XMLField
	answerField string
}

// ParsedXML represents the result of XML parsing
type ParsedXML struct {
	Fields map[string]string // Map of field name to content
}

// NewXMLParser creates a new XML parser with field definitions
func NewXMLParser(fields []interface{}, answerField string) (*XMLParser, error) {
	parser := &XMLParser{
		fields:      make([]XMLField, 0),
		answerField: answerField,
	}

	if answerField == "" {
		parser.answerField = "answer"
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

// Parse extracts XML fields from the response
func (p *XMLParser) Parse(ctx context.Context, response string) (string, error) {
	parsed, err := p.ParseXML(response, true)
	if err != nil {
		return "", err
	}

	// Return the answer field if it exists
	if answer, ok := parsed.Fields[p.answerField]; ok {
		return answer, nil
	}

	// If no answer field, return empty string
	return "", nil
}

// ParseXML parses XML and returns structured data
func (p *XMLParser) ParseXML(text string, strip bool) (*ParsedXML, error) {
	result := &ParsedXML{
		Fields: make(map[string]string),
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
			}
		}
	}

	return result, nil
}

// ParseWithTracking returns parsed content with metadata
func (p *XMLParser) ParseWithTracking(ctx context.Context, response string) (string, map[string]interface{}, error) {
	parsed, err := p.ParseXML(response, true)
	if err != nil {
		return "", nil, err
	}

	answer := ""
	if val, ok := parsed.Fields[p.answerField]; ok {
		answer = val
	}

	metadata := map[string]interface{}{
		"parser_type":  "xml",
		"fields_found": len(parsed.Fields),
		"all_fields":   parsed.Fields,
	}

	return answer, metadata, nil
}

// GetFormatStr returns a string describing the expected XML format
func (p *XMLParser) GetFormatStr() string {
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
func (p *XMLParser) Format(values map[string]string) (string, error) {
	var parts []string
	
	for _, field := range p.fields {
		value := ""
		found := false

		// Check canonical name first
		if val, ok := values[field.Canonical]; ok {
			value = val
			found = true
		} else {
			// Check alternatives
			for _, alt := range field.Alternatives {
				if val, ok := values[alt]; ok {
					value = val
					found = true
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
func (p *XMLParser) GetFields() []string {
	fields := make([]string, len(p.fields))
	for i, field := range p.fields {
		fields[i] = field.Canonical
	}
	return fields
}

// HasField checks if a field name is valid (canonical or alternative)
func (p *XMLParser) HasField(name string) bool {
	for _, field := range p.fields {
		if field.Canonical == name {
			return true
		}
		for _, alt := range field.Alternatives {
			if alt == name {
				return true
			}
		}
	}
	return false
}