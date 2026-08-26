package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestStagedGenerationJobsCompletePersistedPipeline(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 13, 14, 0, 0, 0, time.UTC)
	store := newPipelineMemoryStore()
	ai := &pipelineAIStub{}
	service := NewCourseGeneratorService(ai, pipelineMemoryUnitOfWork{store: store}, fixedClock{now: now}, CourseGeneratorConfig{})

	started, err := service.StartFullCourseGeneration(authenticatedTestContext(), contract.StartGenerationParams{Prompt: "Build a Linux course"})
	if err != nil {
		t.Fatalf("start generation: %v", err)
	}
	requestID := started.RequestID
	analysisJob := pipelineJobByKind(t, store, requestID, domain.GenerationJobKindAnalysis)
	if err := service.runAnalysisJob(authenticatedTestContext(), analysisJob); err != nil {
		t.Fatalf("runAnalysisJob() error = %v", err)
	}
	architectureJob := pipelineJobByKind(t, store, requestID, domain.GenerationJobKindArchitecture)
	if err := service.runArchitectureJob(authenticatedTestContext(), architectureJob); err != nil {
		t.Fatalf("runArchitectureJob() error = %v", err)
	}
	lessonPlanJob := pipelineJobByKind(t, store, requestID, domain.GenerationJobKindLessonPlan)
	if err := service.runLessonPlanJob(authenticatedTestContext(), lessonPlanJob); err != nil {
		t.Fatalf("runLessonPlanJob() error = %v", err)
	}
	lessonContentJob := pipelineJobByKind(t, store, requestID, domain.GenerationJobKindLessonContent)
	if err := service.runLessonContentJob(authenticatedTestContext(), lessonContentJob); err != nil {
		t.Fatalf("runLessonContentJob() error = %v", err)
	}
	finalizeJob := pipelineJobByKind(t, store, requestID, domain.GenerationJobKindFinalizeCourse)
	if err := service.runFinalizeCourseJob(authenticatedTestContext(), finalizeJob); err != nil {
		t.Fatalf("runFinalizeCourseJob() error = %v", err)
	}
	completedRequest := store.requests[requestID]
	if completedRequest.PipelineStatus != domain.PipelineStatusCompleted || completedRequest.ProgressPercent != 100 {
		t.Fatalf("unexpected request state: %+v", completedRequest)
	}
	course, err := store.courseByRequestID(requestID)
	if err != nil {
		t.Fatalf("find generated course: %v", err)
	}
	if course.Status != domain.CourseStatusCompleted || len(course.Modules) != 1 || len(course.Modules[0].Lessons) != 1 {
		t.Fatalf("unexpected generated course: %+v", course)
	}
	lesson := course.Modules[0].Lessons[0]
	if lesson.ContentMarkdown == nil || *lesson.ContentMarkdown != "# Linux filesystem" {
		t.Fatalf("lesson content was not persisted: %+v", lesson)
	}
	if ai.analysisCalls != 1 || ai.architectureCalls != 1 || ai.lessonPlanCalls != 1 || ai.lessonContentCalls != 1 {
		t.Fatalf("unexpected AI calls: %+v", ai)
	}

	if err := service.runAnalysisJob(authenticatedTestContext(), analysisJob); err != nil {
		t.Fatalf("idempotent completed analysis job error = %v", err)
	}
	if ai.lessonContentCalls != 1 {
		t.Fatalf("completed pipeline should not call AI again, calls=%d", ai.lessonContentCalls)
	}
}

