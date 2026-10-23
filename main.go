package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
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

	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	store.MaxAge(86400 * 30)
	store.Options.Path = "/"
	store.Options.HttpOnly = false // HttpOnly should always be enabled
	store.Options.Secure = false   // cfg.IsProd
	store.Options.SameSite = http.SameSiteLaxMode
	gothic.Store = store

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
		cfg.JWTIssuer,
		cfg.JWTAudience,
	)

	userService := services.NewUserService(db)
	tokenService := services.NewTokenService(db)
	authHandler := handlers.NewAuthHandler(userService, tokenService, jwtUtil, cfg.CallbackWhitelist)
	profileHandler := handlers.NewProfileHandler(userService)

	// 配置 Goth
	if cfg.GithubKey != "" {
		goth.UseProviders(
			github.New(
				cfg.GithubKey,
				cfg.GithubSecret,
				handlers.ProviderCallbackURL(cfg.BaseURL, "github"),
			),
		)
	}
	if cfg.OpenidConnectKey != "" {
		oidc, err := newOIDCProvider(cfg.BaseURL, cfg.OpenidConnectKey, cfg.OpenidConnectSecret, cfg.OpenidConnectDiscoveryUrl)
		if err != nil {
			log.Fatal(err)
		}
		goth.UseProviders(oidc)
	}

	// 设置 Gin
	r := gin.Default()

	// 配置HTML模板
	r.LoadHTMLGlob("templates/*")

	// 配置静态文件服务
	r.Static("/static", "./static")

	// 页面路由
	r.GET("/login", authHandler.ShowLoginPage)
	r.GET("/register", authHandler.ShowRegisterPage)

	// 认证路由
	auth := r.Group("/auth")
	{
		auth.GET("/:provider", authHandler.BeginAuth)
		auth.GET("/:provider/callback", authHandler.Callback)
		auth.POST("/refresh", authHandler.RefreshToken)

		auth.POST("/local/login", authHandler.Login)
		auth.POST("/local/register", authHandler.Register)

		auth.POST("/logout", authHandler.Logout)
		auth.GET("/logout", authHandler.ShowLogoutPage)
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

// newOIDCProvider 创建 OpenID Connect provider。goth 的 NewNamed 会给传入的名字自动加
// -oidc 后缀，这里不依赖这条规则，构造后显式 SetName，保证注册名、登录入口
// /auth/<名> 与回调 /auth/<名>/callback 三处一致
func newOIDCProvider(baseURL, key, secret, discoveryURL string) (*openidConnect.Provider, error) {
	p, err := openidConnect.NewNamed(
		"",
		key,
		secret,
		handlers.ProviderCallbackURL(baseURL, handlers.OIDCProvider),
		discoveryURL,
	)
	if err != nil {
		return nil, err
	}
	p.SetName(handlers.OIDCProvider)
	return p, nil
}
