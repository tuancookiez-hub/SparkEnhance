package enhance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestEnhanceRoundTrip spins up a fake GMI server and verifies the client
// pipeline: auth header, payload shape, response parsing, cleaner + scorer.
func TestEnhanceRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate the request.
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("auth header missing or wrong format")
		}
		var body struct {
			Model    string `json:"model"`
			Messages []map[string]any `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Model != modelName {
			t.Errorf("model = %s, want %s", body.Model, modelName)
		}
		if len(body.Messages) != 2 {
			t.Errorf("messages = %d, want 2", len(body.Messages))
		}
		if body.Messages[0]["role"] != "system" {
			t.Errorf("first message role = %v, want system", body.Messages[0]["role"])
		}

		// Respond with a fake M3 output.
		resp := gmiChatResponse{
			ID:      "test",
			Object:  "chat.completion",
			Model:   modelName,
			Choices: []struct {
				Index        int `json:"index"`
				Message      struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{{
				Index: 0,
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: "Goal: ship a new endpoint\n\n1. Add a POST /login handler to the user controller\n2. Validate credentials against the users table\n3. Return a JWT on success with a 30-minute expiry\n4. Write pytest cases for the happy path and bad-password case",
				},
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	// Override the base URL by re-creating the client with a custom one.
	c := NewClient("test-key-fake")
	c.baseURL = srv.URL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := c.Enhance(ctx, "fix the login flow")
	if err != nil {
		t.Fatalf("Enhance: %v", err)
	}
	if !strings.HasPrefix(res.Output, "Goal:") {
		t.Errorf("output missing goal line: %q", res.Output)
	}
	if res.Score < 70 {
		t.Errorf("score = %d, want >= 70 (output has goal + numbered + specifics)", res.Score)
	}
}

// TestValidateKey exercises the /models validator.
func TestValidateKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ok-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	c := NewClient("ok-key")
	c.baseURL = srv.URL
	if err := c.ValidateKey(context.Background()); err != nil {
		t.Errorf("ValidateKey ok-key: %v", err)
	}
	c2 := NewClient("bad-key")
	c2.baseURL = srv.URL
	if err := c2.ValidateKey(context.Background()); err == nil {
		t.Errorf("ValidateKey bad-key: expected error, got nil")
	}
}
