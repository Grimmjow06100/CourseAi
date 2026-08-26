package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/gin-gonic/gin"
)

func TestUserRateLimiterRejectsExcessGenerationCommands(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter, err := NewUserRateLimiter(1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(ErrorHandler())
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(contract.ContextWithPrincipal(c.Request.Context(), contract.Principal{UserID: "user_test"}))
		c.Next()
	})
	router.POST("/generation", limiter.Middleware(), func(c *gin.Context) { c.Status(http.StatusAccepted) })

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/generation", nil))
	if first.Code != http.StatusAccepted {
		t.Fatalf("first status = %d", first.Code)
	}
	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/generation", nil))
	if second.Code != http.StatusTooManyRequests || second.Header().Get("Retry-After") == "" {
		t.Fatalf("second response = %d headers=%v body=%s", second.Code, second.Header(), second.Body.String())
	}
}

func TestBodyLimitRejectsOversizedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandler())
	router.Use(BodyLimit(4))
	router.POST("/body", func(c *gin.Context) { c.Status(http.StatusAccepted) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("12345")))
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
}
