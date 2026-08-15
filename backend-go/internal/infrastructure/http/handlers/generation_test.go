package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/dto"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestGenerationHandlerStartReturnsAcceptedJobAndForwardsIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jobID := uuid.New()
	requestID := uuid.New()
	service := &generationServiceStub{}
	service.start = func(_ context.Context, params contract.StartGenerationParams) (contract.GenerationStarted, error) {
		if params.Prompt != "Build a Linux course" || params.IdempotencyKey != "command-123" {
			t.Fatalf("unexpected params: %+v", params)
		}
		return acceptedGeneration(jobID, requestID), nil
	}
	router := generationTestRouter(service)

	request := httptest.NewRequest(http.MethodPost, "/api/generations", bytes.NewBufferString(`{"prompt":"Build a Linux course"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "command-123")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var body dto.GenerationStartedResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.JobID != jobID.String() || body.RequestID != requestID.String() || body.JobStatus != string(domain.GenerationJobStatusQueued) {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestGenerationHandlerStructureEnqueuesAndReturnsAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	requestID := uuid.New()
	jobID := uuid.New()
	service := &generationServiceStub{}
	service.enqueueStructure = func(_ context.Context, params contract.GenerateStructureParams) (contract.GenerationStarted, error) {
		if params.RequestID != requestID || params.Title != "Linux" || params.Language != domain.CourseLanguageFR {
			t.Fatalf("unexpected structure params: %+v", params)
		}
		return acceptedGeneration(jobID, requestID), nil
	}
	router := generationTestRouter(service)
	body := `{
		"title":"Linux",
		"synopsis":"Linux administration",
		"currentLevel":"beginner",
		"targetLevel":"advanced",
		"goals":["Administer servers"],
		"language":"fr"
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/generations/"+requestID.String()+"/structure", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
}

func TestGenerationHandlerSubmitsClarifications(t *testing.T) {
	gin.SetMode(gin.TestMode)

	requestID := uuid.New()
	jobID := uuid.New()
	service := &generationServiceStub{}
	service.submitClarifications = func(_ context.Context, params contract.SubmitClarificationsParams) (contract.GenerationStarted, error) {
		if params.RequestID != requestID || params.Title != "Linux" || params.Language != domain.CourseLanguageFR || len(params.Answers) != 1 {
			t.Fatalf("unexpected clarification params: %+v", params)
		}
		if params.Answers[0].QuestionID != domain.ClarificationIDCurrentLevel || params.Answers[0].SelectedValues[0] != "beginner" {
			t.Fatalf("unexpected clarification answer: %+v", params.Answers[0])
		}
		return acceptedGeneration(jobID, requestID), nil
	}
	router := generationTestRouter(service)
	body := `{
		"answers":[{"questionId":"currentLevel","selectedValues":["beginner"]}],
		"title":"Linux",
		"synopsis":"Linux administration",
		"language":"fr"
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/generations/"+requestID.String()+"/clarifications", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
}

func TestGenerationHandlerRemainingRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	requestID := uuid.New()
	jobID := uuid.New()
	lessonID := uuid.New()
	moduleID := uuid.New()
	service := &generationServiceStub{}
	service.analyze = func(_ context.Context, params contract.AnalyzePromptParams) (contract.GenerationAnalysisResult, error) {
		if params.Prompt != "Learn Linux" {
			t.Fatalf("unexpected analysis params: %+v", params)
		}
		return contract.GenerationAnalysisResult{Request: domain.GenerationRequest{ID: requestID}}, nil
	}
	service.retryStructure = func(_ context.Context, params contract.GenerateStructureParams) (contract.GenerationStarted, error) {
		if params.RequestID != requestID {
			t.Fatalf("unexpected retry request id: %s", params.RequestID)
		}
		return acceptedGeneration(jobID, requestID), nil
	}
	service.lessonContent = func(_ context.Context, id uuid.UUID) (contract.GenerationStarted, error) {
		if id != lessonID {
			t.Fatalf("unexpected lesson id: %s", id)
		}
		return acceptedGeneration(jobID, requestID), nil
	}
	service.moduleContent = func(_ context.Context, id uuid.UUID) (contract.GenerationStarted, error) {
		if id != moduleID {
			t.Fatalf("unexpected module id: %s", id)
		}
		return acceptedGeneration(jobID, requestID), nil
	}
	service.job = func(_ context.Context, id uuid.UUID) (domain.GenerationJob, error) {
		if id != jobID {
			t.Fatalf("unexpected job id: %s", id)
		}
		return domain.GenerationJob{ID: jobID, RequestID: requestID, Status: domain.GenerationJobStatusQueued}, nil
	}
	service.status = func(_ context.Context, id uuid.UUID) (contract.GenerationStatus, error) {
		return contract.GenerationStatus{RequestID: id, PipelineStatus: domain.PipelineStatusRunning}, nil
	}
	service.result = func(_ context.Context, id uuid.UUID) (contract.GenerationResult, error) {
		return contract.GenerationResult{Request: domain.GenerationRequest{ID: id}, Course: domain.Course{ID: uuid.New(), RequestID: id}}, nil
	}
	service.retry = func(_ context.Context, id uuid.UUID) (contract.GenerationStarted, error) {
		return acceptedGeneration(jobID, id), nil
	}
	router := generationTestRouter(service)
	structureBody := `{
		"title":"Linux","synopsis":"Linux administration","currentLevel":"beginner",
		"targetLevel":"advanced","goals":["Administer servers"],"language":"fr"
	}`
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{name: "analyze", method: http.MethodPost, path: "/api/generations/analyze", body: `{"prompt":"Learn Linux"}`, wantStatus: http.StatusCreated},
		{name: "retry structure", method: http.MethodPost, path: "/api/generations/" + requestID.String() + "/structure/retry", body: structureBody, wantStatus: http.StatusAccepted},
		{name: "lesson content", method: http.MethodPost, path: "/api/generations/lessons/" + lessonID.String() + "/content", wantStatus: http.StatusAccepted},
		{name: "module content", method: http.MethodPost, path: "/api/generations/modules/" + moduleID.String() + "/contents", wantStatus: http.StatusAccepted},
		{name: "job status", method: http.MethodGet, path: "/api/generation-jobs/" + jobID.String(), wantStatus: http.StatusOK},
		{name: "generation status", method: http.MethodGet, path: "/api/generations/" + requestID.String() + "/status", wantStatus: http.StatusOK},
		{name: "generation result", method: http.MethodGet, path: "/api/generations/" + requestID.String() + "/result", wantStatus: http.StatusOK},
		{name: "retry generation", method: http.MethodPost, path: "/api/generations/" + requestID.String() + "/retry", wantStatus: http.StatusAccepted},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, bytes.NewBufferString(test.body))
			if test.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestGenerationHandlerRejectsInvalidStructureEnumsAndUnavailableService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	requestID := uuid.New()
	body := `{"title":"Linux","synopsis":"Course","currentLevel":"novice","targetLevel":"advanced","goals":["Learn"],"language":"fr"}`
	request := httptest.NewRequest(http.MethodPost, "/api/generations/"+requestID.String()+"/structure", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	generationTestRouter(&generationServiceStub{}).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid enum status = %d, body=%s", response.Code, response.Body.String())
	}

	unavailable := httptest.NewRecorder()
	generationTestRouter(nil).ServeHTTP(unavailable, httptest.NewRequest(http.MethodPost, "/api/generations", nil))
	if unavailable.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable status = %d", unavailable.Code)
	}
}

func generationTestRouter(service contract.CourseGenerationService) *gin.Engine {
	router := gin.New()
	router.Use(middlewares.ErrorHandler())
	handler := NewGenerationHandler(service)
	router.POST("/api/generations", handler.Start)
	router.POST("/api/generations/analyze", handler.Analyze)
	router.POST("/api/generations/:requestID/clarifications", handler.SubmitClarifications)
	router.POST("/api/generations/:requestID/structure", handler.Structure)
	router.POST("/api/generations/:requestID/structure/retry", handler.RetryStructure)
	router.POST("/api/generations/lessons/:lessonID/content", handler.LessonContent)
	router.POST("/api/generations/modules/:moduleID/contents", handler.ModuleLessonContents)
	router.GET("/api/generation-jobs/:jobID", handler.JobStatus)
	router.GET("/api/generations/:requestID/status", handler.Status)
	router.GET("/api/generations/:requestID/result", handler.Result)
	router.POST("/api/generations/:requestID/retry", handler.Retry)
	return router
}

func acceptedGeneration(jobID, requestID uuid.UUID) contract.GenerationStarted {
	return contract.GenerationStarted{
		JobID:        jobID,
		RequestID:    requestID,
		Status:       domain.PipelineStatusQueued,
		JobStatus:    domain.GenerationJobStatusQueued,
		StatusURL:    "/api/generations/" + requestID.String() + "/status",
		JobStatusURL: "/api/generation-jobs/" + jobID.String(),
		ResultURL:    "/api/generations/" + requestID.String() + "/result",
	}
}

type generationServiceStub struct {
	start                func(context.Context, contract.StartGenerationParams) (contract.GenerationStarted, error)
	enqueueStructure     func(context.Context, contract.GenerateStructureParams) (contract.GenerationStarted, error)
	analyze              func(context.Context, contract.AnalyzePromptParams) (contract.GenerationAnalysisResult, error)
	submitClarifications func(context.Context, contract.SubmitClarificationsParams) (contract.GenerationStarted, error)
	retryStructure       func(context.Context, contract.GenerateStructureParams) (contract.GenerationStarted, error)
	lessonContent        func(context.Context, uuid.UUID) (contract.GenerationStarted, error)
	moduleContent        func(context.Context, uuid.UUID) (contract.GenerationStarted, error)
	job                  func(context.Context, uuid.UUID) (domain.GenerationJob, error)
	status               func(context.Context, uuid.UUID) (contract.GenerationStatus, error)
	result               func(context.Context, uuid.UUID) (contract.GenerationResult, error)
	retry                func(context.Context, uuid.UUID) (contract.GenerationStarted, error)
}

func (s *generationServiceStub) StartFullCourseGeneration(ctx context.Context, params contract.StartGenerationParams) (contract.GenerationStarted, error) {
	if s.start == nil {
		return contract.GenerationStarted{}, nil
	}
	return s.start(ctx, params)
}

func (s *generationServiceStub) EnqueueCourseStructure(ctx context.Context, params contract.GenerateStructureParams) (contract.GenerationStarted, error) {
	if s.enqueueStructure == nil {
		return contract.GenerationStarted{}, nil
	}
	return s.enqueueStructure(ctx, params)
}

func (s *generationServiceStub) AnalyzePrompt(ctx context.Context, params contract.AnalyzePromptParams) (contract.GenerationAnalysisResult, error) {
	if s.analyze == nil {
		return contract.GenerationAnalysisResult{}, nil
	}
	return s.analyze(ctx, params)
}

func (s *generationServiceStub) SubmitClarifications(ctx context.Context, params contract.SubmitClarificationsParams) (contract.GenerationStarted, error) {
	if s.submitClarifications == nil {
		return contract.GenerationStarted{}, nil
	}
	return s.submitClarifications(ctx, params)
}

func (s *generationServiceStub) EnqueueStructureRetry(ctx context.Context, params contract.GenerateStructureParams) (contract.GenerationStarted, error) {
	if s.retryStructure == nil {
		return contract.GenerationStarted{}, nil
	}
	return s.retryStructure(ctx, params)
}

func (s *generationServiceStub) EnqueueLessonContentGeneration(ctx context.Context, id uuid.UUID) (contract.GenerationStarted, error) {
	if s.lessonContent == nil {
		return contract.GenerationStarted{}, nil
	}
	return s.lessonContent(ctx, id)
}

func (s *generationServiceStub) EnqueueModuleContentGeneration(ctx context.Context, id uuid.UUID) (contract.GenerationStarted, error) {
	if s.moduleContent == nil {
		return contract.GenerationStarted{}, nil
	}
	return s.moduleContent(ctx, id)
}

func (s *generationServiceStub) GetGenerationJob(ctx context.Context, id uuid.UUID) (domain.GenerationJob, error) {
	if s.job == nil {
		return domain.GenerationJob{}, nil
	}
	return s.job(ctx, id)
}

func (*generationServiceStub) GenerateCourseStructure(context.Context, contract.GenerateStructureParams) (contract.GenerationResult, error) {
	return contract.GenerationResult{}, nil
}

func (*generationServiceStub) RetryCourseStructure(context.Context, contract.GenerateStructureParams) (contract.GenerationResult, error) {
	return contract.GenerationResult{}, nil
}

func (*generationServiceStub) GenerateLessonContent(context.Context, uuid.UUID) (domain.Lesson, error) {
	return domain.Lesson{}, nil
}

func (*generationServiceStub) GenerateModuleLessonContents(context.Context, uuid.UUID) (domain.Module, error) {
	return domain.Module{}, nil
}

func (s *generationServiceStub) GetGenerationStatus(ctx context.Context, id uuid.UUID) (contract.GenerationStatus, error) {
	if s.status == nil {
		return contract.GenerationStatus{}, nil
	}
	return s.status(ctx, id)
}

func (s *generationServiceStub) GetGenerationResult(ctx context.Context, id uuid.UUID) (contract.GenerationResult, error) {
	if s.result == nil {
		return contract.GenerationResult{}, nil
	}
	return s.result(ctx, id)
}

func (s *generationServiceStub) RetryFullCourseGeneration(ctx context.Context, id uuid.UUID) (contract.GenerationStarted, error) {
	if s.retry == nil {
		return contract.GenerationStarted{}, nil
	}
	return s.retry(ctx, id)
}

var _ contract.CourseGenerationService = (*generationServiceStub)(nil)
