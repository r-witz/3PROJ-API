package config

import (
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

	MinioEndpoint  string `mapstructure:"MINIO_ENDPOINT"`
	MinioAccessKey string `mapstructure:"MINIO_ACCESS_KEY"`
	MinioSecretKey string `mapstructure:"MINIO_SECRET_KEY"`
	MinioBucket    string `mapstructure:"MINIO_BUCKET"`
	MinioUseSSL    bool   `mapstructure:"MINIO_USE_SSL"`
	MinioPublicURL string `mapstructure:"MINIO_PUBLIC_URL"`

	RedisURL string `mapstructure:"REDIS_URL"`

	GitHubClientID     string `mapstructure:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `mapstructure:"GITHUB_CLIENT_SECRET"`
	GoogleClientID     string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	OAuthRedirectBase  string `mapstructure:"OAUTH_REDIRECT_BASE"`
	OAuthStateSecret   string `mapstructure:"OAUTH_STATE_SECRET"`

	CORSAllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`

	BrevoAPIKey      string `mapstructure:"BREVO_API_KEY"`
	EmailFromAddress string `mapstructure:"EMAIL_FROM_ADDRESS"`
	EmailFromName    string `mapstructure:"EMAIL_FROM_NAME"`

	SeedAdminEmail    string `mapstructure:"SEED_ADMIN_EMAIL"`
	SeedAdminUsername string `mapstructure:"SEED_ADMIN_USERNAME"`
	SeedAdminPassword string `mapstructure:"SEED_ADMIN_PASSWORD"`
}

func LoadConfig() (Config, error) {
	viper.AutomaticEnv()

	viper.BindEnv("DATABASE_URL")
	viper.BindEnv("TMDB_API_KEY")
	viper.BindEnv("ACCESS_TOKEN_SECRET")
	viper.BindEnv("REFRESH_TOKEN_SECRET")
	viper.BindEnv("MINIO_ACCESS_KEY")
	viper.BindEnv("MINIO_SECRET_KEY")
	viper.BindEnv("MINIO_PUBLIC_URL")
	viper.BindEnv("GITHUB_CLIENT_ID")
	viper.BindEnv("GITHUB_CLIENT_SECRET")
	viper.BindEnv("GOOGLE_CLIENT_ID")
	viper.BindEnv("GOOGLE_CLIENT_SECRET")
	viper.BindEnv("OAUTH_STATE_SECRET")
	viper.BindEnv("BREVO_API_KEY")

	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("ACCESS_TOKEN_EXPIRY", 15*time.Minute)
	viper.SetDefault("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour)
	viper.SetDefault("MINIO_ENDPOINT", "minio:9000")
	viper.SetDefault("MINIO_BUCKET", "duskforge")
	viper.SetDefault("MINIO_USE_SSL", false)
	viper.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	viper.SetDefault("OAUTH_REDIRECT_BASE", "http://localhost:8080")
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "*")
	viper.SetDefault("EMAIL_FROM_ADDRESS", "noreply@duskforge.studio")
	viper.SetDefault("EMAIL_FROM_NAME", "Duskforge")
	viper.SetDefault("SEED_ADMIN_EMAIL", "admin@duskforge.studio")
	viper.SetDefault("SEED_ADMIN_USERNAME", "superadmin")
	viper.SetDefault("SEED_ADMIN_PASSWORD", "Admin123!")

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return config, err
	}

	return config, nil
}
