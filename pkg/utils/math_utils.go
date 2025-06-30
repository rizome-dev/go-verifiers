package utils

import (
	"regexp"
	"strconv"
	"strings"
)

// ExtractBoxedAnswer extracts content from \boxed{...} format
func ExtractBoxedAnswer(text string) string {
	// Find \boxed{
	boxedStart := strings.Index(text, "\\boxed{")
	if boxedStart == -1 {
		return text
	}

	// Find matching brace
	contentStart := boxedStart + 7 // len("\\boxed{")
	count := 1
	i := contentStart
	
	for i < len(text) && count > 0 {
		if text[i] == '{' {
			count++
		} else if text[i] == '}' {
			count--
		}
		i++
	}
	
	if count == 0 {
		return text[contentStart : i-1]
	}
	
	return text
}

// ExtractHashAnswer extracts answer after #### marker
func ExtractHashAnswer(text string) string {
	if !strings.Contains(text, "####") {
		return text
	}
	parts := strings.Split(text, "####")
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return text
}

// StripNonNumeric removes all non-numeric characters except dots
func StripNonNumeric(text string) string {
	var result strings.Builder
	for _, r := range text {
		if (r >= '0' && r <= '9') || r == '.' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// NormalizeNumber attempts to normalize numeric answers for comparison
func NormalizeNumber(text string) string {
	// Remove common formatting
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, "$", "")
	text = strings.ReplaceAll(text, " ", "")
	
	// Try to parse as number and format consistently
	if f, err := strconv.ParseFloat(text, 64); err == nil {
		// Format to remove trailing zeros
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	
	return text
}

// ExtractFirstNumber extracts the first number from text
func ExtractFirstNumber(text string) string {
	re := regexp.MustCompile(`-?\d+\.?\d*`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// CompareMathAnswers performs fuzzy comparison of mathematical answers
func CompareMathAnswers(answer1, answer2 string) bool {
	// Direct string comparison
	if answer1 == answer2 {
		return true
	}
	
	// Normalize and compare
	norm1 := NormalizeNumber(answer1)
	norm2 := NormalizeNumber(answer2)
	
	if norm1 == norm2 {
		return true
	}
	
	// Try numeric comparison
	num1, err1 := strconv.ParseFloat(norm1, 64)
	num2, err2 := strconv.ParseFloat(norm2, 64)
	
	if err1 == nil && err2 == nil {
		// Compare with small epsilon for floating point
		epsilon := 1e-9
		return num1-num2 < epsilon && num2-num1 < epsilon
	}
	
	return false
}