package biz

import (
	"context"
	"time"
)

// BadImage represents a flagged/NSFW image stored by its perceptual hash.
type BadImage struct {
	ID        int64
	PHash     int64   // 64-bit perceptual hash
	Category  string  // e.g., "nsfw", "violence"
	NSFWScore float64 // Original NSFW detection score
	SourceURL string  // Optional: original image URL
	AddedBy   string  // "auto" or admin username
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BadImageRepo is a repository interface for bad images.
type BadImageRepo interface {
	// Create creates or updates a bad image entry.
	Create(ctx context.Context, img *BadImage) (*BadImage, error)
	// FindByPHash finds a bad image by its perceptual hash.
	FindByPHash(ctx context.Context, phash int64) (*BadImage, error)
	// FindByPHashes finds bad images by multiple perceptual hashes.
	FindByPHashes(ctx context.Context, phashes []int64) ([]*BadImage, error)
	// ListAll returns all bad images (for rebuilding bloom filter).
	ListAll(ctx context.Context) ([]*BadImage, error)
	// Count returns the total count of bad images.
	Count(ctx context.Context) (int64, error)
	// Delete removes a bad image by its perceptual hash.
	Delete(ctx context.Context, phash int64) error
}
