package http

import (
	"net/http"
	"time"

	"duskforge-api/internal/adapters/handlers"
	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type RouterConfig struct {
	AccessTokenSecret  string
	CORSAllowedOrigins string
}

type Router struct {
	engine            *gin.Engine
	config            RouterConfig
	authHandler       *handlers.AuthHandler
	oauthHandler      *handlers.OAuthHandler
	userHandler       *handlers.UserHandler
	movieHandler      *handlers.MovieHandler
	actorHandler      *handlers.ActorHandler
	collectionHandler *handlers.CollectionHandler
	reviewHandler     *handlers.ReviewHandler
	commentHandler    *handlers.CommentHandler
	followHandler     *handlers.FollowHandler
	messageHandler    *handlers.MessageHandler
	blockHandler      *handlers.BlockHandler
	statsHandler      *handlers.StatsHandler
	activityHandler   *handlers.ActivityHandler
	wsHandler            *handlers.WebSocketHandler
	adminHandler         *handlers.AdminHandler
	importHandler        *handlers.ImportHandler
	notificationHandler  *handlers.NotificationHandler
	exportHandler        *handlers.ExportHandler
	achievementHandler   *handlers.AchievementHandler
	userService          ports.UserService
	activityRepo         ports.ActivityRepository
	achievementService   ports.AchievementService
	banCache             ports.BanCache
}

func NewRouter(
	config RouterConfig,
	authHandler *handlers.AuthHandler,
	oauthHandler *handlers.OAuthHandler,
	userHandler *handlers.UserHandler,
	movieHandler *handlers.MovieHandler,
	actorHandler *handlers.ActorHandler,
	collectionHandler *handlers.CollectionHandler,
	reviewHandler *handlers.ReviewHandler,
	commentHandler *handlers.CommentHandler,
	followHandler *handlers.FollowHandler,
	messageHandler *handlers.MessageHandler,
	blockHandler *handlers.BlockHandler,
	statsHandler *handlers.StatsHandler,
	activityHandler *handlers.ActivityHandler,
	wsHandler *handlers.WebSocketHandler,
	adminHandler *handlers.AdminHandler,
	importHandler *handlers.ImportHandler,
	notificationHandler *handlers.NotificationHandler,
	exportHandler *handlers.ExportHandler,
	achievementHandler *handlers.AchievementHandler,
	userService ports.UserService,
	activityRepo ports.ActivityRepository,
	achievementService ports.AchievementService,
	banCache ports.BanCache,
) *Router {
	return &Router{
		engine:            gin.Default(),
		config:            config,
		authHandler:       authHandler,
		oauthHandler:      oauthHandler,
		userHandler:       userHandler,
		movieHandler:      movieHandler,
		actorHandler:      actorHandler,
		collectionHandler: collectionHandler,
		reviewHandler:     reviewHandler,
		commentHandler:    commentHandler,
		followHandler:     followHandler,
		messageHandler:    messageHandler,
		blockHandler:      blockHandler,
		statsHandler:      statsHandler,
		activityHandler:   activityHandler,
		wsHandler:           wsHandler,
		adminHandler:        adminHandler,
		importHandler:       importHandler,
		notificationHandler: notificationHandler,
		exportHandler:       exportHandler,
		achievementHandler:  achievementHandler,
		userService:         userService,
		activityRepo:        activityRepo,
		achievementService:  achievementService,
		banCache:            banCache,
	}
}

func (r *Router) Setup() *gin.Engine {
	r.engine.Use(middleware.CORS(r.config.CORSAllowedOrigins))
	r.engine.Use(middleware.ActivityLogger(r.activityRepo, r.achievementService))

	r.engine.GET("/health", r.healthCheck)
	r.engine.GET("/", r.root)

	r.engine.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.engine.Group("/api/v1")
	{
		r.setupAuthRoutes(v1)
		r.setupUserRoutes(v1)
		r.setupMovieRoutes(v1)
		r.setupActorRoutes(v1)
		r.setupReviewRoutes(v1)
		r.setupCommentRoutes(v1)
		r.setupMessageRoutes(v1)
		r.setupActivityRoutes(v1)
		r.setupReportRoutes(v1)
		r.setupAdminRoutes(v1)
		r.setupImportRoutes(v1)
		r.setupNotificationRoutes(v1)
		r.setupAchievementRoutes(v1)
		v1.GET("/ws", r.wsHandler.Connect)
	}

	return r.engine
}

func (r *Router) setupAuthRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", r.authHandler.Register)
		auth.POST("/login", r.authHandler.Login)
		auth.POST("/refresh", r.authHandler.Refresh)
		auth.POST("/logout", r.authHandler.Logout)
		auth.POST("/verify-email/send", r.authHandler.SendVerificationCode)
		auth.POST("/verify-email", r.authHandler.VerifyEmail)
		auth.POST("/password-reset/request", r.authHandler.RequestPasswordReset)
		auth.POST("/password-reset", r.authHandler.ResetPassword)
		auth.POST("/email-change/request", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.authHandler.RequestEmailChange)
		auth.POST("/email-change/confirm", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.authHandler.ConfirmEmailChange)

		oauth := auth.Group("/oauth")
		{
			oauth.GET("/providers", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.oauthHandler.GetLinkedProviders)

			oauth.GET("/github", r.oauthHandler.GitHubRedirect)
			oauth.GET("/github/callback", r.oauthHandler.GitHubCallback)
			oauth.GET("/github/link", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.oauthHandler.LinkGitHub)
			oauth.DELETE("/github/unlink", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.oauthHandler.UnlinkGitHub)

			oauth.GET("/google", r.oauthHandler.GoogleRedirect)
			oauth.GET("/google/callback", r.oauthHandler.GoogleCallback)
			oauth.GET("/google/link", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.oauthHandler.LinkGoogle)
			oauth.DELETE("/google/unlink", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.oauthHandler.UnlinkGoogle)
		}
	}
}

