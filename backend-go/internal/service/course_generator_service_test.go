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
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/pointer"
	"github.com/google/uuid"
)

func TestNormalizeStructureParamsRequiresRequestID(t *testing.T) {
	_, err := normalizeStructureParams(contract.GenerateStructureParams{
		Title:        "Formation Linux",
		Synopsis:     "Apprendre les bases de Linux.",
		CurrentLevel: domain.LevelBeginner,
		TargetLevel:  domain.LevelIntermediate,
		Goals:        []string{"Comprendre Linux"},
		Language:     domain.CourseLanguageFR,
	})
	if !errors.Is(err, domain.ErrBlankField) {
		t.Fatalf("expected blank field error, got %v", err)
	}
}

func TestNormalizeStructureParamsNormalizesGoals(t *testing.T) {
	requestID := uuid.New()
	params, err := normalizeStructureParams(contract.GenerateStructureParams{
		RequestID:    requestID,
		Title:        " Formation Linux ",
		Synopsis:     " Apprendre Linux. ",
		CurrentLevel: domain.LevelBeginner,
		TargetLevel:  domain.LevelIntermediate,
		Goals:        []string{" ", "Shell", " Administration "},
		Language:     domain.CourseLanguageFR,
	})
	if err != nil {
		t.Fatalf("expected params to be valid: %v", err)
	}
	if params.Title != "Formation Linux" {
		t.Fatalf("unexpected title: %q", params.Title)
	}
	if len(params.Goals) != 2 || params.Goals[0] != "Shell" || params.Goals[1] != "Administration" {
		t.Fatalf("unexpected goals: %#v", params.Goals)
	}
}

func TestNormalizeGeneratedCourseAllowsGeneratedModulesWithoutIDs(t *testing.T) {
	clock := fixedClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)}
	request, err := domain.NewGenerationRequestAt("je veux une formation sur linux", "user_test", clock.Now())
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	service := NewCourseGeneratorService(fakeCourseAI{}, nil, clock, CourseGeneratorConfig{})
	generatedCourse := domain.Course{
		Language:          domain.CourseLanguageFR,
		InitialUserPrompt: request.InitialUserPrompt,
		Title:             "Formation Linux",
		Synopsis:          "Apprendre Linux de maniere progressive.",
		CurrentLevel:      domain.LevelBeginner,
		TargetLevel:       domain.LevelIntermediate,
		Goals:             []string{"Maitriser la ligne de commande"},
		AcquiredSkills:    []string{"Utiliser les commandes Linux essentielles"},
		Modules: []domain.Module{
			{
				Order:             1,
				Title:             "Fondamentaux Linux",
				Description:       "Comprendre le systeme, le shell et les commandes essentielles.",
				KeyLearningPoints: []string{"Shell", "Systeme de fichiers", "Commandes essentielles"},
			},
		},
	}

	course, err := service.normalizeGeneratedCourse(request, generatedCourse)
	if err != nil {
		t.Fatalf("expected generated course with raw modules to be accepted: %v", err)
	}
	if len(course.Modules) != 1 {
		t.Fatalf("expected generated module to be preserved, got %d", len(course.Modules))
	}
	if course.Modules[0].ID != uuid.Nil {
		t.Fatalf("expected raw generated module id to stay empty before module normalization")
	}

	modules, err := service.normalizeGeneratedModules(course.ID, course.Modules)
	if err != nil {
		t.Fatalf("expected modules to be normalized with backend ids: %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("expected one normalized module, got %d", len(modules))
	}
	if modules[0].ID == uuid.Nil {
		t.Fatal("expected normalized module id to be generated")
	}
	if modules[0].CourseID != course.ID {
		t.Fatalf("expected module course id %s, got %s", course.ID, modules[0].CourseID)
	}

	course.Modules = modules
	if err := course.ValidateWithRelations(); err != nil {
		t.Fatalf("expected course with normalized modules to pass relation validation: %v", err)
	}
}

