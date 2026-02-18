package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort         string        `mapstructure:"SERVER_PORT"`
	DatabaseURL        string        `mapstructure:"DATABASE_URL"`
	TMDBAPIKey         string        `mapstructure:"TMDB_API_KEY"`
	LogLevel           string        `mapstructure:"LOG_LEVEL"`
	AccessTokenSecret  string        `mapstructure:"ACCESS_TOKEN_SECRET"`
	AccessTokenExpiry  time.Duration `mapstructure:"ACCESS_TOKEN_EXPIRY"`
	RefreshTokenSecret string        `mapstructure:"REFRESH_TOKEN_SECRET"`
	RefreshTokenExpiry time.Duration `mapstructure:"REFRESH_TOKEN_EXPIRY"`

	// MinIO Configuration
	MinioEndpoint  string `mapstructure:"MINIO_ENDPOINT"`
	MinioAccessKey string `mapstructure:"MINIO_ACCESS_KEY"`
	MinioSecretKey string `mapstructure:"MINIO_SECRET_KEY"`
	MinioBucket    string `mapstructure:"MINIO_BUCKET"`
	MinioUseSSL    bool   `mapstructure:"MINIO_USE_SSL"`
	MinioPublicURL string `mapstructure:"MINIO_PUBLIC_URL"`

	// OAuth Configuration
	GitHubClientID     string `mapstructure:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `mapstructure:"GITHUB_CLIENT_SECRET"`
	GoogleClientID     string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	OAuthRedirectBase  string `mapstructure:"OAUTH_REDIRECT_BASE"`
	OAuthStateSecret   string `mapstructure:"OAUTH_STATE_SECRET"`
}

func LoadConfig() (config Config, err error) {
	viper.AddConfigPath(".")
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	viper.BindEnv("SERVER_PORT")
	viper.BindEnv("DATABASE_URL")
	viper.BindEnv("TMDB_API_KEY")
	viper.BindEnv("LOG_LEVEL")
	viper.BindEnv("ACCESS_TOKEN_SECRET")
	viper.BindEnv("ACCESS_TOKEN_EXPIRY")
	viper.BindEnv("REFRESH_TOKEN_SECRET")
	viper.BindEnv("REFRESH_TOKEN_EXPIRY")

	// MinIO configuration bindings
	viper.BindEnv("MINIO_ENDPOINT")
	viper.BindEnv("MINIO_ACCESS_KEY")
	viper.BindEnv("MINIO_SECRET_KEY")
	viper.BindEnv("MINIO_BUCKET")
	viper.BindEnv("MINIO_USE_SSL")
	viper.BindEnv("MINIO_PUBLIC_URL")

	// OAuth configuration bindings
	viper.BindEnv("GITHUB_CLIENT_ID")
	viper.BindEnv("GITHUB_CLIENT_SECRET")
	viper.BindEnv("GOOGLE_CLIENT_ID")
	viper.BindEnv("GOOGLE_CLIENT_SECRET")
	viper.BindEnv("OAUTH_REDIRECT_BASE")
	viper.BindEnv("OAUTH_STATE_SECRET")

	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("ACCESS_TOKEN_EXPIRY", 15*time.Minute)
	viper.SetDefault("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour)
	viper.SetDefault("MINIO_ENDPOINT", "minio:9000")
	viper.SetDefault("MINIO_BUCKET", "duskforge")
	viper.SetDefault("MINIO_USE_SSL", false)
	viper.SetDefault("OAUTH_REDIRECT_BASE", "http://localhost:8080")

	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("No app.env file found, loading config from environment variables.")
		} else {
			return
		}
	}

	err = viper.Unmarshal(&config)
	return
}
