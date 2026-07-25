package handlers

import (
	"net/http"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/dto"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
)

type CourseHandler struct {
	service contract.CourseCatalogService
}

func NewCourseHandler(service contract.CourseCatalogService) *CourseHandler {
	return &CourseHandler{service: service}
}

func (h *CourseHandler) ListCourses(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("course catalog service is unavailable", nil))
		return
	}

	filters, ok := parseCourseFilters(c)
	if !ok {
		return
	}

	page, err := h.service.ListCourses(c.Request.Context(), filters)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CoursePageFromDomain(page))
}

func (h *CourseHandler) GetCourse(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("course catalog service is unavailable", nil))
		return
	}

	courseID, ok := parseUUIDParam(c, "courseID")
	if !ok {
		return
	}

	course, err := h.service.GetCourse(c.Request.Context(), courseID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CourseFromDomain(course))
}

func (h *CourseHandler) DeleteCourse(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("course catalog service is unavailable", nil))
		return
	}

	courseID, ok := parseUUIDParam(c, "courseID")
	if !ok {
		return
	}

	if err := h.service.DeleteCourse(c.Request.Context(), courseID); err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *CourseHandler) ListCourseModules(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("course catalog service is unavailable", nil))
		return
	}

	courseID, ok := parseUUIDParam(c, "courseID")
	if !ok {
		return
	}

	modules, err := h.service.ListModulesByCourseID(c.Request.Context(), courseID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	responses := make([]dto.ModuleResponse, 0, len(modules))
	for _, module := range modules {
		responses = append(responses, dto.ModuleFromDomain(module))
	}
	c.JSON(http.StatusOK, responses)
}

func (h *CourseHandler) GetModule(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("course catalog service is unavailable", nil))
		return
	}

	moduleID, ok := parseUUIDParam(c, "moduleID")
	if !ok {
		return
	}

	module, err := h.service.GetModule(c.Request.Context(), moduleID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ModuleFromDomain(module))
}

func (h *CourseHandler) ListModuleLessons(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("course catalog service is unavailable", nil))
		return
	}

	moduleID, ok := parseUUIDParam(c, "moduleID")
	if !ok {
		return
	}

	lessons, err := h.service.ListLessonsByModuleID(c.Request.Context(), moduleID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	responses := make([]dto.LessonResponse, 0, len(lessons))
	for _, lesson := range lessons {
		responses = append(responses, dto.LessonFromDomain(lesson))
	}
	c.JSON(http.StatusOK, responses)
}

func (h *CourseHandler) GetLesson(c *gin.Context) {
	if h.service == nil {
		middlewares.AbortWithError(c, middlewares.ServiceUnavailable("course catalog service is unavailable", nil))
		return
	}

	lessonID, ok := parseUUIDParam(c, "lessonID")
	if !ok {
		return
	}

	lesson, err := h.service.GetLesson(c.Request.Context(), lessonID)
	if err != nil {
		middlewares.AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.LessonFromDomain(lesson))
}
