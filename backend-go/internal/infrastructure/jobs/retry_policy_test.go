package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	openaiinfra "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/openai"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRetryPolicyUsesExponentialBackoff(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	clock := &fixedClock{now: now}
	cfg := testWorkerConfig()
	cfg.RetryBaseDelay = 5 * time.Second
	cfg.RetryMaxDelay = time.Minute
	cfg.RetryJitterFraction = 0
	policy, err := NewRetryPolicy(clock, cfg)
	if err != nil {
		t.Fatalf("new retry policy: %v", err)
	}
	job := testJob(t, now)
	job.AttemptCount = 3
	job.MaxAttempts = 4

	decision := policy.Decide(job, context.DeadlineExceeded)
	if !decision.Retry || decision.Delay != 20*time.Second || !decision.AvailableAt.Equal(now.Add(20*time.Second)) {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestRetryPolicyHonorsRetryAfterWithoutShorteningIt(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	clock := &fixedClock{now: now}
	cfg := testWorkerConfig()
	cfg.RetryJitterFraction = 1
	policy, err := NewRetryPolicy(clock, cfg)
	if err != nil {
		t.Fatalf("new retry policy: %v", err)
	}
	job := testJob(t, now)
	job.AttemptCount = 1

	decision := policy.Decide(job, explicitRetryError{retryable: true, delay: 30 * time.Second})
	if !decision.Retry || decision.Delay < 30*time.Second {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestRetryPolicyClassifiesInfrastructureAndApplicationErrors(t *testing.T) {
	t.Parallel()
	now := time.Now()
	clock := &fixedClock{now: now}
	cfg := testWorkerConfig()
	cfg.RetryJitterFraction = 0
	policy, err := NewRetryPolicy(clock, cfg)
	if err != nil {
		t.Fatalf("new retry policy: %v", err)
	}
	job := testJob(t, now)
	job.AttemptCount = 1

	tests := []struct {
		name  string
		cause error
		want  bool
	}{
		{name: "invalid model output", cause: openaiinfra.ErrInvalidModelOutput, want: true},
		{name: "postgres connection", cause: &pgconn.PgError{Code: "08006"}, want: true},
		{name: "domain validation", cause: domain.ErrBlankField, want: false},
		{name: "explicit permanent", cause: explicitRetryError{retryable: false}, want: false},
		{name: "unknown", cause: errors.New("unknown"), want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if decision := policy.Decide(job, test.cause); decision.Retry != test.want {
				t.Fatalf("decision = %+v, want retry=%t", decision, test.want)
			}
		})
	}
}

func TestRetryPolicyStopsAtMaximumAttempts(t *testing.T) {
	t.Parallel()
	now := time.Now()
	policy, err := NewRetryPolicy(&fixedClock{now: now}, testWorkerConfig())
	if err != nil {
		t.Fatalf("new retry policy: %v", err)
	}
	job := testJob(t, now)
	job.AttemptCount = job.MaxAttempts
	if decision := policy.Decide(job, context.DeadlineExceeded); decision.Retry {
		t.Fatalf("unexpected retry: %+v", decision)
	}
}
