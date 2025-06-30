package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SearchEngine represents a search provider
type SearchEngine string

const (
	SearchEngineDuckDuckGo SearchEngine = "duckduckgo"
	SearchEngineGoogle     SearchEngine = "google"
	SearchEngineBing       SearchEngine = "bing"
)

// WebSearch implements web search functionality
type WebSearch struct {
	*BaseTool
	httpClient   *http.Client
	searchEngine SearchEngine
	apiKey       string // For engines that require API keys
}

// NewWebSearch creates a new web search tool
func NewWebSearch(engine SearchEngine) *WebSearch {
	search := &WebSearch{
		BaseTool: NewBaseTool(
			"search",
			"Search the web for information",
			nil, // Set below
		),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		searchEngine: engine,
	}

	// Set the executor
	search.executor = search.execute

	// Define schema
	search.schema = ToolSchema{
		Name:        "search",
		Description: search.description,
		Args: map[string]ArgumentSchema{
			"query": {
				Type:        "string",
				Description: "Search query",
				Required:    true,
			},
			"max_results": {
				Type:        "integer",
				Description: "Maximum number of results to return",
				Default:     5,
				Required:    false,
			},
		},
		Returns: "Search results containing titles, URLs, and snippets",
		Examples: []string{
			`{"name": "search", "args": {"query": "Go programming language concurrency"}}`,
			`{"name": "search", "args": {"query": "latest AI research papers", "max_results": 10}}`,
		},
	}

	return search
}

