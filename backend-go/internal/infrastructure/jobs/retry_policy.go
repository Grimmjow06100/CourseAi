package jobs

import (
	"context"
	"errors"
	"hash/fnv"
	"math"
	"net"
	"strconv"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

type RetryDecision struct {
	Retry       bool
	AvailableAt time.Time
	Delay       time.Duration
	Reason      string
}

type RetryPolicy struct {
	clock          contract.Clock
	baseDelay      time.Duration
	maxDelay       time.Duration
	jitterFraction float64
}

type retryableError interface {
	Retryable() bool
}

type retryAfterError interface {
	RetryAfter() time.Duration
}

func NewRetryPolicy(clock contract.Clock, config WorkerConfig) (*RetryPolicy, error) {
	if clock == nil {
		return nil, ErrMissingWorkerClock
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &RetryPolicy{
		clock:          clock,
		baseDelay:      config.RetryBaseDelay,
		maxDelay:       config.RetryMaxDelay,
		jitterFraction: config.RetryJitterFraction,
	}, nil
}

func (p *RetryPolicy) Decide(job domain.GenerationJob, cause error) RetryDecision {
	if p == nil || p.clock == nil || cause == nil || !job.CanRetry() {
		return RetryDecision{Reason: "retry_not_available"}
	}

	retryable, reason := classifyRetryableError(cause)
	if !retryable {
		return RetryDecision{Reason: reason}
	}

	delay := applyDeterministicJitter(p.exponentialDelay(job.AttemptCount), p.jitterFraction, job)
	if requestedDelay, ok := retryAfterDelay(cause); ok && requestedDelay > delay {
		delay = requestedDelay
	}
	if delay < 0 {
		delay = 0
	}

	return RetryDecision{
		Retry:       true,
		AvailableAt: p.clock.Now().Add(delay),
		Delay:       delay,
		Reason:      reason,
	}
}

func (p *RetryPolicy) exponentialDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return p.baseDelay
	}

	delay := p.baseDelay
	for current := 1; current < attempt; current++ {
		if delay >= p.maxDelay/2 {
			return p.maxDelay
		}
		delay *= 2
	}
	if delay > p.maxDelay {
		return p.maxDelay
	}
	return delay
}

func classifyRetryableError(cause error) (bool, string) {
	var explicit retryableError
	if errors.As(cause, &explicit) {
		if explicit.Retryable() {
			return true, "explicitly_retryable"
		}
		return false, "explicitly_permanent"
	}

	var postgresError *pgconn.PgError
	if errors.As(cause, &postgresError) {
		if isTransientPostgresCode(postgresError.Code) {
			return true, "postgres_transient_error"
		}
		return false, "postgres_permanent_error"
	}

	var networkError net.Error
	if errors.As(cause, &networkError) && (networkError.Timeout() || networkError.Temporary()) {
		return true, "network_transient_error"
	}
	if errors.Is(cause, context.DeadlineExceeded) {
		return true, "deadline_exceeded"
	}
	if errors.Is(cause, context.Canceled) || isPermanentApplicationError(cause) {
		return false, "permanent_application_error"
	}
	return false, "unclassified_error"
}

func isPermanentApplicationError(cause error) bool {
	permanentErrors := []error{
		contract.ErrCourseNotFound,
		contract.ErrGenerationRequestNotFound,
		contract.ErrLessonNotFound,
		contract.ErrModuleNotFound,
		contract.ErrGenerationJobNotFound,
		contract.ErrGenerationJobClaimLost,
		contract.ErrGenerationJobNotCancellable,
		contract.ErrGenerationJobIdempotencyConflict,
		domain.ErrBlankField,
		domain.ErrInvalidCollection,
		domain.ErrInvalidCourseLanguage,
		domain.ErrInvalidCourseStatus,
		domain.ErrInvalidGenerationStatus,
		domain.ErrInvalidLessonType,
		domain.ErrInvalidExerciseType,
		domain.ErrInvalidQuizType,
		domain.ErrInvalidQuizQuestionType,
		domain.ErrInvalidDifficulty,
		domain.ErrInvalidQuizAnswer,
		domain.ErrInvalidLevel,
		domain.ErrInvalidOrder,
		domain.ErrInvalidProgress,
		domain.ErrInvalidDuration,
		domain.ErrInvalidStatusTransition,
		domain.ErrInvalidGenerationJobStatus,
		domain.ErrInvalidGenerationJobKind,
		domain.ErrInvalidGenerationJobPayload,
		domain.ErrInvalidGenerationJobAttempts,
		domain.ErrInvalidGenerationJobState,
		domain.ErrGenerationRequestNotReady,
	}
	for _, permanent := range permanentErrors {
		if errors.Is(cause, permanent) {
			return true
		}
	}
	return false
}

func isTransientPostgresCode(code string) bool {
	if len(code) >= 2 {
		switch code[:2] {
		case "08", "40", "53":
			return true
		}
	}
	switch code {
	case "55P03", "57P01", "57P02", "57P03":
		return true
	default:
		return false
	}
}

func retryAfterDelay(cause error) (time.Duration, bool) {
	var provider retryAfterError
	if errors.As(cause, &provider) {
		delay := provider.RetryAfter()
		return max(delay, 0), true
	}
	return 0, false
}

func applyDeterministicJitter(delay time.Duration, fraction float64, job domain.GenerationJob) time.Duration {
	if delay <= 0 || fraction <= 0 {
		return delay
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(job.ID.String()))
	_, _ = hasher.Write([]byte(strconv.Itoa(job.AttemptCount)))
	unit := float64(hasher.Sum64()) / float64(math.MaxUint64)
	factor := 1 + ((unit*2)-1)*fraction
	return time.Duration(float64(delay) * factor)
}
