package handlers

import (
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"

	"github.com/hanxi/gamepass/services"
	"github.com/hanxi/gamepass/utils"
)

type AuthHandler struct {
	userService       *services.UserService
	tokenService      *services.TokenService
	jwtUtil           *utils.JWTUtil
	callbackWhitelist []*regexp.Regexp
}

func NewAuthHandler(us *services.UserService, ts *services.TokenService, ju *utils.JWTUtil, whitelist []*regexp.Regexp) *AuthHandler {
	return &AuthHandler{
		userService:       us,
		tokenService:      ts,
		jwtUtil:           ju,
		callbackWhitelist: whitelist,
	}
}

func (h *AuthHandler) BeginAuth(c *gin.Context) {
	// 获取回调URL参数
	callbackURL := c.Query("callback_url")
	// 如果提供了回调URL，进行安全和白名单验证
	if callbackURL != "" {
		isValidCallback := false
		for _, re := range h.callbackWhitelist {
			if re.MatchString(callbackURL) {
				isValidCallback = true
				break // 找到匹配项，无需继续检查
			}
		}

		if !isValidCallback {
			// 出于安全考虑，生产环境中的日志可能需要更谨慎，但开发时打印很有用。
			fmt.Printf("SECURITY ALERT: Callback URL validation failed. Provided URL: '%s'\n", callbackURL)
			c.JSON(400, gin.H{"error": "Provided client_callback_url is not allowed"})
			return
		}

		// 将你的数据存入cookie
		c.SetCookie(
			"auth_callback_url", // cookie 名称
			callbackURL,         // cookie 值
			3600,                // 过期时间（秒）
			"/",                 // 路径
			"",                  // 域名（留空使用当前域名）
			false,               // secure
			false,               // httpOnly
		)
	}

	provider := c.Param("provider")
	c.Request = gothic.GetContextWithProvider(c.Request, provider)
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func (h *AuthHandler) Callback(c *gin.Context) {
	// 从 cookie 中获取 callback_url
	hasCallbackURL := true
	callbackURL, err := c.Cookie("auth_callback_url")
	if err != nil || callbackURL == "" {
		hasCallbackURL = false
	}

	// 清除 cookie
	c.SetCookie("auth_callback_url", "", -1, "/", "", false, false)

	// 核心认证逻辑 (这部分是共用的)
	provider := c.Param("provider")
	c.Request = gothic.GetContextWithProvider(c.Request, provider)

	// 完成第三方认证
	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		// 如果认证失败，我们需要根据是否有重定向URL来决定如何返回错误
		if hasCallbackURL {
			h.redirectWithError(c, callbackURL, "authentication_failed", err.Error())
		} else {
			c.JSON(500, gin.H{"error": "Authentication failed", "details": err.Error()})
		}
		return
	}

	// 在我们自己的数据库中创建或更新用户
	dbUser, err := h.userService.CreateOrUpdateUser(user)
	if err != nil {
		if hasCallbackURL {
			h.redirectWithError(c, callbackURL, "user_service_error", err.Error())
		} else {
			c.JSON(500, gin.H{"error": "User service error", "details": err.Error()})
		}
		return
	}

	// 为我们的用户生成 Token Pair
	tokens, err := h.jwtUtil.GenerateTokenPair(dbUser.ID.Hex(), dbUser.Email)
	if err != nil {
		if hasCallbackURL {
			h.redirectWithError(c, callbackURL, "token_generation_failed", err.Error())
		} else {
			c.JSON(500, gin.H{"error": "Token generation failed", "details": err.Error()})
		}
		return
	}

	// 根据情况返回响应 (逻辑分叉)
	if hasCallbackURL {
		// 情况 A: 重定向到客户端
		redirectURL, parseErr := url.Parse(callbackURL)
		if parseErr != nil {
			// 这个错误比较严重，因为URL在BeginAuth时已经验证过
			c.JSON(500, gin.H{"error": "Internal server error: stored callback URL is invalid"})
			return
		}

		query := redirectURL.Query()
		query.Set("access_token", tokens.AccessToken)
		query.Set("refresh_token", tokens.RefreshToken)
		redirectURL.RawQuery = query.Encode()

		c.Redirect(302, redirectURL.String())
	} else {
		// 情况 B: 直接返回 JSON
		c.JSON(200, tokens)
	}
}

// redirectWithError 是一个辅助函数，用于将错误信息附加到重定向 URL 上
func (h *AuthHandler) redirectWithError(c *gin.Context, baseURL, errorCode, errorDescription string) {
	redirectURL, err := url.Parse(baseURL)
	if err != nil {
		// 如果基础URL都解析不了，只能返回JSON错误
		c.JSON(500, gin.H{
			"error":                      "Internal server error: invalid base redirect URL",
			"original_error_code":        errorCode,
			"original_error_description": errorDescription,
		})
		return
	}

	query := redirectURL.Query()
	query.Set("error", errorCode)
	query.Set("error_description", errorDescription)
	redirectURL.RawQuery = query.Encode()

	c.Redirect(302, redirectURL.String())
}

// 刷新 token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	refreshToken := c.GetHeader("Refresh-Token")
	if refreshToken == "" {
		c.JSON(400, gin.H{"error": "refresh token required"})
		return
	}

	// 验证刷新令牌
	claims, err := h.jwtUtil.ValidateToken(refreshToken, h.jwtUtil.GetRefreshSecret())
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid refresh token"})
		return
	}

	// 检查令牌是否被加入黑名单
	if h.tokenService.IsTokenBlacklisted(refreshToken) {
		c.JSON(401, gin.H{"error": "token has been revoked"})
		return
	}

	// 生成新的 token pair
	newTokens, err := h.jwtUtil.GenerateTokenPair(claims.UserID, claims.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 将旧的刷新令牌加入黑名单
	if err := h.tokenService.BlacklistToken(refreshToken, time.Unix(claims.ExpiresAt, 0)); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, newTokens)
}

// 注销
func (h *AuthHandler) Logout(c *gin.Context) {
	accessToken := c.GetHeader("Authorization")
	refreshToken := c.GetHeader("Refresh-Token")

	if accessToken != "" {
		claims, err := h.jwtUtil.ValidateToken(accessToken, h.jwtUtil.GetAccessSecret())
		if err == nil {
			h.tokenService.BlacklistToken(accessToken, time.Unix(claims.ExpiresAt, 0))
		}
	}

	if refreshToken != "" {
		claims, err := h.jwtUtil.ValidateToken(refreshToken, h.jwtUtil.GetRefreshSecret())
		if err == nil {
			h.tokenService.BlacklistToken(refreshToken, time.Unix(claims.ExpiresAt, 0))
		}
	}

	c.JSON(200, gin.H{"message": "logged out successfully"})
}

func (h *AuthHandler) GetProviders(c *gin.Context) {
	providers := goth.GetProviders()
	providerList := make([]map[string]interface{}, 0)

	for providerName, _ := range providers {
		providerInfo := map[string]interface{}{
			"name":     providerName,
			"auth_url": fmt.Sprintf("/auth/%s", providerName),
		}
		providerList = append(providerList, providerInfo)
	}

	c.JSON(200, gin.H{
		"providers": providerList,
		"total":     len(providers),
	})
}
