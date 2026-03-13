package config

import (
	"os"
	"fmt"
	"github.com/joho/godotenv"
)

type Config struct {
	TGToken string
	DBHost string
	DBPort string
	DBUser string
	DBPassword string
	DBName string		
}

func Load()(*Config, error) {
	if err :=godotenv.Load(); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}
	
	cfg := &Config {
		TGToken: os.Getenv("TG_TOKEN"),
		DBHost: os.Getenv("DB_HOST"),
		DBPort: os.Getenv("DB_PORT"),
		DBUser: os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName: os.Getenv("DB_NAME"),
	}

	if cfg.TGToken== "" {
		return nil, fmt.Errorf("TG_TOKEN is required")
	}
	if cfg.DBHost== "" {
		return nil, fmt.Errorf("DB_HOST is required")
	}
	if cfg.DBName== "" {
		return nil, fmt.Errorf("DB_NAME is required")
	}
	return cfg, nil
}
