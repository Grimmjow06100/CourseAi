package middlewares

import (
	"errors"
	"net/http"

	"log/slog"
	"runtime/debug"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/dto"
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

type errorMapping struct {
	target  error
	status  int
	code    string
	message string
}

var applicationErrorMappings = []errorMapping{
	{contract.ErrUnauthenticated, http.StatusUnauthorized, "unauthenticated", "authentication is required"},
	{contract.ErrCourseNotFound, http.StatusNotFound, "course_not_found", "course not found"},
	{contract.ErrGenerationRequestNotFound, http.StatusNotFound, "generation_request_not_found", "generation request not found"},
	{contract.ErrGenerationJobNotFound, http.StatusNotFound, "generation_job_not_found", "generation job not found"},
	{contract.ErrGenerationJobIdempotencyConflict, http.StatusConflict, "idempotency_conflict", "idempotency key is already used by another generation"},
	{contract.ErrGenerationActiveLimitExceeded, http.StatusTooManyRequests, "active_generation_limit_exceeded", "too many active generations"},
	{contract.ErrGenerationDailyLimitExceeded, http.StatusTooManyRequests, "daily_generation_limit_exceeded", "daily generation limit exceeded"},
	{contract.ErrGenerationQueueSaturated, http.StatusServiceUnavailable, "generation_queue_saturated", "generation capacity is temporarily exhausted"},
	{contract.ErrModuleNotFound, http.StatusNotFound, "module_not_found", "module not found"},
	{contract.ErrLessonNotFound, http.StatusNotFound, "lesson_not_found", "lesson not found"},
	{contract.ErrPromptRequired, http.StatusBadRequest, "prompt_required", ""},
	{contract.ErrPromptTooLong, http.StatusBadRequest, "prompt_too_long", ""},
	{contract.ErrGenerationOutOfScope, http.StatusUnprocessableEntity, "generation_out_of_scope", ""},
	{contract.ErrGenerationNotCompleted, http.StatusConflict, "generation_not_completed", ""},
	{contract.ErrGenerationAnalysisRequired, http.StatusConflict, "generation_analysis_required", ""},
	{contract.ErrGenerationBriefRequired, http.StatusConflict, "generation_brief_required", ""},
	{contract.ErrGenerationAwaitingClarification, http.StatusConflict, "generation_awaiting_clarification", ""},
	{contract.ErrGenerationNotAwaitingClarification, http.StatusConflict, "generation_not_awaiting_clarification", ""},
	{contract.ErrClarificationAlreadySubmitted, http.StatusConflict, "clarification_already_submitted", ""},
	{contract.ErrGenerationNotRetryable, http.StatusConflict, "generation_not_retryable", ""},
	{contract.ErrGenerationStructureRetryNotAllowed, http.StatusConflict, "structure_retry_not_allowed", ""},
	{contract.ErrGenerationStructureRetryStepMismatch, http.StatusConflict, "structure_retry_step_mismatch", ""},
	{contract.ErrServiceDependency, http.StatusServiceUnavailable, "service_unavailable", "service is unavailable"},
}

var validationErrors = []error{
	domain.ErrBlankField,
	domain.ErrInvalidCollection,
	domain.ErrInvalidCourseLanguage,
	domain.ErrInvalidCourseStatus,
	domain.ErrInvalidGenerationStatus,
	domain.ErrInvalidLessonType,
	domain.ErrInvalidLevel,
	domain.ErrInvalidOrder,
	domain.ErrInvalidProgress,
	domain.ErrInvalidDuration,
	domain.ErrInvalidClerkUserID,
	domain.ErrInvalidClarification,
	domain.ErrInvalidClarificationAnswer,
	domain.ErrClarificationAnswerMissing,
	domain.ErrClarificationAnswerUnknown,
	domain.ErrClarificationValueNotAllowed,
	domain.ErrGenerationBriefIncomplete,
	domain.ErrGenerationRequestNotReady,
}

func classifyError(err error) (int, string, string) {
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Status, httpErr.Code, httpErr.Message
	}
	for _, mapping := range applicationErrorMappings {
		if errors.Is(err, mapping.target) {
			message := mapping.message
			if message == "" {
				message = err.Error()
			}
			return mapping.status, mapping.code, message
		}
	}
	for _, target := range validationErrors {
		if errors.Is(err, target) {
			return http.StatusBadRequest, "validation_error", err.Error()
		}
	}
	return http.StatusInternalServerError, "internal_error", "internal server error"
}
