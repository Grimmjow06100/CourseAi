package contract

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

// Tracking uses metadata only. Lesson bodies, answers and provider diagnostics
// must never be included in this frequently polled projection.
type GenerationTracking struct {
	RequestID           uuid.UUID                       `json:"requestId"`
	GenerationAttempt   int                             `json:"generationAttempt"`
	Revision            string                          `json:"revision"`
	ObservedAt          time.Time                       `json:"observedAt"`
	PipelineStatus      domain.GenerationPipelineStatus `json:"pipelineStatus"`
	Title               string                          `json:"title"`
	CourseID            *uuid.UUID                      `json:"courseId"`
	CreatedAt           time.Time                       `json:"createdAt"`
	CompletedAt         *time.Time                      `json:"completedAt"`
	IsOutOfScope        bool                            `json:"isOutOfScope"`
	HistoryComplete     bool                            `json:"historyComplete"`
	ContentComplete     bool                            `json:"contentComplete"`
	ContentAvailability string                          `json:"contentAvailability"`
	HasActiveWork       bool                            `json:"hasActiveWork"`
	Reconciliation      string                          `json:"reconciliation"`
	Counts              TrackingCounts                  `json:"counts"`
	Phases              []TrackingPhase                 `json:"phases"`
	Modules             []TrackingModule                `json:"modules"`
	Operations          []TrackingOperation             `json:"operations"`
}

type TrackingCounts struct {
	Modules          int  `json:"modules"`
	PlansReady       int  `json:"plansReady"`
	LessonsAvailable int  `json:"lessonsAvailable"`
	LessonsExpected  *int `json:"lessonsExpected"`
	JobsActive       int  `json:"jobsActive"`
	JobsFailed       int  `json:"jobsFailed"`
	JobsCompleted    int  `json:"jobsCompleted"`
}
type TrackingPhase struct {
	Kind   string `json:"kind"`
	Status string `json:"status"`
}
type TrackingModule struct {
	ID      uuid.UUID        `json:"id"`
	Title   string           `json:"title"`
	Order   int              `json:"order"`
	Lessons []TrackingLesson `json:"lessons"`
}
type TrackingLesson struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Order      int       `json:"order"`
	HasContent bool      `json:"hasContent"`
}
type TrackingOperation struct {
	ID               uuid.UUID                  `json:"id"`
	ParentJobID      *uuid.UUID                 `json:"parentJobId"`
	TargetID         *uuid.UUID                 `json:"targetId"`
	Kind             domain.GenerationJobKind   `json:"kind"`
	Status           domain.GenerationJobStatus `json:"status"`
	OperationVersion int                        `json:"operationVersion"`
	SupersedesJobID  *uuid.UUID                 `json:"supersedesJobId"`
	AttemptCount     int                        `json:"attemptCount"`
	MaxAttempts      int                        `json:"maxAttempts"`
	AvailableAt      time.Time                  `json:"availableAt"`
	StartedAt        *time.Time                 `json:"startedAt"`
	CompletedAt      *time.Time                 `json:"completedAt"`
	Retryable        bool                       `json:"retryable"`
	FailureCode      *string                    `json:"failureCode"`
}
type GenerationEvent struct {
	ID                string     `json:"id"`
	GenerationAttempt int        `json:"generationAttempt"`
	JobID             *uuid.UUID `json:"jobId"`
	Kind              string     `json:"kind"`
	Status            string     `json:"status"`
	TargetID          *uuid.UUID `json:"targetId"`
	OperationVersion  *int       `json:"operationVersion"`
	AttemptCount      *int       `json:"attemptCount"`
	OccurredAt        time.Time  `json:"occurredAt"`
}
type GenerationEventPage struct {
	Items      []GenerationEvent `json:"items"`
	NextCursor *string           `json:"nextCursor"`
}
type RetryOperation struct {
	JobID            uuid.UUID `json:"jobId"`
	OperationVersion int       `json:"operationVersion"`
}
type RetryOperationsParams struct {
	RequestID      uuid.UUID
	IdempotencyKey string
	Operations     []RetryOperation
}
type RetryOperationsResult struct {
	Revision          string                 `json:"revision"`
	GenerationAttempt int                    `json:"generationAttempt"`
	RequestID         uuid.UUID              `json:"requestId"`
	Replacements      []OperationReplacement `json:"replacements"`
}
type OperationReplacement struct {
	PreviousJobID    uuid.UUID `json:"previousJobId"`
	JobID            uuid.UUID `json:"jobId"`
	OperationVersion int       `json:"operationVersion"`
}

type GenerationTrackingRepository interface {
	Snapshot(context.Context, uuid.UUID) (GenerationTracking, error)
	Events(context.Context, uuid.UUID, int64, int) ([]GenerationEvent, error)
	Supersede(context.Context, uuid.UUID, int) error
	RetryReceipt(context.Context, uuid.UUID, string) (json.RawMessage, json.RawMessage, error)
	SaveRetryReceipt(context.Context, uuid.UUID, string, json.RawMessage, json.RawMessage) error
}
type GenerationTrackingService interface {
	GetGenerationTracking(context.Context, uuid.UUID) (GenerationTracking, error)
	GetGenerationEvents(context.Context, uuid.UUID, int64, int) (GenerationEventPage, error)
	RetryGenerationOperations(context.Context, RetryOperationsParams) (RetryOperationsResult, error)
}
