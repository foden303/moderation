package biz

import (
	"context"
	"time"
)

// TextCache represents a cached text moderation result.
type TextCache struct {
	ContentHash       string
	NormalizedContent string
	DetectResult      []byte // JSON raw message
	Category          string
	NSFWScore         float64
	ModelVersion      string
	AddedBy           string
	ExpiresAt         *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ImageCache represents a cached image moderation result.
type ImageCache struct {
	FileHash     string
	PHash        int64
	DetectResult []byte // JSON raw message
	Category     string
	NSFWScore    float64
	ModelVersion string
	SourceURL    string
	AddedBy      string
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TextCacheRepo is a repository interface for text caches.
type TextCacheRepo interface {
	Upsert(ctx context.Context, cache *TextCache) error
	Get(ctx context.Context, contentHash string) (*TextCache, error)
	Delete(ctx context.Context, contentHash string) error
	DeleteExpired(ctx context.Context) (int64, error)
	List(ctx context.Context, category string, limit, offset int32) ([]*TextCache, error)
	ListAll(ctx context.Context) ([]*TextCache, error)
	Count(ctx context.Context, category string) (int64, error)
}

// ImageCacheRepo is a repository interface for image caches.
type ImageCacheRepo interface {
	Upsert(ctx context.Context, cache *ImageCache) error
	Get(ctx context.Context, fileHash string) (*ImageCache, error)
	FindSimilarByPHash(ctx context.Context, phash int64, maxDistance int32) ([]*ImageCache, error)
	Delete(ctx context.Context, fileHash string) error
	DeleteExpired(ctx context.Context) (int64, error)
	List(ctx context.Context, category string, limit, offset int32) ([]*ImageCache, error)
	ListAll(ctx context.Context) ([]*ImageCache, error)
	Count(ctx context.Context, category string) (int64, error)
}
