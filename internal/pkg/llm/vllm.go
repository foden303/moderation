package llm

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

// VLLMConfig contains configuration for vLLM server.
type VLLMConfig struct {
	BaseURL string // e.g., "http://localhost:8000" (vLLM default)
	Model   string // e.g., "meta-llama/Llama-Guard-3-1B"
	APIKey  string // Optional API key
	Timeout time.Duration
}

// DefaultVLLMConfig returns default configuration for local vLLM.
func DefaultVLLMConfig() VLLMConfig {
	return VLLMConfig{
		BaseURL: "http://localhost:8000",
		Model:   "meta-llama/Llama-Guard-3-1B",
		Timeout: 30 * time.Second,
	}
}

// VLLMClient is a client for vLLM server (OpenAI-compatible API).
type VLLMClient struct {
	config     VLLMConfig
	httpClient *http.Client
}

// NewVLLMClient creates a new vLLM client.
func NewVLLMClient(config VLLMConfig) *VLLMClient {
	return &VLLMClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// OpenAI-compatible request/response structures
type vllmChatRequest struct {
	Model       string        `json:"model"`
	Messages    []vllmMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type vllmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type vllmChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
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
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// ModerateText checks text content for safety violations using LlamaGuard via vLLM.
func (c *VLLMClient) ModerateText(ctx context.Context, text string) (*ModerationResult, error) {
	prompt := c.formatModerationPrompt(text, "")
	return c.callAPI(ctx, prompt)
}

// ModerateConversation checks both user input and assistant response.
func (c *VLLMClient) ModerateConversation(ctx context.Context, userInput, assistantResponse string) (*ModerationResult, error) {
	prompt := c.formatModerationPrompt(userInput, assistantResponse)
	return c.callAPI(ctx, prompt)
}

// formatModerationPrompt formats the prompt for LlamaGuard.
func (c *VLLMClient) formatModerationPrompt(userInput, assistantResponse string) string {
	if assistantResponse == "" {
		return fmt.Sprintf(`<|begin_of_text|><|start_header_id|>user<|end_header_id|>

Task: Check if there is unsafe content in 'User' message.

<BEGIN CONVERSATION>
User: %s
<END CONVERSATION>

Provide your safety assessment for ONLY the last User message.
<|eot_id|><|start_header_id|>assistant<|end_header_id|>`, userInput)
	}

	return fmt.Sprintf(`<|begin_of_text|><|start_header_id|>user<|end_header_id|>

Task: Check if there is unsafe content in the conversation.

<BEGIN CONVERSATION>
User: %s
Assistant: %s
<END CONVERSATION>

Provide your safety assessment for ONLY the last message.
<|eot_id|><|start_header_id|>assistant<|end_header_id|>`, userInput, assistantResponse)
}

func (c *VLLMClient) callAPI(ctx context.Context, prompt string) (*ModerationResult, error) {
	reqBody := vllmChatRequest{
		Model: c.config.Model,
		Messages: []vllmMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   100,
		Temperature: 0.0,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(c.config.BaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call vLLM API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vLLM API error (status %d): %s", resp.StatusCode, string(body))
	}

	var vllmResp vllmChatResponse
	if err := json.Unmarshal(body, &vllmResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if vllmResp.Error != nil {
		return nil, fmt.Errorf("vLLM error: %s", vllmResp.Error.Message)
	}

	if len(vllmResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices from vLLM")
	}

	content := vllmResp.Choices[0].Message.Content
	return c.parseResponse(content, vllmResp.Model), nil
}

func (c *VLLMClient) parseResponse(response, model string) *ModerationResult {
	result := &ModerationResult{
		Response: response,
		Model:    model,
		IsSafe:   true,
	}

	response = strings.ToLower(strings.TrimSpace(response))

	if strings.HasPrefix(response, "unsafe") {
		result.IsSafe = false

		parts := strings.Split(response, "\n")
		if len(parts) > 1 {
			categories := strings.Split(parts[1], ",")
			for _, cat := range categories {
				cat = strings.TrimSpace(strings.ToUpper(cat))
				if cat != "" && strings.HasPrefix(cat, "S") {
					result.ViolatedCategories = append(result.ViolatedCategories, GuardCategory(cat))
				}
			}
		}
	}

	return result
}

// Ping checks if vLLM server is running.
func (c *VLLMClient) Ping(ctx context.Context) error {
	url := strings.TrimSuffix(c.config.BaseURL, "/") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vLLM not reachable at %s: %w", c.config.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vLLM returned status %d", resp.StatusCode)
	}

	return nil
}

// ListModels returns available models from vLLM.
func (c *VLLMClient) ListModels(ctx context.Context) ([]string, error) {
	url := strings.TrimSuffix(c.config.BaseURL, "/") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Data))
	for i, m := range result.Data {
		models[i] = m.ID
	}
	return models, nil
}
