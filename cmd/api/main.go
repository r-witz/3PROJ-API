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
	"duskforge-api/internal/core/ports"
	"duskforge-api/internal/core/services"
	"duskforge-api/pkg/cache"
	"duskforge-api/pkg/database"
	"duskforge-api/pkg/email"
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
	banCache := repositories.NewBanCache(redisClient)
	verificationRepo := repositories.NewVerificationCodeRepo(redisClient)

	emailSender := email.NewBrevoSender(cfg.BrevoAPIKey, cfg.EmailFromAddress, cfg.EmailFromName)
	logger.Logger.Info("Brevo email sender initialized")

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
	reportRepo := repositories.NewReportRepository(db)
	notifRepo := repositories.NewNotificationRepository(db)
	notifPrefRepo := repositories.NewNotificationPreferencesRepository(db)
	exportRepo := repositories.NewExportRepository(db)
	achievementRepo := repositories.NewAchievementRepository(db)

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
	if err := minioStorage.EnsureBucket(context.Background()); err != nil {
		logger.Logger.Fatal("Failed to ensure MinIO bucket", zap.Error(err))
	}
	logger.Logger.Info("MinIO storage client initialized", zap.String("endpoint", cfg.MinioEndpoint))

	collectionService := services.NewCollectionService(collectionRepo, collectionItemRepo, cachedTMDB, reviewRepo)

	tokenConfig := services.TokenConfig{
		AccessTokenSecret:  cfg.AccessTokenSecret,
		AccessTokenExpiry:  cfg.AccessTokenExpiry,
		RefreshTokenSecret: cfg.RefreshTokenSecret,
		RefreshTokenExpiry: cfg.RefreshTokenExpiry,
	}

	authService := services.NewAuthService(userRepo, sessionRepo, collectionService, emailSender, verificationRepo, tokenConfig)
	userService := services.NewUserService(userRepo, verificationRepo)

	followService := services.NewFollowService(followRepo, userRepo)
	blockService := services.NewBlockService(blockRepo, followRepo, userRepo, convStateRepo)
	reviewService := services.NewReviewService(reviewRepo, reviewLikeRepo, commentRepo, collectionService, userRepo, blockRepo)
	commentService := services.NewCommentService(commentRepo, commentLikeRepo, reviewRepo, userRepo, blockRepo)
	activityService := services.NewActivityService(activityRepo, userRepo, reviewRepo, collectionRepo, commentRepo)
	messageService := services.NewMessageService(messageRepo, followRepo, userRepo, blockRepo, attachmentRepo, reactionRepo, convStateRepo, minioStorage)
	movieService := services.NewMovieService(cachedTMDB, reviewRepo)
	actorService := services.NewActorService(cachedTMDB, reviewRepo)
	reportService := services.NewReportService(reportRepo, userRepo, reviewRepo, commentRepo)
	adminService := services.NewAdminService(userRepo, reviewRepo, commentRepo, sessionRepo, banCache)
	notifService := services.NewNotificationService(notifRepo, notifPrefRepo)
	exportService := services.NewExportService(exportRepo)

	bannedIDs, err := userRepo.GetBannedUserIDs(context.Background())
	if err != nil {
		logger.Logger.Warn("Failed to load banned users", zap.Error(err))
	} else if err := banCache.SyncBannedUsers(context.Background(), bannedIDs); err != nil {
		logger.Logger.Warn("Failed to sync banned users to cache", zap.Error(err))
	}

	if err := adminService.SeedSuperAdmin(context.Background(), ports.SeedSuperAdminInput{
		Email:    cfg.SeedAdminEmail,
		Username: cfg.SeedAdminUsername,
		Password: cfg.SeedAdminPassword,
	}); err != nil {
		logger.Logger.Fatal("Failed to seed superadmin", zap.Error(err))
	}

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
	userHandler := handlers.NewUserHandler(userService, followService, blockService, minioStorage, banCache)
	movieHandler := handlers.NewMovieHandler(movieService)
	actorHandler := handlers.NewActorHandler(actorService)
	collectionHandler := handlers.NewCollectionHandler(collectionService, blockService, banCache)
	statsHandler := handlers.NewStatsHandler(statsService, blockService, banCache)
	activityHandler := handlers.NewActivityHandler(activityService, movieService, blockService, banCache)
	hub := ws.NewHub()
	go hub.Run()

	achievementService := services.NewAchievementService(achievementRepo, statsRepo, notifService, hub)
	achievementHandler := handlers.NewAchievementHandler(achievementService)

	reviewHandler := handlers.NewReviewHandler(reviewService, movieService, userService, blockService, banCache, notifService, hub)
	commentHandler := handlers.NewCommentHandler(commentService, userService, blockService, banCache, notifService, hub, reviewService)
	followHandler := handlers.NewFollowHandler(followService, blockService, hub, banCache, notifService)
	messageHandler := handlers.NewMessageHandler(messageService, hub)
	blockHandler := handlers.NewBlockHandler(blockService, hub)
	importService := services.NewImportService(collectionRepo, collectionItemRepo, reviewRepo, cachedTMDB, hub, achievementService)

	adminHandler := handlers.NewAdminHandler(adminService, reportService, messageRepo, hub)
	importHandler := handlers.NewImportHandler(importService)
	exportHandler := handlers.NewExportHandler(exportService)
	wsHandler := handlers.NewWebSocketHandler(hub, cfg.AccessTokenSecret)
	notificationHandler := handlers.NewNotificationHandler(notifService)

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
		adminHandler,
		importHandler,
		notificationHandler,
		exportHandler,
		achievementHandler,
		userService,
		activityRepo,
		achievementService,
		banCache,
	)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Logger.Error("unverified-cleanup panic", zap.Any("panic", r))
			}
		}()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			ctx := context.Background()
			cutoff := time.Now().Add(-24 * time.Hour)
			users, err := userRepo.GetUnverifiedBefore(ctx, cutoff)
			if err != nil {
				logger.Logger.Error("Failed to fetch unverified accounts for cleanup", zap.Error(err))
				continue
			}
			if len(users) == 0 {
				continue
			}
			deleted, err := userRepo.DeleteUnverifiedBefore(ctx, cutoff)
			if err != nil {
				logger.Logger.Error("Failed to cleanup unverified accounts", zap.Error(err))
				continue
			}
			for _, u := range users {
				_ = verificationRepo.DeleteAllForEmail(ctx, u.Email)
			}
			logger.Logger.Info("Cleaned up unverified accounts", zap.Int64("deleted", deleted))
		}
	}()

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
