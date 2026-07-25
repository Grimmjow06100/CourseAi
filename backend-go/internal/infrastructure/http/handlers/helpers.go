package handlers

import (
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

	page, ok := parseOptionalPositiveInt(c, "page", 1)
	if !ok {
		return contract.CourseFilters{}, false
	}
	pageSize, ok := parseOptionalPositiveInt(c, "pageSize", 20)
	if !ok {
		return contract.CourseFilters{}, false
	}
	filters.Pagination = contract.Pagination{Page: page, PageSize: pageSize}.Normalize()

	return filters, true
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
