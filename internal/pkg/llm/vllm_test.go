package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVLLMClient_ModerateText_Safe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected /v1/chat/completions, got %s", r.URL.Path)
		}

		resp := vllmChatResponse{
			ID:    "test-id",
			Model: "Qwen/Qwen3Guard-Gen-0.6B",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Safety: Safe\nCategories: None",
					},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewVLLMClient(VLLMConfig{
		BaseURL: server.URL,
		Model:   "Qwen/Qwen3Guard-Gen-0.6B",
	})

	result, err := client.ModerateText(context.Background(), "Hello, how are you?")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.IsSafe {
		t.Error("Expected content to be safe")
	}
	if result.Severity != SeveritySafe {
		t.Errorf("Expected severity Safe, got %d", result.Severity)
	}
}

func TestVLLMClient_ModerateText_Unsafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := vllmChatResponse{
			ID:    "test-id",
			Model: "Qwen/Qwen3Guard-Gen-0.6B",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Safety: Unsafe\nCategories: Violent",
					},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewVLLMClient(VLLMConfig{
		BaseURL: server.URL,
		Model:   "Qwen/Qwen3Guard-Gen-0.6B",
	})

	result, err := client.ModerateText(context.Background(), "harmful content")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.IsSafe {
		t.Error("Expected content to be unsafe")
	}
	if result.Severity != SeverityUnsafe {
		t.Errorf("Expected severity Unsafe, got %d", result.Severity)
	}
	if len(result.ViolatedCategories) != 1 {
		t.Fatalf("Expected 1 violated category, got %d", len(result.ViolatedCategories))
	}
	if result.ViolatedCategories[0] != CategoryViolent {
		t.Errorf("Expected Violent category, got %s", result.ViolatedCategories[0])
	}
}

func TestVLLMClient_ModerateText_Controversial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := vllmChatResponse{
			ID:    "test-id",
			Model: "Qwen/Qwen3Guard-Gen-0.6B",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Safety: Controversial\nCategories: Politically Sensitive Topics",
					},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewVLLMClient(VLLMConfig{
		BaseURL: server.URL,
		Model:   "Qwen/Qwen3Guard-Gen-0.6B",
	})

	result, err := client.ModerateText(context.Background(), "controversial topic")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.IsSafe {
		t.Error("Expected content to be not safe")
	}
	if result.Severity != SeverityControversial {
		t.Errorf("Expected severity Controversial, got %d", result.Severity)
	}
	if len(result.ViolatedCategories) != 1 {
		t.Fatalf("Expected 1 violated category, got %d", len(result.ViolatedCategories))
	}
	if result.ViolatedCategories[0] != CategoryPoliticallySensitive {
		t.Errorf("Expected Politically Sensitive Topics, got %s", result.ViolatedCategories[0])
	}
}

func TestVLLMClient_Ping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("Expected /v1/models, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"id": "Qwen/Qwen3Guard-Gen-0.6B"}]}`))
	}))
	defer server.Close()

	client := NewVLLMClient(VLLMConfig{BaseURL: server.URL})

	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestVLLMClient_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data": [{"id": "Qwen/Qwen3Guard-Gen-0.6B"}, {"id": "Qwen/Qwen3-0.6B"}]}`))
	}))
	defer server.Close()

	client := NewVLLMClient(VLLMConfig{BaseURL: server.URL})

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}
}

func TestDefaultVLLMConfig(t *testing.T) {
	config := DefaultVLLMConfig()

	if config.BaseURL != "http://localhost:8000" {
		t.Errorf("Expected BaseURL http://localhost:8000, got %s", config.BaseURL)
	}
	if config.Model != "Qwen/Qwen3Guard-Gen-0.6B" {
		t.Errorf("Expected Model Qwen/Qwen3Guard-Gen-0.6B, got %s", config.Model)
	}
}
