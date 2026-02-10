package service

import (
	"context"

	v1 "moderation/api/moderation/v1"
	"moderation/internal/biz"
)

// AdminService implements the AdminService gRPC service.
type AdminService struct {
	v1.UnimplementedAdminServiceServer

	badwordUc    *biz.BadwordUsecase
	moderationUc *biz.ModerationUsecase
}

// NewAdminService creates a new AdminService.
func NewAdminService(badwordUc *biz.BadwordUsecase, moderationUc *biz.ModerationUsecase) *AdminService {
	return &AdminService{
		badwordUc:    badwordUc,
		moderationUc: moderationUc,
	}
}

// AddBadWord adds a word to the blocklist.
func (s *AdminService) AddBadWord(ctx context.Context, in *v1.AddBadWordRequest) (*v1.AddBadWordResponse, error) {
	// Add to database
	_, err := s.badwordUc.AddBadword(ctx, in.Word, in.Category, in.AddedBy, in.Severity)
	if err != nil {
		return &v1.AddBadWordResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Update moderation filters
	if err := s.moderationUc.AddBadWord(ctx, in.Word, in.Category, in.AddedBy, in.Severity); err != nil {
		return &v1.AddBadWordResponse{
			Success: false,
			Message: "Word added to database but failed to update filters: " + err.Error(),
		}, nil
	}

	return &v1.AddBadWordResponse{
		Success: true,
		Message: "Bad word added successfully",
	}, nil
}

// RemoveBadWord removes a word from the blocklist.
func (s *AdminService) RemoveBadWord(ctx context.Context, in *v1.RemoveBadWordRequest) (*v1.RemoveBadWordResponse, error) {
	err := s.badwordUc.RemoveBadword(ctx, in.Word)
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
	words, total, err := s.badwordUc.ListBadwords(ctx, in.Category, in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}

	entries := make([]*v1.BadWordEntry, len(words))
	for i, w := range words {
		entries[i] = &v1.BadWordEntry{
			Word:     w.Word,
			Category: w.Category,
			Severity: w.Severity,
			AddedBy:  w.AddedBy,
			AddedAt:  w.CreatedAt.Unix(),
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
