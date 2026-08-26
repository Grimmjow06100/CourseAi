package postgres

import (
	"context"

	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/google/uuid"
)

type OwnershipRepository struct {
	queries *dbsqlc.Queries
}

func NewOwnershipRepository(db DBTX) *OwnershipRepository {
	return &OwnershipRepository{queries: dbsqlc.New(db)}
}

func (r *OwnershipRepository) OwnsGenerationRequest(ctx context.Context, id uuid.UUID, clerkUserID string) (bool, error) {
	return r.queries.OwnsGenerationRequest(ctx, dbsqlc.OwnsGenerationRequestParams{ID: id, ClerkUserID: clerkUserID})
}

func (r *OwnershipRepository) OwnsGenerationJob(ctx context.Context, id uuid.UUID, clerkUserID string) (bool, error) {
	return r.queries.OwnsGenerationJob(ctx, dbsqlc.OwnsGenerationJobParams{ID: id, ClerkUserID: clerkUserID})
}

func (r *OwnershipRepository) OwnsCourse(ctx context.Context, id uuid.UUID, clerkUserID string) (bool, error) {
	return r.queries.OwnsCourse(ctx, dbsqlc.OwnsCourseParams{ID: id, ClerkUserID: clerkUserID})
}

func (r *OwnershipRepository) OwnsModule(ctx context.Context, id uuid.UUID, clerkUserID string) (bool, error) {
	return r.queries.OwnsModule(ctx, dbsqlc.OwnsModuleParams{ID: id, ClerkUserID: clerkUserID})
}

func (r *OwnershipRepository) OwnsLesson(ctx context.Context, id uuid.UUID, clerkUserID string) (bool, error) {
	return r.queries.OwnsLesson(ctx, dbsqlc.OwnsLessonParams{ID: id, ClerkUserID: clerkUserID})
}
