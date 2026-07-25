package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/config"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/database"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/auth"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/clock"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http"
	openaiinfra "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/openai"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	promptinfra "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/prompts"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/service"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := config.Load(); err != nil {
		logger.Error(err.Error())
		return
	}

	ctx := context.Background()
	db, err := database.Open(ctx)
	if err != nil {
		logger.Error("erreur impossible d'ouvrir la connexion à la bd", "erreur", err)
		return
	}
	defer db.Close()

	httpAddr, err := config.GetEnv[string]("HTTP_ADDR")
	if err != nil {
		logger.Error("impossible de charger le port http", "erreur", err)
		return
	}

	jwtSecret, err := config.GetEnv[string]("JWT_SECRET")
	if err != nil {
		logger.Error("impossible de charger le secret jwt", "erreur", err)
		return
	}
	tokenTTL := durationEnvOrDefault("JWT_TOKEN_TTL", 24*time.Hour, logger)

	tokenManager, err := auth.NewTokenManager(jwtSecret, tokenTTL)
	if err != nil {
		logger.Error("impossible d'initialiser le token manager", "erreur", err)
		return
	}

	promptsDir := stringEnvOrDefault("PROMPTS_DIR", "./prompts")
	promptStore, err := promptinfra.Load(promptsDir)
	if err != nil {
		logger.Error("impossible de charger les prompts", "directory", promptsDir, "erreur", err)
		return
	}

	openAIKey, err := config.GetEnv[string]("OPENAI_API_KEY")
	if err != nil {
		logger.Error("impossible de charger la cle OpenAI", "erreur", err)
		return
	}
	openAIClient := openaisdk.NewClient(option.WithAPIKey(openAIKey))
	courseAIGenerator := openaiinfra.NewCourseAIGenerator(&openAIClient, promptStore, openaiinfra.Config{
		Model:           stringEnvOrDefault("OPENAI_MODEL", "gpt-5.6"),
		MaxOutputTokens: int64EnvOrDefault("OPENAI_MAX_OUTPUT_TOKENS", 12000, logger),
	})

	repositories := postgres.NewRepositories(db)
	unitOfWork := postgres.NewUnitOfWork(db)
	passwordManager := new(auth.PasswordManager)
	authService := service.NewAuthService(tokenManager, repositories.Users(), passwordManager)
	courseCatalogService := service.NewCourseCatalogService(unitOfWork)
	courseGenerationService := service.NewCourseGeneratorService(
		courseAIGenerator,
		unitOfWork,
		clock.NewSystemClock(),
		service.CourseGeneratorConfig{},
	)

	router := http.NewRouter(http.RouterConfig{
		AuthService:             authService,
		CourseCatalogService:    courseCatalogService,
		CourseGenerationService: courseGenerationService,
		TokenManager:            tokenManager,
	})

	if err := router.Run(httpAddr); err != nil {
		logger.Error("erreur au démarrage du serveur http", "erreur", err)
	}
}

func stringEnvOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func durationEnvOrDefault(key string, fallback time.Duration, logger *slog.Logger) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		logger.Warn("duration env invalide, utilisation de la valeur par defaut", "key", key, "value", value, "fallback", fallback.String())
		return fallback
	}
	return duration
}

func int64EnvOrDefault(key string, fallback int64, logger *slog.Logger) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		logger.Warn("int env invalide, utilisation de la valeur par defaut", "key", key, "value", value, "fallback", fallback)
		return fallback
	}
	return parsed
}
