package service

import (
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
)

func TestPrepareFailedRequestRetryResumesAnalysis(t *testing.T) {
	now := time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC)
	request, err := domain.NewGenerationRequestAt("Build a Linux course", "user_test", now)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if err := request.MarkFailed("provider unavailable", now); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	service := NewCourseGeneratorService(nil, nil, fixedClock{now: now.Add(time.Minute)}, CourseGeneratorConfig{})
	continuation, err := service.prepareFailedRequestRetry(&request)
	if err != nil {
		t.Fatalf("prepare retry: %v", err)
	}
	if continuation != retryFromAnalysis {
		t.Fatalf("continuation = %d, want retryFromAnalysis", continuation)
	}
	if request.PipelineStatus != domain.PipelineStatusRunning || request.CurrentStep == nil || *request.CurrentStep != stepAnalysis {
		t.Fatalf("unexpected resumed request: %+v", request)
	}
}

func TestPrepareFailedRequestRetryResumesArchitectureWithPersistedBrief(t *testing.T) {
	now := time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC)
	request := analyzedGenerationRequest(t, now)
	if err := request.ConfirmDetectedBrief(now); err != nil {
		t.Fatalf("confirm detected brief: %v", err)
	}
	confirmedTitle := request.ConfirmedBrief.Title
	if err := request.MarkFailed("architecture failed", now); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	service := NewCourseGeneratorService(nil, nil, fixedClock{now: now.Add(time.Minute)}, CourseGeneratorConfig{})
	continuation, err := service.prepareFailedRequestRetry(&request)
	if err != nil {
		t.Fatalf("prepare retry: %v", err)
	}
	if continuation != retryFromArchitecture {
		t.Fatalf("continuation = %d, want retryFromArchitecture", continuation)
	}
	if request.PipelineStatus != domain.PipelineStatusRunning || request.ConfirmedBrief == nil || request.ConfirmedBrief.Title != confirmedTitle {
		t.Fatalf("unexpected resumed request: %+v", request)
	}
}