func (r *Router) setupUserRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	users.Use(middleware.Locale(r.userService))
	{
		users.GET("/search", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.userHandler.Search)
		users.GET("/me", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.userHandler.GetCurrentUser)
		users.PATCH("/me", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.userHandler.UpdateCurrentUser)
		users.PUT("/me/avatar", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.userHandler.UploadAvatar)
		users.DELETE("/me/avatar", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.userHandler.DeleteAvatar)
		users.PUT("/me/password", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.userHandler.ChangePassword)
		users.DELETE("/me", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.userHandler.DeleteCurrentUser)
		users.GET("/me/export", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.exportHandler.ExportUserData)
		users.GET("/me/blocked", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.blockHandler.GetBlockedUsers)
		users.GET("/:userId", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.userHandler.GetByID)
		users.GET("/:userId/stats", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.statsHandler.GetUserStats)
		users.GET("/:userId/reviews", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.reviewHandler.GetByUserID)
		users.POST("/:userId/follow", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.followHandler.Follow)
		users.DELETE("/:userId/follow", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.followHandler.Unfollow)
		users.DELETE("/:userId/followers", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.followHandler.RemoveFollower)
		users.GET("/:userId/followers", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.followHandler.GetFollowers)
		users.GET("/:userId/following", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.followHandler.GetFollowing)
		users.POST("/:userId/block", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.blockHandler.BlockUser)
		users.DELETE("/:userId/block", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.blockHandler.UnblockUser)
		users.GET("/me/achievements/recent", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.achievementHandler.RecentForMe)
		users.GET("/:userId/activities", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.activityHandler.GetByUserID)
		users.GET("/:userId/achievements", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.achievementHandler.ListForUser)

		collections := users.Group("/:userId/collections")
		{
			collections.POST("", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.collectionHandler.Create)
			collections.GET("", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.collectionHandler.GetByUserID)
			collections.GET("/:slug", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.collectionHandler.GetBySlug)
			collections.PATCH("/:slug", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.collectionHandler.Update)
			collections.DELETE("/:slug", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.collectionHandler.Delete)
			collections.POST("/:slug/items", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.collectionHandler.AddItem)
			collections.GET("/:slug/items", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.collectionHandler.GetItems)
			collections.DELETE("/:slug/items/:tmdbId", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.collectionHandler.RemoveItem)
		}
	}
}

