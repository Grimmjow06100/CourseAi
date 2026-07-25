package middlewares

import (
	"errors"
	"net/http"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/dto"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/service"
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

func AbortWithError(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
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
	c.JSON(status, dto.ErrorResponse{Code: code, Message: message})
}

func classifyError(err error) (int, string, string) {
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Status, httpErr.Code, httpErr.Message
	}

	switch {
	case errors.Is(err, contract.ErrCourseNotFound):
		return http.StatusNotFound, "course_not_found", "course not found"
	case errors.Is(err, contract.ErrGenerationRequestNotFound):
		return http.StatusNotFound, "generation_request_not_found", "generation request not found"
	case errors.Is(err, contract.ErrModuleNotFound):
		return http.StatusNotFound, "module_not_found", "module not found"
	case errors.Is(err, contract.ErrLessonNotFound):
		return http.StatusNotFound, "lesson_not_found", "lesson not found"
	case errors.Is(err, domain.ErrUserNotFound):
		return http.StatusNotFound, "user_not_found", "user not found"
	case errors.Is(err, domain.ErrUsernameAlreadyExists):
		return http.StatusConflict, "username_already_exists", err.Error()
	case errors.Is(err, service.ErrAuthentification):
		return http.StatusUnauthorized, "invalid_credentials", err.Error()
	case errors.Is(err, service.ErrPromptRequired):
		return http.StatusBadRequest, "prompt_required", err.Error()
	case errors.Is(err, service.ErrGenerationOutOfScope):
		return http.StatusUnprocessableEntity, "generation_out_of_scope", err.Error()
	case errors.Is(err, service.ErrGenerationNotCompleted):
		return http.StatusConflict, "generation_not_completed", err.Error()
	case errors.Is(err, service.ErrGenerationNotRetryable):
		return http.StatusConflict, "generation_not_retryable", err.Error()
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
		errors.Is(err, domain.ErrInvalidPassword),
		errors.Is(err, domain.ErrInvalidUsername),
		errors.Is(err, domain.ErrInvalidClarification):
		return http.StatusBadRequest, "validation_error", err.Error()
	default:
		return http.StatusInternalServerError, "internal_error", "internal server error"
	}
}
