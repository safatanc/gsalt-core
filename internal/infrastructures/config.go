package infrastructures

import (
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	DATABASE_URL     string
	CONNECT_BASE_URL string
	FlipConfig       *FlipConfig
}

var Config *AppConfig

func LoadConfig() *AppConfig {
	godotenv.Load()

	Config = &AppConfig{
		DATABASE_URL:     os.Getenv("DATABASE_URL"),
		CONNECT_BASE_URL: os.Getenv("CONNECT_BASE_URL"),
		FlipConfig: &FlipConfig{
			SecretKey:          os.Getenv("FLIP_SECRET_KEY"),
			WebhookToken:       os.Getenv("FLIP_WEBHOOK_TOKEN"),
			CallbackURL:        os.Getenv("FLIP_CALLBACK_URL"),
			SuccessURL:         os.Getenv("FLIP_SUCCESS_URL"),
			FailureURL:         os.Getenv("FLIP_FAILURE_URL"),
			Environment:        os.Getenv("FLIP_ENVIRONMENT"),
			DefaultRedirectURL: os.Getenv("FLIP_DEFAULT_REDIRECT_URL"),
		},
	}

	return Config
}
