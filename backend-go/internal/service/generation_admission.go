package service

import (
	"context"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
)

func (s *CourseGeneratorService) enforceGenerationAdmission(ctx context.Context, repository contract.GenerationRequestRepository, owner string) error {
	return s.enforceGenerationRetryAdmission(ctx, repository, owner, false)
}

// An operation retry within an active request consumes daily/queue capacity,
// but does not reserve a second active-generation slot for the same request.
func (s *CourseGeneratorService) enforceGenerationRetryAdmission(ctx context.Context, repository contract.GenerationRequestRepository, owner string, alreadyActive bool) error {
	usage, err := repository.GetGenerationAdmissionUsage(ctx, owner, s.now().Add(-24*time.Hour))
	if err != nil {
		return err
	}
	if !alreadyActive && s.config.MaxActivePerUser > 0 && usage.ActiveRequests >= s.config.MaxActivePerUser {
		return contract.ErrGenerationActiveLimitExceeded
	}
	if s.config.MaxDailyPerUser > 0 && usage.DailyRequests >= s.config.MaxDailyPerUser {
		return contract.ErrGenerationDailyLimitExceeded
	}
	if s.config.MaxPendingJobs > 0 && usage.PendingJobs >= s.config.MaxPendingJobs {
		return contract.ErrGenerationQueueSaturated
	}
	return nil
}
