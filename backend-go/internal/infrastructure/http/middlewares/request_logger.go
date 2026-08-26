package middlewares

import (
	"log/slog"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/gin-gonic/gin"
)

func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()
		attributes := []any{
			"event", "http_request_completed",
			"request_id", RequestIDFromContext(c.Request.Context()),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"client_ip", c.ClientIP(),
		}
		if principal, ok := contract.PrincipalFromContext(c.Request.Context()); ok {
			attributes = append(attributes, "clerk_user_id", principal.UserID)
		}
		logger.InfoContext(c.Request.Context(), "http request completed", attributes...)
	}
}
