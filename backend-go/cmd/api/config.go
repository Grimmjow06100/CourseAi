package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/config"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/auth"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/jobs"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/textutil"
)

type applicationConfig struct {
	Environment           string
	AllowedOrigins        []string
	Clerk                 auth.ClerkConfig
	PromptsDirectory      string
	OpenAIAPIKey          string
	OpenAIModel           string
	OpenAIMaxOutputTokens int
	HTTP                  httpServerConfig
	Worker                jobs.WorkerConfig
	Guardrails            apiGuardrailConfig
}

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

func loadApplicationConfig() (applicationConfig, error) {
	httpConfig, err := loadHTTPServerConfig()
	if err != nil {
		return applicationConfig{}, err
	}
	workerConfig, err := jobs.LoadWorkerConfig()
	if err != nil {
		return applicationConfig{}, err
	}
	guardrails, err := loadAPIGuardrailConfig()
	if err != nil {
		return applicationConfig{}, err
	}
	environment, err := config.GetEnvWithDefault("APP_ENV", "development")
	if err != nil {
		return applicationConfig{}, err
	}
	origins, err := config.GetEnvWithDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	if err != nil {
		return applicationConfig{}, err
	}
	clerkSecretKey, err := config.GetEnv[string]("CLERK_SECRET_KEY")
	if err != nil {
		return applicationConfig{}, err
	}
	clerkParties, err := config.GetEnv[string]("CLERK_AUTHORIZED_PARTIES")
	if err != nil {
		return applicationConfig{}, err
	}
	promptsDirectory, err := config.GetEnv[string]("PROMPTS_DIR")
	if err != nil {
		return applicationConfig{}, err
	}
	openAIAPIKey, err := config.GetEnv[string]("OPENAI_API_KEY")
	if err != nil {
		return applicationConfig{}, err
	}
	openAIMaxOutputTokens, err := config.GetEnv[int]("OPENAI_MAX_OUTPUT_TOKENS")
	if err != nil {
		return applicationConfig{}, err
	}
	openAIModel, err := config.GetEnv[string]("OPENAI_MODEL")
	if err != nil {
		return applicationConfig{}, err
	}

	loaded := applicationConfig{
		Environment:           environment,
		AllowedOrigins:        textutil.SplitNonBlank(origins, ","),
		Clerk:                 auth.ClerkConfig{SecretKey: clerkSecretKey, AuthorizedParties: textutil.SplitNonBlank(clerkParties, ",")},
		PromptsDirectory:      promptsDirectory,
		OpenAIAPIKey:          openAIAPIKey,
		OpenAIModel:           openAIModel,
		OpenAIMaxOutputTokens: openAIMaxOutputTokens,
		HTTP:                  httpConfig,
		Worker:                workerConfig,
		Guardrails:            guardrails,
	}
	if err := validateProductionConfig(
		loaded.Environment,
		loaded.AllowedOrigins,
		loaded.Clerk,
		loaded.Worker,
		loaded.OpenAIModel,
		loaded.OpenAIMaxOutputTokens,
		loaded.Guardrails,
	); err != nil {
		return applicationConfig{}, err
	}
	return loaded, nil
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
