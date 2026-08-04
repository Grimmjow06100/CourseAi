package handlers

import (
	"net/http"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/dto"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

func (h *GenerationHandler) Analyze(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	var request dto.AnalyzeGenerationRequest
	if !bindJSON(c, &request) {
		return
	}

	result, err := h.service.AnalyzePrompt(c.Request.Context(), contract.AnalyzePromptParams{Prompt: request.Prompt})
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.GenerationAnalysisFromContract(result))
}

func (h *GenerationHandler) Structure(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	requestID, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}

	var request dto.GenerateStructureRequest
	if !bindJSON(c, &request) {
		return
	}

	params, err := generateStructureParamsFromRequest(requestID, request)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	result, err := h.service.GenerateCourseStructure(c.Request.Context(), params)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.GenerationResultFromContract(result))
}

func (h *GenerationHandler) RetryStructure(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	requestID, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}

	var request dto.GenerateStructureRequest
	if !bindJSON(c, &request) {
		return
	}

	params, err := generateStructureParamsFromRequest(requestID, request)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	result, err := h.service.RetryCourseStructure(c.Request.Context(), params)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.GenerationResultFromContract(result))
}
func (h *GenerationHandler) LessonContent(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	lessonID, ok := parseUUIDParam(c, "lessonID")
	if !ok {
		return
	}

	lesson, err := h.service.GenerateLessonContent(c.Request.Context(), lessonID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.LessonFromDomain(lesson))
}

func (h *GenerationHandler) ModuleLessonContents(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	moduleID, ok := parseUUIDParam(c, "moduleID")
	if !ok {
		return
	}

	module, err := h.service.GenerateModuleLessonContents(c.Request.Context(), moduleID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ModuleFromDomain(module))
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

func generateStructureParamsFromRequest(requestID uuid.UUID, request dto.GenerateStructureRequest) (contract.GenerateStructureParams, error) {
	currentLevel, err := domain.ParseLevel(request.CurrentLevel)
	if err != nil {
		return contract.GenerateStructureParams{}, err
	}
	targetLevel, err := domain.ParseLevel(request.TargetLevel)
	if err != nil {
		return contract.GenerateStructureParams{}, err
	}
	language, err := domain.ParseCourseLanguage(request.Language)
	if err != nil {
		return contract.GenerateStructureParams{}, err
	}

	return contract.GenerateStructureParams{
		RequestID:    requestID,
		Title:        request.Title,
		Synopsis:     request.Synopsis,
		CurrentLevel: currentLevel,
		TargetLevel:  targetLevel,
		Goals:        request.Goals,
		Language:     language,
	}, nil
}