func TestAnalysisJobWaitsForClarificationsBeforeEnqueuingArchitecture(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 13, 15, 0, 0, 0, time.UTC)
	store := newPipelineMemoryStore()
	ai := &pipelineAIStub{needsClarification: true}
	service := NewCourseGeneratorService(ai, pipelineMemoryUnitOfWork{store: store}, fixedClock{now: now}, CourseGeneratorConfig{})

	started, err := service.StartFullCourseGeneration(authenticatedTestContext(), contract.StartGenerationParams{Prompt: "Build a Linux course"})
	if err != nil {
		t.Fatalf("start generation: %v", err)
	}
	analysisJob := pipelineJobByKind(t, store, started.RequestID, domain.GenerationJobKindAnalysis)
	if err := service.runAnalysisJob(authenticatedTestContext(), analysisJob); err != nil {
		t.Fatalf("runAnalysisJob() error = %v", err)
	}

	waiting := store.requests[started.RequestID]
	if waiting.PipelineStatus != domain.PipelineStatusAwaitingClarification || len(waiting.ClarificationQuestions) != 1 {
		t.Fatalf("request should wait for clarification: %+v", waiting)
	}
	jobs, err := store.jobs.ListByRequestID(authenticatedTestContext(), started.RequestID)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	for _, job := range jobs {
		if job.Kind == domain.GenerationJobKindArchitecture {
			t.Fatal("architecture job must not exist before clarification")
		}
	}
	if store.jobs.count() != 1 || ai.analysisCalls != 1 {
		t.Fatalf("unexpected work while waiting: jobs=%d analysisCalls=%d", store.jobs.count(), ai.analysisCalls)
	}

	accepted, err := service.SubmitClarifications(authenticatedTestContext(), contract.SubmitClarificationsParams{
		RequestID: started.RequestID,
		Answers: []domain.ClarificationAnswer{{
			QuestionID:     domain.ClarificationIDCurrentLevel,
			SelectedValues: []string{"beginner"},
		}},
		Title:    "Linux fundamentals",
		Synopsis: "Learn Linux progressively",
		Language: domain.CourseLanguageEN,
	})
	if err != nil {
		t.Fatalf("SubmitClarifications() error = %v", err)
	}
	architectureJob, err := store.jobs.FindByID(authenticatedTestContext(), accepted.JobID)
	if err != nil || architectureJob.Kind != domain.GenerationJobKindArchitecture {
		t.Fatalf("architecture continuation = %+v, %v", architectureJob, err)
	}
	resumed := store.requests[started.RequestID]
	if resumed.PipelineStatus != domain.PipelineStatusQueued || resumed.ConfirmedBrief == nil || resumed.ClarificationVersion != 1 {
		t.Fatalf("request was not resumed with a confirmed brief: %+v", resumed)
	}
	if ai.analysisCalls != 1 {
		t.Fatalf("clarification submission should not repeat analysis, calls=%d", ai.analysisCalls)
	}
}

func TestRetryResumesPersistedClarificationWithoutCallingAIAgain(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 13, 16, 0, 0, 0, time.UTC)
	store := newPipelineMemoryStore()
	ai := &pipelineAIStub{needsClarification: true}
	service := NewCourseGeneratorService(ai, pipelineMemoryUnitOfWork{store: store}, fixedClock{now: now}, CourseGeneratorConfig{})

	started, err := service.StartFullCourseGeneration(authenticatedTestContext(), contract.StartGenerationParams{Prompt: "Build a Linux course"})
	if err != nil {
		t.Fatalf("start generation: %v", err)
	}
	analysisJob := pipelineJobByKind(t, store, started.RequestID, domain.GenerationJobKindAnalysis)
	if err := service.runAnalysisJob(authenticatedTestContext(), analysisJob); err != nil {
		t.Fatalf("run analysis: %v", err)
	}
	failed := store.requests[started.RequestID]
	if err := failed.MarkFailed("transition persistence failed", now); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	store.requests[started.RequestID] = failed

	retry, err := service.RetryFullCourseGeneration(authenticatedTestContext(), started.RequestID)
	if err != nil {
		t.Fatalf("retry generation: %v", err)
	}
	retryJob, err := store.jobs.FindByID(authenticatedTestContext(), retry.JobID)
	if err != nil || retryJob.Kind != domain.GenerationJobKindAnalysis {
		t.Fatalf("clarification resume job = %+v, %v", retryJob, err)
	}
	if err := service.runAnalysisJob(authenticatedTestContext(), retryJob); err != nil {
		t.Fatalf("resume analysis state: %v", err)
	}
	resumed := store.requests[started.RequestID]
	if resumed.PipelineStatus != domain.PipelineStatusAwaitingClarification || ai.analysisCalls != 1 {
		t.Fatalf("unexpected resumed clarification: request=%+v analysisCalls=%d", resumed, ai.analysisCalls)
	}
}

