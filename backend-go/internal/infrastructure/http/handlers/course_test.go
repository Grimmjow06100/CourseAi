package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestCourseHandlerRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	courseID := uuid.New()
	moduleID := uuid.New()
	lessonID := uuid.New()
	service := &courseCatalogStub{
		course:  domain.Course{ID: courseID, RequestID: uuid.New()},
		module:  domain.Module{ID: moduleID, CourseID: courseID},
		lesson:  domain.Lesson{ID: lessonID, ModuleID: moduleID},
		modules: []domain.Module{{ID: moduleID, CourseID: courseID}},
		lessons: []domain.Lesson{{ID: lessonID, ModuleID: moduleID}},
		page:    contract.Page[domain.Course]{Items: []domain.Course{{ID: courseID}}, Page: 2, PageSize: 5},
	}
	router := courseTestRouter(service)

	tests := []struct {
		method     string
		path       string
		wantStatus int
	}{
		{method: http.MethodGet, path: "/api/courses?status=completed&language=fr&orderBy=updatedAt&orderDirection=DESC&page=2&pageSize=5&search=linux", wantStatus: http.StatusOK},
		{method: http.MethodGet, path: "/api/courses/" + courseID.String(), wantStatus: http.StatusOK},
		{method: http.MethodDelete, path: "/api/courses/" + courseID.String(), wantStatus: http.StatusNoContent},
		{method: http.MethodGet, path: "/api/courses/" + courseID.String() + "/modules", wantStatus: http.StatusOK},
		{method: http.MethodGet, path: "/api/modules/" + moduleID.String(), wantStatus: http.StatusOK},
		{method: http.MethodGet, path: "/api/modules/" + moduleID.String() + "/lessons", wantStatus: http.StatusOK},
		{method: http.MethodGet, path: "/api/lessons/" + lessonID.String(), wantStatus: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
	if service.filters.Status == nil || *service.filters.Status != domain.CourseStatusCompleted ||
		service.filters.Language == nil || *service.filters.Language != domain.CourseLanguageFR ||
		service.filters.OrderBy != contract.CourseOrderByUpdatedAt || service.filters.OrderDirection != contract.SortDescending ||
		service.filters.Pagination.Page != 2 || service.filters.Pagination.PageSize != 5 || service.filters.Search != "linux" {
		t.Fatalf("unexpected parsed filters: %+v", service.filters)
	}
	if service.deletedID != courseID {
		t.Fatalf("deleted id = %s, want %s", service.deletedID, courseID)
	}
}

func TestCourseHandlerRejectsInvalidParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := courseTestRouter(&courseCatalogStub{})
	paths := []string{
		"/api/courses/not-a-uuid",
		"/api/courses?status=published",
		"/api/courses?language=de",
		"/api/courses?orderBy=duration",
		"/api/courses?orderDirection=sideways",
		"/api/courses?page=zero",
		"/api/courses?pageSize=0",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400, body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestCourseHandlerRejectsUnavailableService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	courseTestRouter(nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/courses", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.Code)
	}
}

func courseTestRouter(service contract.CourseCatalogService) *gin.Engine {
	router := gin.New()
	router.Use(middlewares.ErrorHandler())
	handler := NewCourseHandler(service)
	router.GET("/api/courses", handler.ListCourses)
	router.GET("/api/courses/:courseID", handler.GetCourse)
	router.DELETE("/api/courses/:courseID", handler.DeleteCourse)
	router.GET("/api/courses/:courseID/modules", handler.ListCourseModules)
	router.GET("/api/modules/:moduleID", handler.GetModule)
	router.GET("/api/modules/:moduleID/lessons", handler.ListModuleLessons)
	router.GET("/api/lessons/:lessonID", handler.GetLesson)
	return router
}

type courseCatalogStub struct {
	course    domain.Course
	module    domain.Module
	lesson    domain.Lesson
	modules   []domain.Module
	lessons   []domain.Lesson
	page      contract.Page[domain.Course]
	filters   contract.CourseFilters
	deletedID uuid.UUID
	err       error
}

func (s *courseCatalogStub) GetCourse(context.Context, uuid.UUID) (domain.Course, error) {
	return s.course, s.err
}
func (s *courseCatalogStub) ListCourses(_ context.Context, filters contract.CourseFilters) (contract.Page[domain.Course], error) {
	s.filters = filters
	return s.page, s.err
}
func (s *courseCatalogStub) DeleteCourse(_ context.Context, id uuid.UUID) error {
	s.deletedID = id
	return s.err
}
func (s *courseCatalogStub) GetModule(context.Context, uuid.UUID) (domain.Module, error) {
	return s.module, s.err
}
func (s *courseCatalogStub) ListModulesByCourseID(context.Context, uuid.UUID) ([]domain.Module, error) {
	return s.modules, s.err
}
func (s *courseCatalogStub) GetLesson(context.Context, uuid.UUID) (domain.Lesson, error) {
	return s.lesson, s.err
}
func (s *courseCatalogStub) ListLessonsByModuleID(context.Context, uuid.UUID) ([]domain.Lesson, error) {
	return s.lessons, s.err
}
