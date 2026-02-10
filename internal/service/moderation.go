package service

import (
	"context"
	"time"

	v1 "moderation/api/moderation/v1"
	"moderation/internal/biz"
)

// ModerationService implements the ModerationService gRPC service.
type ModerationService struct {
	v1.UnimplementedModerationServiceServer

	uc *biz.ModerationUsecase
}

// NewModerationService creates a new ModerationService.
func NewModerationService(uc *biz.ModerationUsecase) *ModerationService {
	return &ModerationService{uc: uc}
}

// ModeratePost checks a post for harmful content.
func (s *ModerationService) ModeratePost(ctx context.Context, in *v1.ModeratePostRequest) (*v1.ModerationResponse, error) {
	result, err := s.uc.ModeratePost(ctx, in.RequestId, in.Content, in.ImageUrls)
	if err != nil {
		return nil, err
	}

	return toProtoResponse(result), nil
}

// ModerateComment checks a comment for harmful content.
func (s *ModerationService) ModerateComment(ctx context.Context, in *v1.ModerateCommentRequest) (*v1.ModerationResponse, error) {
	result, err := s.uc.ModerateText(ctx, in.RequestId, in.Content)
	if err != nil {
		return nil, err
	}

	return toProtoResponse(result), nil
}

// BatchModerate checks multiple items at once.
func (s *ModerationService) BatchModerate(ctx context.Context, in *v1.BatchModerateRequest) (*v1.BatchModerateResponse, error) {
	results := make([]*v1.ModerationResponse, len(in.Items))

	for i, item := range in.Items {
		var result *biz.ModerationResult
		var err error

		switch item.ContentType {
		case v1.ContentType_CONTENT_TYPE_POST:
			result, err = s.uc.ModeratePost(ctx, item.RequestId, item.Content, nil)
		case v1.ContentType_CONTENT_TYPE_COMMENT:
			result, err = s.uc.ModerateText(ctx, item.RequestId, item.Content)
		default:
			result, err = s.uc.ModerateText(ctx, item.RequestId, item.Content)
		}

		if err != nil {
			return nil, err
		}
		results[i] = toProtoResponse(result)
	}

	return &v1.BatchModerateResponse{Results: results}, nil
}

// toProtoResponse converts biz.ModerationResult to v1.ModerationResponse.
func toProtoResponse(result *biz.ModerationResult) *v1.ModerationResponse {
	resp := &v1.ModerationResponse{
		RequestId:   result.RequestID,
		Action:      toProtoAction(result.Action),
		Verdict:     toProtoVerdict(result.Verdict),
		Reason:      result.Reason,
		Categories:  result.Categories,
		Scores:      result.Scores,
		ProcessedAt: result.ProcessedAt.Unix(),
	}

	if resp.ProcessedAt == 0 {
		resp.ProcessedAt = time.Now().Unix()
	}

	return resp
}

func toProtoAction(action biz.ModerationAction) v1.ModerationAction {
	switch action {
	case biz.ModerationActionAutoApprove:
		return v1.ModerationAction_MODERATION_ACTION_AUTO_APPROVE
	case biz.ModerationActionAutoReject:
		return v1.ModerationAction_MODERATION_ACTION_AUTO_REJECT
	case biz.ModerationActionPendingReview:
		return v1.ModerationAction_MODERATION_ACTION_PENDING_REVIEW
	default:
		return v1.ModerationAction_MODERATION_ACTION_UNSPECIFIED
	}
}

func toProtoVerdict(verdict biz.Verdict) v1.Verdict {
	switch verdict {
	case biz.VerdictClean:
		return v1.Verdict_VERDICT_CLEAN
	case biz.VerdictReject:
		return v1.Verdict_VERDICT_REJECT
	case biz.VerdictReview:
		return v1.Verdict_VERDICT_REVIEW
	default:
		return v1.Verdict_VERDICT_UNSPECIFIED
	}
}
