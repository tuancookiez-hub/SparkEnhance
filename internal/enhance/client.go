package enhance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	gmiBaseURL = "https://api.gmi-serving.com/v1"
	modelName  = "MiniMax-M3"
)

// Client calls the GMI Cloud OpenAI-compatible endpoint.
type Client struct {
	apiKey   string
	baseURL  string
	model    string
	httpCL   *http.Client
	timeout  time.Duration
}

// NewClient returns a GMI Cloud client. apiKey must be set before use.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: gmiBaseURL,
		model:   modelName,
		httpCL: &http.Client{
			Timeout: 60 * time.Second,
		},
		timeout: 90 * time.Second,
	}
}

// BaseURL returns the configured base URL (for tests/diagnostics).
func (c *Client) BaseURL() string { return c.baseURL }

// Model returns the configured model name (for tests/diagnostics).
func (c *Client) Model() string { return c.model }

// EnhanceRequest is the input to the enhance flow.
type EnhanceRequest struct {
	Input string
}

// EnhanceResult is the output of the enhance flow.
type EnhanceResult struct {
	Output  string
	Score   int
	Raw     string // un-cleaned output for debugging
}

// Enhance rewrites rawText into an agent-quality brief using M3.
// It returns the cleaned text, a quality score, and any error.
func (c *Client) Enhance(ctx context.Context, rawText string) (*EnhanceResult, error) {
	if strings.TrimSpace(rawText) == "" {
		return nil, fmt.Errorf("input text is empty")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	messages := []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": UserMessage(rawText)},
	}

	payload := map[string]any{
		"model":    c.model,
		"messages": messages,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpCL.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GMI API returned %d: %s", resp.StatusCode, string(b))
	}

	var gmiResp gmiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&gmiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	raw, err := gmiResp.Content()
	if err != nil {
		return nil, fmt.Errorf("extract content: %w", err)
	}

	cleaned := Clean(raw)
	if cleaned == "" {
		return nil, fmt.Errorf("model returned empty output")
	}

	return &EnhanceResult{
		Output: cleaned,
		Score:  Score(cleaned),
		Raw:    raw,
	}, nil
}

// gmiChatResponse mirrors the OpenAI chat/completions response shape.
type gmiChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int `json:"index"`
		Message      struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Content extracts the assistant message string from the GMI response.
func (r *gmiChatResponse) Content() (string, error) {
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return r.Choices[0].Message.Content, nil
}

// ValidateKey checks whether the API key is valid by hitting the models endpoint.
func (c *Client) ValidateKey(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpCL.Do(req)
	if err != nil {
		return fmt.Errorf("validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
