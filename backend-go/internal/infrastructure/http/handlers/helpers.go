package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			middlewares.AbortWithError(c, middlewares.PayloadTooLarge("request body is too large", err))
			return false
		}
		middlewares.AbortWithError(c, middlewares.BadRequest("invalid request body", err))
		return false
	}
	return true
}

func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	value := strings.TrimSpace(c.Param(name))
	id, err := uuid.Parse(value)
	if err != nil {
		middlewares.AbortWithError(c, middlewares.BadRequest("invalid "+name, err))
		return uuid.Nil, false
	}
	return id, true
}

func writeUUIDCollection[Entity, Response any](
	c *gin.Context,
	parameter string,
	load func(context.Context, uuid.UUID) ([]Entity, error),
	mapResponse func(Entity) Response,
) {
	id, ok := parseUUIDParam(c, parameter)
	if !ok {
		return
	}
	entities, err := load(c.Request.Context(), id)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}
	responses := make([]Response, 0, len(entities))
	for _, entity := range entities {
		responses = append(responses, mapResponse(entity))
	}
	c.JSON(http.StatusOK, responses)
}

func parseCourseFilters(c *gin.Context) (contract.CourseFilters, bool) {
	filters := contract.CourseFilters{
		Search: strings.TrimSpace(c.Query("search")),
		Pagination: contract.Pagination{
			Page:     1,
			PageSize: 20,
		},
	}

	if value := strings.TrimSpace(c.Query("status")); value != "" {
		status, err := domain.ParseCourseGenerationStatus(value)
		if err != nil {
			middlewares.AbortWithError(c, middlewares.BadRequest("invalid status query parameter", err))
			return contract.CourseFilters{}, false
		}
		filters.Status = &status
	}

	if value := strings.TrimSpace(c.Query("language")); value != "" {
		language, err := domain.ParseCourseLanguage(value)
		if err != nil {
			middlewares.AbortWithError(c, middlewares.BadRequest("invalid language query parameter", err))
			return contract.CourseFilters{}, false
		}
		filters.Language = &language
	}

	if value := strings.TrimSpace(c.Query("orderBy")); value != "" {
		orderBy, ok := parseCourseOrderField(value)
		if !ok {
			middlewares.AbortWithError(c, middlewares.BadRequest("invalid orderBy query parameter", nil))
			return contract.CourseFilters{}, false
		}
		filters.OrderBy = orderBy
	}

	if value := strings.TrimSpace(c.Query("orderDirection")); value != "" {
		direction := contract.SortDirection(strings.ToLower(value))
		if err := direction.Validate(); err != nil {
			middlewares.AbortWithError(c, middlewares.BadRequest("invalid orderDirection query parameter", err))
			return contract.CourseFilters{}, false
		}
		filters.OrderDirection = direction
	}

	pagination, ok := parsePagination(c)
	if !ok {
		return contract.CourseFilters{}, false
	}
	filters.Pagination = pagination

	return filters, true
}

func parseGenerationHistoryFilters(c *gin.Context) (contract.GenerationHistoryFilters, bool) {
	filters := contract.GenerationHistoryFilters{}
	if value := strings.TrimSpace(c.Query("status")); value != "" {
		status, err := domain.ParseGenerationPipelineStatus(value)
		if err != nil {
			middlewares.AbortWithError(c, middlewares.BadRequest("invalid status query parameter", err))
			return contract.GenerationHistoryFilters{}, false
		}
		filters.PipelineStatus = &status
	}
	pagination, ok := parsePagination(c)
	if !ok {
		return contract.GenerationHistoryFilters{}, false
	}
	filters.Pagination = pagination
	return filters, true
}

func parsePagination(c *gin.Context) (contract.Pagination, bool) {
	page, ok := parseOptionalPositiveInt(c, "page", 1)
	if !ok {
		return contract.Pagination{}, false
	}
	pageSize, ok := parseOptionalPositiveInt(c, "pageSize", 20)
	if !ok {
		return contract.Pagination{}, false
	}
	return contract.Pagination{Page: page, PageSize: pageSize}.Normalize(), true
}

func parseOptionalPositiveInt(c *gin.Context, name string, fallback int) (int, bool) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return fallback, true
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		middlewares.AbortWithError(c, middlewares.BadRequest("invalid "+name+" query parameter", err))
		return 0, false
	}
	return parsed, true
}

func parseCourseOrderField(value string) (contract.CourseOrderField, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "created_at", "createdat":
		return contract.CourseOrderByCreatedAt, true
	case "updated_at", "updatedat":
		return contract.CourseOrderByUpdatedAt, true
	case "title":
		return contract.CourseOrderByTitle, true
	case "status":
		return contract.CourseOrderByStatus, true
	default:
		return "", false
	}
}
