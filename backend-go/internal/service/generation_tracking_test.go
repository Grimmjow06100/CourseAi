package service

import (
	"errors"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/google/uuid"
)

func TestTrackingQueriesRejectMissingDependencies(t *testing.T) {
	for _, service := range []*CourseGeneratorService{nil, {}} {
		if _, err := service.GetGenerationTracking(authenticatedTestContext(), uuid.New()); !errors.Is(err, contract.ErrServiceDependency) {
			t.Fatalf("tracking error = %v, want unavailable dependency", err)
		}
		if _, err := service.GetGenerationEvents(authenticatedTestContext(), uuid.New(), 0, 30); !errors.Is(err, contract.ErrServiceDependency) {
			t.Fatalf("events error = %v, want unavailable dependency", err)
		}
	}
}
