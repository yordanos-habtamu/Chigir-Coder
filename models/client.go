package models

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// Client wraps the OpenAI-compatible API for LLM calls.
type Client struct {
	api       *openai.Client
	model     string
	maxTokens int
}

// NewClient creates a new LLM client configured for OpenRouter or any
// OpenAI-compatible endpoint.
func NewClient(baseURL, apiKey, model string, maxTokens int) *Client {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = normalizeBaseURL(baseURL)
	return &Client{
		api:       openai.NewClientWithConfig(cfg),
		model:     model,
		maxTokens: maxTokens,
	}
}

// Chat sends a system + user message pair and returns the assistant response.
func (c *Client) Chat(systemPrompt, userPrompt string) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		resp, err := c.api.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:     c.model,
				MaxTokens: c.maxTokens,
				Messages: []openai.ChatCompletionMessage{
					{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
					{Role: openai.ChatMessageRoleUser, Content: userPrompt},
				},
			},
		)
		if err != nil {
			lastErr = err
			if attempt == 1 && isRetryable(err) {
				continue
			}
			return "", fmt.Errorf("LLM call failed: %w", err)
		}
		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("LLM returned no choices")
		}
		return strings.TrimSpace(resp.Choices[0].Message.Content), nil
	}
	return "", fmt.Errorf("LLM call failed after retry: %w", lastErr)
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "504") ||
		strings.Contains(msg, "gateway timeout") ||
		strings.Contains(msg, "unexpected end of json input")
}

// normalizeBaseURL allows users to pass either the API root (..../v1)
// or the full chat completions endpoint (..../v1/chat/completions).
func normalizeBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return trimmed
	}
	const suffix = "/chat/completions"
	if strings.HasSuffix(trimmed, suffix) {
		return strings.TrimSuffix(trimmed, suffix)
	}
	return trimmed
}
