package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/config"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/db"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/auth"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/clock"
	httpinfra "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http"
	httpmiddlewares "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/jobs"
	openaiinfra "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/openai"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	promptinfra "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/prompts"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/service"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func run(logger *slog.Logger) error {
	if err := config.Load(); err != nil {
		return err
	}
	appConfig, err := loadApplicationConfig()
	if err != nil {
		return err
	}
	if err := auth.ConfigureClerk(appConfig.Clerk); err != nil {
		return fmt.Errorf("configure Clerk: %w", err)
	}

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	processCtx, cancelProcesses := context.WithCancel(signalCtx)
	defer cancelProcesses()

	pool, err := db.Open(processCtx)
	if err != nil {
		return err
	}
	defer pool.Close()

	promptStore, err := promptinfra.Load(appConfig.PromptsDirectory)
	if err != nil {
		return fmt.Errorf("load prompts from %s: %w", appConfig.PromptsDirectory, err)
	}
	openAIClient := openaisdk.NewClient(
		option.WithAPIKey(appConfig.OpenAIAPIKey),
		option.WithMaxRetries(appConfig.Guardrails.OpenAIMaxRetries),
	)
	courseAIGenerator := openaiinfra.NewCourseAIGenerator(&openAIClient, promptStore, openaiinfra.Config{
		Model:           appConfig.OpenAIModel,
		MaxOutputTokens: int64(appConfig.OpenAIMaxOutputTokens),
	})

	repositories := postgres.NewRepositories(pool)
	unitOfWork := postgres.NewUnitOfWork(pool)
	systemClock := clock.NewSystemClock()
	courseCatalogService := service.NewCourseCatalogService(unitOfWork)
	courseGenerationService := service.NewCourseGeneratorService(
		courseAIGenerator,
		unitOfWork,
		systemClock,
		service.CourseGeneratorConfig{
			MaxActivePerUser: appConfig.Guardrails.MaxActivePerUser,
			MaxDailyPerUser:  appConfig.Guardrails.MaxDailyPerUser,
			MaxPendingJobs:   appConfig.Guardrails.MaxPendingJobs,
		},
	)
	jobExecutor := service.NewGenerationJobExecutor(courseGenerationService)
	workerPool, err := jobs.NewWorkerPool(
		repositories.GenerationJobs(),
		jobExecutor,
		systemClock,
		appConfig.Worker,
		logger,
	)
	if err != nil {
		return err
	}

	router := httpinfra.NewRouter(httpinfra.RouterConfig{
		CourseCatalogService:      courseCatalogService,
		GenerationCommandService:  courseGenerationService,
		GenerationQueryService:    courseGenerationService,
		GenerationTrackingService: courseGenerationService,
		Authentication:            httpmiddlewares.ClerkAuthentication(appConfig.Clerk.AuthorizedParties),
		AllowedOrigins:            appConfig.AllowedOrigins,
		AppEnv:                    appConfig.Environment,
		MaxBodyBytes:              appConfig.Guardrails.MaxBodyBytes,
		GenerationRateRequests:    appConfig.Guardrails.RateLimitRequests,
		GenerationRateWindow:      appConfig.Guardrails.RateLimitWindow,
		ReadyCheck:                pool.Ping,
		WorkerEnabled:             appConfig.Worker.Enabled,
	})
	server := &http.Server{
		Addr:              appConfig.HTTP.Address,
		Handler:           router,
		ReadHeaderTimeout: appConfig.HTTP.ReadHeaderTimeout,
		ReadTimeout:       appConfig.HTTP.ReadTimeout,
		WriteTimeout:      appConfig.HTTP.WriteTimeout,
		IdleTimeout:       appConfig.HTTP.IdleTimeout,
	}

	return runProcesses(processCtx, signalCtx, cancelProcesses, server, workerPool, appConfig, logger)
}
