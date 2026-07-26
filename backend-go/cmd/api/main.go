package main

import (
	"context"
	"log/slog"
	"os"
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
	"github.com/openai/openai-go/v3"
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
		logger.Error(err.Error())
		return
	}
	defer db.Close()

	httpAddr, err := config.GetEnv[string]("HTTP_ADDR")
	if err != nil {
		logger.Error(err.Error())
		return
	}

	jwtSecret, err := config.GetEnv[string]("JWT_SECRET")
	if err != nil {
		logger.Error(err.Error())
		return
	}
	tokenTTL,err := config.GetEnv[time.Duration]("JWT_TOKEN_TTL")
	if err != nil {
		logger.Error(err.Error())
		return
	}

	tokenManager, err := auth.NewTokenManager(jwtSecret, tokenTTL)
	
	if err != nil {
		logger.Error(err.Error())
		return
	}


	promptsDir,err := config.GetEnv[string]("PROMPTS_DIR")
	if err != nil {
		logger.Error(err.Error())
		return
	}
	promptStore, err := promptinfra.Load(promptsDir)
	if err != nil {
		logger.Error("impossible de charger les prompts", "directory", promptsDir, "erreur", err)
		return
	}

	openAIKey, err := config.GetEnv[string]("OPENAI_API_KEY")
	if err != nil {
		logger.Error(err.Error())
		return
	}
	openAIClient := openaisdk.NewClient(option.WithAPIKey(openAIKey))
	courseAIGenerator := openaiinfra.NewCourseAIGenerator(&openAIClient, promptStore, openaiinfra.Config{
		Model:           openai.ChatModelGPT5_6Luna,
		MaxOutputTokens: 12000,
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

