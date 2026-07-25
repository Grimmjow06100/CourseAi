package handlers

import (
	"net/http"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/dto"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
)

type GenerationHandler struct {
	service contract.CourseGenerationService
}

func NewGenerationHandler(service contract.CourseGenerationService) *GenerationHandler {
	return &GenerationHandler{service: service}
}

func (h *GenerationHandler) Start(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	var request dto.StartGenerationRequest
	if !bindJSON(c, &request) {
		return
	}

	started, err := h.service.StartFullCourseGeneration(c.Request.Context(), contract.StartGenerationParams{Prompt: request.Prompt})
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, dto.GenerationStartedFromContract(started))
}

func (h *GenerationHandler) Status(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	requestID, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}

	status, err := h.service.GetGenerationStatus(c.Request.Context(), requestID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.GenerationStatusFromContract(status))
}

func (h *GenerationHandler) Result(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	requestID, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}

	result, err := h.service.GetGenerationResult(c.Request.Context(), requestID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.GenerationResultFromContract(result))
}

func (h *GenerationHandler) Retry(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	requestID, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}

	started, err := h.service.RetryFullCourseGeneration(c.Request.Context(), requestID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, dto.GenerationStartedFromContract(started))
}
