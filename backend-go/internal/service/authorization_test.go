package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/google/uuid"
)

func TestAuthorizeOwnedResourceHidesCrossOwnerResources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		resource ownedResource
		want     error
	}{
		{resource: ownedGenerationRequest, want: contract.ErrGenerationRequestNotFound},
		{resource: ownedGenerationJob, want: contract.ErrGenerationJobNotFound},
		{resource: ownedCourse, want: contract.ErrCourseNotFound},
		{resource: ownedModule, want: contract.ErrModuleNotFound},
		{resource: ownedLesson, want: contract.ErrLessonNotFound},
	}
	for _, test := range tests {
		if err := authorizeOwnedResource(context.Background(), deniedOwnership{}, test.resource, uuid.New(), "user_b"); !errors.Is(err, test.want) {
			t.Errorf("resource %d error = %v, want %v", test.resource, err, test.want)
		}
	}
}

type deniedOwnership struct{ contract.OwnershipRepository }

func (deniedOwnership) OwnsGenerationRequest(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}
func (deniedOwnership) OwnsGenerationJob(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}
func (deniedOwnership) OwnsCourse(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}
func (deniedOwnership) OwnsModule(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}
func (deniedOwnership) OwnsLesson(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}
