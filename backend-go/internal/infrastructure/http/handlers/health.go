package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	readyCheck    func(context.Context) error
	workerEnabled bool
}

func NewHealthHandler(readyCheck func(context.Context) error, workerEnabled bool) *HealthHandler {
	return &HealthHandler{readyCheck: readyCheck, workerEnabled: workerEnabled}
}

func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	if !h.workerEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "worker_disabled"})
		return
	}
	if h.readyCheck != nil {
		if err := h.readyCheck(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "database_unavailable"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
