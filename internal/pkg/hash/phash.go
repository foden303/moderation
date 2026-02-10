package hash

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"time"

	"github.com/corona10/goimagehash"
)

// HashType represents the type of perceptual hash.
type HashType int

const (
	// PHash uses DCT-based perceptual hash (most accurate).
	PHash HashType = iota
	// AHash uses average hash (fastest).
	AHash
	// DHash uses difference hash (good balance).
	DHash
)

// ImageHash represents a computed image hash.
type ImageHash struct {
	Hash     uint64
	HashType HashType
	Width    int
	Height   int
}

// PerceptualHasher provides image hashing functionality.
type PerceptualHasher struct {
	httpClient *http.Client
}

// NewPerceptualHasher creates a new PerceptualHasher.
func NewPerceptualHasher() *PerceptualHasher {
	return &PerceptualHasher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ComputePHash computes the DCT-based perceptual hash of an image.
func (ph *PerceptualHasher) ComputePHash(img image.Image) (*ImageHash, error) {
	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return nil, fmt.Errorf("failed to compute pHash: %w", err)
	}
	return &ImageHash{
		Hash:     hash.GetHash(),
		HashType: PHash,
		Width:    img.Bounds().Dx(),
		Height:   img.Bounds().Dy(),
	}, nil
}

// ComputeAHash computes the average hash of an image.
func (ph *PerceptualHasher) ComputeAHash(img image.Image) (*ImageHash, error) {
	hash, err := goimagehash.AverageHash(img)
	if err != nil {
		return nil, fmt.Errorf("failed to compute aHash: %w", err)
	}
	return &ImageHash{
		Hash:     hash.GetHash(),
		HashType: AHash,
		Width:    img.Bounds().Dx(),
		Height:   img.Bounds().Dy(),
	}, nil
}

// ComputeDHash computes the difference hash of an image.
func (ph *PerceptualHasher) ComputeDHash(img image.Image) (*ImageHash, error) {
	hash, err := goimagehash.DifferenceHash(img)
	if err != nil {
		return nil, fmt.Errorf("failed to compute dHash: %w", err)
	}
	return &ImageHash{
		Hash:     hash.GetHash(),
		HashType: DHash,
		Width:    img.Bounds().Dx(),
		Height:   img.Bounds().Dy(),
	}, nil
}

// ComputeHashFromBytes computes a perceptual hash from image bytes.
func (ph *PerceptualHasher) ComputeHashFromBytes(data []byte, hashType HashType) (*ImageHash, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	return ph.computeHash(img, hashType)
}

// ComputeHashFromReader computes a perceptual hash from an io.Reader.
func (ph *PerceptualHasher) ComputeHashFromReader(r io.Reader, hashType HashType) (*ImageHash, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	return ph.computeHash(img, hashType)
}

// ComputeHashFromURL computes a perceptual hash from an image URL.
func (ph *PerceptualHasher) ComputeHashFromURL(ctx context.Context, url string, hashType HashType) (*ImageHash, error) {
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

	return ph.ComputeHashFromReader(resp.Body, hashType)
}

func (ph *PerceptualHasher) computeHash(img image.Image, hashType HashType) (*ImageHash, error) {
	switch hashType {
	case PHash:
		return ph.ComputePHash(img)
	case AHash:
		return ph.ComputeAHash(img)
	case DHash:
		return ph.ComputeDHash(img)
	default:
		return ph.ComputePHash(img)
	}
}

// HammingDistance calculates the Hamming distance between two hashes.
// Returns the number of different bits (0 = identical images).
func HammingDistance(hash1, hash2 uint64) int {
	xor := hash1 ^ hash2
	count := 0
	for xor != 0 {
		count++
		xor &= xor - 1
	}
	return count
}

// CompareHashes compares two ImageHash objects.
// Returns the Hamming distance (0 = identical, higher = more different).
func CompareHashes(h1, h2 *ImageHash) int {
	return HammingDistance(h1.Hash, h2.Hash)
}

// IsSimilar checks if two hashes are similar within a threshold.
// Typical thresholds:
//   - 0: Identical
//   - 1-5: Very similar (likely same image with minor edits)
//   - 6-10: Somewhat similar
//   - 11+: Different images
func IsSimilar(h1, h2 *ImageHash, threshold int) bool {
	return HammingDistance(h1.Hash, h2.Hash) <= threshold
}

// SimilarityScore returns a similarity percentage (0-100).
// 100 = identical, 0 = completely different.
func SimilarityScore(h1, h2 *ImageHash) float64 {
	distance := HammingDistance(h1.Hash, h2.Hash)
	// Max distance for 64-bit hash is 64
	return (1 - float64(distance)/64.0) * 100
}

// String returns a hex string representation of the hash.
func (h *ImageHash) String() string {
	return fmt.Sprintf("%016x", h.Hash)
}

// HashTypeString returns the name of the hash type.
func (h *ImageHash) HashTypeString() string {
	switch h.HashType {
	case PHash:
		return "pHash"
	case AHash:
		return "aHash"
	case DHash:
		return "dHash"
	default:
		return "unknown"
	}
}
