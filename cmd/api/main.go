package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"duskforge-api/internal/adapters/handlers"
	"duskforge-api/internal/adapters/http"
	"duskforge-api/internal/adapters/repositories"
	"duskforge-api/internal/config"
	"duskforge-api/internal/core/services"
	"duskforge-api/pkg/cache"
	"duskforge-api/pkg/database"
	"duskforge-api/pkg/logger"
	"duskforge-api/pkg/oauth"
	"duskforge-api/pkg/storage"
	"duskforge-api/pkg/tmdb"
	ws "duskforge-api/pkg/websocket"

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

	redisClient, err := cache.New(cfg.RedisURL)
	if err != nil {
		logger.Logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	cachedTMDB := tmdb.NewCachedClient(tmdbClient, redisClient)

	userRepo := repositories.NewUserRepository(db)
	sessionRepo := repositories.NewSessionRepository(db)
	oauthRepo := repositories.NewOAuthAccountRepository(db)
	followRepo := repositories.NewFollowRepository(db)
	reviewRepo := repositories.NewReviewRepository(db)
	reviewLikeRepo := repositories.NewReviewLikeRepository(db)
	commentRepo := repositories.NewCommentRepository(db)
	commentLikeRepo := repositories.NewCommentLikeRepository(db)
	collectionRepo := repositories.NewCollectionRepository(db)
	collectionItemRepo := repositories.NewCollectionItemRepository(db)
	messageRepo := repositories.NewMessageRepository(db)
	blockRepo := repositories.NewBlockRepository(db)
	attachmentRepo := repositories.NewMessageAttachmentRepository(db)
	reactionRepo := repositories.NewMessageReactionRepository(db)
	convStateRepo := repositories.NewConversationStateRepository(db)
	statsRepo := repositories.NewStatsRepository(db)
	activityRepo := repositories.NewActivityRepository(db)

	minioStorage, err := storage.NewMinioStorage(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
		cfg.MinioUseSSL,
		cfg.MinioPublicURL,
	)
	if err != nil {
		logger.Logger.Fatal("Failed to create MinIO storage client", zap.Error(err))
	}
	logger.Logger.Info("MinIO storage client initialized", zap.String("endpoint", cfg.MinioEndpoint))

	collectionService := services.NewCollectionService(collectionRepo, collectionItemRepo, cachedTMDB, reviewRepo)

	tokenConfig := services.TokenConfig{
		AccessTokenSecret:  cfg.AccessTokenSecret,
		AccessTokenExpiry:  cfg.AccessTokenExpiry,
		RefreshTokenSecret: cfg.RefreshTokenSecret,
		RefreshTokenExpiry: cfg.RefreshTokenExpiry,
	}

	authService := services.NewAuthService(userRepo, sessionRepo, collectionService, tokenConfig)
	userService := services.NewUserService(userRepo)

	followService := services.NewFollowService(followRepo, userRepo)
	blockService := services.NewBlockService(blockRepo, followRepo, userRepo, convStateRepo)
	reviewService := services.NewReviewService(reviewRepo, reviewLikeRepo, commentRepo, collectionService, userRepo, blockRepo)
	commentService := services.NewCommentService(commentRepo, commentLikeRepo, reviewRepo, userRepo, blockRepo)
	activityService := services.NewActivityService(activityRepo, userRepo, reviewRepo, collectionRepo, commentRepo)
	messageService := services.NewMessageService(messageRepo, followRepo, userRepo, blockRepo, attachmentRepo, reactionRepo, convStateRepo, minioStorage)
	movieService := services.NewMovieService(cachedTMDB, reviewRepo)
	actorService := services.NewActorService(cachedTMDB, reviewRepo)
	statsService := services.NewStatsService(statsRepo, userRepo)

	providers := make(map[oauth.OAuthProvider]oauth.Provider)
	if cfg.GitHubClientID != "" && cfg.GitHubClientSecret != "" {
		providers[oauth.ProviderGitHub] = oauth.NewGitHubProvider(cfg.GitHubClientID, cfg.GitHubClientSecret)
		logger.Logger.Info("GitHub OAuth provider initialized")
	}
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		providers[oauth.ProviderGoogle] = oauth.NewGoogleProvider(cfg.GoogleClientID, cfg.GoogleClientSecret)
		logger.Logger.Info("Google OAuth provider initialized")
	}

	stateSecret := cfg.OAuthStateSecret
	if stateSecret == "" {
		stateSecret = cfg.AccessTokenSecret
	}
	stateManager := oauth.NewStateManager(stateSecret, 10*time.Minute)

	oauthService := services.NewOAuthService(
		userRepo,
		oauthRepo,
		sessionRepo,
		collectionService,
		stateManager,
		providers,
		tokenConfig,
	)

	authHandler := handlers.NewAuthHandler(authService)
	oauthHandler := handlers.NewOAuthHandler(oauthService, cfg.OAuthRedirectBase)
	userHandler := handlers.NewUserHandler(userService, followService, blockService, minioStorage)
	movieHandler := handlers.NewMovieHandler(movieService)
	actorHandler := handlers.NewActorHandler(actorService)
	collectionHandler := handlers.NewCollectionHandler(collectionService, blockService)
	reviewHandler := handlers.NewReviewHandler(reviewService, movieService, userService, blockService)
	commentHandler := handlers.NewCommentHandler(commentService, userService, blockService)
	statsHandler := handlers.NewStatsHandler(statsService, blockService)
	activityHandler := handlers.NewActivityHandler(activityService, movieService, blockService)
	hub := ws.NewHub()
	go hub.Run()

	followHandler := handlers.NewFollowHandler(followService, blockService, hub)
	messageHandler := handlers.NewMessageHandler(messageService, hub)
	blockHandler := handlers.NewBlockHandler(blockService, hub)
	wsHandler := handlers.NewWebSocketHandler(hub, cfg.AccessTokenSecret)

	router := http.NewRouter(
		http.RouterConfig{
			AccessTokenSecret:  cfg.AccessTokenSecret,
			CORSAllowedOrigins: cfg.CORSAllowedOrigins,
		},
		authHandler,
		oauthHandler,
		userHandler,
		movieHandler,
		actorHandler,
		collectionHandler,
		reviewHandler,
		commentHandler,
		followHandler,
		messageHandler,
		blockHandler,
		statsHandler,
		activityHandler,
		wsHandler,
		userService,
		activityRepo,
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Logger.Info("Server exited properly")
}
