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
)

var (
	ErrGenerationJobExecutorDependency = errors.New("generation job executor dependency is missing")
	ErrGenerationJobNotExecutable      = errors.New("generation job is not executable")
)

type generationJobRunner interface {
	runAnalysisJob(ctx context.Context, job domain.GenerationJob) error
	runArchitectureJob(ctx context.Context, job domain.GenerationJob) error
	runLessonPlanJob(ctx context.Context, job domain.GenerationJob) error
	runLessonContentJob(ctx context.Context, job domain.GenerationJob) error
	runModuleContentJob(ctx context.Context, job domain.GenerationJob) error
	runFinalizeCourseJob(ctx context.Context, job domain.GenerationJob) error
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
	run, err := e.handlerFor(job.Kind)
	if err != nil {
		return err
	}
	if err := requireEmptyJobPayload(job.Payload); err != nil {
		return err
	}
	err = run(context.WithValue(ctx, generationJobContextKey{}, job), job)
	if errors.Is(err, errGenerationSuperseded) {
		return nil
	}
	return err
}

// ReconcileCompletedGenerations delegates maintenance without running any AI job.
func (e *GenerationJobExecutor) ReconcileCompletedGenerations(ctx context.Context, limit int) error {
	if e == nil || e.runner == nil {
		return ErrGenerationJobExecutorDependency
	}
	if reconciler, ok := e.runner.(contract.GenerationSuccessReconciler); ok {
		return reconciler.ReconcileCompletedGenerations(ctx, limit)
	}
	return nil
}

func (e *GenerationJobExecutor) handlerFor(kind domain.GenerationJobKind) (func(context.Context, domain.GenerationJob) error, error) {
	switch kind {
	case domain.GenerationJobKindAnalysis:
		return e.runner.runAnalysisJob, nil
	case domain.GenerationJobKindArchitecture:
		return e.runner.runArchitectureJob, nil
	case domain.GenerationJobKindLessonPlan:
		return e.runner.runLessonPlanJob, nil
	case domain.GenerationJobKindLessonContent:
		return e.runner.runLessonContentJob, nil
	case domain.GenerationJobKindModuleContent:
		return e.runner.runModuleContentJob, nil
	case domain.GenerationJobKindFinalizeCourse:
		return e.runner.runFinalizeCourseJob, nil
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidGenerationJobKind, kind)
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
