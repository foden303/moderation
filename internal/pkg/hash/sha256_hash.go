package hash

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ImageHash represents a computed image hash.
type ImageSha256Hash struct {
	Hash string
}

// Sha256Hasher provides image hashing functionality.
type Sha256Hasher struct {
	httpClient *http.Client
}

// NewSha256Hasher creates a new Sha256Hasher.
func NewSha256Hasher() *Sha256Hasher {
	return &Sha256Hasher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ComputeHashFromURL computes a perceptual hash from an image URL.
func (ph *Sha256Hasher) ComputeHashFromURL(ctx context.Context, url string) (*ImageSha256Hash, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ph.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return ph.ComputeHashFromReader(resp.Body)
}

func (ph *Sha256Hasher) ComputeHashFromReader(r io.Reader) (*ImageSha256Hash, error) {
	hasher := sha256.New()

	// Copy data from reader to hasher
	if _, err := io.Copy(hasher, r); err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	hashBytes := hasher.Sum(nil)

	return &ImageSha256Hash{
		Hash: hex.EncodeToString(hashBytes),
	}, nil
}