func TestAnalyzePromptPersistsFailureAndOutOfScopeCompletion(t *testing.T) {
	t.Parallel()

	now := time.Now()
	t.Run("provider failure", func(t *testing.T) {
		store := newPipelineMemoryStore()
		ai := &pipelineAIStub{analysisErr: errors.New("provider unavailable")}
		service := NewCourseGeneratorService(ai, pipelineMemoryUnitOfWork{store: store}, fixedClock{now: now}, CourseGeneratorConfig{})
		if _, err := service.AnalyzePrompt(authenticatedTestContext(), contract.AnalyzePromptParams{Prompt: "Linux"}); err == nil {
			t.Fatal("expected analysis failure")
		}
		for _, request := range store.requests {
			if request.PipelineStatus != domain.PipelineStatusFailed || request.FailureMessage == nil {
				t.Fatalf("failure was not persisted: %+v", request)
			}
			return
		}
		t.Fatal("generation request was not persisted")
	})

	t.Run("out of scope", func(t *testing.T) {
		store := newPipelineMemoryStore()
		ai := &pipelineAIStub{outOfScope: true}
		service := NewCourseGeneratorService(ai, pipelineMemoryUnitOfWork{store: store}, fixedClock{now: now}, CourseGeneratorConfig{})
		result, err := service.AnalyzePrompt(authenticatedTestContext(), contract.AnalyzePromptParams{Prompt: "Write a poem"})
		if err != nil {
			t.Fatalf("AnalyzePrompt() error = %v", err)
		}
		if !result.Request.IsOutOfScope || result.Request.PipelineStatus != domain.PipelineStatusCompleted {
			t.Fatalf("unexpected out-of-scope result: %+v", result.Request)
		}
	})
}

type pipelineAIStub struct {
	analysisCalls      int
	architectureCalls  int
	lessonPlanCalls    int
	lessonContentCalls int
	analysisErr        error
	outOfScope         bool
	needsClarification bool
}

func (a *pipelineAIStub) AnalyzePrompt(context.Context, contract.AnalysisInput) (contract.AnalysisOutput, error) {
	a.analysisCalls++
	if a.analysisErr != nil {
		return contract.AnalysisOutput{}, a.analysisErr
	}
	current := domain.LevelBeginner
	questions := []domain.ClarificationQuestion(nil)
	if a.needsClarification {
		current = domain.LevelUnknown
		questions = []domain.ClarificationQuestion{{
			ID:       domain.ClarificationIDCurrentLevel,
			Question: "What is your current Linux level?",
			Options: []domain.ClarificationOption{
				{Value: "beginner", Label: "Beginner"},
				{Value: "intermediate", Label: "Intermediate"},
			},
		}}
	}
	target := domain.LevelIntermediate
	language := domain.CourseLanguageEN
	title := "Linux fundamentals"
	synopsis := "Learn Linux progressively"
	goal := "Use Linux autonomously"
	return contract.AnalysisOutput{
		Summary: domain.AnalysisSummary{
			IsOutOfScope: a.outOfScope, SuggestedTitle: &title, ShortSynopsis: &synopsis,
			DetectedCurrentLevel: &current, DetectedTargetLevel: &target,
			DetectedGoal: &goal, DetectedLanguage: &language, ClarificationQuestions: questions,
		},
		Raw: json.RawMessage(`{"analysis":true}`),
	}, nil
}

