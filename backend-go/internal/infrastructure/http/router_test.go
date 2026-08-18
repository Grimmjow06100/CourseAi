package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewRouterRegistersHealthAndAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(RouterConfig{AllowedOrigins: []string{"https://app.example.com"}})

	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK || health.Body.String() != `{"status":"ok"}` {
		t.Fatalf("health response = %d %s", health.Code, health.Body.String())
	}

	api := httptest.NewRecorder()
	router.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/courses", nil))
	if api.Code != http.StatusServiceUnavailable {
		t.Fatalf("registered API route status = %d, want 503", api.Code)
	}
}
