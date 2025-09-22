package config

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	Port             string
	BaseURL          string
	MongoURI         string
	DBName           string
	GithubKey        string
	GithubSecret     string
	JWTAccessSecret  string
	JWTRefreshSecret string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	return &Config{
		Port:             os.Getenv("PORT"),
		BaseURL:          os.Getenv("BASE_URL"),
		MongoURI:         os.Getenv("MONGODB_URI"),
		DBName:           os.Getenv("DB_NAME"),
		GithubKey:        os.Getenv("GITHUB_KEY"),
		GithubSecret:     os.Getenv("GITHUB_SECRET"),
		JWTAccessSecret:  os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
	}, nil
}
