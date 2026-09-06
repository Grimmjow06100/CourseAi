package service

import (
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestListGenerationRequestsScopesAndPaginatesByAuthenticatedOwner(t *testing.T) {
	now := time.Now().UTC()
	requests := make(map[uuid.UUID]domain.GenerationRequest)
	for index, owner := range []string{"user_test", "user_other", "user_test"} {
		request, err := domain.NewGenerationRequestAt("Build course", owner, now.Add(time.Duration(index)*time.Minute))
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		requests[request.ID] = request
	}
	service := NewCourseGeneratorService(fakeCourseAI{}, &fakeUnitOfWork{requests: requests}, fixedClock{now: now}, CourseGeneratorConfig{})

	page, err := service.ListGenerationRequests(authenticatedTestContext(), contract.GenerationHistoryFilters{
		Pagination: contract.Pagination{Page: 1, PageSize: 1},
	})
	if err != nil {
		t.Fatalf("list generation history: %v", err)
	}
	if len(page.Items) != 1 || page.TotalItems != 2 || page.TotalPages != 2 || !page.HasNext {
		t.Fatalf("unexpected history page: %+v", page)
	}
}