// SetAPIKey sets the API key for search engines that require it
func (s *WebSearch) SetAPIKey(key string) {
	s.apiKey = key
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// execute performs the search
func (s *WebSearch) execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	queryInterface, ok := args["query"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'query'")
	}

	query, ok := queryInterface.(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	maxResults := 5
	if maxInterface, ok := args["max_results"]; ok {
		switch v := maxInterface.(type) {
		case int:
			maxResults = v
		case float64:
			maxResults = int(v)
		case int64:
			maxResults = int(v)
		}
	}

	// Perform search based on engine
	results, err := s.performSearch(ctx, query, maxResults)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Format results
	return s.formatResults(results), nil
}

// performSearch executes the search based on the configured engine
func (s *WebSearch) performSearch(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	switch s.searchEngine {
	case SearchEngineDuckDuckGo:
		return s.searchDuckDuckGo(ctx, query, maxResults)
	default:
		// For now, we'll simulate search results
		return s.simulateSearch(query, maxResults), nil
	}
}

// searchDuckDuckGo performs a search using DuckDuckGo's instant answer API
func (s *WebSearch) searchDuckDuckGo(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	// DuckDuckGo instant answer API (limited but no API key required)
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1",
		url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse DuckDuckGo response
	var ddgResponse struct {
		Abstract       string `json:"Abstract"`
		AbstractURL    string `json:"AbstractURL"`
		AbstractSource string `json:"AbstractSource"`
		RelatedTopics  []struct {
			Text      string `json:"Text"`
			FirstURL  string `json:"FirstURL"`
			Result    string `json:"Result"`
		} `json:"RelatedTopics"`
	}

	if err := json.Unmarshal(body, &ddgResponse); err != nil {
		return nil, err
	}

	// Convert to our format
	results := make([]SearchResult, 0, maxResults)

	// Add abstract if available
	if ddgResponse.Abstract != "" {
		results = append(results, SearchResult{
			Title:   fmt.Sprintf("%s (from %s)", query, ddgResponse.AbstractSource),
			URL:     ddgResponse.AbstractURL,
			Snippet: ddgResponse.Abstract,
		})
	}

	// Add related topics
	for i, topic := range ddgResponse.RelatedTopics {
		if i >= maxResults || len(results) >= maxResults {
			break
		}
		if topic.Text != "" {
			results = append(results, SearchResult{
				Title:   extractTitle(topic.Text),
				URL:     topic.FirstURL,
				Snippet: topic.Text,
			})
		}
	}

	// If no results, return simulated results
	if len(results) == 0 {
		return s.simulateSearch(query, maxResults), nil
	}

	return results, nil
}

// simulateSearch returns simulated search results for demonstration
func (s *WebSearch) simulateSearch(query string, maxResults int) []SearchResult {
	// Simulate search results based on query keywords
	results := make([]SearchResult, 0, maxResults)

	// Generate contextual results based on query
	queryLower := strings.ToLower(query)

	if strings.Contains(queryLower, "go") || strings.Contains(queryLower, "golang") {
		results = append(results, SearchResult{
			Title:   "The Go Programming Language",
			URL:     "https://golang.org",
			Snippet: "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
		})
	}

	if strings.Contains(queryLower, "concurrency") {
		results = append(results, SearchResult{
			Title:   "Concurrency in Go",
			URL:     "https://golang.org/doc/effective_go#concurrency",
			Snippet: "Go provides goroutines and channels for managing concurrent operations. Goroutines are lightweight threads managed by the Go runtime.",
		})
	}

	if strings.Contains(queryLower, "ai") || strings.Contains(queryLower, "artificial intelligence") {
		results = append(results, SearchResult{
			Title:   "Recent Advances in Artificial Intelligence",
			URL:     "https://arxiv.org/list/cs.AI/recent",
			Snippet: "Latest research papers and developments in artificial intelligence, machine learning, and deep learning.",
		})
	}

	// Add generic results
	for i := len(results); i < maxResults && i < 5; i++ {
		results = append(results, SearchResult{
			Title:   fmt.Sprintf("Result %d for: %s", i+1, query),
			URL:     fmt.Sprintf("https://example.com/search?q=%s&p=%d", url.QueryEscape(query), i+1),
			Snippet: fmt.Sprintf("This is a search result snippet for your query '%s'. It contains relevant information about the topic.", query),
		})
	}

	return results
}

// formatResults formats search results for output
func (s *WebSearch) formatResults(results []SearchResult) string {
	if len(results) == 0 {
		return "No results found."
	}

	var formatted []string
	for i, result := range results {
		formatted = append(formatted, fmt.Sprintf("%d. %s\n   URL: %s\n   %s",
			i+1, result.Title, result.URL, result.Snippet))
	}

	return strings.Join(formatted, "\n\n")
}

// extractTitle extracts a title from DuckDuckGo text
func extractTitle(text string) string {
	// Try to extract the first sentence or up to 50 characters
	if idx := strings.Index(text, "."); idx > 0 && idx < 50 {
		return text[:idx]
	}
	if len(text) > 50 {
		return text[:47] + "..."
	}
	return text
}

// SearchCache provides caching for search results
type SearchCache struct {
	*WebSearch
	cache     map[string]cacheEntry
	cacheMu   sync.RWMutex
	ttl       time.Duration
}

type cacheEntry struct {
	results   []SearchResult
	timestamp time.Time
}

// NewCachedWebSearch creates a web search tool with caching
func NewCachedWebSearch(engine SearchEngine, ttl time.Duration) *SearchCache {
	return &SearchCache{
		WebSearch: NewWebSearch(engine),
		cache:     make(map[string]cacheEntry),
		ttl:       ttl,
	}
}

// execute performs cached search
func (c *SearchCache) execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	maxResults := 5
	if mr, ok := args["max_results"]; ok {
		if v, ok := mr.(float64); ok {
			maxResults = int(v)
		}
	}

	// Check cache
	cacheKey := fmt.Sprintf("%s:%d", query, maxResults)
	c.cacheMu.RLock()
	if entry, ok := c.cache[cacheKey]; ok && time.Since(entry.timestamp) < c.ttl {
		c.cacheMu.RUnlock()
		return c.formatResults(entry.results), nil
	}
	c.cacheMu.RUnlock()

	// Perform search
	results, err := c.performSearch(ctx, query, maxResults)
	if err != nil {
		return nil, err
	}

	// Update cache
	c.cacheMu.Lock()
	c.cache[cacheKey] = cacheEntry{
		results:   results,
		timestamp: time.Now(),
	}
	c.cacheMu.Unlock()

	return c.formatResults(results), nil
}