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
			Model: "meta-llama/Llama-Guard-3-1B",
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
						Content: "safe",
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
		Model:   "meta-llama/Llama-Guard-3-1B",
	})

	result, err := client.ModerateText(context.Background(), "Hello, how are you?")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.IsSafe {
		t.Error("Expected content to be safe")
	}
}

func TestVLLMClient_ModerateText_Unsafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := vllmChatResponse{
			ID:    "test-id",
			Model: "meta-llama/Llama-Guard-3-1B",
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
						Content: "unsafe\nS1, S9",
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
		Model:   "meta-llama/Llama-Guard-3-1B",
	})

	result, err := client.ModerateText(context.Background(), "harmful content")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.IsSafe {
		t.Error("Expected content to be unsafe")
	}

	if len(result.ViolatedCategories) != 2 {
		t.Errorf("Expected 2 violated categories, got %d", len(result.ViolatedCategories))
	}
}

func TestVLLMClient_Ping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("Expected /v1/models, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"id": "meta-llama/Llama-Guard-3-1B"}]}`))
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
		w.Write([]byte(`{"data": [{"id": "meta-llama/Llama-Guard-3-1B"}, {"id": "Qwen/Qwen2.5-7B"}]}`))
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
	if config.Model != "meta-llama/Llama-Guard-3-1B" {
		t.Errorf("Expected Model meta-llama/Llama-Guard-3-1B, got %s", config.Model)
	}
}
