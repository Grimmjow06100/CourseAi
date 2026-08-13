package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
)

type Heartbeat struct {
	queue         contract.GenerationJobQueue
	clock         contract.Clock
	interval      time.Duration
	leaseDuration time.Duration
}

func NewHeartbeat(queue contract.GenerationJobQueue, clock contract.Clock, interval, leaseDuration time.Duration) (*Heartbeat, error) {
	if queue == nil {
		return nil, ErrMissingWorkerQueue
	}
	if clock == nil {
		return nil, ErrMissingWorkerClock
	}
	if interval <= 0 || leaseDuration <= 0 || interval >= leaseDuration/2 {
		return nil, fmt.Errorf("%w: invalid heartbeat timing", ErrInvalidWorkerConfig)
	}
	return &Heartbeat{queue: queue, clock: clock, interval: interval, leaseDuration: leaseDuration}, nil
}

// Run renews a claim until the execution context ends or ownership is lost.
func (h *Heartbeat) Run(ctx context.Context, claim domain.JobClaim) error {
	if err := claim.Validate(); err != nil {
		return err
	}
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			lockedUntil := h.clock.Now().Add(h.leaseDuration)
			if err := h.queue.RenewLease(ctx, claim, lockedUntil); err != nil {
				return fmt.Errorf("renew generation job lease: %w", err)
			}
		}
	}
}
