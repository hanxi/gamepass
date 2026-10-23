package config

import (
	"fmt"
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

	compiledWhitelist, err := ParseCallbackWhitelist(os.Getenv("CALLBACK_WHITELIST"))
	if err != nil {
		// 如果配置中的正则表达式无法编译，这是一个致命错误，服务器不应该启动。
		log.Fatal(err)
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

// ParseCallbackWhitelist 解析 CALLBACK_WHITELIST：逗号分隔的正则列表，每项两端空白忽略、空项跳过。
func ParseCallbackWhitelist(raw string) ([]*regexp.Regexp, error) {
	var compiled []*regexp.Regexp
	for _, p := range strings.Split(raw, ",") {
		pattern := strings.TrimSpace(p)
		if pattern == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile whitelist regex '%s': %w", pattern, err)
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}
