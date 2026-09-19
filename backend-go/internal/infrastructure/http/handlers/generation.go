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
	commands contract.GenerationCommandService
	queries  contract.GenerationQueryService
	tracking contract.GenerationTrackingService
}

type structureOperation uint8

const (
	createStructure structureOperation = iota
	retryStructure
)

func NewGenerationHandler(commands contract.GenerationCommandService, queries contract.GenerationQueryService) *GenerationHandler {
	tracking, _ := queries.(contract.GenerationTrackingService)
	return &GenerationHandler{commands: commands, queries: queries, tracking: tracking}
}

func (h *GenerationHandler) List(c *gin.Context) {
	if h.queries == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}
	filters, ok := parseGenerationHistoryFilters(c)
	if !ok {
		return
	}
	page, err := h.queries.ListGenerationRequests(c.Request.Context(), filters)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.GenerationPageFromContract(page))
}

func (h *GenerationHandler) Start(c *gin.Context) {
	if h.commands == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	var request dto.StartGenerationRequest
	if !bindJSON(c, &request) {
		return
	}

	started, err := h.commands.StartFullCourseGeneration(c.Request.Context(), contract.StartGenerationParams{
		Prompt:         request.Prompt,
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
	})
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, dto.GenerationStartedFromContract(started))
}

func (h *GenerationHandler) SubmitClarifications(c *gin.Context) {
	if h.commands == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}
	requestID, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}
	var request dto.SubmitClarificationsRequest
	if !bindJSON(c, &request) {
		return
	}
	language, err := domain.ParseCourseLanguage(request.Language)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}
	answers := make([]domain.ClarificationAnswer, 0, len(request.Answers))
	for _, answer := range request.Answers {
		answers = append(answers, domain.ClarificationAnswer{
			QuestionID:     answer.QuestionID,
			SelectedValues: answer.SelectedValues,
		})
	}
	started, err := h.commands.SubmitClarifications(c.Request.Context(), contract.SubmitClarificationsParams{
		RequestID: requestID,
		Answers:   answers,
		Title:     request.Title,
		Synopsis:  request.Synopsis,
		Language:  language,
	})
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, dto.GenerationStartedFromContract(started))
}

func (h *GenerationHandler) Structure(c *gin.Context) {
	h.handleStructure(c, createStructure)
}

func (h *GenerationHandler) RetryStructure(c *gin.Context) {
	h.handleStructure(c, retryStructure)
}

func (h *GenerationHandler) handleStructure(c *gin.Context, operation structureOperation) {
	if h.commands == nil {
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

	var started contract.GenerationStarted
	if operation == retryStructure {
		started, err = h.commands.EnqueueStructureRetry(c.Request.Context(), params)
	} else {
		started, err = h.commands.EnqueueCourseStructure(c.Request.Context(), params)
	}
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, dto.GenerationStartedFromContract(started))
}
func (h *GenerationHandler) LessonContent(c *gin.Context) {
	if h.commands == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	lessonID, ok := parseUUIDParam(c, "lessonID")
	if !ok {
		return
	}

	started, err := h.commands.EnqueueLessonContentGeneration(c.Request.Context(), lessonID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, dto.GenerationStartedFromContract(started))
}

func (h *GenerationHandler) ModuleLessonContents(c *gin.Context) {
	if h.commands == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	moduleID, ok := parseUUIDParam(c, "moduleID")
	if !ok {
		return
	}

	started, err := h.commands.EnqueueModuleContentGeneration(c.Request.Context(), moduleID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, dto.GenerationStartedFromContract(started))
}

func (h *GenerationHandler) JobStatus(c *gin.Context) {
	if h.queries == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	jobID, ok := parseUUIDParam(c, "jobID")
	if !ok {
		return
	}
	job, err := h.queries.GetGenerationJob(c.Request.Context(), jobID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.GenerationJobFromDomain(job))
}

func (h *GenerationHandler) Jobs(c *gin.Context) {
	if h.queries == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}
	requestID, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}
	jobs, err := h.queries.ListGenerationJobs(c.Request.Context(), requestID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}
	response := make([]dto.GenerationJobResponse, 0, len(jobs))
	for _, job := range jobs {
		response = append(response, dto.GenerationJobFromDomain(job))
	}
	c.JSON(http.StatusOK, response)
}

func (h *GenerationHandler) Status(c *gin.Context) {
	if h.queries == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	requestID, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}

	status, err := h.queries.GetGenerationStatus(c.Request.Context(), requestID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.GenerationStatusFromContract(status))
}

func (h *GenerationHandler) Result(c *gin.Context) {
	if h.queries == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	requestID, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}

	result, err := h.queries.GetGenerationResult(c.Request.Context(), requestID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.GenerationResultFromContract(result))
}

func (h *GenerationHandler) Retry(c *gin.Context) {
	if h.commands == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}

	requestID, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}

	started, err := h.commands.RetryFullCourseGeneration(c.Request.Context(), requestID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, dto.GenerationStartedFromContract(started))
}

func (h *GenerationHandler) Delete(c *gin.Context) {
	if h.commands == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("generation service is unavailable", nil))
		return
	}
	requestID, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}
	if err := h.commands.DeleteGenerationRequest(c.Request.Context(), requestID); err != nil {
		middlewares.AbortWithError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
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
