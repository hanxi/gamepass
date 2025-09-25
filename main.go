package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/openidConnect"

	"github.com/hanxi/gamepass/config"
	"github.com/hanxi/gamepass/db"
	"github.com/hanxi/gamepass/handlers"
	"github.com/hanxi/gamepass/middlewares"
	"github.com/hanxi/gamepass/services"
	"github.com/hanxi/gamepass/utils"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// 初始化数据库
	mongodb, err := db.NewMongoDB(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatal(err)
	}
	defer mongodb.Close(context.Background())

	// 获取数据库实例
	db := mongodb.GetDB()

	// 初始化服务
	jwtUtil := utils.NewJWTUtil(
		cfg.JWTAccessSecret,
		cfg.JWTRefreshSecret,
		15*time.Minute,
		7*24*time.Hour,
	)

	userService := services.NewUserService(db)
	tokenService := services.NewTokenService(db)
	authHandler := handlers.NewAuthHandler(userService, tokenService, jwtUtil)
	profileHandler := handlers.NewProfileHandler(userService)

	// 配置 Goth
	goth.UseProviders(
		github.New(
			cfg.GithubKey,
			cfg.GithubSecret,
			fmt.Sprintf("%s/auth/github/callback", cfg.BaseURL),
		),
	)
	openidConnect, err := openidConnect.NewNamed(
		"local",
		os.Getenv("OPENID_CONNECT_KEY"),
		os.Getenv("OPENID_CONNECT_SECRET"),
		fmt.Sprintf("%s/auth/local-oidc/callback", cfg.BaseURL),
		os.Getenv("OPENID_CONNECT_DISCOVERY_URL"),
	)
	if openidConnect == nil {
		log.Fatal(err)
	}
	goth.UseProviders(openidConnect)

	// 设置 Gin
	r := gin.Default()

	// 认证路由
	auth := r.Group("/auth")
	{
		auth.GET("/:provider", authHandler.BeginAuth)
		auth.GET("/:provider/callback", authHandler.Callback)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/logout", authHandler.Logout)
	}

	// 公开路由组
	public := r.Group("/api")
	{
		public.GET("/providers", authHandler.GetProviders)
	}

	// API 路由
	api := r.Group("/api")
	api.Use(middlewares.JWTAuth(jwtUtil, tokenService))
	{
		api.GET("/profile", profileHandler.GetProfile)
		// 游戏数据相关端点
		api.PUT("/game-data", profileHandler.UpdateGameData)
		api.PUT("/game-data/:field", profileHandler.UpdateGameDataField)
	}

	// 启动服务器
	addr := fmt.Sprintf(":%s", cfg.Port)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
