package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestWorkerJobLifecycleLogsContainCorrelationAndOutcome(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		executionErr error
		endEvent     string
		outcome      string
		status       domain.GenerationJobStatus
	}{
		{name: "success", endEvent: "generation_job_completed", outcome: "success", status: domain.GenerationJobStatusCompleted},
		{name: "retry", executionErr: context.DeadlineExceeded, endEvent: "generation_job_retry_scheduled", outcome: "retry_scheduled", status: domain.GenerationJobStatusRetryScheduled},
		{name: "terminal failure", executionErr: domain.ErrBlankField, endEvent: "generation_job_failed", outcome: "failed", status: domain.GenerationJobStatusFailed},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			now := time.Now().UTC()
			job := claimedLoggingTestJob(t, now)
			queue := newFakeJobQueue()
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			pool, err := NewWorkerPool(queue, executorFunc(func(context.Context, domain.GenerationJob) error {
				return testCase.executionErr
			}), &fixedClock{now: now}, testWorkerConfig(), logger)
			if err != nil {
				t.Fatalf("new worker pool: %v", err)
			}

			pool.executeClaimedJob(context.Background(), job)
			records := decodeJSONLogRecords(t, output.Bytes())
			started := findJSONLogEvent(t, records, "generation_job_started")
			finished := findJSONLogEvent(t, records, testCase.endEvent)

			assertJobLogCorrelation(t, started, job)
			assertJobLogCorrelation(t, finished, job)
			assertJSONLogValue(t, started, "outcome", "running")
			assertJSONLogValue(t, finished, "outcome", testCase.outcome)
			assertJSONLogValue(t, finished, "job_status", string(testCase.status))
			if _, exists := finished["duration_ms"]; !exists {
				t.Fatalf("finished log is missing duration_ms: %#v", finished)
			}
			if testCase.executionErr != nil {
				if _, exists := finished["error"]; !exists {
					t.Fatalf("failed log is missing error: %#v", finished)
				}
				if _, exists := finished["error_type"]; !exists {
					t.Fatalf("failed log is missing error_type: %#v", finished)
				}
			}
		})
	}
}

func TestWorkerLifecycleLogsContainWorkerIDAndDuration(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	queue := newFakeJobQueue()
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	pool, err := NewWorkerPool(queue, executorFunc(func(context.Context, domain.GenerationJob) error {
		return nil
	}), &fixedClock{now: now}, testWorkerConfig(), logger)
	if err != nil {
		t.Fatalf("new worker pool: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		pool.runWorker(ctx, "worker-log-test")
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}

	records := decodeJSONLogRecords(t, output.Bytes())
	started := findJSONLogEvent(t, records, "generation_worker_started")
	stopped := findJSONLogEvent(t, records, "generation_worker_stopped")
	assertJSONLogValue(t, started, "worker_id", "worker-log-test")
	assertJSONLogValue(t, stopped, "worker_id", "worker-log-test")
	if _, exists := stopped["duration_ms"]; !exists {
		t.Fatalf("worker stop log is missing duration_ms: %#v", stopped)
	}
}

func claimedLoggingTestJob(t *testing.T, now time.Time) domain.GenerationJob {
	t.Helper()

	job := testJob(t, now)
	workerID := "worker-log-test"
	parentJobID := uuid.New()
	lockedUntil := now.Add(time.Minute)
	job.ParentJobID = &parentJobID
	job.Status = domain.GenerationJobStatusRunning
	job.AttemptCount = 1
	job.LockedBy = &workerID
	job.LockedUntil = &lockedUntil
	job.StartedAt = &now
	return job
}

func decodeJSONLogRecords(t *testing.T, data []byte) []map[string]any {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(data))
	records := make([]map[string]any, 0)
	for {
		var record map[string]any
		if err := decoder.Decode(&record); err != nil {
			if errors.Is(err, io.EOF) {
				return records
			}
			t.Fatalf("decode JSON log: %v\n%s", err, string(data))
		}
		records = append(records, record)
	}
}

func findJSONLogEvent(t *testing.T, records []map[string]any, event string) map[string]any {
	t.Helper()

	for _, record := range records {
		if record["event"] == event {
			return record
		}
	}
	t.Fatalf("log event %q not found in %#v", event, records)
	return nil
}

func assertJobLogCorrelation(t *testing.T, record map[string]any, job domain.GenerationJob) {
	t.Helper()

	assertJSONLogValue(t, record, "component", workerLogComponent)
	assertJSONLogValue(t, record, "worker_id", *job.LockedBy)
	assertJSONLogValue(t, record, "job_id", job.ID.String())
	assertJSONLogValue(t, record, "request_id", job.RequestID.String())
	assertJSONLogValue(t, record, "parent_job_id", job.ParentJobID.String())
	assertJSONLogValue(t, record, "job_kind", string(job.Kind))
	assertJSONLogValue(t, record, "attempt", float64(job.AttemptCount))
	assertJSONLogValue(t, record, "max_attempts", float64(job.MaxAttempts))
}

func assertJSONLogValue(t *testing.T, record map[string]any, key string, expected any) {
	t.Helper()
	if actual, exists := record[key]; !exists || actual != expected {
		t.Fatalf("log field %q = %#v, want %#v in %#v", key, actual, expected, record)
	}
}

var _ contract.GenerationJobQueue = (*fakeJobQueue)(nil)
