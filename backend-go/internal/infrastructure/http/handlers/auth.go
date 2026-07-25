package handlers

import (
	"net/http"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/dto"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service contract.AuthService
}

func NewAuthHandler(service contract.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) SignUp(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("auth service is unavailable", nil))
		return
	}

	var request dto.SignupRequest
	if !bindJSON(c, &request) {
		return
	}

	token, err := h.service.SignUp(c.Request.Context(), contract.SignupParams{
		Username: request.Username,
		Password: request.Password,
	})
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.AuthResponse{Token: token})
}

func (h *AuthHandler) Login(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("auth service is unavailable", nil))
		return
	}

	var request dto.LoginRequest
	if !bindJSON(c, &request) {
		return
	}

	token, err := h.service.Login(c.Request.Context(), request.Username, request.Password)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.AuthResponse{Token: token})
}
