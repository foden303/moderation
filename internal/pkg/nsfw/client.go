package nsfw

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// DetectionResult represents the result of NSFW detection.
type DetectionResult struct {
	IsNSFW      bool
	NSFWScore   float64 // 0.0 to 1.0
	NormalScore float64
	Label       string
	Confidence  float64
	ProcessedAt time.Time
}

// Config holds configuration for NSFW detector.
type Config struct {
	BaseURL   string // NSFW API URL, e.g., "http://localhost:8080"
	Timeout   time.Duration
	Threshold float64 // Score threshold to consider NSFW (default 0.5)
}

// DefaultConfig returns default configuration.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "http://localhost:8080",
		Timeout:   30 * time.Second,
		Threshold: 0.5,
	}
}

// Client is a client for NSFW detection API (Falconsai/nsfw_image_detection).
type Client struct {
	config     Config
	httpClient *http.Client
}

// NewClient creates a new NSFW detection client.
func NewClient(config Config) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// apiResponse represents the API response from Falconsai model.
type apiResponse struct {
	IsNSFW      bool    `json:"is_nsfw"`
	NSFWScore   float64 `json:"nsfw_score"`
	NormalScore float64 `json:"normal_score"`
	Label       string  `json:"label"`
	Confidence  float64 `json:"confidence"`
}

// DetectFromBytes detects NSFW content from image bytes.
func (c *Client) DetectFromBytes(ctx context.Context, imageData []byte) (*DetectionResult, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "image.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(imageData); err != nil {
		return nil, fmt.Errorf("failed to write image data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	return c.doRequest(ctx, body, writer.FormDataContentType())
}

// DetectFromBase64 detects NSFW content from base64 encoded image.
func (c *Client) DetectFromBase64(ctx context.Context, base64Image string) (*DetectionResult, error) {
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	return c.DetectFromBytes(ctx, imageData)
}

// DetectFromReader detects NSFW content from io.Reader.
func (c *Client) DetectFromReader(ctx context.Context, reader io.Reader) (*DetectionResult, error) {
	imageData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}
	return c.DetectFromBytes(ctx, imageData)
}

func (c *Client) doRequest(ctx context.Context, body *bytes.Buffer, contentType string) (*DetectionResult, error) {
	url := c.config.BaseURL + "/predict"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call NSFW API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NSFW API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &DetectionResult{
		IsNSFW:      apiResp.IsNSFW,
		NSFWScore:   apiResp.NSFWScore,
		NormalScore: apiResp.NormalScore,
		Label:       apiResp.Label,
		Confidence:  apiResp.Confidence,
		ProcessedAt: time.Now(),
	}, nil
}

// BatchURLResult represents a single result in batch URL detection.
type BatchURLResult struct {
	URL    string
	Result *DetectionResult
	Error  error
}

// DetectFromURLs detects NSFW content from multiple public image URLs.
func (c *Client) DetectFromURLs(ctx context.Context, imageURLs []string) ([]BatchURLResult, error) {
	url := c.config.BaseURL + "/predict/batch/url"

	reqBody := struct {
		URLs []string `json:"urls"`
	}{
		URLs: imageURLs,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call NSFW API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NSFW API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var batchResp struct {
		Predictions []struct {
			URL         string  `json:"url"`
			IsNSFW      bool    `json:"is_nsfw"`
			NSFWScore   float64 `json:"nsfw_score"`
			NormalScore float64 `json:"normal_score"`
			Label       string  `json:"label"`
			Confidence  float64 `json:"confidence"`
			Error       string  `json:"error,omitempty"`
		} `json:"predictions"`
	}

	if err := json.Unmarshal(respBody, &batchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	results := make([]BatchURLResult, len(batchResp.Predictions))
	for i, pred := range batchResp.Predictions {
		results[i].URL = pred.URL
		if pred.Error != "" {
			results[i].Error = fmt.Errorf("%s", pred.Error)
		} else {
			results[i].Result = &DetectionResult{
				IsNSFW:      pred.IsNSFW,
				NSFWScore:   pred.NSFWScore,
				NormalScore: pred.NormalScore,
				Label:       pred.Label,
				Confidence:  pred.Confidence,
				ProcessedAt: time.Now(),
			}
		}
	}

	return results, nil
}

// Ping checks if the NSFW API is reachable.
func (c *Client) Ping(ctx context.Context) error {
	url := c.config.BaseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("NSFW API not reachable at %s: %w", c.config.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("NSFW API returned status %d", resp.StatusCode)
	}

	return nil
}

// IsSafe returns true if the image is safe (not NSFW).
func (r *DetectionResult) IsSafe() bool {
	return !r.IsNSFW
}

// DetectFromURL detects NSFW content from a public image URL.
func (c *Client) DetectFromURL(ctx context.Context, imageURL string) (*DetectionResult, error) {
	url := c.config.BaseURL + "/predict/url"

	reqBody := struct {
		URL string `json:"url"`
	}{
		URL: imageURL,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call NSFW API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NSFW API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &DetectionResult{
		IsNSFW:      apiResp.IsNSFW,
		NSFWScore:   apiResp.NSFWScore,
		NormalScore: apiResp.NormalScore,
		Label:       apiResp.Label,
		Confidence:  apiResp.Confidence,
		ProcessedAt: time.Now(),
	}, nil
}
