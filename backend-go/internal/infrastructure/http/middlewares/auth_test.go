package middlewares

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		header string
		want   string
	}{
		{header: "Bearer token", want: "token"},
		{header: " bearer   token ", want: "token"},
		{header: "Basic token"},
		{header: "Bearer"},
		{header: "Bearer   "},
		{header: ""},
	}
	for _, test := range tests {
		if got := bearerToken(test.header); got != test.want {
			t.Fatalf("bearerToken(%q) = %q, want %q", test.header, got, test.want)
		}
	}
}

func TestAuthRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	tests := []struct {
		name       string
		manager    *middlewareTokenManager
		header     string
		wantStatus int
	}{
		{name: "valid", manager: &middlewareTokenManager{userID: userID}, header: "Bearer valid", wantStatus: http.StatusNoContent},
		{name: "missing token", manager: &middlewareTokenManager{}, wantStatus: http.StatusUnauthorized},
		{name: "invalid token", manager: &middlewareTokenManager{err: errors.New("invalid")}, header: "Bearer invalid", wantStatus: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ErrorHandler(), AuthRequired(test.manager))
			router.GET("/protected", func(c *gin.Context) {
				if got, exists := c.Get(UserIDContextKey); !exists || got != userID {
					t.Fatalf("user id context = %v, %t", got, exists)
				}
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", test.header)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestAuthRequiredRejectsMissingManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandler(), AuthRequired(nil))
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

type middlewareTokenManager struct {
	userID uuid.UUID
	err    error
}

func (*middlewareTokenManager) GenerateToken(uuid.UUID) (string, error) { return "", nil }
func (m *middlewareTokenManager) VerifyToken(string) (uuid.UUID, error) { return m.userID, m.err }
