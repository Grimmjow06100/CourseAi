package middlewares

import (
	"errors"
	"net/http"

	"log/slog"
	"runtime/debug"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/dto"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/service"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/errtrace"
	"github.com/gin-gonic/gin"
)

type HTTPError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e HTTPError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e HTTPError) Unwrap() error {
	return e.Err
}

func BadRequest(message string, err error) error {
	return HTTPError{Status: http.StatusBadRequest, Code: "bad_request", Message: message, Err: err}
}

func Unauthorized(message string, err error) error {
	return HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: message, Err: err}
}

func ServiceUnavailable(message string, err error) error {
	return HTTPError{Status: http.StatusServiceUnavailable, Code: "service_unavailable", Message: message, Err: err}
}

func PayloadTooLarge(message string, err error) error {
	return HTTPError{Status: http.StatusRequestEntityTooLarge, Code: "payload_too_large", Message: message, Err: err}
}

func TooManyRequests(message string, err error) error {
	return HTTPError{Status: http.StatusTooManyRequests, Code: "rate_limit_exceeded", Message: message, Err: err}
}

func AbortWithError(c *gin.Context, err error) {
	_ = c.Error(errtrace.Capture(err))
	c.Abort()
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(c.Request.Context(), "request panic",
					"event", "http_request_panic",
					"request_id", RequestIDFromContext(c.Request.Context()),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"panic", recovered,
					"stack", string(debug.Stack()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "internal_error", Message: "internal server error"})
			}
		}()
		c.Next()
	}
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		writeError(c, c.Errors.Last().Err)
	}
}

func writeError(c *gin.Context, err error) {
	status, code, message := classifyError(err)

	slog.ErrorContext(
		c.Request.Context(),
		"request failed",
		"status", status,
		"code", code,
		"error", err.Error(),
		"request_id", RequestIDFromContext(c.Request.Context()),
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"stack", errtrace.Stack(err),
	)

	c.JSON(status, dto.ErrorResponse{Code: code, Message: message})
}
func classifyError(err error) (int, string, string) {
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Status, httpErr.Code, httpErr.Message
	}

	switch {
	case errors.Is(err, contract.ErrUnauthenticated):
		return http.StatusUnauthorized, "unauthenticated", "authentication is required"
	case errors.Is(err, contract.ErrCourseNotFound):
		return http.StatusNotFound, "course_not_found", "course not found"
	case errors.Is(err, contract.ErrGenerationRequestNotFound):
		return http.StatusNotFound, "generation_request_not_found", "generation request not found"
	case errors.Is(err, contract.ErrGenerationJobNotFound):
		return http.StatusNotFound, "generation_job_not_found", "generation job not found"
	case errors.Is(err, contract.ErrGenerationJobIdempotencyConflict):
		return http.StatusConflict, "idempotency_conflict", "idempotency key is already used by another generation"
	case errors.Is(err, contract.ErrGenerationActiveLimitExceeded):
		return http.StatusTooManyRequests, "active_generation_limit_exceeded", "too many active generations"
	case errors.Is(err, contract.ErrGenerationDailyLimitExceeded):
		return http.StatusTooManyRequests, "daily_generation_limit_exceeded", "daily generation limit exceeded"
	case errors.Is(err, contract.ErrGenerationQueueSaturated):
		return http.StatusServiceUnavailable, "generation_queue_saturated", "generation capacity is temporarily exhausted"
	case errors.Is(err, contract.ErrModuleNotFound):
		return http.StatusNotFound, "module_not_found", "module not found"
	case errors.Is(err, contract.ErrLessonNotFound):
		return http.StatusNotFound, "lesson_not_found", "lesson not found"
	case errors.Is(err, service.ErrPromptRequired):
		return http.StatusBadRequest, "prompt_required", err.Error()
	case errors.Is(err, service.ErrPromptTooLong):
		return http.StatusBadRequest, "prompt_too_long", err.Error()
	case errors.Is(err, service.ErrGenerationOutOfScope):
		return http.StatusUnprocessableEntity, "generation_out_of_scope", err.Error()
	case errors.Is(err, service.ErrGenerationNotCompleted):
		return http.StatusConflict, "generation_not_completed", err.Error()
	case errors.Is(err, service.ErrGenerationAnalysisRequired):
		return http.StatusConflict, "generation_analysis_required", err.Error()
	case errors.Is(err, service.ErrGenerationBriefRequired):
		return http.StatusConflict, "generation_brief_required", err.Error()
	case errors.Is(err, service.ErrGenerationAwaitingClarification):
		return http.StatusConflict, "generation_awaiting_clarification", err.Error()
	case errors.Is(err, service.ErrGenerationNotAwaitingClarification):
		return http.StatusConflict, "generation_not_awaiting_clarification", err.Error()
	case errors.Is(err, service.ErrClarificationAlreadySubmitted):
		return http.StatusConflict, "clarification_already_submitted", err.Error()
	case errors.Is(err, service.ErrGenerationNotRetryable):
		return http.StatusConflict, "generation_not_retryable", err.Error()
	case errors.Is(err, service.ErrGenerationStructureRetryNotAllowed):
		return http.StatusConflict, "structure_retry_not_allowed", err.Error()
	case errors.Is(err, service.ErrGenerationStructureRetryStepMismatch):
		return http.StatusConflict, "structure_retry_step_mismatch", err.Error()
	case errors.Is(err, service.ErrCourseCatalogDependency), errors.Is(err, service.ErrCourseGeneratorDependency):
		return http.StatusServiceUnavailable, "service_unavailable", err.Error()
	case errors.Is(err, domain.ErrBlankField),
		errors.Is(err, domain.ErrInvalidCollection),
		errors.Is(err, domain.ErrInvalidCourseLanguage),
		errors.Is(err, domain.ErrInvalidCourseStatus),
		errors.Is(err, domain.ErrInvalidGenerationStatus),
		errors.Is(err, domain.ErrInvalidLessonType),
		errors.Is(err, domain.ErrInvalidLevel),
		errors.Is(err, domain.ErrInvalidOrder),
		errors.Is(err, domain.ErrInvalidProgress),
		errors.Is(err, domain.ErrInvalidDuration),
		errors.Is(err, domain.ErrInvalidClerkUserID),
		errors.Is(err, domain.ErrInvalidClarification),
		errors.Is(err, domain.ErrInvalidClarificationAnswer),
		errors.Is(err, domain.ErrClarificationAnswerMissing),
		errors.Is(err, domain.ErrClarificationAnswerUnknown),
		errors.Is(err, domain.ErrClarificationValueNotAllowed),
		errors.Is(err, domain.ErrGenerationBriefIncomplete),
		errors.Is(err, domain.ErrGenerationRequestNotReady):
		return http.StatusBadRequest, "validation_error", err.Error()
	default:
		return http.StatusInternalServerError, "internal_error", "internal server error"
	}
}
