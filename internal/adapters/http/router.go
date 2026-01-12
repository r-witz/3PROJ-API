package http

import (
	"net/http"
	"time"

	"duskforge-api/internal/adapters/handlers"
	"duskforge-api/internal/adapters/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type RouterConfig struct {
	AccessTokenSecret string
}

type Router struct {
	engine       *gin.Engine
	config       RouterConfig
	authHandler  *handlers.AuthHandler
	oauthHandler *handlers.OAuthHandler
	userHandler  *handlers.UserHandler
	movieHandler *handlers.MovieHandler
}

func NewRouter(
	config RouterConfig,
	authHandler *handlers.AuthHandler,
	oauthHandler *handlers.OAuthHandler,
	userHandler *handlers.UserHandler,
	movieHandler *handlers.MovieHandler,
) *Router {
	return &Router{
		engine:       gin.Default(),
		config:       config,
		authHandler:  authHandler,
		oauthHandler: oauthHandler,
		userHandler:  userHandler,
		movieHandler: movieHandler,
	}
}

func (r *Router) Setup() *gin.Engine {
	r.engine.Use(middleware.CORS())

	r.engine.GET("/health", r.healthCheck)
	r.engine.GET("/", r.root)

	r.engine.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.engine.Group("/api/v1")
	{
		r.setupAuthRoutes(v1)
		r.setupUserRoutes(v1)
		r.setupMovieRoutes(v1)
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

		// OAuth routes
		oauth := auth.Group("/oauth")
		{
			// GitHub OAuth
			oauth.GET("/github", r.oauthHandler.GitHubRedirect)
			oauth.GET("/github/callback", r.oauthHandler.GitHubCallback)
			oauth.POST("/github/link", middleware.Auth(r.config.AccessTokenSecret), r.oauthHandler.LinkGitHub)
			oauth.DELETE("/github/unlink", middleware.Auth(r.config.AccessTokenSecret), r.oauthHandler.UnlinkGitHub)

			// Google OAuth
			oauth.GET("/google", r.oauthHandler.GoogleRedirect)
			oauth.GET("/google/callback", r.oauthHandler.GoogleCallback)
			oauth.POST("/google/link", middleware.Auth(r.config.AccessTokenSecret), r.oauthHandler.LinkGoogle)
			oauth.DELETE("/google/unlink", middleware.Auth(r.config.AccessTokenSecret), r.oauthHandler.UnlinkGoogle)
		}
	}
}

func (r *Router) setupUserRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.GET("/me", middleware.Auth(r.config.AccessTokenSecret), r.userHandler.GetCurrentUser)
		users.PATCH("/me", middleware.Auth(r.config.AccessTokenSecret), r.userHandler.UpdateCurrentUser)
		users.GET("/:id", r.userHandler.GetByID)
	}
}

func (r *Router) setupMovieRoutes(rg *gin.RouterGroup) {
	movies := rg.Group("/movies")
	{
		movies.GET("/search", r.movieHandler.Search)
		movies.GET("/popular", r.movieHandler.GetPopular)
		movies.GET("/:id", r.movieHandler.GetByID)
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
