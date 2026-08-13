package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrGenerationJobExecutorDependency = errors.New("generation job executor dependency is missing")
	ErrGenerationJobNotExecutable      = errors.New("generation job is not executable")
)

type generationJobRunner interface {
	runFullCourseJob(ctx context.Context, requestID uuid.UUID) error
	runAnalysisJob(ctx context.Context, requestID uuid.UUID) error
	runArchitectureJob(ctx context.Context, requestID uuid.UUID, payload contract.ArchitectureJobPayload) error
	runLessonPlanJob(ctx context.Context, requestID, moduleID uuid.UUID) error
	runLessonContentJob(ctx context.Context, requestID, lessonID uuid.UUID) error
	runModuleContentJob(ctx context.Context, requestID, moduleID uuid.UUID) error
	runFinalizeCourseJob(ctx context.Context, requestID, courseID uuid.UUID) error
	handleTerminalJobFailure(ctx context.Context, job domain.GenerationJob, cause error) error
}

// GenerationJobExecutor dispatches a claimed job to its application use case.
type GenerationJobExecutor struct {
	runner generationJobRunner
}

// NewGenerationJobExecutor builds the application executor used by the worker pool.
func NewGenerationJobExecutor(generator *CourseGeneratorService) *GenerationJobExecutor {
	if generator == nil {
		return &GenerationJobExecutor{}
	}
	return &GenerationJobExecutor{runner: generator}
}

func newGenerationJobExecutor(runner generationJobRunner) *GenerationJobExecutor {
	return &GenerationJobExecutor{runner: runner}
}

// Execute validates a claimed job and dispatches it according to its kind.
func (e *GenerationJobExecutor) Execute(ctx context.Context, job domain.GenerationJob) error {
	if e == nil || e.runner == nil {
		return ErrGenerationJobExecutorDependency
	}
	if err := validateExecutableJob(job); err != nil {
		return err
	}

	switch job.Kind {
	case domain.GenerationJobKindFullCourse:
		if err := requireEmptyJobPayload(job.Payload); err != nil {
			return err
		}
		return e.runner.runFullCourseJob(ctx, job.RequestID)
	case domain.GenerationJobKindAnalysis:
		if err := requireEmptyJobPayload(job.Payload); err != nil {
			return err
		}
		return e.runner.runAnalysisJob(ctx, job.RequestID)
	case domain.GenerationJobKindArchitecture:
		var payload contract.ArchitectureJobPayload
		if err := decodeJobPayload(job.Payload, &payload); err != nil {
			return err
		}
		return e.runner.runArchitectureJob(ctx, job.RequestID, payload)
	case domain.GenerationJobKindLessonPlan:
		if err := requireEmptyJobPayload(job.Payload); err != nil {
			return err
		}
		return e.runner.runLessonPlanJob(ctx, job.RequestID, *job.TargetID)
	case domain.GenerationJobKindLessonContent:
		if err := requireEmptyJobPayload(job.Payload); err != nil {
			return err
		}
		return e.runner.runLessonContentJob(ctx, job.RequestID, *job.TargetID)
	case domain.GenerationJobKindModuleContent:
		if err := requireEmptyJobPayload(job.Payload); err != nil {
			return err
		}
		return e.runner.runModuleContentJob(ctx, job.RequestID, *job.TargetID)
	case domain.GenerationJobKindFinalizeCourse:
		if err := requireEmptyJobPayload(job.Payload); err != nil {
			return err
		}
		return e.runner.runFinalizeCourseJob(ctx, job.RequestID, *job.TargetID)
	default:
		return fmt.Errorf("%w: %s", domain.ErrInvalidGenerationJobKind, job.Kind)
	}
}

// HandleTerminalFailure updates the request and course after retries are exhausted.
func (e *GenerationJobExecutor) HandleTerminalFailure(ctx context.Context, job domain.GenerationJob, cause error) error {
	if e == nil || e.runner == nil {
		return ErrGenerationJobExecutorDependency
	}
	if cause == nil {
		return fmt.Errorf("%w: terminal failure cause", domain.ErrBlankField)
	}
	return e.runner.handleTerminalJobFailure(ctx, job, cause)
}

func validateExecutableJob(job domain.GenerationJob) error {
	if err := job.Validate(); err != nil {
		return err
	}
	if job.Status != domain.GenerationJobStatusRunning {
		return fmt.Errorf("%w: status=%s", ErrGenerationJobNotExecutable, job.Status)
	}
	return nil
}

func requireEmptyJobPayload(raw json.RawMessage) error {
	var payload map[string]json.RawMessage
	if err := decodeJobPayload(raw, &payload); err != nil {
		return err
	}
	if len(payload) != 0 {
		return fmt.Errorf("%w: this job kind does not accept payload fields", domain.ErrInvalidGenerationJobPayload)
	}
	return nil
}

func decodeJobPayload(raw json.RawMessage, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidGenerationJobPayload, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: payload must contain exactly one JSON object", domain.ErrInvalidGenerationJobPayload)
	}
	return nil
}
