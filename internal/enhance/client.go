// Package enhance calls the MiniMax M3 API to transform raw text into
// a structured numbered brief.
//
// No secrets are stored here. The API key is passed in at construction time.
package enhance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ─── Client ─────────────────────────────────────────────────────────────────

type Client struct {
	apiKey  string
	baseURL string
	model   string
	httpClient *http.Client
}

// New returns a client configured with the given credentials.
// If any argument is empty, New will use a sensible default.
func New(apiKey, baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = "https://api.gmi-serving.com/v1"
	}
	if model == "" {
		model = "MiniMaxAI/MiniMax-M3"
	}
	return &Client{
		apiKey:    apiKey,
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		model:     model,
		httpClient: &http.Client{Timeout: 90e9},
	}
}

// Enhance sends the raw selection to the model and returns the formatted brief.
func (c *Client) Enhance(ctx context.Context, text string) (string, error) {
	systemPrompt := `You are an expert writing assistant. Transform the user's raw text into a clean, numbered agent brief.

Format rules:
- Start each point with "1.", "2.", "3." etc (no other numbering)
- Each point must be self-contained and actionable
- Preserve the original intent exactly — no hallucinations
- Max 7 points total
- Output only the numbered list — no preamble, no follow-up offer
- Language: match the input language`

	payload := map[string]any{
		"model": c.model,
		"messages": []map[string]any{
			{"role": "system", "content": systemPrompt},
			{"role": "user",   "content": text},
		},
		"max_tokens": 1024,
		"temperature": 0.3,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(b))
	}

	var reply struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	if len(reply.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}
	return strings.TrimSpace(reply.Choices[0].Message.Content), nil
}
