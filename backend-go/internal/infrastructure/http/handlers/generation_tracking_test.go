package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type trackingServiceStub struct {
	reads      int
	retries    int
	params     contract.RetryOperationsParams
	retryError error
}

func (s *trackingServiceStub) GetGenerationTracking(_ context.Context, id uuid.UUID) (contract.GenerationTracking, error) {
	s.reads++
	return contract.GenerationTracking{RequestID: id, Revision: "9007199254740993"}, nil
}
func (s *trackingServiceStub) GetGenerationEvents(_ context.Context, _ uuid.UUID, _ int64, _ int) (contract.GenerationEventPage, error) {
	s.reads++
	return contract.GenerationEventPage{Items: []contract.GenerationEvent{}}, nil
}
func (s *trackingServiceStub) RetryGenerationOperations(_ context.Context, params contract.RetryOperationsParams) (contract.RetryOperationsResult, error) {
	s.retries++
	s.params = params
	return contract.RetryOperationsResult{RequestID: params.RequestID, Revision: "42", Replacements: []contract.OperationReplacement{}}, s.retryError
}

func trackingTestRouter(service contract.GenerationTrackingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middlewares.ErrorHandler())
	handler := NewGenerationHandler(nil, nil, service)
	router.GET("/api/generations/:requestID/tracking", handler.Tracking)
	router.GET("/api/generations/:requestID/events", handler.Events)
	router.POST("/api/generations/:requestID/jobs/:jobID/retry", handler.RetryJob)
	router.POST("/api/generations/:requestID/retry-failed", handler.RetryFailed)
	return router
}

func TestTrackingReadsDisableCachingAndNeverRetry(t *testing.T) {
	service := &trackingServiceStub{}
	router := trackingTestRouter(service)
	for _, resource := range []string{"tracking", "events"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/generations/"+uuid.NewString()+"/"+resource, nil))
		if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("unsafe tracking read: %d %v", response.Code, response.Header())
		}
		if resource == "tracking" && !strings.Contains(response.Body.String(), `"revision":"9007199254740993"`) {
			t.Fatal("revision must retain precision as a string")
		}
	}
	if service.reads != 2 || service.retries != 0 {
		t.Fatalf("calls: %+v", service)
	}
}

func TestTrackingRejectsInvalidPaginationBeforeService(t *testing.T) {
	service := &trackingServiceStub{}
	router := trackingTestRouter(service)
	for _, query := range []string{"cursor=-1", "cursor=9223372036854775808", "cursor=abc", "limit=0", "limit=101"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/generations/"+uuid.NewString()+"/events?"+query, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", query, response.Code)
		}
	}
	if service.reads != 0 {
		t.Fatal("invalid pagination reached the service")
	}
}

func TestTrackingRetryForwardsObservedVersionAndIdempotencyKey(t *testing.T) {
	service := &trackingServiceStub{}
	router := trackingTestRouter(service)
	requestID, jobID := uuid.New(), uuid.New()
	for _, version := range []string{"0", "3"} {
		request := httptest.NewRequest(http.MethodPost, "/api/generations/"+requestID.String()+"/jobs/"+jobID.String()+"/retry", strings.NewReader(`{"operationVersion":`+version+`}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "retry-observed-version")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if version == "0" {
			if response.Code != http.StatusBadRequest || service.retries != 0 {
				t.Fatal("invalid version accepted")
			}
		} else if response.Code != http.StatusAccepted {
			t.Fatalf("retry response: %d %s", response.Code, response.Body.String())
		}
	}
	if service.params.RequestID != requestID || service.params.IdempotencyKey != "retry-observed-version" || len(service.params.Operations) != 1 || service.params.Operations[0].JobID != jobID || service.params.Operations[0].OperationVersion != 3 {
		t.Fatalf("retry context lost: %+v", service.params)
	}

	service.retryError = contract.ErrGenerationJobIdempotencyConflict
	request := httptest.NewRequest(http.MethodPost, "/api/generations/"+requestID.String()+"/retry-failed", strings.NewReader(`{"operations":[{"jobId":"`+jobID.String()+`","operationVersion":3}]}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("stale operation must return 409: %d", response.Code)
	}
}
