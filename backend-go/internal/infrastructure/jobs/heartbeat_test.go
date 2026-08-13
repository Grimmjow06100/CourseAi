package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestHeartbeatRenewsLeaseUntilContextEnds(t *testing.T) {
	t.Parallel()
	queue := newFakeJobQueue()
	heartbeat, err := NewHeartbeat(queue, &fixedClock{now: time.Now()}, 5*time.Millisecond, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("new heartbeat: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Millisecond)
	defer cancel()

	err = heartbeat.Run(ctx, domain.JobClaim{JobID: uuid.New(), WorkerID: "worker", AttemptCount: 1})
	if err != nil {
		t.Fatalf("run heartbeat: %v", err)
	}
	renewals, _, _, _, _ := queue.snapshot()
	if renewals < 2 {
		t.Fatalf("renewals = %d, want at least 2", renewals)
	}
}

func TestHeartbeatReturnsClaimLoss(t *testing.T) {
	t.Parallel()
	queue := newFakeJobQueue()
	queue.renewErr = contract.ErrGenerationJobClaimLost
	heartbeat, err := NewHeartbeat(queue, &fixedClock{now: time.Now()}, 5*time.Millisecond, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("new heartbeat: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = heartbeat.Run(ctx, domain.JobClaim{JobID: uuid.New(), WorkerID: "worker", AttemptCount: 1})
	if !errors.Is(err, contract.ErrGenerationJobClaimLost) {
		t.Fatalf("error = %v, want ErrGenerationJobClaimLost", err)
	}
}
