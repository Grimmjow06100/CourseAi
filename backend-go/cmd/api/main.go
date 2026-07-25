package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/config"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/database"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/auth"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/http"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/service"
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
	tokenTtl,err:=config.GetEnv[time.Duration]("JWT_TOKEN_TTL")
	if err!=nil{
		logger.Error("impossible de charger le token time to live")
	}

	tokenManager, err := auth.NewTokenManager(jwtSecret, tokenTtl)
	if err != nil {
		logger.Error("impossible d'initialiser le token manager", "erreur", err)
		return
	}

	repositories := postgres.NewRepositories(db)
	unitOfWork := postgres.NewUnitOfWork(db)
	passwordManager := new(auth.PasswordManager)
	authService := service.NewAuthService(tokenManager, repositories.Users(), passwordManager)
	courseCatalogService := service.NewCourseCatalogService(unitOfWork)

	router := http.NewRouter(http.RouterConfig{
		AuthService:          authService,
		CourseCatalogService: courseCatalogService,
		TokenManager:         tokenManager,
	})

	if err := router.Run(httpAddr); err != nil {
		logger.Error("erreur au démarrage du serveur http", "erreur", err)
	}
}

