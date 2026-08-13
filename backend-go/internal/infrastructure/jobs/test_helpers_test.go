package jobs

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type fixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

type executorFunc func(context.Context, domain.GenerationJob) error

func (f executorFunc) Execute(ctx context.Context, job domain.GenerationJob) error {
	return f(ctx, job)
}

type retryCall struct {
	claim   domain.JobClaim
	retryAt time.Time
	cause   error
}

type failCall struct {
	claim  domain.JobClaim
	cause  error
	failAt time.Time
}

type fakeJobQueue struct {
	mu sync.Mutex

	pending       []domain.GenerationJob
	renewErr      error
	claimErr      error
	renewals      []time.Time
	completed     []domain.JobClaim
	retries       []retryCall
	failures      []failCall
	requeueCalls  int
	completedChan chan domain.JobClaim
	retryChan     chan retryCall
	failChan      chan failCall
}

func newFakeJobQueue(jobs ...domain.GenerationJob) *fakeJobQueue {
	return &fakeJobQueue{
		pending:       append([]domain.GenerationJob(nil), jobs...),
		completedChan: make(chan domain.JobClaim, max(len(jobs), 1)),
		retryChan:     make(chan retryCall, max(len(jobs), 1)),
		failChan:      make(chan failCall, max(len(jobs), 1)),
	}
}

func (q *fakeJobQueue) Enqueue(_ context.Context, job domain.GenerationJob) (domain.GenerationJob, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending = append(q.pending, job)
	return job, nil
}

func (q *fakeJobQueue) FindByID(context.Context, uuid.UUID) (domain.GenerationJob, error) {
	return domain.GenerationJob{}, contract.ErrGenerationJobNotFound
}

func (q *fakeJobQueue) FindByIdempotencyKey(context.Context, string) (domain.GenerationJob, error) {
	return domain.GenerationJob{}, contract.ErrGenerationJobNotFound
}

func (q *fakeJobQueue) ListByRequestID(context.Context, uuid.UUID) ([]domain.GenerationJob, error) {
	return nil, nil
}

func (q *fakeJobQueue) ClaimNext(ctx context.Context, workerID string, lockedUntil time.Time) (domain.GenerationJob, error) {
	if err := ctx.Err(); err != nil {
		return domain.GenerationJob{}, err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.claimErr != nil {
		return domain.GenerationJob{}, q.claimErr
	}
	if len(q.pending) == 0 {
		return domain.GenerationJob{}, contract.ErrGenerationJobUnavailable
	}

	job := q.pending[0]
	q.pending = q.pending[1:]
	now := time.Now()
	job.Status = domain.GenerationJobStatusRunning
	job.AttemptCount++
	job.LockedBy = &workerID
	job.LockedUntil = &lockedUntil
	job.StartedAt = &now
	job.LastErrorCode = nil
	job.LastErrorMessage = nil
	job.UpdatedAt = now
	return job, nil
}

func (q *fakeJobQueue) RenewLease(_ context.Context, _ domain.JobClaim, lockedUntil time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.renewals = append(q.renewals, lockedUntil)
	return q.renewErr
}

func (q *fakeJobQueue) Complete(_ context.Context, claim domain.JobClaim, _ time.Time) error {
	q.mu.Lock()
	q.completed = append(q.completed, claim)
	q.mu.Unlock()
	q.completedChan <- claim
	return nil
}

func (q *fakeJobQueue) Retry(_ context.Context, claim domain.JobClaim, retryAt time.Time, cause error) error {
	call := retryCall{claim: claim, retryAt: retryAt, cause: cause}
	q.mu.Lock()
	q.retries = append(q.retries, call)
	q.mu.Unlock()
	q.retryChan <- call
	return nil
}

func (q *fakeJobQueue) Fail(_ context.Context, claim domain.JobClaim, cause error, failedAt time.Time) error {
	call := failCall{claim: claim, cause: cause, failAt: failedAt}
	q.mu.Lock()
	q.failures = append(q.failures, call)
	q.mu.Unlock()
	q.failChan <- call
	return nil
}

func (q *fakeJobQueue) Cancel(context.Context, uuid.UUID, time.Time) error {
	return nil
}

func (q *fakeJobQueue) RequeueExpired(context.Context, time.Time) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.requeueCalls++
	return 0, nil
}

func (q *fakeJobQueue) snapshot() (renewals, completed, retries, failures, requeues int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.renewals), len(q.completed), len(q.retries), len(q.failures), q.requeueCalls
}

func testJob(t interface {
	Helper()
	Fatalf(string, ...any)
}, now time.Time) domain.GenerationJob {
	t.Helper()
	job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
		RequestID:      uuid.New(),
		Kind:           domain.GenerationJobKindAnalysis,
		IdempotencyKey: uuid.NewString(),
		Payload:        json.RawMessage(`{}`),
		MaxAttempts:    3,
	}, now)
	if err != nil {
		t.Fatalf("new test job: %v", err)
	}
	return job
}

type explicitRetryError struct {
	retryable bool
	delay     time.Duration
}

func (e explicitRetryError) Error() string {
	return "explicit retry error"
}

func (e explicitRetryError) Retryable() bool {
	return e.retryable
}

func (e explicitRetryError) RetryAfter() time.Duration {
	return e.delay
}

var _ contract.GenerationJobQueue = (*fakeJobQueue)(nil)
var _ contract.GenerationJobExecutor = executorFunc(nil)
var _ error = explicitRetryError{}