func (a *pipelineAIStub) GenerateArchitecture(_ context.Context, input contract.ArchitectureInput) (contract.ArchitectureOutput, error) {
	a.architectureCalls++
	return contract.ArchitectureOutput{
		Course: domain.Course{
			Language: input.Language, InitialUserPrompt: input.Request.InitialUserPrompt,
			Title: input.Title, Synopsis: input.Synopsis, CurrentLevel: input.CurrentLevel,
			TargetLevel: input.TargetLevel, Goals: input.Goals,
			Modules: []domain.Module{{Order: 1, Title: "Linux basics", Description: "Filesystem and shell"}},
		},
		Raw: json.RawMessage(`{"architecture":true}`),
	}, nil
}

func (a *pipelineAIStub) GenerateLessonPlan(_ context.Context, input contract.LessonPlanInput) (contract.LessonPlanOutput, error) {
	a.lessonPlanCalls++
	return contract.LessonPlanOutput{
		Lessons: []domain.Lesson{{
			Order: 1, Title: "Filesystem", Type: domain.LessonTypeTheory,
			EstimatedDurationMinutes: 20, LearningGoal: "Navigate Linux files",
		}},
		Raw: json.RawMessage(`{"lessons":true}`),
	}, nil
}

func (a *pipelineAIStub) GenerateLessonContent(_ context.Context, input contract.LessonContentInput) (contract.LessonContentOutput, error) {
	a.lessonContentCalls++
	return contract.LessonContentOutput{
		Lesson: input.Lesson, ContentMarkdown: "# Linux filesystem",
		Raw: json.RawMessage(`{"contentMarkdown":"# Linux filesystem"}`),
	}, nil
}

type pipelineMemoryStore struct {
	requests  map[uuid.UUID]domain.GenerationRequest
	courses   map[uuid.UUID]domain.Course
	modules   map[uuid.UUID]domain.Module
	lessons   map[uuid.UUID]domain.Lesson
	exercises map[uuid.UUID][]domain.Exercise
	quizzes   map[uuid.UUID][]domain.Quiz
	jobs      *memoryGenerationJobQueue
}

func newPipelineMemoryStore() *pipelineMemoryStore {
	return &pipelineMemoryStore{
		requests: make(map[uuid.UUID]domain.GenerationRequest), courses: make(map[uuid.UUID]domain.Course),
		modules: make(map[uuid.UUID]domain.Module), lessons: make(map[uuid.UUID]domain.Lesson),
		exercises: make(map[uuid.UUID][]domain.Exercise), quizzes: make(map[uuid.UUID][]domain.Quiz),
		jobs: newMemoryGenerationJobQueue(),
	}
}

func (s *pipelineMemoryStore) hydratedCourse(id uuid.UUID) (domain.Course, error) {
	course, ok := s.courses[id]
	if !ok {
		return domain.Course{}, contract.ErrCourseNotFound
	}
	course.Modules = nil
	for _, module := range s.modules {
		if module.CourseID != id {
			continue
		}
		hydrated, _ := s.hydratedModule(module.ID)
		course.Modules = append(course.Modules, hydrated)
	}
	sort.Slice(course.Modules, func(i, j int) bool { return course.Modules[i].Order < course.Modules[j].Order })
	return course, nil
}

func (s *pipelineMemoryStore) courseByRequestID(requestID uuid.UUID) (domain.Course, error) {
	for id, course := range s.courses {
		if course.RequestID == requestID {
			return s.hydratedCourse(id)
		}
	}
	return domain.Course{}, contract.ErrCourseNotFound
}

func (s *pipelineMemoryStore) hydratedModule(id uuid.UUID) (domain.Module, error) {
	module, ok := s.modules[id]
	if !ok {
		return domain.Module{}, contract.ErrModuleNotFound
	}
	module.Lessons = nil
	for _, lesson := range s.lessons {
		if lesson.ModuleID == id {
			lesson.Exercises = append([]domain.Exercise(nil), s.exercises[lesson.ID]...)
			lesson.Quizzes = append([]domain.Quiz(nil), s.quizzes[lesson.ID]...)
			module.Lessons = append(module.Lessons, lesson)
		}
	}
	sort.Slice(module.Lessons, func(i, j int) bool { return module.Lessons[i].Order < module.Lessons[j].Order })
	return module, nil
}

