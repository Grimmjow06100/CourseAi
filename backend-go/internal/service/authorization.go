package service

import (
	"context"
	"errors"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/google/uuid"
)

type ownedResource int

const (
	ownedGenerationRequest ownedResource = iota
	ownedGenerationJob
	ownedCourse
	ownedModule
	ownedLesson
)

func authenticatedOwner(ctx context.Context) (string, error) {
	principal, ok := contract.PrincipalFromContext(ctx)
	if !ok {
		return "", contract.ErrUnauthenticated
	}
	return principal.UserID, nil
}

func authorizeOwnedResource(ctx context.Context, repository contract.OwnershipRepository, resource ownedResource, id uuid.UUID, owner string) error {
	var (
		owned bool
		err   error
	)
	switch resource {
	case ownedGenerationRequest:
		owned, err = repository.OwnsGenerationRequest(ctx, id, owner)
	case ownedGenerationJob:
		owned, err = repository.OwnsGenerationJob(ctx, id, owner)
	case ownedCourse:
		owned, err = repository.OwnsCourse(ctx, id, owner)
	case ownedModule:
		owned, err = repository.OwnsModule(ctx, id, owner)
	case ownedLesson:
		owned, err = repository.OwnsLesson(ctx, id, owner)
	default:
		err = errors.New("unsupported owned resource")
	}
	if err != nil {
		return err
	}
	if !owned {
		switch resource {
		case ownedGenerationRequest:
			return contract.ErrGenerationRequestNotFound
		case ownedGenerationJob:
			return contract.ErrGenerationJobNotFound
		case ownedCourse:
			return contract.ErrCourseNotFound
		case ownedModule:
			return contract.ErrModuleNotFound
		case ownedLesson:
			return contract.ErrLessonNotFound
		}
	}
	return nil
}
