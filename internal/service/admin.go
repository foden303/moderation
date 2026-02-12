package service

import (
	"context"

	v1 "moderation/api/moderation/v1"
	"moderation/internal/biz"
)

// AdminService implements the AdminService gRPC service.
type AdminService struct {
	v1.UnimplementedAdminServiceServer

	moderationUc *biz.ModerationUsecase
}

// NewAdminService creates a new AdminService.
func NewAdminService(moderationUc *biz.ModerationUsecase) *AdminService {
	return &AdminService{
		moderationUc: moderationUc,
	}
}

// AddBadWord adds a word to the blocklist.
func (s *AdminService) AddBadWord(ctx context.Context, in *v1.AddBadWordRequest) (*v1.AddBadWordResponse, error) {
	// Add to moderation (both DB and Bloom Filter)
	if err := s.moderationUc.AddBadWord(ctx, in.Word, in.Category, in.NsfwScore, in.AddedBy, in.ModelVersion); err != nil {
		return &v1.AddBadWordResponse{
			Success: false,
			Message: "Failed to add bad word: " + err.Error(),
		}, nil
	}

	return &v1.AddBadWordResponse{
		Success: true,
		Message: "Bad word added successfully",
	}, nil
}

// RemoveBadWord removes a word from the blocklist.
func (s *AdminService) RemoveBadWord(ctx context.Context, in *v1.RemoveBadWordRequest) (*v1.RemoveBadWordResponse, error) {
	err := s.moderationUc.RemoveBadWord(ctx, in.Word)
	if err != nil {
		return &v1.RemoveBadWordResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &v1.RemoveBadWordResponse{
		Success: true,
		Message: "Bad word removed successfully. Note: Bloom filter will be updated on next rebuild.",
	}, nil
}

// ListBadWords lists all bad words.
func (s *AdminService) ListBadWords(ctx context.Context, in *v1.ListBadWordsRequest) (*v1.ListBadWordsResponse, error) {
	words, total, err := s.moderationUc.ListBadWords(ctx, in.Category, in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}

	entries := make([]*v1.BadWordEntry, len(words))
	for i, w := range words {
		entries[i] = &v1.BadWordEntry{
			Word:         w.NormalizedContent,
			Category:     w.Category,
			NsfwScore:    w.NSFWScore,
			AddedBy:      &w.AddedBy,
			ModelVersion: &w.ModelVersion,
		}
	}

	return &v1.ListBadWordsResponse{
		Words: entries,
		Total: int32(total),
	}, nil
}

// RebuildBloomFilter rebuilds the bloom filter.
func (s *AdminService) RebuildBloomFilter(ctx context.Context, in *v1.RebuildBloomFilterRequest) (*v1.RebuildBloomFilterResponse, error) {
	count, err := s.moderationUc.RebuildFilters(ctx)
	if err != nil {
		return &v1.RebuildBloomFilterResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &v1.RebuildBloomFilterResponse{
		Success:    true,
		Message:    "Bloom filter rebuilt successfully",
		WordsCount: int32(count),
	}, nil
}