type pipelineMemoryUnitOfWork struct{ store *pipelineMemoryStore }

func (u pipelineMemoryUnitOfWork) WithinTx(ctx context.Context, fn func(context.Context, contract.TransactionalRepositories) error) error {
	return fn(ctx, pipelineMemoryRepositories{store: u.store})
}

type pipelineMemoryRepositories struct {
	contract.TransactionalRepositories
	store *pipelineMemoryStore
}

func (r pipelineMemoryRepositories) GenerationRequests() contract.GenerationRequestRepository {
	return pipelineRequestRepository{store: r.store}
}
func (r pipelineMemoryRepositories) Courses() contract.CourseRepository {
	return pipelineCourseRepository{store: r.store}
}
func (r pipelineMemoryRepositories) Modules() contract.ModuleRepository {
	return pipelineModuleRepository{store: r.store}
}
func (r pipelineMemoryRepositories) Lessons() contract.LessonRepository {
	return pipelineLessonRepository{store: r.store}
}
func (r pipelineMemoryRepositories) Exercises() contract.ExerciseRepository {
	return pipelineExerciseRepository{store: r.store}
}
func (r pipelineMemoryRepositories) Quizzes() contract.QuizRepository {
	return pipelineQuizRepository{store: r.store}
}
func (r pipelineMemoryRepositories) GenerationJobs() contract.GenerationJobQueue {
	return r.store.jobs
}
func (r pipelineMemoryRepositories) Ownership() contract.OwnershipRepository {
	return allowAllOwnership{}
}

func pipelineJobByKind(t *testing.T, store *pipelineMemoryStore, requestID uuid.UUID, kind domain.GenerationJobKind) domain.GenerationJob {
	t.Helper()
	jobs, err := store.jobs.ListByRequestID(authenticatedTestContext(), requestID)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	for _, job := range jobs {
		if job.Kind == kind {
			return job
		}
	}
	t.Fatalf("job kind %s not found in %+v", kind, jobs)
	return domain.GenerationJob{}
}

type pipelineRequestRepository struct {
	contract.GenerationRequestRepository
	store *pipelineMemoryStore
}

func (r pipelineRequestRepository) SaveGenerationRequest(_ context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	r.store.requests[request.ID] = request
	return request, nil
}
func (r pipelineRequestRepository) UpdateGenerationRequest(_ context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	r.store.requests[request.ID] = request
	return request, nil
}
func (r pipelineRequestRepository) FindGenerationRequestByID(_ context.Context, id uuid.UUID) (domain.GenerationRequest, error) {
	request, ok := r.store.requests[id]
	if !ok {
		return domain.GenerationRequest{}, contract.ErrGenerationRequestNotFound
	}
	return request, nil
}

func (r pipelineRequestRepository) FindGenerationRequestForUpdate(ctx context.Context, id uuid.UUID) (domain.GenerationRequest, error) {
	return r.FindGenerationRequestByID(ctx, id)
}

func (r pipelineRequestRepository) GetGenerationAdmissionUsage(context.Context, string, time.Time) (contract.GenerationAdmissionUsage, error) {
	return contract.GenerationAdmissionUsage{}, nil
}

func (r pipelineRequestRepository) DeleteGenerationRequest(_ context.Context, id uuid.UUID) error {
	if _, ok := r.store.requests[id]; !ok {
		return contract.ErrGenerationRequestNotFound
	}
	delete(r.store.requests, id)
	return nil
}

type pipelineCourseRepository struct {
	contract.CourseRepository
	store *pipelineMemoryStore
}

