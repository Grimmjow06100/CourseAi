package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

const (
	stepAnalysis                = "analysis"
	stepAnalysisCompleted       = "analysis_completed"
	stepAwaitingClarification   = "awaiting_clarification"
	stepClarificationsCompleted = "clarifications_completed"
	stepArchitecture            = "architecture_generation"
	stepLessonPlan              = "lesson_plan_generation"
	stepLessonContent           = "lesson_content_generation"
	stepGenerationSuccess       = "generation_completed"
)

var (
	ErrCourseGeneratorDependency            = fmt.Errorf("course generator service: %w", contract.ErrServiceDependency)
	ErrPromptRequired                       = contract.ErrPromptRequired
	ErrPromptTooLong                        = contract.ErrPromptTooLong
	ErrGenerationOutOfScope                 = contract.ErrGenerationOutOfScope
	ErrGenerationAnalysisRequired           = contract.ErrGenerationAnalysisRequired
	ErrGenerationBriefRequired              = contract.ErrGenerationBriefRequired
	ErrGenerationAwaitingClarification      = contract.ErrGenerationAwaitingClarification
	ErrGenerationNotAwaitingClarification   = contract.ErrGenerationNotAwaitingClarification
	ErrClarificationAlreadySubmitted        = contract.ErrClarificationAlreadySubmitted
	ErrGenerationNotCompleted               = contract.ErrGenerationNotCompleted
	ErrGenerationNotRetryable               = contract.ErrGenerationNotRetryable
	ErrGenerationStructureRetryNotAllowed   = contract.ErrGenerationStructureRetryNotAllowed
	ErrGenerationStructureRetryStepMismatch = contract.ErrGenerationStructureRetryStepMismatch
	ErrMissingGeneratedCourse               = errors.New("AI generator returned no course")
	ErrMissingGeneratedModules              = errors.New("AI generator returned no modules")
	ErrMissingGeneratedLessons              = errors.New("AI generator returned no lessons")
	ErrMissingGeneratedContent              = errors.New("AI generator returned no lesson content")
)

type CourseGeneratorConfig struct {
	StatusURLFormat        string
	JobStatusURLFormat     string
	ResultURLFormat        string
	ClarificationURLFormat string
	MaxActivePerUser       int64
	MaxDailyPerUser        int64
	MaxPendingJobs         int64
}

type CourseGeneratorService struct {
	ai     contract.CourseAIGenerator
	uow    contract.UnitOfWork
	clock  contract.Clock
	config CourseGeneratorConfig
}

func NewCourseGeneratorService(
	ai contract.CourseAIGenerator,
	uow contract.UnitOfWork,
	clock contract.Clock,
	config CourseGeneratorConfig,
) *CourseGeneratorService {
	if config.MaxActivePerUser == 0 {
		config.MaxActivePerUser = 2
	}
	if config.MaxDailyPerUser == 0 {
		config.MaxDailyPerUser = 10
	}
	if config.MaxPendingJobs == 0 {
		config.MaxPendingJobs = 500
	}
	return &CourseGeneratorService{
		ai:     ai,
		uow:    uow,
		clock:  clock,
		config: config,
	}
}

func (s *CourseGeneratorService) generationStarted(request domain.GenerationRequest, job domain.GenerationJob) contract.GenerationStarted {
	return contract.GenerationStarted{
		JobID:        job.ID,
		RequestID:    request.ID,
		Status:       request.PipelineStatus,
		JobStatus:    job.Status,
		StatusURL:    formatGenerationURL(s.config.StatusURLFormat, "/api/generations/%s/status", request.ID),
		JobStatusURL: formatGenerationURL(s.config.JobStatusURLFormat, "/api/generation-jobs/%s", job.ID),
		ResultURL:    formatGenerationURL(s.config.ResultURLFormat, "/api/generations/%s/result", request.ID),
	}
}

func (s *CourseGeneratorService) validateDependencies() error {
	if s == nil || s.ai == nil || s.uow == nil {
		return ErrCourseGeneratorDependency
	}
	return nil
}

func (s *CourseGeneratorService) now() time.Time {
	if s == nil || s.clock == nil {
		return time.Now()
	}
	return s.clock.Now()
}

func formatGenerationURL(format string, fallbackFormat string, requestID uuid.UUID) string {
	format = strings.TrimSpace(format)
	if format == "" {
		return fmt.Sprintf(fallbackFormat, requestID.String())
	}
	if strings.Contains(format, "%s") {
		return fmt.Sprintf(format, requestID.String())
	}
	return strings.TrimRight(format, "/") + "/" + requestID.String()
}
