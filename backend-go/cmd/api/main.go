package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/config"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/db"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/auth"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/clock"
	httpinfra "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/jobs"
	openaiinfra "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/openai"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	promptinfra "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/prompts"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/service"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/textutil"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type httpServerConfig struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	if err := config.Load(); err != nil {
		return err
	}

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithCancel(signalCtx)
	defer cancel()

	httpConfig, err := loadHTTPServerConfig()
	if err != nil {
		return err
	}
	workerConfig, err := jobs.LoadWorkerConfig()
	if err != nil {
		return err
	}
	allowedOriginsValue, err := config.GetEnvWithDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	if err != nil {
		return err
	}
	allowedOrigins := textutil.SplitNonBlank(allowedOriginsValue, ",")

	pool, err := db.Open(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	jwtSecret, err := config.GetEnv[string]("JWT_SECRET")
	if err != nil {
		return err
	}
	tokenTTL, err := config.GetEnv[time.Duration]("JWT_TOKEN_TTL")
	if err != nil {
		return err
	}
	tokenManager, err := auth.NewTokenManager(jwtSecret, tokenTTL)
	if err != nil {
		return err
	}

	promptsDir, err := config.GetEnv[string]("PROMPTS_DIR")
	if err != nil {
		return err
	}
	promptStore, err := promptinfra.Load(promptsDir)
	if err != nil {
		return fmt.Errorf("load prompts from %s: %w", promptsDir, err)
	}

	openAIKey, err := config.GetEnv[string]("OPENAI_API_KEY")
	if err != nil {
		return err
	}
	openAIMaxOutputTokens, err := config.GetEnv[int]("OPENAI_MAX_OUTPUT_TOKENS")
	if err != nil {
		return err
	}
	openAIModel, err := config.GetEnv[string]("OPENAI_MODEL")
	if err != nil {
		return err
	}

	openAIClient := openaisdk.NewClient(option.WithAPIKey(openAIKey))
	courseAIGenerator := openaiinfra.NewCourseAIGenerator(&openAIClient, promptStore, openaiinfra.Config{
		Model:           openAIModel,
		MaxOutputTokens: int64(openAIMaxOutputTokens),
	})

	repositories := postgres.NewRepositories(pool)
	unitOfWork := postgres.NewUnitOfWork(pool)
	systemClock := clock.NewSystemClock()
	authService := service.NewAuthService(tokenManager, repositories.Users(), new(auth.PasswordManager))
	courseCatalogService := service.NewCourseCatalogService(unitOfWork)
	courseGenerationService := service.NewCourseGeneratorService(
		courseAIGenerator,
		unitOfWork,
		systemClock,
		service.CourseGeneratorConfig{},
	)
	jobExecutor := service.NewGenerationJobExecutor(courseGenerationService)
	workerPool, err := jobs.NewWorkerPool(
		repositories.GenerationJobs(),
		jobExecutor,
		systemClock,
		workerConfig,
		logger,
	)
	if err != nil {
		return err
	}

	router := httpinfra.NewRouter(httpinfra.RouterConfig{
		AuthService:             authService,
		CourseCatalogService:    courseCatalogService,
		CourseGenerationService: courseGenerationService,
		TokenManager:            tokenManager,
		AllowedOrigins:          allowedOrigins,
	})
	server := &http.Server{
		Addr:              httpConfig.Address,
		Handler:           router,
		ReadHeaderTimeout: httpConfig.ReadHeaderTimeout,
		ReadTimeout:       httpConfig.ReadTimeout,
		WriteTimeout:      httpConfig.WriteTimeout,
		IdleTimeout:       httpConfig.IdleTimeout,
	}

	serverDone := make(chan error, 1)
	go func() {
		logger.Info("http server started", "address", httpConfig.Address)
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverDone <- err
	}()

	var workerDone chan error
	if workerConfig.Enabled {
		workerDone = make(chan error, 1)
		go func() {
			logger.Info("generation worker pool started", "concurrency", workerConfig.Concurrency)
			workerDone <- workerPool.Run(ctx)
		}()
	} else {
		logger.Warn("generation worker pool is disabled")
	}

	var runErr error
	serverConsumed := false
	workerConsumed := false
	select {
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverDone:
		serverConsumed = true
		if err != nil {
			runErr = fmt.Errorf("http server: %w", err)
		} else if signalCtx.Err() == nil {
			runErr = errors.New("http server stopped unexpectedly")
		}
	case err := <-workerDone:
		workerConsumed = true
		if err != nil {
			runErr = fmt.Errorf("generation worker pool: %w", err)
		} else if signalCtx.Err() == nil {
			runErr = errors.New("generation worker pool stopped unexpectedly")
		}
	}
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), httpConfig.ShutdownTimeout)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("shutdown http server: %w", err))
	}
	if !serverConsumed {
		select {
		case err := <-serverDone:
			if err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("http server: %w", err))
			}
		case <-shutdownCtx.Done():
			runErr = errors.Join(runErr, shutdownCtx.Err())
		}
	}
	if workerDone != nil && !workerConsumed {
		select {
		case err := <-workerDone:
			if err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("generation worker pool: %w", err))
			}
		case <-shutdownCtx.Done():
			runErr = errors.Join(runErr, shutdownCtx.Err())
		}
	}

	logger.Info("application shutdown completed")
	return runErr
}

func loadHTTPServerConfig() (httpServerConfig, error) {
	port, err := config.GetEnvWithDefault("PORT", "")
	if err != nil {
		return httpServerConfig{}, err
	}
	defaultAddress := ":8080"
	if strings.TrimSpace(port) != "" {
		defaultAddress = ":" + strings.TrimSpace(port)
	}

	address, err := config.GetEnvWithDefault("HTTP_ADDR", defaultAddress)
	if err != nil {
		return httpServerConfig{}, err
	}
	readHeaderTimeout, err := config.GetEnvWithDefault("HTTP_READ_HEADER_TIMEOUT", 5*time.Second)
	if err != nil {
		return httpServerConfig{}, err
	}
	readTimeout, err := config.GetEnvWithDefault("HTTP_READ_TIMEOUT", 30*time.Second)
	if err != nil {
		return httpServerConfig{}, err
	}
	writeTimeout, err := config.GetEnvWithDefault("HTTP_WRITE_TIMEOUT", 2*time.Minute)
	if err != nil {
		return httpServerConfig{}, err
	}
	idleTimeout, err := config.GetEnvWithDefault("HTTP_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return httpServerConfig{}, err
	}
	shutdownTimeout, err := config.GetEnvWithDefault("HTTP_SHUTDOWN_TIMEOUT", 20*time.Second)
	if err != nil {
		return httpServerConfig{}, err
	}

	values := []struct {
		name  string
		value time.Duration
	}{
		{name: "HTTP_READ_HEADER_TIMEOUT", value: readHeaderTimeout},
		{name: "HTTP_READ_TIMEOUT", value: readTimeout},
		{name: "HTTP_WRITE_TIMEOUT", value: writeTimeout},
		{name: "HTTP_IDLE_TIMEOUT", value: idleTimeout},
		{name: "HTTP_SHUTDOWN_TIMEOUT", value: shutdownTimeout},
	}
	for _, value := range values {
		if value.value <= 0 {
			return httpServerConfig{}, fmt.Errorf("%s must be positive", value.name)
		}
	}
	if strings.TrimSpace(address) == "" {
		return httpServerConfig{}, errors.New("HTTP_ADDR must not be blank")
	}
	return httpServerConfig{
		Address:           strings.TrimSpace(address),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ShutdownTimeout:   shutdownTimeout,
	}, nil
}
