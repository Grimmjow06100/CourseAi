//go:build integration

package postgresintegration_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	"github.com/Grimmjow06100/course-ai/backend-go/tests/testkit"
	"github.com/google/uuid"
)

func TestGenerationJobQueueAgainstPostgres(t *testing.T) {
	ctx, pool := testkit.OpenPostgres(t, 30*time.Second)
	tx := testkit.BeginRollback(t, ctx, pool)

	if _, err := tx.Exec(ctx, `LOCK TABLE generation_jobs IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatalf("lock generation jobs: %v", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM generation_jobs`); err != nil {
		t.Fatalf("clear generation jobs in test transaction: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	requestID := uuid.New()
	if _, err := tx.Exec(ctx, `
		INSERT INTO generation_requests (id, clerk_user_id, initial_user_prompt, updated_at)
		VALUES ($1, 'user_integration', 'job integration test', $2)`, requestID, now); err != nil {
		t.Fatalf("insert generation request: %v", err)
	}

	repository := postgres.NewGenerationJobRepository(tx)
	job := newIntegrationGenerationJob(t, requestID, "request:analysis", now)
	saved, err := repository.Enqueue(ctx, job)
	if err != nil {
		t.Fatalf("enqueue job: %v", err)
	}

	duplicate := job
	duplicate.ID = uuid.New()
	duplicate.Payload = json.RawMessage(`{ "force": false }`)
	idempotentResult, err := repository.Enqueue(ctx, duplicate)
	if err != nil {
		t.Fatalf("idempotent enqueue: %v", err)
	}
	if idempotentResult.ID != saved.ID {
		t.Fatalf("idempotent enqueue returned %s, want %s", idempotentResult.ID, saved.ID)
	}

	conflict := duplicate
	conflict.Payload = json.RawMessage(`{"force":true}`)
	if _, err := repository.Enqueue(ctx, conflict); !errors.Is(err, contract.ErrGenerationJobIdempotencyConflict) {
		t.Fatalf("conflicting enqueue error = %v, want ErrGenerationJobIdempotencyConflict", err)
	}

	claimed, err := repository.ClaimNext(ctx, "worker-1", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	claim, err := claimed.Claim()
	if err != nil {
		t.Fatalf("build claim token: %v", err)
	}
	if claim.AttemptCount != 1 {
		t.Fatalf("attempt count = %d, want 1", claim.AttemptCount)
	}
	if _, err := repository.ClaimNext(ctx, "worker-2", time.Now().Add(time.Minute)); !errors.Is(err, contract.ErrGenerationJobUnavailable) {
		t.Fatalf("second claim error = %v, want ErrGenerationJobUnavailable", err)
	}

	staleClaim := claim
	staleClaim.AttemptCount++
	if err := repository.RenewLease(ctx, staleClaim, time.Now().Add(2*time.Minute)); !errors.Is(err, contract.ErrGenerationJobClaimLost) {
		t.Fatalf("stale renew error = %v, want ErrGenerationJobClaimLost", err)
	}
	if err := repository.RenewLease(ctx, claim, time.Now().Add(2*time.Minute)); err != nil {
		t.Fatalf("renew lease: %v", err)
	}

	if err := repository.Retry(ctx, claim, time.Now().Add(-time.Second), errors.New("temporary OpenAI error")); err != nil {
		t.Fatalf("retry job: %v", err)
	}
	retrying, err := repository.FindByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("find retrying job: %v", err)
	}
	if retrying.Status != domain.GenerationJobStatusRetryScheduled || retrying.LastErrorMessage == nil {
		t.Fatalf("unexpected retry state: %+v", retrying)
	}

	claimedAgain, err := repository.ClaimNext(ctx, "worker-2", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("claim retry: %v", err)
	}
	secondClaim, err := claimedAgain.Claim()
	if err != nil {
		t.Fatalf("build second claim token: %v", err)
	}
	if secondClaim.AttemptCount != 2 {
		t.Fatalf("second attempt count = %d, want 2", secondClaim.AttemptCount)
	}
	if err := repository.Complete(ctx, secondClaim, time.Now()); err != nil {
		t.Fatalf("complete job: %v", err)
	}
	completed, err := repository.FindByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("find completed job: %v", err)
	}
	if completed.Status != domain.GenerationJobStatusCompleted || completed.CompletedAt == nil {
		t.Fatalf("unexpected completed state: %+v", completed)
	}

	cancellable := newIntegrationGenerationJob(t, requestID, "request:architecture", time.Now())
	cancellable.Kind = domain.GenerationJobKindArchitecture
	cancellable, err = repository.Enqueue(ctx, cancellable)
	if err != nil {
		t.Fatalf("enqueue cancellable job: %v", err)
	}
	if err := repository.Cancel(ctx, cancellable.ID, time.Now()); err != nil {
		t.Fatalf("cancel job: %v", err)
	}
	cancelled, err := repository.FindByID(ctx, cancellable.ID)
	if err != nil {
		t.Fatalf("find cancelled job: %v", err)
	}
	if cancelled.Status != domain.GenerationJobStatusCancelled || cancelled.CompletedAt == nil {
		t.Fatalf("unexpected cancelled state: %+v", cancelled)
	}

	expiring := newIntegrationGenerationJob(t, requestID, "request:expired", time.Now())
	expiring.Priority = 10
	expiring, err = repository.Enqueue(ctx, expiring)
	if err != nil {
		t.Fatalf("enqueue expiring job: %v", err)
	}
	expiring, err = repository.ClaimNext(ctx, "worker-expired", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("claim expiring job: %v", err)
	}
	requeuedAt := time.Now()
	expireResult, err := tx.Exec(ctx, `UPDATE generation_jobs SET locked_until = $2 WHERE id = $1`, expiring.ID, requeuedAt.Add(-time.Second))
	if err != nil {
		t.Fatalf("expire lease: %v", err)
	}
	if expireResult.RowsAffected() != 1 {
		t.Fatalf("expired lease rows = %d, want 1", expireResult.RowsAffected())
	}
	requeued, err := repository.RequeueExpired(ctx, requeuedAt)
	if err != nil {
		t.Fatalf("requeue expired job: %v", err)
	}
	if requeued != 1 {
		t.Fatalf("requeued jobs = %d, want 1", requeued)
	}
	expired, err := repository.FindByID(ctx, expiring.ID)
	if err != nil {
		t.Fatalf("find expired job: %v", err)
	}
	if expired.Status != domain.GenerationJobStatusRetryScheduled || expired.LockedBy != nil || expired.LockedUntil != nil {
		t.Fatalf("unexpected expired lease state: %+v", expired)
	}
	if err := repository.Cancel(ctx, expired.ID, time.Now()); err != nil {
		t.Fatalf("cancel requeued job: %v", err)
	}

	exhausted := newIntegrationGenerationJob(t, requestID, "request:exhausted", time.Now())
	exhausted.MaxAttempts = 1
	exhausted.Priority = 20
	exhausted, err = repository.Enqueue(ctx, exhausted)
	if err != nil {
		t.Fatalf("enqueue exhausted job: %v", err)
	}
	exhausted, err = repository.ClaimNext(ctx, "worker-exhausted", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("claim exhausted job: %v", err)
	}
	exhaustedClaim, err := exhausted.Claim()
	if err != nil {
		t.Fatalf("build exhausted claim: %v", err)
	}
	if err := repository.Retry(ctx, exhaustedClaim, time.Now(), errors.New("last retryable error")); err != nil {
		t.Fatalf("exhaust retries: %v", err)
	}
	exhausted, err = repository.FindByID(ctx, exhausted.ID)
	if err != nil {
		t.Fatalf("find exhausted job: %v", err)
	}
	if exhausted.Status != domain.GenerationJobStatusFailed || exhausted.CompletedAt == nil || exhausted.LastErrorCode == nil || *exhausted.LastErrorCode != "max_attempts_exhausted" {
		t.Fatalf("unexpected exhausted state: %+v", exhausted)
	}
	unreconciled, err := repository.ListUnreconciledFailures(ctx, 10)
	if err != nil || len(unreconciled) != 1 || unreconciled[0].ID != exhausted.ID {
		t.Fatalf("unreconciled failures = %+v, %v", unreconciled, err)
	}
	if err := repository.MarkFailureHandled(ctx, exhausted.ID, time.Now()); err != nil {
		t.Fatalf("mark failure handled: %v", err)
	}

	crashed := newIntegrationGenerationJob(t, requestID, "request:crashed-last-attempt", time.Now())
	crashed.MaxAttempts = 1
	crashed.Priority = 30
	crashed, err = repository.Enqueue(ctx, crashed)
	if err != nil {
		t.Fatalf("enqueue crashed job: %v", err)
	}
	crashed, err = repository.ClaimNext(ctx, "worker-crashed", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("claim crashed job: %v", err)
	}
	crashedAt := time.Now()
	if _, err := tx.Exec(ctx, `UPDATE generation_jobs SET locked_until = $2 WHERE id = $1`, crashed.ID, crashedAt.Add(-time.Second)); err != nil {
		t.Fatalf("expire crashed job lease: %v", err)
	}
	if count, err := repository.RequeueExpired(ctx, crashedAt); err != nil || count != 1 {
		t.Fatalf("reap crashed job = %d, %v", count, err)
	}
	unreconciled, err = repository.ListUnreconciledFailures(ctx, 10)
	if err != nil || len(unreconciled) != 1 || unreconciled[0].ID != crashed.ID {
		t.Fatalf("crashed unreconciled failures = %+v, %v", unreconciled, err)
	}
}

func TestGenerationJobConcurrentClaimsReturnDifferentJobs(t *testing.T) {
	ctx, pool := testkit.OpenPostgres(t, 20*time.Second)

	now := time.Now().UTC().Truncate(time.Millisecond)
	requestID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO generation_requests (id, clerk_user_id, initial_user_prompt, updated_at)
		VALUES ($1, 'user_integration', 'concurrent claim integration test', $2)`, requestID, now); err != nil {
		t.Fatalf("insert generation request: %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM generation_requests WHERE id = $1`, requestID)
	}()

	repository := postgres.NewGenerationJobRepository(pool)
	expectedIDs := make(map[uuid.UUID]struct{}, 2)
	for index := 0; index < 2; index++ {
		job := newIntegrationGenerationJob(t, requestID, "request:concurrent:"+uuid.NewString(), now)
		job.Priority = int(1<<31 - 1)
		job.AvailableAt = time.Unix(1, int64(index)).UTC()
		job, err := repository.Enqueue(ctx, job)
		if err != nil {
			t.Fatalf("enqueue concurrent job %d: %v", index+1, err)
		}
		expectedIDs[job.ID] = struct{}{}
	}

	type claimResult struct {
		job domain.GenerationJob
		err error
	}
	start := make(chan struct{})
	results := make(chan claimResult, 2)
	for index := 0; index < 2; index++ {
		workerID := "concurrent-worker-" + uuid.NewString()
		go func(workerID string) {
			<-start
			job, claimErr := repository.ClaimNext(ctx, workerID, time.Now().Add(time.Minute))
			// A sibling worker may hold the shared request lock for its claim transaction.
			for attempt := 0; errors.Is(claimErr, contract.ErrGenerationJobUnavailable) && attempt < 20; attempt++ {
				time.Sleep(10 * time.Millisecond)
				job, claimErr = repository.ClaimNext(ctx, workerID, time.Now().Add(time.Minute))
			}
			results <- claimResult{job: job, err: claimErr}
		}(workerID)
	}
	close(start)

	claimedIDs := make(map[uuid.UUID]struct{}, 2)
	for index := 0; index < 2; index++ {
		result := <-results
		if result.err != nil {
			t.Fatalf("concurrent claim %d: %v", index+1, result.err)
		}
		if _, expected := expectedIDs[result.job.ID]; !expected {
			t.Fatalf("claimed unexpected job %s", result.job.ID)
		}
		if _, duplicate := claimedIDs[result.job.ID]; duplicate {
			t.Fatalf("job %s was claimed by two workers", result.job.ID)
		}
		claimedIDs[result.job.ID] = struct{}{}

		claim, claimErr := result.job.Claim()
		if claimErr != nil {
			t.Fatalf("build concurrent claim: %v", claimErr)
		}
		if completeErr := repository.Complete(ctx, claim, time.Now()); completeErr != nil {
			t.Fatalf("complete concurrent job: %v", completeErr)
		}
	}
}

func newIntegrationGenerationJob(t *testing.T, requestID uuid.UUID, key string, now time.Time) domain.GenerationJob {
	t.Helper()
	targetID := uuid.New()
	job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
		TargetID:       &targetID,
		RequestID:      requestID,
		Kind:           domain.GenerationJobKindLessonContent,
		IdempotencyKey: key,
		Payload:        json.RawMessage(`{"force":false}`),
		AvailableAt:    now.Add(-time.Second),
	}, now)
	if err != nil {
		t.Fatalf("new generation job: %v", err)
	}
	return job
}
