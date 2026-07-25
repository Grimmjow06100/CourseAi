package middlewares

import (
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/gin-gonic/gin"
)

const UserIDContextKey = "user_id"

func AuthRequired(tokenManager contract.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tokenManager == nil {
			AbortWithError(c, ServiceUnavailable("auth service is unavailable", nil))
			return
		}

		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			AbortWithError(c, Unauthorized("missing bearer token", nil))
			return
		}

		userID, err := tokenManager.VerifyToken(token)
		if err != nil {
			AbortWithError(c, Unauthorized("invalid bearer token", err))
			return
		}

		c.Set(UserIDContextKey, userID)
		c.Next()
	}
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
