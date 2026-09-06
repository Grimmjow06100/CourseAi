package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestListGenerationJobsAuthorizesBeforeReading(t *testing.T) {
	requestID := uuid.New()
	otherUser := contract.ContextWithPrincipal(context.Background(), contract.Principal{UserID: "user_other"})
	tests := []struct {
		name      string
		ctx       context.Context
		requestID uuid.UUID
		wantErr   error
	}{
		{"owner", authenticatedTestContext(), requestID, nil},
		{"anonymous", context.Background(), requestID, contract.ErrUnauthenticated},
		{"other user", otherUser, requestID, contract.ErrGenerationRequestNotFound},
		{"missing request", authenticatedTestContext(), uuid.New(), contract.ErrGenerationRequestNotFound},
		{"blank id", authenticatedTestContext(), uuid.Nil, domain.ErrBlankField},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queue := &jobReadQueue{requestID: requestID}
			uow := jobReadUnitOfWork{queue: queue, ownership: jobReadOwnership{requestID: requestID}}
			service := NewCourseGeneratorService(fakeCourseAI{}, uow, fixedClock{now: time.Now()}, CourseGeneratorConfig{})
			jobs, err := service.ListGenerationJobs(tt.ctx, tt.requestID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil && queue.called {
				t.Fatal("job data read before authorization")
			}
			if tt.wantErr == nil && (len(jobs) != 1 || jobs[0].RequestID != requestID) {
				t.Fatalf("unexpected jobs: %+v", jobs)
			}
		})
	}
}

type jobReadQueue struct {
	contract.GenerationJobQueue
	requestID uuid.UUID
	called    bool
}

func (q *jobReadQueue) ListByRequestID(_ context.Context, id uuid.UUID) ([]domain.GenerationJob, error) {
	q.called = true
	if id != q.requestID {
		return nil, errors.New("unexpected request id")
	}
	return []domain.GenerationJob{{ID: uuid.New(), RequestID: id}}, nil
}

type jobReadOwnership struct {
	contract.OwnershipRepository
	requestID uuid.UUID
}

func (o jobReadOwnership) OwnsGenerationRequest(_ context.Context, id uuid.UUID, owner string) (bool, error) {
	return id == o.requestID && owner == "user_test", nil
}

type jobReadUnitOfWork struct {
	queue     contract.GenerationJobQueue
	ownership contract.OwnershipRepository
}

func (u jobReadUnitOfWork) WithinTx(ctx context.Context, fn func(context.Context, contract.TransactionalRepositories) error) error {
	return fn(ctx, jobReadRepositories{queue: u.queue, ownership: u.ownership})
}

type jobReadRepositories struct {
	contract.TransactionalRepositories
	queue     contract.GenerationJobQueue
	ownership contract.OwnershipRepository
}

func (r jobReadRepositories) GenerationJobs() contract.GenerationJobQueue { return r.queue }
func (r jobReadRepositories) Ownership() contract.OwnershipRepository     { return r.ownership }
