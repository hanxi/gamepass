package config

import (
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                      string
	BaseURL                   string
	SessionSecret             string
	IsProd                    bool
	CallbackWhitelist         []*regexp.Regexp
	MongoURI                  string
	DBName                    string
	JWTAccessSecret           string
	JWTRefreshSecret          string
	JWTIssuer                 string
	JWTAudience               string
	GithubKey                 string
	GithubSecret              string
	OpenidConnectKey          string
	OpenidConnectSecret       string
	OpenidConnectDiscoveryUrl string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	callbackWhitelist := os.Getenv("CALLBACK_WHITELIST")
	patterns := strings.Split(callbackWhitelist, ",")
	var compiledWhitelist []*regexp.Regexp
	for _, p := range patterns {
		// Trim space in case of "pattern1, pattern2"
		trimmedPattern := strings.TrimSpace(p)
		if trimmedPattern == "" {
			continue
		}
		re, err := regexp.Compile(trimmedPattern)
		if err != nil {
			// 如果配置中的正则表达式无法编译，这是一个致命错误，服务器不应该启动。
			log.Fatalf("Failed to compile whitelist regex '%s': %v", trimmedPattern, err)
		}
		compiledWhitelist = append(compiledWhitelist, re)
	}
	isProd := false
	if os.Getenv("PROD") == "true" {
		isProd = true
	}

	return &Config{
		Port:                      os.Getenv("PORT"),
		BaseURL:                   os.Getenv("BASE_URL"),
		IsProd:                    isProd,
		SessionSecret:             os.Getenv("SESSION_SECRET"),
		CallbackWhitelist:         compiledWhitelist,
		MongoURI:                  os.Getenv("MONGODB_URI"),
		DBName:                    os.Getenv("DB_NAME"),
		JWTAccessSecret:           os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:          os.Getenv("JWT_REFRESH_SECRET"),
		JWTIssuer:                 os.Getenv("JWT_ISSUER"),
		JWTAudience:               os.Getenv("JWT_AUDIENCE"),
		GithubKey:                 os.Getenv("GITHUB_KEY"),
		GithubSecret:              os.Getenv("GITHUB_SECRET"),
		OpenidConnectKey:          os.Getenv("OPENID_CONNECT_KEY"),
		OpenidConnectSecret:       os.Getenv("OPENID_CONNECT_SECRET"),
		OpenidConnectDiscoveryUrl: os.Getenv("OPENID_CONNECT_DISCOVERY_URL"),
	}, nil
}
