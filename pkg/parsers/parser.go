package parsers

import (
	"context"
	"strings"
)

// Parser is the interface for parsing model outputs
type Parser interface {
	// Parse extracts the final answer from model output
	Parse(ctx context.Context, response string) (string, error)
	
	// ParseWithTracking extracts answer and tracks parsing metadata
	ParseWithTracking(ctx context.Context, response string) (string, map[string]interface{}, error)
}

// BaseParser provides a default implementation that returns the response as-is
type BaseParser struct{}

// NewBaseParser creates a new base parser
func NewBaseParser() *BaseParser {
	return &BaseParser{}
}

// Parse returns the response with whitespace trimmed
func (p *BaseParser) Parse(ctx context.Context, response string) (string, error) {
	return strings.TrimSpace(response), nil
}

// ParseWithTracking returns the response with metadata
func (p *BaseParser) ParseWithTracking(ctx context.Context, response string) (string, map[string]interface{}, error) {
	parsed, err := p.Parse(ctx, response)
	if err != nil {
		return "", nil, err
	}
	
	metadata := map[string]interface{}{
		"original_length": len(response),
		"parsed_length":   len(parsed),
		"parser_type":     "base",
	}
	
	return parsed, metadata, nil
}

// RegexParser parses responses using regular expressions
type RegexParser struct {
	pattern string
}

// NewRegexParser creates a parser that extracts content matching a regex pattern
func NewRegexParser(pattern string) *RegexParser {
	return &RegexParser{pattern: pattern}
}

// LastLineParser extracts the last non-empty line
type LastLineParser struct{}

// NewLastLineParser creates a parser that returns the last non-empty line
func NewLastLineParser() *LastLineParser {
	return &LastLineParser{}
}

// Parse returns the last non-empty line
func (p *LastLineParser) Parse(ctx context.Context, response string) (string, error) {
	lines := strings.Split(response, "\n")
	
	// Find last non-empty line
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line, nil
		}
	}
	
	return "", nil
}

// ParseWithTracking returns the last line with metadata
func (p *LastLineParser) ParseWithTracking(ctx context.Context, response string) (string, map[string]interface{}, error) {
	parsed, err := p.Parse(ctx, response)
	if err != nil {
		return "", nil, err
	}
	
	lines := strings.Split(response, "\n")
	metadata := map[string]interface{}{
		"total_lines":   len(lines),
		"parser_type":   "last_line",
		"parsed_length": len(parsed),
	}
	
	return parsed, metadata, nil
}