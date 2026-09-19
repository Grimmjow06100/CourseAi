package handlers

import (
	"net/http"
	"strconv"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
)

func (h *GenerationHandler) trackingAvailable(c *gin.Context) bool {
	if h.tracking != nil {
		return true
	}
	middlewares.AbortWithError(c, middlewares.ServiceUnavailable("tracking service is unavailable", nil))
	return false
}

func (h *GenerationHandler) Tracking(c *gin.Context) {
	if !h.trackingAvailable(c) {
		return
	}
	id, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}
	result, err := h.tracking.GetGenerationTracking(c.Request.Context(), id)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, result)
}

func (h *GenerationHandler) Events(c *gin.Context) {
	if !h.trackingAvailable(c) {
		return
	}
	id, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}
	cursor, err := strconv.ParseInt(c.DefaultQuery("cursor", "0"), 10, 64)
	if err != nil || cursor < 0 {
		middlewares.AbortWithError(c, middlewares.BadRequest("invalid cursor", err))
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "30"))
	if err != nil || limit < 1 || limit > 100 {
		middlewares.AbortWithError(c, middlewares.BadRequest("invalid limit", err))
		return
	}
	result, err := h.tracking.GetGenerationEvents(c.Request.Context(), id, cursor, limit)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, result)
}

func (h *GenerationHandler) RetryJob(c *gin.Context) {
	if !h.trackingAvailable(c) {
		return
	}
	id, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}
	jobID, ok := parseUUIDParam(c, "jobID")
	if !ok {
		return
	}
	var body struct {
		OperationVersion int `json:"operationVersion" binding:"required,min=1"`
	}
	if !bindJSON(c, &body) {
		return
	}
	h.retryOperations(c, contract.RetryOperationsParams{RequestID: id, IdempotencyKey: c.GetHeader("Idempotency-Key"), Operations: []contract.RetryOperation{{JobID: jobID, OperationVersion: body.OperationVersion}}})
}

func (h *GenerationHandler) RetryFailed(c *gin.Context) {
	if !h.trackingAvailable(c) {
		return
	}
	id, ok := parseUUIDParam(c, "requestID")
	if !ok {
		return
	}
	var body struct {
		Operations []contract.RetryOperation `json:"operations" binding:"required,min=1,max=500"`
	}
	if !bindJSON(c, &body) {
		return
	}
	h.retryOperations(c, contract.RetryOperationsParams{RequestID: id, IdempotencyKey: c.GetHeader("Idempotency-Key"), Operations: body.Operations})
}

func (h *GenerationHandler) retryOperations(c *gin.Context, params contract.RetryOperationsParams) {
	result, err := h.tracking.RetryGenerationOperations(c.Request.Context(), params)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, result)
}
