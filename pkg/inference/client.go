package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rizome-dev/go-verifiers/pkg/types"
)

// HTTPClient implements the types.Client interface using HTTP
type HTTPClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewHTTPClient creates a new HTTP-based inference client
func NewHTTPClient(baseURL string, apiKey string) *HTTPClient {
	if baseURL == "" {
		baseURL = "http://localhost:8000/v1"
	}
	if apiKey == "" {
		apiKey = "local"
	}

	return &HTTPClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// ChatCompletionRequest represents the request structure for chat completions
type ChatCompletionRequest struct {
	Model       string                 `json:"model"`
	Messages    []types.Message        `json:"messages"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	N           int                    `json:"n,omitempty"`
	Stop        []string               `json:"stop,omitempty"`
	ExtraBody   map[string]interface{} `json:"extra_body,omitempty"`
}

// CompletionRequest represents the request structure for completions
type CompletionRequest struct {
	Model       string                 `json:"model"`
	Prompt      string                 `json:"prompt"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	N           int                    `json:"n,omitempty"`
	Stop        []string               `json:"stop,omitempty"`
	ExtraBody   map[string]interface{} `json:"extra_body,omitempty"`
}

// ChatCompletionResponse represents the response from chat completion
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int             `json:"index"`
		Message      types.Message   `json:"message"`
		FinishReason string          `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CompletionResponse represents the response from completion
type CompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CreateChatCompletion creates a chat completion
func (c *HTTPClient) CreateChatCompletion(ctx context.Context, model string, messages []types.Message, args types.SamplingArgs) (string, error) {
	req := ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		Temperature: args.Temperature,
		MaxTokens:   args.MaxTokens,
		TopP:        args.TopP,
		N:           args.N,
		Stop:        args.Stop,
		ExtraBody:   args.ExtraBody,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for context length error
		if resp.StatusCode == http.StatusBadRequest && bytes.Contains(body, []byte("context_length_exceeded")) {
			return "[ERROR] context_length_exceeded", nil
		}
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	// Check if generation was truncated
	if chatResp.Choices[0].FinishReason == "length" {
		return "[ERROR] max_tokens_reached", nil
	}

	return chatResp.Choices[0].Message.Content, nil
}

// CreateCompletion creates a text completion
func (c *HTTPClient) CreateCompletion(ctx context.Context, model string, prompt string, args types.SamplingArgs) (string, error) {
	req := CompletionRequest{
		Model:       model,
		Prompt:      prompt,
		Temperature: args.Temperature,
		MaxTokens:   args.MaxTokens,
		TopP:        args.TopP,
		N:           args.N,
		Stop:        args.Stop,
		ExtraBody:   args.ExtraBody,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for context length error
		if resp.StatusCode == http.StatusBadRequest && bytes.Contains(body, []byte("context_length_exceeded")) {
			return "[ERROR] context_length_exceeded", nil
		}
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var compResp CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&compResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(compResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	// Check if generation was truncated
	if compResp.Choices[0].FinishReason == "length" {
		return "[ERROR] max_tokens_reached", nil
	}

	return compResp.Choices[0].Text, nil
}

// CheckServer checks if the server is available
func (c *HTTPClient) CheckServer(ctx context.Context, totalTimeout time.Duration, retryInterval time.Duration) error {
	if retryInterval == 0 {
		retryInterval = 2 * time.Second
	}

	deadline := time.Now().Add(totalTimeout)
	
	for {
		req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/models", nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.HTTPClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("server not available after %v", totalTimeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
			continue
		}
	}
}