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
	httpmiddlewares "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http/middlewares"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/jobs"
	openaiinfra "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/openai"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	promptinfra "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/prompts"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/service"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/errtrace"
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

type apiGuardrailConfig struct {
	MaxBodyBytes      int64
	RateLimitRequests int
	RateLimitWindow   time.Duration
	MaxActivePerUser  int64
	MaxDailyPerUser   int64
	MaxPendingJobs    int64
	OpenAIMaxRetries  int
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		traced := errtrace.Capture(err)
		logger.Error("application stopped", "error", err, "stack", errtrace.Stack(traced))
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
	guardrails, err := loadAPIGuardrailConfig()
	if err != nil {
		return err
	}
	appEnv, err := config.GetEnvWithDefault("APP_ENV", "development")
	if err != nil {
		return err
	}
	allowedOriginsValue, err := config.GetEnvWithDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	if err != nil {
		return err
	}
	allowedOrigins := textutil.SplitNonBlank(allowedOriginsValue, ",")
	clerkSecretKey, err := config.GetEnv[string]("CLERK_SECRET_KEY")
	if err != nil {
		return err
	}
	clerkAuthorizedPartiesValue, err := config.GetEnv[string]("CLERK_AUTHORIZED_PARTIES")
	if err != nil {
		return err
	}
	clerkConfig := auth.ClerkConfig{
		SecretKey:         clerkSecretKey,
		AuthorizedParties: textutil.SplitNonBlank(clerkAuthorizedPartiesValue, ","),
	}
	if err := auth.ConfigureClerk(clerkConfig); err != nil {
		return fmt.Errorf("configure Clerk: %w", err)
	}
	pool, err := db.Open(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

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
	if err := validateProductionConfig(appEnv, allowedOrigins, clerkConfig, workerConfig, openAIModel, openAIMaxOutputTokens, guardrails); err != nil {
		return err
	}

	openAIClient := openaisdk.NewClient(
		option.WithAPIKey(openAIKey),
		option.WithMaxRetries(guardrails.OpenAIMaxRetries),
	)
	courseAIGenerator := openaiinfra.NewCourseAIGenerator(&openAIClient, promptStore, openaiinfra.Config{
		Model:           openAIModel,
		MaxOutputTokens: int64(openAIMaxOutputTokens),
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
			MaxActivePerUser: guardrails.MaxActivePerUser,
			MaxDailyPerUser:  guardrails.MaxDailyPerUser,
			MaxPendingJobs:   guardrails.MaxPendingJobs,
		},
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
		CourseCatalogService:    courseCatalogService,
		CourseGenerationService: courseGenerationService,
		Authentication:          httpmiddlewares.ClerkAuthentication(clerkConfig.AuthorizedParties),
		AllowedOrigins:          allowedOrigins,
		AppEnv:                  appEnv,
		MaxBodyBytes:            guardrails.MaxBodyBytes,
		GenerationRateRequests:  guardrails.RateLimitRequests,
		GenerationRateWindow:    guardrails.RateLimitWindow,
		ReadyCheck:              pool.Ping,
		WorkerEnabled:           workerConfig.Enabled,
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

func loadAPIGuardrailConfig() (apiGuardrailConfig, error) {
	maxBodyBytes, err := config.GetEnvWithDefault("HTTP_MAX_BODY_BYTES", 64*1024)
	if err != nil {
		return apiGuardrailConfig{}, err
	}
	rateLimitRequests, err := config.GetEnvWithDefault("GENERATION_RATE_LIMIT_REQUESTS", 10)
	if err != nil {
		return apiGuardrailConfig{}, err
	}
	rateLimitWindow, err := config.GetEnvWithDefault("GENERATION_RATE_LIMIT_WINDOW", time.Minute)
	if err != nil {
		return apiGuardrailConfig{}, err
	}
	maxActive, err := config.GetEnvWithDefault("GENERATION_MAX_ACTIVE_PER_USER", 2)
	if err != nil {
		return apiGuardrailConfig{}, err
	}
	maxDaily, err := config.GetEnvWithDefault("GENERATION_MAX_DAILY_PER_USER", 10)
	if err != nil {
		return apiGuardrailConfig{}, err
	}
	maxPendingJobs, err := config.GetEnvWithDefault("GENERATION_MAX_PENDING_JOBS", 500)
	if err != nil {
		return apiGuardrailConfig{}, err
	}
	openAIMaxRetries, err := config.GetEnvWithDefault("OPENAI_MAX_RETRIES", 0)
	if err != nil {
		return apiGuardrailConfig{}, err
	}
	values := []struct {
		name  string
		value int
	}{
		{name: "HTTP_MAX_BODY_BYTES", value: maxBodyBytes},
		{name: "GENERATION_RATE_LIMIT_REQUESTS", value: rateLimitRequests},
		{name: "GENERATION_MAX_ACTIVE_PER_USER", value: maxActive},
		{name: "GENERATION_MAX_DAILY_PER_USER", value: maxDaily},
		{name: "GENERATION_MAX_PENDING_JOBS", value: maxPendingJobs},
	}
	for _, value := range values {
		if value.value <= 0 {
			return apiGuardrailConfig{}, fmt.Errorf("%s must be positive", value.name)
		}
	}
	if rateLimitWindow <= 0 {
		return apiGuardrailConfig{}, errors.New("GENERATION_RATE_LIMIT_WINDOW must be positive")
	}
	if openAIMaxRetries < 0 || openAIMaxRetries > 1 {
		return apiGuardrailConfig{}, errors.New("OPENAI_MAX_RETRIES must be 0 or 1")
	}
	return apiGuardrailConfig{
		MaxBodyBytes:      int64(maxBodyBytes),
		RateLimitRequests: rateLimitRequests,
		RateLimitWindow:   rateLimitWindow,
		MaxActivePerUser:  int64(maxActive),
		MaxDailyPerUser:   int64(maxDaily),
		MaxPendingJobs:    int64(maxPendingJobs),
		OpenAIMaxRetries:  openAIMaxRetries,
	}, nil
}

func validateProductionConfig(
	appEnv string,
	allowedOrigins []string,
	clerkConfig auth.ClerkConfig,
	workerConfig jobs.WorkerConfig,
	openAIModel string,
	openAIMaxOutputTokens int,
	guardrails apiGuardrailConfig,
) error {
	if !strings.EqualFold(strings.TrimSpace(appEnv), "production") {
		return nil
	}
	if !workerConfig.Enabled {
		return errors.New("GENERATION_WORKER_ENABLED must be true in production")
	}
	if workerConfig.Concurrency != 1 {
		return errors.New("GENERATION_WORKER_CONCURRENCY must be 1 for the initial production deployment")
	}
	for name, origins := range map[string][]string{
		"CORS_ALLOWED_ORIGINS":     allowedOrigins,
		"CLERK_AUTHORIZED_PARTIES": clerkConfig.AuthorizedParties,
	} {
		if len(origins) == 0 {
			return fmt.Errorf("%s must contain at least one HTTPS origin in production", name)
		}
		for _, origin := range origins {
			if origin == "*" || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(origin)), "https://") {
				return fmt.Errorf("%s contains a non-HTTPS production origin: %s", name, origin)
			}
		}
	}
	if strings.TrimSpace(openAIModel) == "" {
		return errors.New("OPENAI_MODEL must not be blank in production")
	}
	if openAIMaxOutputTokens < 256 || openAIMaxOutputTokens > 100000 {
		return errors.New("OPENAI_MAX_OUTPUT_TOKENS must be between 256 and 100000 in production")
	}
	if guardrails.MaxBodyBytes > 1024*1024 {
		return errors.New("HTTP_MAX_BODY_BYTES must not exceed 1 MiB in production")
	}
	return nil
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