func (r *Router) setupMovieRoutes(rg *gin.RouterGroup) {
	movies := rg.Group("/movies")
	movies.Use(middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache))
	movies.Use(middleware.Locale(r.userService))
	{
		movies.GET("/search", r.movieHandler.Search)
		movies.GET("/discover", r.movieHandler.Discover)
		movies.GET("/popular", r.movieHandler.GetPopular)
		movies.GET("/upcoming", r.movieHandler.GetUpcoming)
		movies.GET("/genres", r.movieHandler.GetGenres)
		movies.GET("/:id", r.movieHandler.GetByID)
		movies.GET("/:id/trailer", r.movieHandler.GetTrailer)
		movies.GET("/:id/cast", r.movieHandler.GetCast)
		movies.GET("/:id/release-dates", r.movieHandler.GetReleaseDates)
		movies.GET("/:id/reviews", r.reviewHandler.GetByMovieID)
		movies.POST("/:id/reviews", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.reviewHandler.Create)
	}
}

func (r *Router) setupActorRoutes(rg *gin.RouterGroup) {
	actors := rg.Group("/actors")
	actors.Use(middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache))
	actors.Use(middleware.Locale(r.userService))
	{
		actors.GET("/search", r.actorHandler.Search)
		actors.GET("/:id", r.actorHandler.GetByID)
		actors.GET("/:id/movies", r.actorHandler.GetFilmography)
	}
}

func (r *Router) setupReviewRoutes(rg *gin.RouterGroup) {
	reviews := rg.Group("/reviews")
	reviews.Use(middleware.Locale(r.userService))
	{
		reviews.GET("/:reviewId", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.reviewHandler.GetByID)
		reviews.PATCH("/:reviewId", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.reviewHandler.Update)
		reviews.DELETE("/:reviewId", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.reviewHandler.Delete)
		reviews.POST("/:reviewId/like", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.reviewHandler.Like)
		reviews.DELETE("/:reviewId/like", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.reviewHandler.Unlike)
		reviews.GET("/:reviewId/comments", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.commentHandler.GetByReviewID)
		reviews.POST("/:reviewId/comments", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.commentHandler.Create)
	}
}

func (r *Router) setupCommentRoutes(rg *gin.RouterGroup) {
	comments := rg.Group("/comments")
	{
		comments.GET("/:commentId", middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache), r.commentHandler.GetByID)
		comments.PATCH("/:commentId", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.commentHandler.Update)
		comments.DELETE("/:commentId", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.commentHandler.Delete)
		comments.POST("/:commentId/like", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.commentHandler.Like)
		comments.DELETE("/:commentId/like", middleware.Auth(r.config.AccessTokenSecret, r.banCache), r.commentHandler.Unlike)
	}
}

func (r *Router) setupActivityRoutes(rg *gin.RouterGroup) {
	rg.GET("/feed", middleware.Auth(r.config.AccessTokenSecret, r.banCache), middleware.Locale(r.userService), r.activityHandler.GetFeed)
}

func (r *Router) setupMessageRoutes(rg *gin.RouterGroup) {
	messages := rg.Group("/messages")
	messages.Use(middleware.Auth(r.config.AccessTokenSecret, r.banCache))
	{
		messages.GET("", r.messageHandler.GetConversations)
		messages.GET("/:id", r.messageHandler.GetConversation)
		messages.POST("/:id", r.messageHandler.SendMessage)
		messages.PUT("/:id/read", r.messageHandler.MarkAsRead)
		messages.POST("/:id/close", r.messageHandler.CloseConversation)
		messages.DELETE("/:id/close", r.messageHandler.ReopenConversation)
		messages.PATCH("/:id", r.messageHandler.UpdateMessage)
		messages.DELETE("/:id", r.messageHandler.DeleteMessage)
		messages.POST("/:id/reactions", r.messageHandler.AddReaction)
		messages.DELETE("/:id/reactions", r.messageHandler.RemoveReaction)
	}
}

