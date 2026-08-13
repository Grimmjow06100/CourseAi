package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestGenerationJobExecutorDispatchesEveryJobKind(t *testing.T) {
	t.Parallel()

	requestID := uuid.New()
	targetID := uuid.New()
	architecturePayload := contract.ArchitectureJobPayload{
		Title:        "Linux administration",
		Synopsis:     "A production-oriented Linux course",
		CurrentLevel: domain.LevelBeginner,
		TargetLevel:  domain.LevelAdvanced,
		Goals:        []string{"administer Linux servers"},
		Language:     domain.CourseLanguageEN,
	}
	encodedArchitecturePayload, err := json.Marshal(architecturePayload)
	if err != nil {
		t.Fatalf("marshal architecture payload: %v", err)
	}

	tests := []struct {
		name    string
		kind    domain.GenerationJobKind
		target  *uuid.UUID
		payload json.RawMessage
	}{
		{name: "full course", kind: domain.GenerationJobKindFullCourse, payload: json.RawMessage(`{}`)},
		{name: "analysis", kind: domain.GenerationJobKindAnalysis, payload: json.RawMessage(`{}`)},
		{name: "architecture", kind: domain.GenerationJobKindArchitecture, payload: encodedArchitecturePayload},
		{name: "lesson plan", kind: domain.GenerationJobKindLessonPlan, target: &targetID, payload: json.RawMessage(`{}`)},
		{name: "lesson content", kind: domain.GenerationJobKindLessonContent, target: &targetID, payload: json.RawMessage(`{}`)},
		{name: "module content", kind: domain.GenerationJobKindModuleContent, target: &targetID, payload: json.RawMessage(`{}`)},
		{name: "finalize course", kind: domain.GenerationJobKindFinalizeCourse, target: &targetID, payload: json.RawMessage(`{}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runner := &recordingGenerationJobRunner{}
			executor := newGenerationJobExecutor(runner)
			job := executableGenerationJob(t, requestID, test.kind, test.target, test.payload)

			if err := executor.Execute(context.Background(), job); err != nil {
				t.Fatalf("execute job: %v", err)
			}
			if runner.kind != test.kind || runner.requestID != requestID {
				t.Fatalf("dispatch kind=%s request=%s, want kind=%s request=%s", runner.kind, runner.requestID, test.kind, requestID)
			}
			if test.target != nil && runner.targetID != *test.target {
				t.Fatalf("target = %s, want %s", runner.targetID, *test.target)
			}
			if test.kind == domain.GenerationJobKindArchitecture && runner.architecturePayload.Title != architecturePayload.Title {
				t.Fatalf("architecture payload = %+v", runner.architecturePayload)
			}
		})
	}
}

func TestGenerationJobExecutorRejectsInvalidPayloads(t *testing.T) {
	t.Parallel()

	requestID := uuid.New()
	tests := []struct {
		name    string
		kind    domain.GenerationJobKind
		payload json.RawMessage
	}{
		{name: "unexpected full course field", kind: domain.GenerationJobKindFullCourse, payload: json.RawMessage(`{"prompt":"duplicate"}`)},
		{name: "unknown architecture field", kind: domain.GenerationJobKindArchitecture, payload: json.RawMessage(`{"unexpected":true}`)},
		{name: "malformed architecture json", kind: domain.GenerationJobKindArchitecture, payload: json.RawMessage(`{"title":`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			job := executableGenerationJob(t, requestID, test.kind, nil, test.payload)
			if err := newGenerationJobExecutor(&recordingGenerationJobRunner{}).Execute(context.Background(), job); !errors.Is(err, domain.ErrInvalidGenerationJobPayload) {
				t.Fatalf("error = %v, want ErrInvalidGenerationJobPayload", err)
			}
		})
	}
}

func TestGenerationJobExecutorRequiresClaimedJob(t *testing.T) {
	t.Parallel()

	now := time.Now()
	job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
		RequestID:      uuid.New(),
		Kind:           domain.GenerationJobKindAnalysis,
		IdempotencyKey: "analysis:not-claimed",
		Payload:        json.RawMessage(`{}`),
		AvailableAt:    now,
	}, now)
	if err != nil {
		t.Fatalf("new job: %v", err)
	}

	err = newGenerationJobExecutor(&recordingGenerationJobRunner{}).Execute(context.Background(), job)
	if !errors.Is(err, ErrGenerationJobNotExecutable) {
		t.Fatalf("error = %v, want ErrGenerationJobNotExecutable", err)
	}
}

func TestGenerationJobExecutorRequiresRunner(t *testing.T) {
	t.Parallel()

	job := executableGenerationJob(t, uuid.New(), domain.GenerationJobKindAnalysis, nil, json.RawMessage(`{}`))
	if err := NewGenerationJobExecutor(nil).Execute(context.Background(), job); !errors.Is(err, ErrGenerationJobExecutorDependency) {
		t.Fatalf("error = %v, want ErrGenerationJobExecutorDependency", err)
	}
}

func TestArchitectureParamsFromJobMergesAnalysisDefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	title := "Generated title"
	synopsis := "Generated synopsis"
	detectedGoal := "learn Linux administration"
	detectedLanguage := domain.CourseLanguageFR
	request := domain.GenerationRequest{
		ID:                uuid.New(),
		InitialUserPrompt: "I want a Linux course",
		SuggestedTitle:    &title,
		ShortSynopsis:     &synopsis,
		DetectedGoal:      &detectedGoal,
		DetectedLanguage:  &detectedLanguage,
	}

	params, err := architectureParamsFromJob(request, contract.ArchitectureJobPayload{
		Title:       "Confirmed Linux course",
		TargetLevel: domain.LevelAdvanced,
	})
	if err != nil {
		t.Fatalf("architecture params: %v", err)
	}
	if params.Title != "Confirmed Linux course" || params.Synopsis != synopsis || params.TargetLevel != domain.LevelAdvanced {
		t.Fatalf("unexpected merged params: %+v", params)
	}
	if len(params.Goals) != 1 || params.Goals[0] != detectedGoal || params.Language != detectedLanguage {
		t.Fatalf("analysis defaults were not preserved: %+v", params)
	}
}

func executableGenerationJob(
	t *testing.T,
	requestID uuid.UUID,
	kind domain.GenerationJobKind,
	targetID *uuid.UUID,
	payload json.RawMessage,
) domain.GenerationJob {
	t.Helper()
	now := time.Now()
	job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
		RequestID:      requestID,
		Kind:           kind,
		TargetID:       targetID,
		IdempotencyKey: string(kind) + ":" + uuid.NewString(),
		Payload:        payload,
		AvailableAt:    now,
	}, now)
	if err != nil {
		// Malformed payloads cannot pass the domain constructor, but the executor
		// still needs to defend against corrupted persistence data.
		if !errors.Is(err, domain.ErrInvalidGenerationJobPayload) {
			t.Fatalf("new job: %v", err)
		}
		job = domain.GenerationJob{
			ID:             uuid.New(),
			RequestID:      requestID,
			Kind:           kind,
			TargetID:       targetID,
			IdempotencyKey: string(kind) + ":" + uuid.NewString(),
			Payload:        payload,
			Priority:       0,
			MaxAttempts:    3,
			AvailableAt:    now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}
	workerID := "test-worker"
	lockedUntil := now.Add(time.Minute)
	job.Status = domain.GenerationJobStatusRunning
	job.AttemptCount = 1
	job.LockedBy = &workerID
	job.LockedUntil = &lockedUntil
	job.StartedAt = &now
	return job
}

type recordingGenerationJobRunner struct {
	kind                domain.GenerationJobKind
	requestID           uuid.UUID
	targetID            uuid.UUID
	architecturePayload contract.ArchitectureJobPayload
}

func (r *recordingGenerationJobRunner) record(kind domain.GenerationJobKind, requestID, targetID uuid.UUID) {
	r.kind = kind
	r.requestID = requestID
	r.targetID = targetID
}

func (r *recordingGenerationJobRunner) runFullCourseJob(_ context.Context, requestID uuid.UUID) error {
	r.record(domain.GenerationJobKindFullCourse, requestID, uuid.Nil)
	return nil
}

func (r *recordingGenerationJobRunner) runAnalysisJob(_ context.Context, requestID uuid.UUID) error {
	r.record(domain.GenerationJobKindAnalysis, requestID, uuid.Nil)
	return nil
}

func (r *recordingGenerationJobRunner) runArchitectureJob(_ context.Context, requestID uuid.UUID, payload contract.ArchitectureJobPayload) error {
	r.record(domain.GenerationJobKindArchitecture, requestID, uuid.Nil)
	r.architecturePayload = payload
	return nil
}

func (r *recordingGenerationJobRunner) runLessonPlanJob(_ context.Context, requestID, moduleID uuid.UUID) error {
	r.record(domain.GenerationJobKindLessonPlan, requestID, moduleID)
	return nil
}

func (r *recordingGenerationJobRunner) runLessonContentJob(_ context.Context, requestID, lessonID uuid.UUID) error {
	r.record(domain.GenerationJobKindLessonContent, requestID, lessonID)
	return nil
}

func (r *recordingGenerationJobRunner) runModuleContentJob(_ context.Context, requestID, moduleID uuid.UUID) error {
	r.record(domain.GenerationJobKindModuleContent, requestID, moduleID)
	return nil
}

func (r *recordingGenerationJobRunner) runFinalizeCourseJob(_ context.Context, requestID, courseID uuid.UUID) error {
	r.record(domain.GenerationJobKindFinalizeCourse, requestID, courseID)
	return nil
}

func (r *recordingGenerationJobRunner) handleTerminalJobFailure(_ context.Context, job domain.GenerationJob, _ error) error {
	r.requestID = job.RequestID
	return nil
}
