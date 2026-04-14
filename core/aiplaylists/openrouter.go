package aiplaylists

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/navidrome/navidrome/log"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

type LLMClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewLLMClient(apiKey, model string) *LLMClient {
	return &LLMClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *LLMClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	log.Debug(ctx, "Sending LLM request", "model", c.model, "systemLen", len(systemPrompt), "userLen", len(userPrompt))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request to OpenRouter: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenRouter returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("OpenRouter error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("OpenRouter returned no choices")
	}

	result := chatResp.Choices[0].Message.Content
	log.Debug(ctx, "LLM response received", "responseLen", len(result))
	return result, nil
}
