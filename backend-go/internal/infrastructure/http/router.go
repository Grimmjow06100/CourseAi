package http

import (
	"context"
	"strings"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/apidocs"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/handlers"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type RouterConfig struct {
	CourseCatalogService     contract.CourseCatalogService
	GenerationCommandService contract.GenerationCommandService
	GenerationQueryService   contract.GenerationQueryService
	Authentication           gin.HandlerFunc
	AllowedOrigins           []string
	AppEnv                   string
	MaxBodyBytes             int64
	GenerationRateRequests   int
	GenerationRateWindow     time.Duration
	ReadyCheck               func(ctx context.Context) error
	WorkerEnabled            bool
}

func NewRouter(cfg RouterConfig) *gin.Engine {
	binding.EnableDecoderDisallowUnknownFields = true
	binding.EnableDecoderUseNumber = true
	if cfg.MaxBodyBytes == 0 {
		cfg.MaxBodyBytes = 64 * 1024
	}
	if cfg.GenerationRateRequests == 0 {
		cfg.GenerationRateRequests = 10
	}
	if cfg.GenerationRateWindow == 0 {
		cfg.GenerationRateWindow = time.Minute
	}
	rateLimiter, err := middlewares.NewUserRateLimiter(cfg.GenerationRateRequests, cfg.GenerationRateWindow)
	if err != nil {
		panic(err)
	}
	router := gin.New()
	router.Use(middlewares.RequestID())
	router.Use(middlewares.RequestLogger(nil))
	router.Use(middlewares.Recovery())
	router.Use(middlewares.CORS(cfg.AllowedOrigins))
	router.Use(middlewares.ErrorHandler())
	router.Use(middlewares.BodyLimit(cfg.MaxBodyBytes))
	if !strings.EqualFold(strings.TrimSpace(cfg.AppEnv), "production") {
		apidocs.Register(router)
	}

	healthHandler := handlers.NewHealthHandler(cfg.ReadyCheck, cfg.WorkerEnabled)
	generationHandler := handlers.NewGenerationHandler(cfg.GenerationCommandService, cfg.GenerationQueryService)
	courseHandler := handlers.NewCourseHandler(cfg.CourseCatalogService)

	router.GET("/health", healthHandler.Ready)
	router.GET("/health/live", healthHandler.Live)
	router.GET("/health/ready", healthHandler.Ready)

	api := router.Group("/api")
	if cfg.Authentication == nil {
		cfg.Authentication = middlewares.RejectUnauthenticated()
	}
	api.Use(cfg.Authentication)
	registerGenerationRoutes(api, generationHandler, rateLimiter.Middleware())
	registerCourseRoutes(api, courseHandler)

	return router
}

func registerGenerationRoutes(router gin.IRouter, handler *handlers.GenerationHandler, rateLimit gin.HandlerFunc) {
	generations := router.Group("/generations")
	generations.Use(rateLimit)
	generations.GET("", handler.List)
	generations.POST("", handler.Start)
	generations.POST("/:requestID/clarifications", handler.SubmitClarifications)
	generations.POST("/:requestID/structure", handler.Structure)
	generations.POST("/:requestID/structure/retry", handler.RetryStructure)
	generations.POST("/lessons/:lessonID/content", handler.LessonContent)
	generations.POST("/modules/:moduleID/contents", handler.ModuleLessonContents)
	generations.GET("/:requestID/status", handler.Status)
	generations.GET("/:requestID/jobs", handler.Jobs)
	generations.GET("/:requestID/result", handler.Result)
	generations.POST("/:requestID/retry", handler.Retry)
	generations.DELETE("/:requestID", handler.Delete)

	jobs := router.Group("/generation-jobs")
	jobs.GET("/:jobID", handler.JobStatus)
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
	lessons.GET("/:lessonID/solutions", handler.GetLessonSolutions)
}
