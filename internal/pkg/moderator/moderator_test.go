package moderator

import (
	"testing"
)

func TestTextModerationResult_Clean(t *testing.T) {
	result := &TextModerationResult{
		IsClean:      true,
		Matches:      nil,
		MaxSeverity:  0,
		Categories:   nil,
		ShouldReject: false,
		ShouldReview: false,
	}

	if !result.IsClean {
		t.Error("Expected result to be clean")
	}
	if result.ShouldReject {
		t.Error("Expected result not to reject")
	}
	if result.ShouldReview {
		t.Error("Expected result not to require review")
	}
}

func TestImageModeratorConfig_Defaults(t *testing.T) {
	config := DefaultImageModeratorConfig()

	if config.NSFWThreshold != 0.7 {
		t.Errorf("Expected NSFWThreshold 0.7, got %f", config.NSFWThreshold)
	}
	if config.ViolenceThreshold != 0.8 {
		t.Errorf("Expected ViolenceThreshold 0.8, got %f", config.ViolenceThreshold)
	}
	if !config.EnableOCR {
		t.Error("Expected EnableOCR to be true by default")
	}
}

func TestVideoModeratorConfig_Defaults(t *testing.T) {
	config := DefaultVideoModeratorConfig()

	if config.FrameSampleRate != 5 {
		t.Errorf("Expected FrameSampleRate 5, got %d", config.FrameSampleRate)
	}
	if config.MaxFramesToCheck != 10 {
		t.Errorf("Expected MaxFramesToCheck 10, got %d", config.MaxFramesToCheck)
	}
	if config.EnableAudioCheck {
		t.Error("Expected EnableAudioCheck to be false by default")
	}
}

func TestTextModeratorConfig_Defaults(t *testing.T) {
	config := DefaultTextModeratorConfig()

	if config.BloomBits != 1024*1024*8 {
		t.Errorf("Expected BloomBits 8M, got %d", config.BloomBits)
	}
	if config.BloomHashFunctions != 5 {
		t.Errorf("Expected BloomHashFunctions 5, got %d", config.BloomHashFunctions)
	}
	if config.RejectThreshold != 3 {
		t.Errorf("Expected RejectThreshold 3, got %d", config.RejectThreshold)
	}
	if config.ReviewThreshold != 2 {
		t.Errorf("Expected ReviewThreshold 2, got %d", config.ReviewThreshold)
	}
}

func TestLocalVideoModerator_ModerateVideo(t *testing.T) {
	// Skip test - LocalImageModerator now requires Redis, NSFW client, and BadImageRepo
	// These would need to be mocked for proper testing
	t.Skip("LocalVideoModerator requires mocked dependencies for LocalImageModerator")
}