func (r *Router) setupReportRoutes(rg *gin.RouterGroup) {
	reports := rg.Group("/reports")
	reports.Use(middleware.Auth(r.config.AccessTokenSecret, r.banCache))
	{
		reports.POST("", r.adminHandler.SubmitReport)
	}
}

func (r *Router) setupAdminRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin")
	admin.Use(middleware.Auth(r.config.AccessTokenSecret, r.banCache))
	{
		adminOrSuper := admin.Group("")
		adminOrSuper.Use(middleware.RequireRole(string(domain.UserRoleAdmin), string(domain.UserRoleSuperAdmin)))
		{
			adminOrSuper.POST("/users/:userId/ban", r.adminHandler.BanUser)
			adminOrSuper.DELETE("/users/:userId/ban", r.adminHandler.UnbanUser)
			adminOrSuper.DELETE("/reviews/:reviewId", r.adminHandler.DeleteReview)
			adminOrSuper.DELETE("/comments/:commentId", r.adminHandler.DeleteComment)
			adminOrSuper.GET("/reports", r.adminHandler.ListReports)
			adminOrSuper.GET("/reports/:reportId", r.adminHandler.GetReport)
			adminOrSuper.PATCH("/reports/:reportId", r.adminHandler.ResolveReport)
			adminOrSuper.DELETE("/reports/:reportId", r.adminHandler.DeleteReport)

			adminOrSuper.POST("/achievements", r.achievementHandler.Create)
			adminOrSuper.PATCH("/achievements/:id", r.achievementHandler.Update)
			adminOrSuper.DELETE("/achievements/:id", r.achievementHandler.Delete)
		}

		superOnly := admin.Group("")
		superOnly.Use(middleware.RequireRole(string(domain.UserRoleSuperAdmin)))
		{
			superOnly.PATCH("/users/:userId/role", r.adminHandler.SetUserRole)
		}
	}
}

func (r *Router) setupImportRoutes(rg *gin.RouterGroup) {
	importGroup := rg.Group("/import")
	importGroup.Use(middleware.Auth(r.config.AccessTokenSecret, r.banCache))
	{
		importGroup.POST("/letterboxd", r.importHandler.ImportLetterboxd)
		importGroup.GET("/letterboxd/status", r.importHandler.GetImportStatus)
	}
}

func (r *Router) setupAchievementRoutes(rg *gin.RouterGroup) {
	achievements := rg.Group("/achievements")
	achievements.Use(middleware.OptionalAuth(r.config.AccessTokenSecret, r.banCache))
	{
		achievements.GET("", r.achievementHandler.List)
		achievements.GET("/:id", r.achievementHandler.GetByID)
	}
}

func (r *Router) setupNotificationRoutes(rg *gin.RouterGroup) {
	notifications := rg.Group("/notifications")
	notifications.Use(middleware.Auth(r.config.AccessTokenSecret, r.banCache))
	{
		notifications.GET("", r.notificationHandler.GetNotifications)
		notifications.GET("/unread/count", r.notificationHandler.GetUnreadCount)
		notifications.PUT("/read", r.notificationHandler.MarkAllAsRead)
		notifications.GET("/preferences", r.notificationHandler.GetPreferences)
		notifications.PATCH("/preferences", r.notificationHandler.UpdatePreferences)
		notifications.DELETE("", r.notificationHandler.DeleteAll)
		notifications.PUT("/:notificationId/read", r.notificationHandler.MarkAsRead)
		notifications.DELETE("/:notificationId", r.notificationHandler.Delete)
	}
}

func (r *Router) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Duskforge API is running",
	})
}

func (r *Router) root(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to Duskforge API",
		"version": "v1",
		"docs":    "/docs/index.html",
	})
}

func (r *Router) CreateServer(addr string) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      r.engine,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}
