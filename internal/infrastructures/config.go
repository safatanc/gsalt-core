package infrastructures

import (
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	DATABASE_URL     string
	CONNECT_BASE_URL string
}

var Config *AppConfig

func LoadConfig() *AppConfig {
	godotenv.Load()

	Config = &AppConfig{
		DATABASE_URL:     os.Getenv("DATABASE_URL"),
		CONNECT_BASE_URL: os.Getenv("CONNECT_BASE_URL"),
	}

	return Config
}