func (r pipelineCourseRepository) SaveCourse(_ context.Context, course domain.Course) (domain.Course, error) {
	stored := course
	stored.Modules = nil
	r.store.courses[course.ID] = stored
	return course, nil
}
func (r pipelineCourseRepository) UpdateCourse(_ context.Context, course domain.Course) (domain.Course, error) {
	stored := course
	stored.Modules = nil
	r.store.courses[course.ID] = stored
	return course, nil
}
func (r pipelineCourseRepository) FindCourseByID(_ context.Context, id uuid.UUID) (domain.Course, error) {
	return r.store.hydratedCourse(id)
}
func (r pipelineCourseRepository) FindCourseStateByID(_ context.Context, id uuid.UUID) (domain.Course, error) {
	return r.store.hydratedCourse(id)
}
func (r pipelineCourseRepository) FindCourseByRequestID(_ context.Context, id uuid.UUID) (domain.Course, error) {
	return r.store.courseByRequestID(id)
}
func (r pipelineCourseRepository) FindCourseStateByRequestID(_ context.Context, id uuid.UUID) (domain.Course, error) {
	return r.store.courseByRequestID(id)
}
func (r pipelineCourseRepository) IsCourseContentComplete(_ context.Context, id uuid.UUID) (bool, error) {
	course, err := r.store.hydratedCourse(id)
	if err != nil {
		return false, err
	}
	return course.HasCompleteContent(), nil
}

type pipelineModuleRepository struct {
	contract.ModuleRepository
	store *pipelineMemoryStore
}

func (r pipelineModuleRepository) SaveModules(_ context.Context, modules []domain.Module) ([]domain.Module, error) {
	for _, module := range modules {
		stored := module
		stored.Lessons = nil
		r.store.modules[module.ID] = stored
	}
	return modules, nil
}
func (r pipelineModuleRepository) UpdateModule(_ context.Context, module domain.Module) (domain.Module, error) {
	stored := module
	stored.Lessons = nil
	r.store.modules[module.ID] = stored
	return module, nil
}
func (r pipelineModuleRepository) FindModuleByID(_ context.Context, id uuid.UUID) (domain.Module, error) {
	return r.store.hydratedModule(id)
}

type pipelineLessonRepository struct {
	contract.LessonRepository
	store *pipelineMemoryStore
}

func (r pipelineLessonRepository) SaveLessons(_ context.Context, lessons []domain.Lesson) ([]domain.Lesson, error) {
	for _, lesson := range lessons {
		stored := lesson
		stored.Exercises = nil
		stored.Quizzes = nil
		r.store.lessons[lesson.ID] = stored
	}
	return lessons, nil
}
func (r pipelineLessonRepository) ReplaceLessonContent(_ context.Context, lesson domain.Lesson) (domain.Lesson, error) {
	stored := lesson
	stored.Exercises = nil
	stored.Quizzes = nil
	r.store.lessons[lesson.ID] = stored
	return lesson, nil
}
func (r pipelineLessonRepository) FindLessonByID(_ context.Context, id uuid.UUID) (domain.Lesson, error) {
	lesson, ok := r.store.lessons[id]
	if !ok {
		return domain.Lesson{}, contract.ErrLessonNotFound
	}
	lesson.Exercises = append([]domain.Exercise(nil), r.store.exercises[id]...)
	lesson.Quizzes = append([]domain.Quiz(nil), r.store.quizzes[id]...)
	return lesson, nil
}

type pipelineExerciseRepository struct {
	contract.ExerciseRepository
	store *pipelineMemoryStore
}

func (r pipelineExerciseRepository) SaveExercises(_ context.Context, exercises []domain.Exercise) ([]domain.Exercise, error) {
	for _, exercise := range exercises {
		r.store.exercises[exercise.LessonID] = append(r.store.exercises[exercise.LessonID], exercise)
	}
	return exercises, nil
}

type pipelineQuizRepository struct {
	contract.QuizRepository
	store *pipelineMemoryStore
}

func (r pipelineQuizRepository) SaveQuizzes(_ context.Context, quizzes []domain.Quiz) ([]domain.Quiz, error) {
	for _, quiz := range quizzes {
		r.store.quizzes[quiz.LessonID] = append(r.store.quizzes[quiz.LessonID], quiz)
	}
	return quizzes, nil
}
