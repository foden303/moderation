package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// VLLMConfig contains configuration for vLLM server.
type VLLMConfig struct {
	BaseURL string // e.g., "http://localhost:8000" (vLLM default)
	Model   string // e.g., "Qwen/Qwen3Guard-Gen-0.6B"
	APIKey  string // Optional API key
	Timeout time.Duration
}

// DefaultVLLMConfig returns default configuration for local vLLM.
func DefaultVLLMConfig() VLLMConfig {
	return VLLMConfig{
		BaseURL: "http://localhost:8000",
		Model:   "Qwen/Qwen3Guard-Gen-0.6B",
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

// ModerateText checks text content for safety violations using Qwen3Guard via vLLM.
// Qwen3Guard uses simple chat format: just send user message directly.
func (c *VLLMClient) ModerateText(ctx context.Context, text string) (*ModerationResult, error) {
	messages := []vllmMessage{
		{Role: "user", Content: text},
	}
	return c.callAPI(ctx, messages)
}

// ModerateConversation checks both user input and assistant response.
// For response moderation, send user + assistant messages.
func (c *VLLMClient) ModerateConversation(ctx context.Context, userInput, assistantResponse string) (*ModerationResult, error) {
	messages := []vllmMessage{
		{Role: "user", Content: userInput},
		{Role: "assistant", Content: assistantResponse},
	}
	return c.callAPI(ctx, messages)
}

func (c *VLLMClient) callAPI(ctx context.Context, messages []vllmMessage) (*ModerationResult, error) {
	reqBody := vllmChatRequest{
		Model:       c.config.Model,
		Messages:    messages,
		MaxTokens:   128,
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

// Regex patterns for Qwen3Guard response parsing.
var (
	safetyPattern   = regexp.MustCompile(`(?i)Safety:\s*(Safe|Unsafe|Controversial)`)
	categoryPattern = regexp.MustCompile(`(?i)(Violent|Non-violent Illegal Acts|Sexual Content or Sexual Acts|PII|Suicide & Self-Harm|Unethical Acts|Politically Sensitive Topics|Copyright Violation|Jailbreak|None)`)
)

// parseResponse parses Qwen3Guard-Gen response format:
//
//	Safety: Unsafe
//	Categories: Violent
func (c *VLLMClient) parseResponse(response, model string) *ModerationResult {
	result := &ModerationResult{
		Response: response,
		Model:    model,
		IsSafe:   true,
	}

	// Extract safety label
	safetyMatch := safetyPattern.FindStringSubmatch(response)
	if len(safetyMatch) < 2 {
		// Can't parse → treat as safe to avoid blocking
		return result
	}

	label := strings.ToLower(safetyMatch[1])
	switch label {
	case "unsafe":
		result.IsSafe = false
		result.Severity = SeverityUnsafe
	case "controversial":
		result.IsSafe = false
		result.Severity = SeverityControversial
	case "safe":
		result.IsSafe = true
		result.Severity = SeveritySafe
	}

	// Extract categories
	categories := categoryPattern.FindAllString(response, -1)
	for _, cat := range categories {
		cat = strings.TrimSpace(cat)
		if cat != "" && !strings.EqualFold(cat, "None") {
			result.ViolatedCategories = append(result.ViolatedCategories, GuardCategory(cat))
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
