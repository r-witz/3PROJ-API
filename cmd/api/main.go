package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"duskforge-api/internal/adapters/handlers"
	"duskforge-api/internal/adapters/http"
	"duskforge-api/internal/adapters/repositories"
	"duskforge-api/internal/config"
	"duskforge-api/internal/core/services"
	"duskforge-api/pkg/database"
	"duskforge-api/pkg/logger"
	"duskforge-api/pkg/tmdb"

	_ "duskforge-api/docs" // Import generated docs

	"go.uber.org/zap"
)

// @title           Duskforge API
// @version         1.0
// @description     A movie collection and review API powered by TMDB
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    https://github.com/duskforge/api
// @contact.email  support@duskforge.io

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	logger.InitLogger(cfg.LogLevel)

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		logger.Logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	tmdbClient, err := tmdb.New(cfg.TMDBAPIKey)
	if err != nil {
		logger.Logger.Fatal("Failed to create TMDB client", zap.Error(err))
	}
	defer tmdbClient.Close()

	if err := tmdbClient.InitializeConfiguration(context.Background()); err != nil {
		logger.Logger.Warn("Failed to initialize TMDB configuration", zap.Error(err))
	}

	userRepo := repositories.NewUserRepository(db)
	sessionRepo := repositories.NewSessionRepository(db)

	authService := services.NewAuthService(userRepo, sessionRepo, services.AuthServiceConfig{
		AccessTokenSecret:  cfg.AccessTokenSecret,
		AccessTokenExpiry:  cfg.AccessTokenExpiry,
		RefreshTokenSecret: cfg.RefreshTokenSecret,
		RefreshTokenExpiry: cfg.RefreshTokenExpiry,
	})
	userService := services.NewUserService(userRepo)
	movieService := services.NewMovieService(tmdbClient)

	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	movieHandler := handlers.NewMovieHandler(movieService)

	router := http.NewRouter(
		http.RouterConfig{
			AccessTokenSecret: cfg.AccessTokenSecret,
		},
		authHandler,
		userHandler,
		movieHandler,
	)

	router.Setup()
	srv := router.CreateServer(":" + cfg.ServerPort)

	go func() {
		logger.Logger.Info("Starting server", zap.String("port", cfg.ServerPort))
		logger.Logger.Info("Swagger docs available at", zap.String("url", "http://localhost:"+cfg.ServerPort+"/docs/index.html"))
		if err := srv.ListenAndServe(); err != nil {
			logger.Logger.Info("Server stopped", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Logger.Info("Server exited properly")
}
