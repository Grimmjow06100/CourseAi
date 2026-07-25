package http

import (
	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/handlers"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
)

type RouterConfig struct {
	AuthService             contract.AuthService
	CourseCatalogService    contract.CourseCatalogService
	CourseGenerationService contract.CourseGenerationService
	TokenManager            contract.TokenManager

}

func NewRouter(cfg RouterConfig) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middlewares.ErrorHandler())

	
	healthHandler := handlers.NewHealthHandler()
	authHandler := handlers.NewAuthHandler(cfg.AuthService)
	generationHandler := handlers.NewGenerationHandler(cfg.CourseGenerationService)
	courseHandler := handlers.NewCourseHandler(cfg.CourseCatalogService)
	
	router.GET("/health", healthHandler.Health)

	api := router.Group("/api")
	registerAuthRoutes(api, authHandler)
	registerGenerationRoutes(api, generationHandler)
	registerCourseRoutes(api, courseHandler)

	return router
}

func registerAuthRoutes(router gin.IRouter, handler *handlers.AuthHandler) {
	auth := router.Group("/auth")
	auth.POST("/signup", handler.SignUp)
	auth.POST("/login", handler.Login)
}

func registerGenerationRoutes(router gin.IRouter, handler *handlers.GenerationHandler) {
	generations := router.Group("/generations")
	generations.POST("", handler.Start)
	generations.GET("/:requestID/status", handler.Status)
	generations.GET("/:requestID/result", handler.Result)
	generations.POST("/:requestID/retry", handler.Retry)
}

func registerCourseRoutes(router gin.IRouter, handler *handlers.CourseHandler) {
	courses := router.Group("/courses")
	courses.GET("", handler.ListCourses)
	courses.GET("/:courseID", handler.GetCourse)
	courses.DELETE("/:courseID", handler.DeleteCourse)
	courses.GET("/:courseID/modules", handler.ListCourseModules)

	modules := router.Group("/modules")
	modules.GET("/:moduleID", handler.GetModule)
	modules.GET("/:moduleID/lessons", handler.ListModuleLessons)

	lessons := router.Group("/lessons")
	lessons.GET("/:lessonID", handler.GetLesson)
}

