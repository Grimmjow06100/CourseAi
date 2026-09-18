package service

import (
	"context"
	"errors"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
)

var errGenerationSuperseded = errors.New("generation attempt superseded")

type generationJobContextKey struct{}

// Every worker transaction locks request then claim. AI calls stay outside the
// transaction; their results must pass the same fencing checks before commit.
func (s *CourseGeneratorService) withinTx(ctx context.Context, fn func(context.Context, contract.TransactionalRepositories) error) error {
	return s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		job, executing := ctx.Value(generationJobContextKey{}).(domain.GenerationJob)
		if !executing {
			return fn(ctx, repositories)
		}
		request, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, job.RequestID)
		if err != nil {
			return err
		}
		if job.GenerationAttempt != request.GenerationAttempt {
			return errGenerationSuperseded
		}
		claim, err := job.Claim()
		if err != nil {
			return err
		}
		guard, ok := repositories.GenerationJobs().(contract.GenerationJobClaimGuard)
		if !ok {
			return ErrCourseGeneratorDependency
		}
		if err := guard.LockClaim(ctx, claim); err != nil {
			return err
		}
		if err := fn(ctx, repositories); err != nil {
			return err
		}
		return guard.LockClaim(ctx, claim)
	})
}
