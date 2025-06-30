package parsers

import (
	"context"
	"strings"
)

// ThinkParser extracts content after </think> tags
type ThinkParser struct {
	BaseParser
	extractFn func(string) string
}

// NewThinkParser creates a new think parser
func NewThinkParser() *ThinkParser {
	return &ThinkParser{
		extractFn: func(s string) string { return s }, // Default: identity function
	}
}

// NewThinkParserWithExtractor creates a think parser with custom extraction function
func NewThinkParserWithExtractor(extractFn func(string) string) *ThinkParser {
	return &ThinkParser{
		extractFn: extractFn,
	}
}

// Parse extracts content after </think> tag
func (p *ThinkParser) Parse(ctx context.Context, response string) (string, error) {
	text := response
	
	// If </think> exists, take everything after it
	if strings.Contains(text, "</think>") {
		parts := strings.Split(text, "</think>")
		if len(parts) > 1 {
			text = strings.TrimSpace(parts[len(parts)-1])
		}
	}
	
	// Apply extraction function
	return p.extractFn(strings.TrimSpace(text)), nil
}

// ParseWithTracking returns parsed content with metadata
func (p *ThinkParser) ParseWithTracking(ctx context.Context, response string) (string, map[string]interface{}, error) {
	parsed, err := p.Parse(ctx, response)
	if err != nil {
		return "", nil, err
	}
	
	metadata := map[string]interface{}{
		"parser_type":     "think",
		"has_think_tags":  strings.Contains(response, "<think>") && strings.Contains(response, "</think>"),
		"original_length": len(response),
		"parsed_length":   len(parsed),
	}
	
	return parsed, metadata, nil
}

// FollowsFormat checks if text follows the think format
func (p *ThinkParser) FollowsFormat(text string) bool {
	trimmed := strings.TrimSpace(text)
	
	// Check format requirements:
	// 1. Starts with <think>
	// 2. Exactly one <think> tag
	// 3. Exactly one </think> tag
	// 4. Has content after </think>
	if !strings.HasPrefix(trimmed, "<think>") {
		return false
	}
	
	if strings.Count(text, "<think>") != 1 {
		return false
	}
	
	if strings.Count(text, "</think>") != 1 {
		return false
	}
	
	parts := strings.Split(text, "</think>")
	if len(parts) < 2 || len(strings.TrimSpace(parts[1])) == 0 {
		return false
	}
	
	return true
}

// GetFormatStr returns the expected format
func (p *ThinkParser) GetFormatStr() string {
	return `<think>
...thinking process...
</think>
...final answer...`
}