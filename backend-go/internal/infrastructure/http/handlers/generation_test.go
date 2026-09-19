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

func TestGenerationHandlerListsPaginatedHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	requestID := uuid.New()
	service := &generationServiceStub{}
	service.list = func(_ context.Context, filters contract.GenerationHistoryFilters) (contract.Page[contract.GenerationSummary], error) {
		if filters.PipelineStatus == nil || *filters.PipelineStatus != domain.PipelineStatusRunning || filters.Pagination.Page != 2 || filters.Pagination.PageSize != 5 {
			t.Fatalf("unexpected history filters: %+v", filters)
		}
		return contract.Page[contract.GenerationSummary]{
			Items: []contract.GenerationSummary{{RequestID: requestID, Title: "Linux", PipelineStatus: domain.PipelineStatusRunning}},
			Page:  2, PageSize: 5, TotalItems: 6, TotalPages: 2, HasPrevious: true,
		}, nil
	}
	response := httptest.NewRecorder()
	generationTestRouter(service).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/generations?status=running&page=2&pageSize=5", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var page dto.PageResponse[dto.GenerationSummaryResponse]
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].RequestID != requestID.String() || !page.HasPrevious {
		t.Fatalf("unexpected page: %+v", page)
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
	handler := NewGenerationHandler(service, service, nil)
	router.GET("/api/generations", handler.List)
	router.POST("/api/generations", handler.Start)
	router.POST("/api/generations/:requestID/clarifications", handler.SubmitClarifications)
	router.POST("/api/generations/:requestID/structure", handler.Structure)
	router.POST("/api/generations/:requestID/structure/retry", handler.RetryStructure)
	router.POST("/api/generations/lessons/:lessonID/content", handler.LessonContent)
	router.POST("/api/generations/modules/:moduleID/contents", handler.ModuleLessonContents)
	router.GET("/api/generation-jobs/:jobID", handler.JobStatus)
	router.GET("/api/generations/:requestID/status", handler.Status)
	router.GET("/api/generations/:requestID/jobs", handler.Jobs)
	router.GET("/api/generations/:requestID/result", handler.Result)
	router.POST("/api/generations/:requestID/retry", handler.Retry)
	router.DELETE("/api/generations/:requestID", handler.Delete)
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
	jobs                 func(context.Context, uuid.UUID) ([]domain.GenerationJob, error)
	list                 func(context.Context, contract.GenerationHistoryFilters) (contract.Page[contract.GenerationSummary], error)
	start                func(context.Context, contract.StartGenerationParams) (contract.GenerationStarted, error)
	enqueueStructure     func(context.Context, contract.GenerateStructureParams) (contract.GenerationStarted, error)
	submitClarifications func(context.Context, contract.SubmitClarificationsParams) (contract.GenerationStarted, error)
	retryStructure       func(context.Context, contract.GenerateStructureParams) (contract.GenerationStarted, error)
	lessonContent        func(context.Context, uuid.UUID) (contract.GenerationStarted, error)
	moduleContent        func(context.Context, uuid.UUID) (contract.GenerationStarted, error)
	job                  func(context.Context, uuid.UUID) (domain.GenerationJob, error)
	status               func(context.Context, uuid.UUID) (contract.GenerationStatus, error)
	result               func(context.Context, uuid.UUID) (contract.GenerationResult, error)
	retry                func(context.Context, uuid.UUID) (contract.GenerationStarted, error)
	deleteRequest        func(context.Context, uuid.UUID) error
}

func (s *generationServiceStub) ListGenerationJobs(ctx context.Context, id uuid.UUID) ([]domain.GenerationJob, error) {
	if s.jobs == nil {
		return nil, nil
	}
	return s.jobs(ctx, id)
}

func (s *generationServiceStub) ListGenerationRequests(ctx context.Context, filters contract.GenerationHistoryFilters) (contract.Page[contract.GenerationSummary], error) {
	if s.list == nil {
		return contract.Page[contract.GenerationSummary]{}, nil
	}
	return s.list(ctx, filters)
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

func (s *generationServiceStub) DeleteGenerationRequest(ctx context.Context, id uuid.UUID) error {
	if s.deleteRequest == nil {
		return nil
	}
	return s.deleteRequest(ctx, id)
}

var _ contract.CourseGenerationService = (*generationServiceStub)(nil)

func TestGenerationHandlerJobsReturnsPublicArray(t *testing.T) {
	requestID := uuid.New()
	for _, empty := range []bool{false, true} {
		service := &generationServiceStub{}
		service.jobs = func(_ context.Context, id uuid.UUID) ([]domain.GenerationJob, error) {
			if id != requestID {
				t.Fatalf("request id = %v", id)
			}
			if empty {
				return nil, nil
			}
			return []domain.GenerationJob{{ID: uuid.New(), RequestID: id, Payload: []byte("{\"private\":true}"), Status: domain.GenerationJobStatusRunning}}, nil
		}
		response := httptest.NewRecorder()
		generationTestRouter(service).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/generations/"+requestID.String()+"/jobs", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", response.Code, response.Body.String())
		}
		if bytes.Contains(response.Body.Bytes(), []byte("private")) || bytes.Contains(response.Body.Bytes(), []byte("payload")) {
			t.Fatal("private payload exposed")
		}
		if empty && response.Body.String() != "[]" {
			t.Fatalf("expected array, got %s", response.Body.String())
		}
	}
}
