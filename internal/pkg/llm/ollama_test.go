package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaClient_ModerateText_Safe(t *testing.T) {
	// Mock server returning "safe"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected /api/chat, got %s", r.URL.Path)
		}

		resp := ollamaResponse{
			Model: "llama-guard3:1b",
			Message: ollamaMessage{
				Role:    "assistant",
				Content: "safe",
			},
			Done: true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{
		BaseURL: server.URL,
		Model:   "llama-guard3:1b",
	})

	result, err := client.ModerateText(context.Background(), "Hello, how are you?")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.IsSafe {
		t.Error("Expected content to be safe")
	}
	if len(result.ViolatedCategories) != 0 {
		t.Errorf("Expected no violated categories, got %v", result.ViolatedCategories)
	}
}

func TestOllamaClient_ModerateText_Unsafe(t *testing.T) {
	// Mock server returning "unsafe" with categories
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaResponse{
			Model: "llama-guard3:1b",
			Message: ollamaMessage{
				Role:    "assistant",
				Content: "unsafe\nS1, S9",
			},
			Done: true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{
		BaseURL: server.URL,
		Model:   "llama-guard3:1b",
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

	// Check categories (Ollama parser still uses S-prefix codes)
	hasS1, hasS9 := false, false
	for _, cat := range result.ViolatedCategories {
		if cat == GuardCategory("S1") {
			hasS1 = true
		}
		if cat == GuardCategory("S9") {
			hasS9 = true
		}
	}
	if !hasS1 || !hasS9 {
		t.Errorf("Expected S1 and S9 categories, got %v", result.ViolatedCategories)
	}
}

func TestOllamaClient_Ping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Expected /api/tags, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"models": [{"name": "llama-guard3:1b"}]}`))
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestOllamaClient_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"models": [{"name": "llama-guard3:1b"}, {"name": "qwen2.5:7b"}]}`))
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}
}

func TestCategoryDescription(t *testing.T) {
	tests := []struct {
		cat      GuardCategory
		expected string
	}{
		{CategoryViolent, "Violent"},
		{CategorySuicide, "Suicide & Self-Harm"},
		{CategorySexualContent, "Sexual Content or Sexual Acts"},
		{GuardCategory("UNKNOWN"), "UNKNOWN"},
	}

	for _, tt := range tests {
		result := CategoryDescription(tt.cat)
		if result != tt.expected {
			t.Errorf("CategoryDescription(%s) = %s; want %s", tt.cat, result, tt.expected)
		}
	}
}

func TestDefaultOllamaConfig(t *testing.T) {
	config := DefaultOllamaConfig()

	if config.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected BaseURL http://localhost:11434, got %s", config.BaseURL)
	}
	if config.Model != "llama-guard3:1b" {
		t.Errorf("Expected Model llama-guard3:1b, got %s", config.Model)
	}
}
