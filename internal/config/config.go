package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort string `mapstructure:"SERVER_PORT"`

	PostgresUser     string `mapstructure:"POSTGRES_USER"`
	PostgresPassword string `mapstructure:"POSTGRES_PASSWORD"`
	PostgresDB       string `mapstructure:"POSTGRES_DB"`
	PostgresHost     string `mapstructure:"POSTGRES_HOST"`
	PostgresPort     string `mapstructure:"POSTGRES_PORT"`
	PostgresSSLMode  string `mapstructure:"POSTGRES_SSLMODE"`
	DatabaseURL      string `mapstructure:"-"`

	TMDBAPIKey         string        `mapstructure:"TMDB_API_KEY"`
	LogLevel           string        `mapstructure:"LOG_LEVEL"`
	AccessTokenSecret  string        `mapstructure:"ACCESS_TOKEN_SECRET"`
	AccessTokenExpiry  time.Duration `mapstructure:"ACCESS_TOKEN_EXPIRY"`
	RefreshTokenSecret string        `mapstructure:"REFRESH_TOKEN_SECRET"`
	RefreshTokenExpiry time.Duration `mapstructure:"REFRESH_TOKEN_EXPIRY"`

	MinioHost      string `mapstructure:"MINIO_HOST"`
	MinioPort      string `mapstructure:"MINIO_PORT"`
	MinioAccessKey string `mapstructure:"MINIO_ACCESS_KEY"`
	MinioSecretKey string `mapstructure:"MINIO_SECRET_KEY"`
	MinioBucket    string `mapstructure:"MINIO_BUCKET"`
	MinioUseSSL    bool   `mapstructure:"MINIO_USE_SSL"`
	MinioPublicURL string `mapstructure:"MINIO_PUBLIC_URL"`
	MinioEndpoint  string `mapstructure:"-"`

	RedisHost string `mapstructure:"REDIS_HOST"`
	RedisPort string `mapstructure:"REDIS_PORT"`
	RedisDB   string `mapstructure:"REDIS_DB"`
	RedisURL  string `mapstructure:"-"`

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

var requiredEnvKeys = []string{
	"SERVER_PORT",
	"POSTGRES_USER",
	"POSTGRES_PASSWORD",
	"POSTGRES_DB",
	"POSTGRES_HOST",
	"POSTGRES_PORT",
	"POSTGRES_SSLMODE",
	"TMDB_API_KEY",
	"LOG_LEVEL",
	"ACCESS_TOKEN_SECRET",
	"ACCESS_TOKEN_EXPIRY",
	"REFRESH_TOKEN_SECRET",
	"REFRESH_TOKEN_EXPIRY",
	"MINIO_HOST",
	"MINIO_PORT",
	"MINIO_ACCESS_KEY",
	"MINIO_SECRET_KEY",
	"MINIO_BUCKET",
	"MINIO_USE_SSL",
	"MINIO_PUBLIC_URL",
	"REDIS_HOST",
	"REDIS_PORT",
	"REDIS_DB",
	"OAUTH_REDIRECT_BASE",
	"CORS_ALLOWED_ORIGINS",
	"EMAIL_FROM_ADDRESS",
	"EMAIL_FROM_NAME",
	"BREVO_API_KEY",
	"SEED_ADMIN_EMAIL",
	"SEED_ADMIN_USERNAME",
	"SEED_ADMIN_PASSWORD",
}

var optionalEnvKeys = []string{
	"GITHUB_CLIENT_ID",
	"GITHUB_CLIENT_SECRET",
	"GOOGLE_CLIENT_ID",
	"GOOGLE_CLIENT_SECRET",
	"OAUTH_STATE_SECRET",
}

func LoadConfig() (Config, error) {
	viper.AutomaticEnv()

	for _, key := range requiredEnvKeys {
		_ = viper.BindEnv(key)
	}
	for _, key := range optionalEnvKeys {
		_ = viper.BindEnv(key)
	}

	var missing []string
	for _, key := range requiredEnvKeys {
		if strings.TrimSpace(viper.GetString(key)) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return config, err
	}

	config.DatabaseURL = buildPostgresURL(config)
	config.RedisURL = buildRedisURL(config)
	config.MinioEndpoint = config.MinioHost + ":" + config.MinioPort

	return config, nil
}

func buildPostgresURL(c Config) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.PostgresUser, c.PostgresPassword),
		Host:   c.PostgresHost + ":" + c.PostgresPort,
		Path:   "/" + c.PostgresDB,
	}
	q := u.Query()
	q.Set("sslmode", c.PostgresSSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func buildRedisURL(c Config) string {
	u := &url.URL{
		Scheme: "redis",
		Host:   c.RedisHost + ":" + c.RedisPort,
		Path:   "/" + c.RedisDB,
	}
	return u.String()
}