func TestPrepareStructureRetryDeletesPartialCourseAndRestartsRequest(t *testing.T) {
	clock := fixedClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)}
	request := analyzedGenerationRequest(t, clock.Now())
	if err := request.MarkFailed("lesson plan failed", clock.Now()); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	step := stepLessonPlan
	request.CurrentStep = &step

	courses := &fakeCourseRepository{}
	uow := &fakeUnitOfWork{
		requests: map[uuid.UUID]domain.GenerationRequest{request.ID: request},
		courses:  courses,
	}
	service := NewCourseGeneratorService(fakeCourseAI{}, uow, clock, CourseGeneratorConfig{})

	updatedRequest, err := service.prepareStructureRetry(authenticatedTestContext(), request.ID)
	if err != nil {
		t.Fatalf("prepare structure retry: %v", err)
	}
	if courses.deletedRequestID != request.ID {
		t.Fatalf("expected partial course for request %s to be deleted, got %s", request.ID, courses.deletedRequestID)
	}
	if updatedRequest.PipelineStatus != domain.PipelineStatusRunning {
		t.Fatalf("expected running status, got %s", updatedRequest.PipelineStatus)
	}
	if updatedRequest.FailureMessage != nil {
		t.Fatal("expected failure message to be cleared")
	}
	if updatedRequest.CompletedAt != nil {
		t.Fatal("expected completed at to be cleared")
	}
	if updatedRequest.CurrentStep == nil || *updatedRequest.CurrentStep != stepAnalysisCompleted {
		t.Fatalf("expected step %s, got %#v", stepAnalysisCompleted, updatedRequest.CurrentStep)
	}
}

func TestRestartFromFailureClearsFailureState(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	request := analyzedGenerationRequest(t, now)
	if err := request.MarkFailed("required field is blank: module id", now); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	retryTime := now.Add(time.Hour)
	if err := request.RestartFromFailure(stepAnalysisCompleted, 25, retryTime); err != nil {
		t.Fatalf("restart from failure: %v", err)
	}

	if request.PipelineStatus != domain.PipelineStatusRunning {
		t.Fatalf("expected running status, got %s", request.PipelineStatus)
	}
	if request.FailureMessage != nil {
		t.Fatalf("expected failure message to be cleared")
	}
	if request.CompletedAt != nil {
		t.Fatalf("expected completed at to be cleared")
	}
	if request.CurrentStep == nil || *request.CurrentStep != stepAnalysisCompleted {
		t.Fatalf("expected current step %s, got %#v", stepAnalysisCompleted, request.CurrentStep)
	}
	if request.ProgressPercent != 25 {
		t.Fatalf("expected progress 25, got %d", request.ProgressPercent)
	}
}

func TestGetGenerationStatusExposesAnalysisAndClarificationAction(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 15, 10, 0, 0, 0, time.UTC)
	request, err := domain.NewGenerationRequestAt("Build a Linux course", "user_test", now)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if err := request.MarkRunning(stepAnalysis, now); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	unknown := domain.LevelUnknown
	target := domain.LevelAdvanced
	language := domain.CourseLanguageEN
	title := "Linux administration"
	synopsis := "Learn Linux progressively"
	goal := "Administer Linux"
	if err := request.ApplyAnalysis(domain.AnalysisSummary{
		SuggestedTitle:       &title,
		ShortSynopsis:        &synopsis,
		DetectedCurrentLevel: &unknown,
		DetectedTargetLevel:  &target,
		DetectedGoal:         &goal,
		DetectedLanguage:     &language,
		ClarificationQuestions: []domain.ClarificationQuestion{{
			ID:       domain.ClarificationIDCurrentLevel,
			Question: "What is your current level?",
			Options: []domain.ClarificationOption{
				{Value: "beginner", Label: "Beginner"},
				{Value: "intermediate", Label: "Intermediate"},
			},
		}},
	}, now); err != nil {
		t.Fatalf("apply analysis: %v", err)
	}
	if err := request.MarkAwaitingClarification(now); err != nil {
		t.Fatalf("mark awaiting clarification: %v", err)
	}

	service := NewCourseGeneratorService(fakeCourseAI{}, &fakeUnitOfWork{
		requests: map[uuid.UUID]domain.GenerationRequest{request.ID: request},
	}, fixedClock{now: now}, CourseGeneratorConfig{})
	status, err := service.GetGenerationStatus(authenticatedTestContext(), request.ID)
	if err != nil {
		t.Fatalf("GetGenerationStatus() error = %v", err)
	}
	if status.SuggestedTitle == nil || *status.SuggestedTitle != title || status.DetectedLanguage == nil || *status.DetectedLanguage != language {
		t.Fatalf("analysis draft is missing from status: %+v", status)
	}
	if status.ActionRequired == nil || status.ActionRequired.Type != "submit_clarifications" || status.ActionRequired.URL != "/api/generations/"+request.ID.String()+"/clarifications" {
		t.Fatalf("clarification action is missing from status: %+v", status.ActionRequired)
	}
}

