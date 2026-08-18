package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
)

func TestAuthHandlerSignUpAndLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &authServiceStub{}
	service.signup = func(_ context.Context, params contract.SignupParams) (string, error) {
		if params.Username != "alice" || params.Password != "Strong!1" {
			t.Fatalf("unexpected signup params: %+v", params)
		}
		return "signup-token", nil
	}
	service.login = func(_ context.Context, username, password string) (string, error) {
		if username != "alice" || password != "Strong!1" {
			t.Fatalf("unexpected login params: %q %q", username, password)
		}
		return "login-token", nil
	}
	router := authTestRouter(service)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "signup", path: "/api/auth/signup", wantStatus: http.StatusCreated, wantBody: `{"token":"signup-token"}`},
		{name: "login", path: "/api/auth/login", wantStatus: http.StatusOK, wantBody: `{"token":"login-token"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, bytes.NewBufferString(`{"username":"alice","password":"Strong!1"}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus || response.Body.String() != test.wantBody {
				t.Fatalf("response = %d %s, want %d %s", response.Code, response.Body.String(), test.wantStatus, test.wantBody)
			}
		})
	}
}

func TestAuthHandlerRejectsInvalidBodyAndUnavailableService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	invalidResponse := httptest.NewRecorder()
	invalidRequest := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(`{"username":"a"}`))
	invalidRequest.Header.Set("Content-Type", "application/json")
	authTestRouter(&authServiceStub{}).ServeHTTP(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid body status = %d, body=%s", invalidResponse.Code, invalidResponse.Body.String())
	}

	unavailableResponse := httptest.NewRecorder()
	authTestRouter(nil).ServeHTTP(unavailableResponse, httptest.NewRequest(http.MethodPost, "/api/auth/login", nil))
	if unavailableResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable service status = %d", unavailableResponse.Code)
	}
}

func TestAuthHandlerPropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serviceErr := errors.New("auth backend unavailable")
	service := &authServiceStub{login: func(context.Context, string, string) (string, error) { return "", serviceErr }}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"alice","password":"Strong!1"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	authTestRouter(service).ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
}

func authTestRouter(service contract.AuthService) *gin.Engine {
	router := gin.New()
	router.Use(middlewares.ErrorHandler())
	handler := NewAuthHandler(service)
	router.POST("/api/auth/signup", handler.SignUp)
	router.POST("/api/auth/login", handler.Login)
	return router
}

type authServiceStub struct {
	signup func(context.Context, contract.SignupParams) (string, error)
	login  func(context.Context, string, string) (string, error)
}

func (s *authServiceStub) SignUp(ctx context.Context, params contract.SignupParams) (string, error) {
	if s.signup == nil {
		return "", nil
	}
	return s.signup(ctx, params)
}
func (s *authServiceStub) Login(ctx context.Context, username, password string) (string, error) {
	if s.login == nil {
		return "", nil
	}
	return s.login(ctx, username, password)
}
