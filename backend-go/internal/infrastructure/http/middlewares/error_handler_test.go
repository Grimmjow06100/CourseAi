package middlewares

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/service"
	"github.com/gin-gonic/gin"
)

func TestClassifyError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "explicit HTTP error", err: BadRequest("invalid payload", errors.New("decode")), status: http.StatusBadRequest, code: "bad_request"},
		{name: "course missing", err: contract.ErrCourseNotFound, status: http.StatusNotFound, code: "course_not_found"},
		{name: "request missing", err: contract.ErrGenerationRequestNotFound, status: http.StatusNotFound, code: "generation_request_not_found"},
		{name: "job missing", err: contract.ErrGenerationJobNotFound, status: http.StatusNotFound, code: "generation_job_not_found"},
		{name: "idempotency conflict", err: contract.ErrGenerationJobIdempotencyConflict, status: http.StatusConflict, code: "idempotency_conflict"},
		{name: "module missing", err: contract.ErrModuleNotFound, status: http.StatusNotFound, code: "module_not_found"},
		{name: "lesson missing", err: contract.ErrLessonNotFound, status: http.StatusNotFound, code: "lesson_not_found"},
		{name: "duplicate username", err: domain.ErrUsernameAlreadyExists, status: http.StatusConflict, code: "username_already_exists"},
		{name: "credentials", err: service.ErrAuthentification, status: http.StatusUnauthorized, code: "invalid_credentials"},
		{name: "out of scope", err: service.ErrGenerationOutOfScope, status: http.StatusUnprocessableEntity, code: "generation_out_of_scope"},
		{name: "validation", err: domain.ErrInvalidLevel, status: http.StatusBadRequest, code: "validation_error"},
		{name: "dependency", err: service.ErrCourseCatalogDependency, status: http.StatusServiceUnavailable, code: "service_unavailable"},
		{name: "unknown", err: errors.New("secret database failure"), status: http.StatusInternalServerError, code: "internal_error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			status, code, _ := classifyError(test.err)
			if status != test.status || code != test.code {
				t.Fatalf("classifyError() = %d, %q; want %d, %q", status, code, test.status, test.code)
			}
		})
	}
}

func TestErrorHandlerWritesSanitizedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandler())
	router.GET("/failure", func(c *gin.Context) {
		AbortWithError(c, errors.New("database password must not leak"))
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/failure", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if response.Body.String() != `{"code":"internal_error","message":"internal server error"}` {
		t.Fatalf("unexpected response body: %s", response.Body.String())
	}
}

func TestHTTPErrorMethods(t *testing.T) {
	t.Parallel()
	cause := errors.New("cause")
	err := ServiceUnavailable("fallback", cause)
	if err.Error() != "cause" || !errors.Is(err, cause) {
		t.Fatalf("HTTPError does not expose its cause: %v", err)
	}
	withoutCause := Unauthorized("missing", nil)
	if withoutCause.Error() != "missing" {
		t.Fatalf("HTTPError fallback message = %q", withoutCause.Error())
	}
}
