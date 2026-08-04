package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
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

func TestGenerateCourseStructureRequiresAnalyzedRequest(t *testing.T) {
	clock := fixedClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)}
	request, err := domain.NewGenerationRequestAt("je veux une formation sur linux", clock.Now())
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	service := NewCourseGeneratorService(
		fakeCourseAI{},
		&fakeUnitOfWork{requests: map[uuid.UUID]domain.GenerationRequest{request.ID: request}},
		clock,
		CourseGeneratorConfig{},
	)

	_, err = service.GenerateCourseStructure(context.Background(), validStructureParams(request.ID))
	if !errors.Is(err, ErrGenerationAnalysisRequired) {
		t.Fatalf("expected analysis required error, got %v", err)
	}
}

func TestNormalizeGeneratedCourseAllowsGeneratedModulesWithoutIDs(t *testing.T) {
	clock := fixedClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)}
	request, err := domain.NewGenerationRequestAt("je veux une formation sur linux", clock.Now())
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

func TestRetryCourseStructureRequiresFailedRequest(t *testing.T) {
	clock := fixedClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)}
	request := analyzedGenerationRequest(t, clock.Now())

	service := NewCourseGeneratorService(
		fakeCourseAI{},
		&fakeUnitOfWork{requests: map[uuid.UUID]domain.GenerationRequest{request.ID: request}},
		clock,
		CourseGeneratorConfig{},
	)

	_, err := service.RetryCourseStructure(context.Background(), validStructureParams(request.ID))
	if !errors.Is(err, ErrGenerationStructureRetryNotAllowed) {
		t.Fatalf("expected structure retry not allowed error, got %v", err)
	}
}

func TestRetryCourseStructureRequiresStructureFailureStep(t *testing.T) {
	clock := fixedClock{now: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)}
	request := analyzedGenerationRequest(t, clock.Now())
	if err := request.MarkFailed("analysis failed", clock.Now()); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	step := stepAnalysis
	request.CurrentStep = &step

	service := NewCourseGeneratorService(
		fakeCourseAI{},
		&fakeUnitOfWork{requests: map[uuid.UUID]domain.GenerationRequest{request.ID: request}},
		clock,
		CourseGeneratorConfig{},
	)

	_, err := service.RetryCourseStructure(context.Background(), validStructureParams(request.ID))
	if !errors.Is(err, ErrGenerationStructureRetryStepMismatch) {
		t.Fatalf("expected structure retry step mismatch error, got %v", err)
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

	updatedRequest, err := service.prepareStructureRetry(context.Background(), request.ID)
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

	request, err := domain.NewGenerationRequestAt("je veux une formation sur linux", now)
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
		SuggestedTitle:       testStringPtr("Formation Linux"),
		ShortSynopsis:        testStringPtr("Apprendre les bases de Linux."),
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

func testStringPtr(value string) *string {
	return &value
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
	requests map[uuid.UUID]domain.GenerationRequest
	courses  contract.CourseRepository
}

func (u *fakeUnitOfWork) WithinTx(ctx context.Context, fn func(context.Context, contract.TransactionalRepositories) error) error {
	return fn(ctx, fakeRepositories{requests: fakeGenerationRequestRepository{requests: u.requests}, courses: u.courses})
}

type fakeRepositories struct {
	requests fakeGenerationRequestRepository
	courses  contract.CourseRepository
}

func (r fakeRepositories) Users() contract.UserRepository {
	return nil
}

func (r fakeRepositories) GenerationRequests() contract.GenerationRequestRepository {
	return r.requests
}

func (r fakeRepositories) Courses() contract.CourseRepository {
	return r.courses
}

func (r fakeRepositories) Modules() contract.ModuleRepository {
	return nil
}

func (r fakeRepositories) Lessons() contract.LessonRepository {
	return nil
}

type fakeCourseRepository struct {
	deletedRequestID uuid.UUID
	deleteErr        error
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

func (r *fakeCourseRepository) ListCourses(context.Context, contract.CourseFilters) (contract.Page[domain.Course], error) {
	panic("not used")
}

func (r *fakeCourseRepository) DeleteCourse(context.Context, uuid.UUID) error {
	panic("not used")
}

func (r *fakeCourseRepository) DeleteCourseByRequestID(_ context.Context, requestID uuid.UUID) error {
	r.deletedRequestID = requestID
	return r.deleteErr
}

type fakeGenerationRequestRepository struct {
	requests map[uuid.UUID]domain.GenerationRequest
}

func (r fakeGenerationRequestRepository) SaveGenerationRequest(context.Context, domain.GenerationRequest) (domain.GenerationRequest, error) {
	panic("not used")
}

func (r fakeGenerationRequestRepository) UpdateGenerationRequest(_ context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	r.requests[request.ID] = request
	return request, nil
}

func (r fakeGenerationRequestRepository) FindGenerationRequestByID(_ context.Context, id uuid.UUID) (domain.GenerationRequest, error) {
	request, ok := r.requests[id]
	if !ok {
		return domain.GenerationRequest{}, contract.ErrGenerationRequestNotFound
	}
	return request, nil
}

func (r fakeGenerationRequestRepository) FindGenerationRequestByCourseID(context.Context, uuid.UUID) (domain.GenerationRequest, error) {
	panic("not used")
}