func TestAttachGeneratedContentPreservesRawOutput(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	lesson, err := domain.NewLessonAt(domain.NewLessonParams{
		ModuleID:                 uuid.New(),
		Order:                    1,
		Title:                    "Commandes Linux essentielles",
		Type:                     domain.LessonTypeMixed,
		EstimatedDurationMinutes: 45,
		LearningGoal:             "Utiliser les commandes de base du shell.",
		RequiresDiagram:          false,
		TechnicalKeywords:        []string{"shell", "linux"},
	}, now)
	if err != nil {
		t.Fatalf("create lesson: %v", err)
	}

	exercise, err := domain.NewExerciseAt(domain.NewExerciseParams{
		LessonID:             lesson.ID,
		Type:                 domain.ExerciseTypeGuidedLab,
		Difficulty:           domain.DifficultyBeginner,
		Title:                "Explorer le systeme de fichiers",
		Objective:            "Naviguer entre les dossiers.",
		InstructionsMarkdown: "Utilise `pwd`, `ls` et `cd`.",
		ContentMarkdown:      "Liste le contenu de ton repertoire courant.",
		CorrectionMarkdown:   "`pwd` affiche le chemin courant et `ls` liste les fichiers.",
		Payload:              domain.ExercisePayload{"commands": []string{"pwd", "ls", "cd"}},
	}, now)
	if err != nil {
		t.Fatalf("create exercise: %v", err)
	}

	answer := "ls"
	quiz, err := domain.NewQuizAt(domain.NewQuizParams{
		LessonID:   lesson.ID,
		Type:       domain.QuizTypeSingleChoice,
		Difficulty: domain.DifficultyBeginner,
		Title:      "Quiz shell",
		Objective:  "Verifier les commandes de base.",
		Questions: []domain.QuizQuestion{
			{
				Order:    1,
				Type:     domain.QuizQuestionTypeSingleChoice,
				Question: "Quelle commande liste les fichiers ?",
				Options: []domain.QuizOption{
					{Order: 1, Text: "cd"},
					{Order: 2, Text: "ls"},
				},
				Answer:     domain.QuizAnswer{Answer: &answer},
				Correction: "`ls` liste les fichiers du dossier courant.",
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("create quiz: %v", err)
	}

	raw := json.RawMessage(`{"contentMarkdown":"# Linux","exercises":[{"title":"Explorer"}],"quizzes":[{"title":"Quiz shell"}]}`)
	service := NewCourseGeneratorService(fakeCourseAI{}, nil, fixedClock{now: now}, CourseGeneratorConfig{})

	lessonWithContent, err := service.attachGeneratedContent(lesson, contract.LessonContentOutput{
		ContentMarkdown: "# Linux",
		Exercises:       []domain.Exercise{exercise},
		Quizzes:         []domain.Quiz{quiz},
		Raw:             raw,
	})
	if err != nil {
		t.Fatalf("attach generated content: %v", err)
	}

	if string(lessonWithContent.RawContentOutput) != string(raw) {
		t.Fatalf("unexpected lesson raw output: %s", string(lessonWithContent.RawContentOutput))
	}
	if len(lessonWithContent.Exercises) != 1 || string(lessonWithContent.Exercises[0].RawAIOutput) != string(raw) {
		t.Fatalf("unexpected exercise raw output: %#v", lessonWithContent.Exercises)
	}
	if len(lessonWithContent.Quizzes) != 1 || string(lessonWithContent.Quizzes[0].RawAIOutput) != string(raw) {
		t.Fatalf("unexpected quiz raw output: %#v", lessonWithContent.Quizzes)
	}
}

func TestAttachGeneratedContentRejectsActivityPolicyMismatch(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	lesson, err := domain.NewLessonAt(domain.NewLessonParams{
		ModuleID:                 uuid.New(),
		Order:                    1,
		Title:                    "Pratique Linux",
		Type:                     domain.LessonTypePractice,
		EstimatedDurationMinutes: 45,
		LearningGoal:             "Pratiquer les commandes de base du shell.",
		RequiresDiagram:          false,
		TechnicalKeywords:        []string{"shell", "linux"},
	}, now)
	if err != nil {
		t.Fatalf("create lesson: %v", err)
	}

	service := NewCourseGeneratorService(fakeCourseAI{}, nil, fixedClock{now: now}, CourseGeneratorConfig{})
	_, err = service.attachGeneratedContent(lesson, contract.LessonContentOutput{
		ContentMarkdown: "# Linux",
		Exercises:       nil,
		Quizzes:         nil,
	})
	if !errors.Is(err, domain.ErrInvalidCollection) {
		t.Fatalf("expected activity policy error, got: %v", err)
	}
}

func validStructureParams(requestID uuid.UUID) contract.GenerateStructureParams {
	return contract.GenerateStructureParams{
		RequestID:    requestID,
		Title:        "Formation Linux",
		Synopsis:     "Apprendre les bases de Linux.",
		CurrentLevel: domain.LevelBeginner,
		TargetLevel:  domain.LevelIntermediate,
		Goals:        []string{"Comprendre Linux"},
		Language:     domain.CourseLanguageFR,
	}
}

func analyzedGenerationRequest(t *testing.T, now time.Time) domain.GenerationRequest {
	t.Helper()

	request, err := domain.NewGenerationRequestAt("je veux une formation sur linux", "user_test", now)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if err := request.MarkRunning(stepAnalysis, now); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	level := domain.LevelBeginner
	targetLevel := domain.LevelIntermediate
	goal := "Comprendre Linux"
	language := domain.CourseLanguageFR
	if err := request.ApplyAnalysis(domain.AnalysisSummary{
		SuggestedTitle:       pointer.To("Formation Linux"),
		ShortSynopsis:        pointer.To("Apprendre les bases de Linux."),
		DetectedCurrentLevel: &level,
		DetectedTargetLevel:  &targetLevel,
		DetectedGoal:         &goal,
		DetectedLanguage:     &language,
	}, now); err != nil {
		t.Fatalf("apply analysis: %v", err)
	}
	if err := request.UpdateProgress(stepAnalysisCompleted, 25, now); err != nil {
		t.Fatalf("update progress: %v", err)
	}
	return request
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type fakeCourseAI struct{}

func (fakeCourseAI) AnalyzePrompt(context.Context, contract.AnalysisInput) (contract.AnalysisOutput, error) {
	panic("not used")
}

func (fakeCourseAI) GenerateArchitecture(context.Context, contract.ArchitectureInput) (contract.ArchitectureOutput, error) {
	panic("not used")
}

func (fakeCourseAI) GenerateLessonPlan(context.Context, contract.LessonPlanInput) (contract.LessonPlanOutput, error) {
	panic("not used")
}

func (fakeCourseAI) GenerateLessonContent(context.Context, contract.LessonContentInput) (contract.LessonContentOutput, error) {
	panic("not used")
}

type fakeUnitOfWork struct {
	requests       map[uuid.UUID]domain.GenerationRequest
	courses        contract.CourseRepository
	modules        contract.ModuleRepository
	lessons        contract.LessonRepository
	jobs           contract.GenerationJobQueue
	admissionUsage contract.GenerationAdmissionUsage
}

func (u *fakeUnitOfWork) WithinTx(ctx context.Context, fn func(context.Context, contract.TransactionalRepositories) error) error {
	return fn(ctx, fakeRepositories{
		requests: fakeGenerationRequestRepository{requests: u.requests, admissionUsage: u.admissionUsage},
		courses:  u.courses,
		modules:  u.modules,
		lessons:  u.lessons,
		jobs:     u.jobs,
	})
}

type fakeRepositories struct {
	requests fakeGenerationRequestRepository
	courses  contract.CourseRepository
	modules  contract.ModuleRepository
	lessons  contract.LessonRepository
	jobs     contract.GenerationJobQueue
}

func (r fakeRepositories) Ownership() contract.OwnershipRepository {
	return allowAllOwnership{}
}

func (r fakeRepositories) GenerationRequests() contract.GenerationRequestRepository {
	return r.requests
}

func (r fakeRepositories) GenerationJobs() contract.GenerationJobQueue {
	return r.jobs
}

func (r fakeRepositories) Courses() contract.CourseRepository {
	return r.courses
}

func (r fakeRepositories) Modules() contract.ModuleRepository {
	return r.modules
}

func (r fakeRepositories) Lessons() contract.LessonRepository {
	return r.lessons
}

func (r fakeRepositories) Exercises() contract.ExerciseRepository {
	return nil
}

func (r fakeRepositories) Quizzes() contract.QuizRepository {
	return nil
}

type allowAllOwnership struct{ contract.OwnershipRepository }

func (allowAllOwnership) OwnsGenerationRequest(context.Context, uuid.UUID, string) (bool, error) {
	return true, nil
}

func (allowAllOwnership) OwnsGenerationJob(context.Context, uuid.UUID, string) (bool, error) {
	return true, nil
}

func (allowAllOwnership) OwnsCourse(context.Context, uuid.UUID, string) (bool, error) {
	return true, nil
}

func (allowAllOwnership) OwnsModule(context.Context, uuid.UUID, string) (bool, error) {
	return true, nil
}

func (allowAllOwnership) OwnsLesson(context.Context, uuid.UUID, string) (bool, error) {
	return true, nil
}

type fakeCourseRepository struct {
	deletedRequestID uuid.UUID
	deleteErr        error
	courseByID       map[uuid.UUID]domain.Course
	findErr          error
}

func (r *fakeCourseRepository) SaveCourse(context.Context, domain.Course) (domain.Course, error) {
	panic("not used")
}

func (r *fakeCourseRepository) UpdateCourse(context.Context, domain.Course) (domain.Course, error) {
	panic("not used")
}

func (r *fakeCourseRepository) FindCourseByID(context.Context, uuid.UUID) (domain.Course, error) {
	panic("not used")
}

func (r *fakeCourseRepository) FindCourseByRequestID(context.Context, uuid.UUID) (domain.Course, error) {
	panic("not used")
}

func (r *fakeCourseRepository) FindCourseStateByID(context.Context, uuid.UUID) (domain.Course, error) {
	if r.findErr != nil {
		return domain.Course{}, r.findErr
	}
	for _, course := range r.courseByID {
		return course, nil
	}
	return domain.Course{}, contract.ErrCourseNotFound
}

func (r *fakeCourseRepository) FindCourseStateByRequestID(context.Context, uuid.UUID) (domain.Course, error) {
	panic("not used")
}

func (r *fakeCourseRepository) IsCourseContentComplete(context.Context, uuid.UUID) (bool, error) {
	panic("not used")
}

func (r *fakeCourseRepository) ListCourses(context.Context, contract.CourseFilters) (contract.Page[domain.Course], error) {
	panic("not used")
}

func (r *fakeCourseRepository) DeleteCourse(context.Context, uuid.UUID) error {
	panic("not used")
}

func (r *fakeCourseRepository) DeleteCourseGeneration(context.Context, uuid.UUID) error {
	panic("not used")
}

func (r *fakeCourseRepository) DeleteCourseByRequestID(_ context.Context, requestID uuid.UUID) error {
	r.deletedRequestID = requestID
	return r.deleteErr
}

type fakeGenerationRequestRepository struct {
	requests       map[uuid.UUID]domain.GenerationRequest
	admissionUsage contract.GenerationAdmissionUsage
}

func (r fakeGenerationRequestRepository) SaveGenerationRequest(_ context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	r.requests[request.ID] = request
	return request, nil
}

func (r fakeGenerationRequestRepository) UpdateGenerationRequest(_ context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	r.requests[request.ID] = request
	return request, nil
}

func (r fakeGenerationRequestRepository) ListGenerationRequests(_ context.Context, filters contract.GenerationHistoryFilters) (contract.Page[contract.GenerationSummary], error) {
	items := make([]contract.GenerationSummary, 0, len(r.requests))
	for _, request := range r.requests {
		if request.ClerkUserID != filters.ClerkUserID {
			continue
		}
		if filters.PipelineStatus != nil && request.PipelineStatus != *filters.PipelineStatus {
			continue
		}
		title := request.InitialUserPrompt
		if request.SuggestedTitle != nil {
			title = *request.SuggestedTitle
		}
		items = append(items, contract.GenerationSummary{
			RequestID: request.ID, InitialUserPrompt: request.InitialUserPrompt, Title: title,
			PipelineStatus: request.PipelineStatus, CurrentStep: request.CurrentStep,
			ProgressPercent: request.ProgressPercent, IsOutOfScope: request.IsOutOfScope,
			FailureMessage: request.FailureMessage, CreatedAt: request.CreatedAt, UpdatedAt: request.UpdatedAt,
		})
	}
	sort.Slice(items, func(left, right int) bool { return items[left].CreatedAt.After(items[right].CreatedAt) })
	pagination := filters.Pagination.Normalize()
	totalItems := len(items)
	start := min((pagination.Page-1)*pagination.PageSize, totalItems)
	end := min(start+pagination.PageSize, totalItems)
	totalPages := 0
	if totalItems > 0 {
		totalPages = (totalItems + pagination.PageSize - 1) / pagination.PageSize
	}
	return contract.Page[contract.GenerationSummary]{
		Items: items[start:end], Page: pagination.Page, PageSize: pagination.PageSize,
		TotalItems: totalItems, TotalPages: totalPages, HasNext: pagination.Page < totalPages,
		HasPrevious: pagination.Page > 1,
	}, nil
}

func (r fakeGenerationRequestRepository) FindGenerationRequestByID(_ context.Context, id uuid.UUID) (domain.GenerationRequest, error) {
	request, ok := r.requests[id]
	if !ok {
		return domain.GenerationRequest{}, contract.ErrGenerationRequestNotFound
	}
	return request, nil
}

func (r fakeGenerationRequestRepository) FindGenerationRequestForUpdate(ctx context.Context, id uuid.UUID) (domain.GenerationRequest, error) {
	return r.FindGenerationRequestByID(ctx, id)
}

func (r fakeGenerationRequestRepository) FindGenerationRequestByCourseID(context.Context, uuid.UUID) (domain.GenerationRequest, error) {
	panic("not used")
}

func (r fakeGenerationRequestRepository) FindGenerationStatusByID(_ context.Context, id uuid.UUID) (contract.GenerationStatus, error) {
	request, ok := r.requests[id]
	if !ok {
		return contract.GenerationStatus{}, contract.ErrGenerationRequestNotFound
	}
	return contract.GenerationStatus{
		RequestID:              request.ID,
		PipelineStatus:         request.PipelineStatus,
		CurrentStep:            request.CurrentStep,
		ProgressPercent:        request.ProgressPercent,
		FailureMessage:         request.FailureMessage,
		IsOutOfScope:           request.IsOutOfScope,
		ErrorMessage:           request.ErrorMessage,
		WarningMessage:         request.WarningMessage,
		SuggestedTitle:         request.SuggestedTitle,
		ShortSynopsis:          request.ShortSynopsis,
		DetectedCurrentLevel:   request.DetectedCurrentLevel,
		DetectedTargetLevel:    request.DetectedTargetLevel,
		DetectedGoal:           request.DetectedGoal,
		DetectedLanguage:       request.DetectedLanguage,
		ClarificationQuestions: request.ClarificationQuestions,
	}, nil
}

func (r fakeGenerationRequestRepository) GetGenerationAdmissionUsage(context.Context, string, time.Time) (contract.GenerationAdmissionUsage, error) {
	return r.admissionUsage, nil
}

func (r fakeGenerationRequestRepository) DeleteGenerationRequest(_ context.Context, id uuid.UUID) error {
	if _, ok := r.requests[id]; !ok {
		return contract.ErrGenerationRequestNotFound
	}
	delete(r.requests, id)
	return nil
}
