package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS([]string{"https://app.example.com"}))
	router.GET("/resource", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/resource", nil)
	request.Header.Set("Origin", "https://app.example.com")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials header, got %q", got)
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS([]string{"https://app.example.com"}))
	router.GET("/resource", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/resource", nil)
	request.Header.Set("Origin", "https://malicious.example.com")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.Code)
	}
}

func TestCORSHandlesPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS([]string{"https://app.example.com"}))
	router.POST("/resource", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "Content-Type, Idempotency-Key")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type, Idempotency-Key" {
		t.Fatalf("unexpected allowed headers: %q", got)
	}
}
