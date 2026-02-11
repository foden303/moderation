package service

import (
	"context"

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

// Moderate checks a post for harmful content.
func (s *ModerationService) Moderate(ctx context.Context, in *v1.ModerateRequest) (*v1.ModerationResponse, error) {
	// Map ModerateRequest to biz.Moderate
	result, err := s.uc.Moderate(ctx, in.RequestId, in.Content, in.ImageUrls, in.AudioUrls, in.VideoUrls)
	if err != nil {
		return nil, err
	}

	return toProtoResponse(result), nil
}

// ModerateContent checks text content for harmful content.
func (s *ModerationService) ModerateContent(ctx context.Context, in *v1.ModerateContentRequest) (*v1.ModerationResponse, error) {
	result, err := s.uc.ModerateText(ctx, in.RequestId, in.Content)
	if err != nil {
		return nil, err
	}

	return toProtoResponse(result), nil
}

// ModerateImage checks an image for harmful content.
func (s *ModerationService) ModerateImage(ctx context.Context, in *v1.ModerateImageRequest) (*v1.ModerationResponse, error) {
	result, err := s.uc.ModerateImage(ctx, in.RequestId, in.ImageUrl)
	if err != nil {
		return nil, err
	}

	return toProtoResponse(result), nil
}

// ModerateVideo checks a video for harmful content.
func (s *ModerationService) ModerateVideo(ctx context.Context, in *v1.ModerateVideoRequest) (*v1.ModerationResponse, error) {
	result, err := s.uc.ModerateVideo(ctx, in.RequestId, in.VideoUrl)
	if err != nil {
		return nil, err
	}

	return toProtoResponse(result), nil
}

// ModerateAudio checks an audio for harmful content.
func (s *ModerationService) ModerateAudio(ctx context.Context, in *v1.ModerateAudioRequest) (*v1.ModerationResponse, error) {
	result, err := s.uc.ModerateAudio(ctx, in.RequestId, in.AudioUrl)
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

		// call service moderate
		result, err = s.uc.Moderate(ctx, item.RequestId, item.Content, item.ImageUrls, item.AudioUrls, item.VideoUrls)
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
		RequestId:  result.RequestID,
		Action:     toProtoAction(result.Action),
		Verdict:    toProtoVerdict(result.Verdict),
		Reason:     result.Reason,
		Categories: result.Categories,
		Scores:     result.Scores,
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
