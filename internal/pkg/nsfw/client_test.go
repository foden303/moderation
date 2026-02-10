package nsfw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_DetectFromBytes_Safe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/predict" {
			t.Errorf("Expected /predict, got %s", r.URL.Path)
		}

		resp := apiResponse{
			IsNSFW:      false,
			NSFWScore:   0.05,
			NormalScore: 0.95,
			Label:       "normal",
			Confidence:  0.95,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		Threshold: 0.5,
	})

	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0}

	result, err := client.DetectFromBytes(context.Background(), imageData)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.IsNSFW {
		t.Error("Expected image to be safe")
	}

	if result.NSFWScore > 0.1 {
		t.Errorf("Expected low NSFW score, got %f", result.NSFWScore)
	}
}

func TestClient_DetectFromBytes_Unsafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			IsNSFW:      true,
			NSFWScore:   0.92,
			NormalScore: 0.08,
			Label:       "nsfw",
			Confidence:  0.92,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		Threshold: 0.5,
	})

	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0}

	result, err := client.DetectFromBytes(context.Background(), imageData)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.IsNSFW {
		t.Error("Expected image to be NSFW")
	}

	if result.NSFWScore < 0.9 {
		t.Errorf("Expected high NSFW score, got %f", result.NSFWScore)
	}
}

func TestClient_Ping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected /health, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})

	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.BaseURL != "http://localhost:8080" {
		t.Errorf("Expected BaseURL http://localhost:8080, got %s", config.BaseURL)
	}
	if config.Threshold != 0.5 {
		t.Errorf("Expected Threshold 0.5, got %f", config.Threshold)
	}
}

func TestDetectionResult_IsSafe(t *testing.T) {
	safeResult := &DetectionResult{IsNSFW: false}
	if !safeResult.IsSafe() {
		t.Error("Expected IsSafe() to return true for non-NSFW")
	}

	unsafeResult := &DetectionResult{IsNSFW: true}
	if unsafeResult.IsSafe() {
		t.Error("Expected IsSafe() to return false for NSFW")
	}
}
