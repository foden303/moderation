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

// GuardCategory represents a content safety category.
type GuardCategory string

const (
	CategoryViolentCrimes     GuardCategory = "S1"  // Violent Crimes
	CategoryNonViolentCrimes  GuardCategory = "S2"  // Non-Violent Crimes
	CategorySexCrimes         GuardCategory = "S3"  // Sex-Related Crimes
	CategoryChildExploitation GuardCategory = "S4"  // Child Sexual Exploitation
	CategorySpecializedAdvice GuardCategory = "S5"  // Defamation/Specialized Advice
	CategoryPrivacy           GuardCategory = "S6"  // Privacy
	CategoryIntellectualProp  GuardCategory = "S7"  // Intellectual Property
	CategoryIndiscriminate    GuardCategory = "S8"  // Indiscriminate Weapons
	CategoryHateSpeech        GuardCategory = "S9"  // Hate
	CategorySuicide           GuardCategory = "S10" // Suicide & Self-Harm
	CategorySexualContent     GuardCategory = "S11" // Sexual Content
)

// ModerationResult represents the moderation result.
type ModerationResult struct {
	IsSafe             bool
	ViolatedCategories []GuardCategory
	Response           string
	Model              string
}

// OllamaConfig contains configuration for Ollama client.
type OllamaConfig struct {
	BaseURL string // e.g., "http://localhost:11434"
	Model   string // e.g., "llama-guard3", "qwen2.5-coder"
	Timeout time.Duration
}

// DefaultOllamaConfig returns default configuration for local Ollama.
func DefaultOllamaConfig() OllamaConfig {
	return OllamaConfig{
		BaseURL: "http://localhost:11434",
		Model:   "llama-guard3:1b", // or "qwen2.5:7b-instruct"
		Timeout: 60 * time.Second,
	}
}

// OllamaClient is a client for Ollama API (self-hosted).
type OllamaClient struct {
	config     OllamaConfig
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama client.
func NewOllamaClient(config OllamaConfig) *OllamaClient {
	return &OllamaClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// ollamaRequest represents an Ollama chat request.
type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// ollamaResponse represents an Ollama chat response.
type ollamaResponse struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   ollamaMessage `json:"message"`
	Done      bool          `json:"done"`
	Error     string        `json:"error,omitempty"`
}

// ModerateText checks text content for safety violations using LlamaGuard or QwenGuard.
func (c *OllamaClient) ModerateText(ctx context.Context, text string) (*ModerationResult, error) {
	prompt := c.formatModerationPrompt(text, "")
	return c.callAPI(ctx, prompt)
}

// ModerateConversation checks both user input and assistant response.
func (c *OllamaClient) ModerateConversation(ctx context.Context, userInput, assistantResponse string) (*ModerationResult, error) {
	prompt := c.formatModerationPrompt(userInput, assistantResponse)
	return c.callAPI(ctx, prompt)
}

// formatModerationPrompt formats the moderation prompt for LlamaGuard/QwenGuard.
func (c *OllamaClient) formatModerationPrompt(userInput, assistantResponse string) string {
	// LlamaGuard prompt format
	if strings.Contains(strings.ToLower(c.config.Model), "llama-guard") || strings.Contains(strings.ToLower(c.config.Model), "llamaguard") {
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

	// Qwen Guard / Generic LLM prompt format
	if assistantResponse == "" {
		return fmt.Sprintf(`You are a content safety classifier. Analyze the following user message and determine if it contains unsafe content.

Categories to check:
- S1: Violent Crimes
- S2: Non-Violent Crimes  
- S3: Sex-Related Crimes
- S4: Child Sexual Exploitation
- S5: Defamation
- S6: Privacy Violations
- S7: Intellectual Property
- S8: Weapons
- S9: Hate Speech
- S10: Suicide/Self-Harm
- S11: Sexual Content

User message: "%s"

Respond with ONLY one of:
- "safe" if the content is safe
- "unsafe\nS1,S2,..." listing violated category codes if unsafe`, userInput)
	}

	return fmt.Sprintf(`You are a content safety classifier. Analyze the following conversation.

Categories to check:
- S1: Violent Crimes
- S2: Non-Violent Crimes
- S3: Sex-Related Crimes
- S4: Child Sexual Exploitation
- S5: Defamation
- S6: Privacy Violations
- S7: Intellectual Property
- S8: Weapons
- S9: Hate Speech
- S10: Suicide/Self-Harm
- S11: Sexual Content

User: "%s"
Assistant: "%s"

Respond with ONLY one of:
- "safe" if the content is safe
- "unsafe\nS1,S2,..." listing violated category codes if unsafe`, userInput, assistantResponse)
}

func (c *OllamaClient) callAPI(ctx context.Context, prompt string) (*ModerationResult, error) {
	reqBody := ollamaRequest{
		Model: c.config.Model,
		Messages: []ollamaMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false, // Disable streaming for simplicity
		Options: &ollamaOptions{
			Temperature: 0.0, // Deterministic output
			NumPredict:  100, // Limit output tokens
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if ollamaResp.Error != "" {
		return nil, fmt.Errorf("Ollama error: %s", ollamaResp.Error)
	}

	return c.parseResponse(ollamaResp.Message.Content, ollamaResp.Model), nil
}

func (c *OllamaClient) parseResponse(response, model string) *ModerationResult {
	result := &ModerationResult{
		Response: response,
		Model:    model,
		IsSafe:   true,
	}

	response = strings.ToLower(strings.TrimSpace(response))

	// Check if content is unsafe
	if strings.HasPrefix(response, "unsafe") {
		result.IsSafe = false

		// Extract violated categories (e.g., "unsafe\nS1, S3")
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

// Ping checks if Ollama is running and the model is available.
func (c *OllamaClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama not reachable at %s: %w", c.config.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	return nil
}

// ListModels returns available models from Ollama.
func (c *OllamaClient) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}
	return models, nil
}

// CategoryDescription returns a human-readable description of the category.
func CategoryDescription(cat GuardCategory) string {
	descriptions := map[GuardCategory]string{
		CategoryViolentCrimes:     "Violent Crimes",
		CategoryNonViolentCrimes:  "Non-Violent Crimes",
		CategorySexCrimes:         "Sex-Related Crimes",
		CategoryChildExploitation: "Child Sexual Exploitation",
		CategorySpecializedAdvice: "Defamation/Specialized Advice",
		CategoryPrivacy:           "Privacy Violations",
		CategoryIntellectualProp:  "Intellectual Property",
		CategoryIndiscriminate:    "Indiscriminate Weapons",
		CategoryHateSpeech:        "Hate Speech",
		CategorySuicide:           "Suicide & Self-Harm",
		CategorySexualContent:     "Sexual Content",
	}
	if desc, ok := descriptions[cat]; ok {
		return desc
	}
	return string(cat)
}
