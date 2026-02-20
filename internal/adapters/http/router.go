package http

import (
	"net/http"
	"time"

	"duskforge-api/internal/adapters/handlers"
	"duskforge-api/internal/adapters/middleware"
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
	userService       ports.UserService
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
	userService ports.UserService,
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
		userService:       userService,
	}
}

func (r *Router) Setup() *gin.Engine {
	r.engine.Use(middleware.CORS(r.config.CORSAllowedOrigins))

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

		oauth := auth.Group("/oauth")
		{
			oauth.GET("/providers", middleware.Auth(r.config.AccessTokenSecret), r.oauthHandler.GetLinkedProviders)

			oauth.GET("/github", r.oauthHandler.GitHubRedirect)
			oauth.GET("/github/callback", r.oauthHandler.GitHubCallback)
			oauth.GET("/github/link", middleware.Auth(r.config.AccessTokenSecret), r.oauthHandler.LinkGitHub)
			oauth.DELETE("/github/unlink", middleware.Auth(r.config.AccessTokenSecret), r.oauthHandler.UnlinkGitHub)

			oauth.GET("/google", r.oauthHandler.GoogleRedirect)
			oauth.GET("/google/callback", r.oauthHandler.GoogleCallback)
			oauth.GET("/google/link", middleware.Auth(r.config.AccessTokenSecret), r.oauthHandler.LinkGoogle)
			oauth.DELETE("/google/unlink", middleware.Auth(r.config.AccessTokenSecret), r.oauthHandler.UnlinkGoogle)
		}
	}
}

func (r *Router) setupUserRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.GET("/search", r.userHandler.Search)
		users.GET("/me", middleware.Auth(r.config.AccessTokenSecret), r.userHandler.GetCurrentUser)
		users.PATCH("/me", middleware.Auth(r.config.AccessTokenSecret), r.userHandler.UpdateCurrentUser)
		users.PUT("/me/avatar", middleware.Auth(r.config.AccessTokenSecret), r.userHandler.UploadAvatar)
		users.DELETE("/me/avatar", middleware.Auth(r.config.AccessTokenSecret), r.userHandler.DeleteAvatar)
		users.PUT("/me/password", middleware.Auth(r.config.AccessTokenSecret), r.userHandler.ChangePassword)
		users.DELETE("/me", middleware.Auth(r.config.AccessTokenSecret), r.userHandler.DeleteCurrentUser)
		users.GET("/:userId", middleware.OptionalAuth(r.config.AccessTokenSecret), r.userHandler.GetByID)
		users.GET("/:userId/reviews", middleware.OptionalAuth(r.config.AccessTokenSecret), r.reviewHandler.GetByUserID)

		collections := users.Group("/:userId/collections")
		{
			collections.POST("", middleware.Auth(r.config.AccessTokenSecret), r.collectionHandler.Create)
			collections.GET("", middleware.OptionalAuth(r.config.AccessTokenSecret), r.collectionHandler.GetByUserID)
			collections.GET("/:slug", middleware.OptionalAuth(r.config.AccessTokenSecret), r.collectionHandler.GetBySlug)
			collections.PATCH("/:slug", middleware.Auth(r.config.AccessTokenSecret), r.collectionHandler.Update)
			collections.DELETE("/:slug", middleware.Auth(r.config.AccessTokenSecret), r.collectionHandler.Delete)
			collections.POST("/:slug/items", middleware.Auth(r.config.AccessTokenSecret), r.collectionHandler.AddItem)
			collections.GET("/:slug/items", middleware.OptionalAuth(r.config.AccessTokenSecret), r.collectionHandler.GetItems)
			collections.DELETE("/:slug/items/:tmdbId", middleware.Auth(r.config.AccessTokenSecret), r.collectionHandler.RemoveItem)
		}
	}
}

func (r *Router) setupMovieRoutes(rg *gin.RouterGroup) {
	movies := rg.Group("/movies")
	movies.Use(middleware.OptionalAuth(r.config.AccessTokenSecret))
	movies.Use(middleware.Locale(r.userService))
	{
		movies.GET("/search", r.movieHandler.Search)
		movies.GET("/discover", r.movieHandler.Discover)
		movies.GET("/popular", r.movieHandler.GetPopular)
		movies.GET("/genres", r.movieHandler.GetGenres)
		movies.GET("/:id", r.movieHandler.GetByID)
		movies.GET("/:id/trailer", r.movieHandler.GetTrailer)
		movies.GET("/:id/cast", r.movieHandler.GetCast)
		movies.GET("/:id/release-dates", r.movieHandler.GetReleaseDates)
		movies.GET("/:id/reviews", r.reviewHandler.GetByMovieID)
		movies.POST("/:id/reviews", middleware.Auth(r.config.AccessTokenSecret), r.reviewHandler.Create)
	}
}

func (r *Router) setupActorRoutes(rg *gin.RouterGroup) {
	actors := rg.Group("/actors")
	actors.Use(middleware.OptionalAuth(r.config.AccessTokenSecret))
	actors.Use(middleware.Locale(r.userService))
	{
		actors.GET("/search", r.actorHandler.Search)
		actors.GET("/:id", r.actorHandler.GetByID)
		actors.GET("/:id/movies", r.actorHandler.GetFilmography)
	}
}

func (r *Router) setupReviewRoutes(rg *gin.RouterGroup) {
	reviews := rg.Group("/reviews")
	{
		reviews.GET("/:reviewId", middleware.OptionalAuth(r.config.AccessTokenSecret), r.reviewHandler.GetByID)
		reviews.PATCH("/:reviewId", middleware.Auth(r.config.AccessTokenSecret), r.reviewHandler.Update)
		reviews.DELETE("/:reviewId", middleware.Auth(r.config.AccessTokenSecret), r.reviewHandler.Delete)
		reviews.POST("/:reviewId/like", middleware.Auth(r.config.AccessTokenSecret), r.reviewHandler.Like)
		reviews.DELETE("/:reviewId/like", middleware.Auth(r.config.AccessTokenSecret), r.reviewHandler.Unlike)
		reviews.GET("/:reviewId/comments", middleware.OptionalAuth(r.config.AccessTokenSecret), r.commentHandler.GetByReviewID)
		reviews.POST("/:reviewId/comments", middleware.Auth(r.config.AccessTokenSecret), r.commentHandler.Create)
	}
}

func (r *Router) setupCommentRoutes(rg *gin.RouterGroup) {
	comments := rg.Group("/comments")
	{
		comments.PATCH("/:commentId", middleware.Auth(r.config.AccessTokenSecret), r.commentHandler.Update)
		comments.DELETE("/:commentId", middleware.Auth(r.config.AccessTokenSecret), r.commentHandler.Delete)
		comments.POST("/:commentId/like", middleware.Auth(r.config.AccessTokenSecret), r.commentHandler.Like)
		comments.DELETE("/:commentId/like", middleware.Auth(r.config.AccessTokenSecret), r.commentHandler.Unlike)
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
