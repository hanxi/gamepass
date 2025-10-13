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

const (
	// Cookie 过期时间常量
	CookieExpireTime = 3600 // 1小时

	// Cookie 名称常量
	AuthCallbackURLCookie = "auth_callback_url"
	AuthSourceCookie      = "auth_source"
	AccessTokenCookie     = "access_token"
	RefreshTokenCookie    = "refresh_token"
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

// 辅助方法：验证回调URL是否在白名单中
func (h *AuthHandler) validateCallbackURL(callbackURL string) bool {
	if callbackURL == "" {
		return true // 空URL是允许的
	}

	for _, re := range h.callbackWhitelist {
		if re.MatchString(callbackURL) {
			return true
		}
	}
	return false
}

// 辅助方法：处理回调URL验证和存储
func (h *AuthHandler) handleCallbackURL(c *gin.Context, callbackURL string) error {
	if callbackURL == "" {
		return nil
	}

	if !h.validateCallbackURL(callbackURL) {
		fmt.Printf("SECURITY ALERT: Callback URL validation failed. Provided URL: '%s'\n", callbackURL)
		c.JSON(400, gin.H{"error": "Provided client_callback_url is not allowed"})
		return fmt.Errorf("invalid callback URL")
	}

	// 将回调URL存入cookie
	c.SetCookie(AuthCallbackURLCookie, callbackURL, CookieExpireTime, "/", "", false, false)
	return nil
}

// 辅助方法：检查用户是否已登录（从cookie获取token）
func (h *AuthHandler) checkUserLoggedIn(c *gin.Context) (*utils.TokenPair, error) {
	accessToken, err := c.Cookie(AccessTokenCookie)
	if err != nil || accessToken == "" {
		return nil, fmt.Errorf("no access token")
	}

	// 检查黑名单
	if h.tokenService.IsTokenBlacklisted(accessToken) {
		return nil, fmt.Errorf("token blacklisted")
	}

	// 验证token
	claims, err := h.jwtUtil.ValidateToken(accessToken, h.jwtUtil.GetAccessSecret())
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	// 生成新的token pair
	newTokens, err := h.jwtUtil.GenerateTokenPair(claims.UserID, claims.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new tokens")
	}

	return newTokens, nil
}

func (h *AuthHandler) BeginAuth(c *gin.Context) {
	provider := c.Param("provider")

	// 处理本地认证
	if provider == "local" {
		// 处理回调URL
		callbackURL := c.Query("callback_url")
		if err := h.handleCallbackURL(c, callbackURL); err != nil {
			return // 错误已在handleCallbackURL中处理
		}

		// 检查用户是否已经登录
		if newTokens, err := h.checkUserLoggedIn(c); err == nil {
			// 用户已登录，直接跳转到callback
			callbackUrl := fmt.Sprintf("/auth/local/callback?access_token=%s&refresh_token=%s",
				url.QueryEscape(newTokens.AccessToken),
				url.QueryEscape(newTokens.RefreshToken))
			c.Redirect(302, callbackUrl)
			return
		}

		// 设置标记表示来自 /auth/local
		c.SetCookie(AuthSourceCookie, "local", CookieExpireTime, "/", "", false, false)

		// 跳转到登录页面
		c.Redirect(302, "/login")
		return
	}

	// 处理第三方OAuth认证
	callbackURL := c.Query("callback_url")
	if err := h.handleCallbackURL(c, callbackURL); err != nil {
		return // 错误已在handleCallbackURL中处理
	}

	c.Request = gothic.GetContextWithProvider(c.Request, provider)
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func (h *AuthHandler) Callback(c *gin.Context) {
	provider := c.Param("provider")

	// 处理本地认证回调
	if provider == "local" {
		h.handleLocalCallback(c)
		return
	}

	// 处理第三方OAuth认证回调
	h.handleOAuthCallback(c, provider)
}

// 处理本地认证回调
func (h *AuthHandler) handleLocalCallback(c *gin.Context) {
	// 从 cookie 中获取 callback_url
	hasCallbackURL := true
	callbackURL, err := c.Cookie(AuthCallbackURLCookie)
	if err != nil || callbackURL == "" {
		hasCallbackURL = false
	}

	// 清除 callback URL cookie
	c.SetCookie(AuthCallbackURLCookie, "", -1, "/", "", false, false)

	// 检查是否有认证token（从URL参数获取）
	accessToken := c.Query("access_token")
	refreshToken := c.Query("refresh_token")

	if accessToken == "" || refreshToken == "" {
		// 没有token，重定向到登录页面
		if hasCallbackURL {
			h.redirectWithError(c, callbackURL, "authentication_required", "Please login first")
		} else {
			c.Redirect(302, "/login")
		}
		return
	}

	// 有token，处理成功回调
	if hasCallbackURL {
		redirectURL, parseErr := url.Parse(callbackURL)
		if parseErr != nil {
			c.JSON(500, gin.H{"error": "Internal server error: stored callback URL is invalid"})
			return
		}

		query := redirectURL.Query()
		query.Set("access_token", accessToken)
		query.Set("refresh_token", refreshToken)
		redirectURL.RawQuery = query.Encode()

		c.Redirect(302, redirectURL.String())
	} else {
		// 返回JSON格式的token
		c.JSON(200, gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

// 处理第三方OAuth认证回调
func (h *AuthHandler) handleOAuthCallback(c *gin.Context, provider string) {
	// 从 cookie 中获取 callback_url
	hasCallbackURL := true
	callbackURL, err := c.Cookie(AuthCallbackURLCookie)
	if err != nil || callbackURL == "" {
		hasCallbackURL = false
	}

	// 清除 callback URL cookie
	c.SetCookie(AuthCallbackURLCookie, "", -1, "/", "", false, false)

	// 核心认证逻辑
	c.Request = gothic.GetContextWithProvider(c.Request, provider)

	// 完成第三方认证
	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		if hasCallbackURL {
			h.redirectWithError(c, callbackURL, "authentication_failed", err.Error())
		} else {
			c.JSON(500, gin.H{"error": "Authentication failed", "details": err.Error()})
		}
		return
	}

	// 在数据库中创建或更新用户
	dbUser, err := h.userService.CreateOrUpdateUser(user)
	if err != nil {
		if hasCallbackURL {
			h.redirectWithError(c, callbackURL, "user_service_error", err.Error())
		} else {
			c.JSON(500, gin.H{"error": "User service error", "details": err.Error()})
		}
		return
	}

	// 生成 Token Pair
	tokens, err := h.jwtUtil.GenerateTokenPair(dbUser.ID.Hex(), dbUser.Email)
	if err != nil {
		if hasCallbackURL {
			h.redirectWithError(c, callbackURL, "token_generation_failed", err.Error())
		} else {
			c.JSON(500, gin.H{"error": "Token generation failed", "details": err.Error()})
		}
		return
	}

	// 根据情况返回响应
	if hasCallbackURL {
		// 重定向到客户端
		redirectURL, parseErr := url.Parse(callbackURL)
		if parseErr != nil {
			c.JSON(500, gin.H{"error": "Internal server error: stored callback URL is invalid"})
			return
		}

		query := redirectURL.Query()
		query.Set("access_token", tokens.AccessToken)
		query.Set("refresh_token", tokens.RefreshToken)
		redirectURL.RawQuery = query.Encode()

		c.Redirect(302, redirectURL.String())
	} else {
		// 直接返回 JSON
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

// 本地用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required"`
		CallbackURL string `json:"callback_url,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// 验证回调URL（如果提供）
	hasCallbackURL := req.CallbackURL != ""
	if hasCallbackURL {
		isValidCallback := false
		for _, re := range h.callbackWhitelist {
			if re.MatchString(req.CallbackURL) {
				isValidCallback = true
				break
			}
		}

		if !isValidCallback {
			fmt.Printf("SECURITY ALERT: Callback URL validation failed. Provided URL: '%s'\n", req.CallbackURL)
			c.JSON(400, gin.H{"error": "Provided callback_url is not allowed"})
			return
		}
	}

	// 创建本地用户
	user, err := h.userService.CreateLocalUser(req.Name, req.Email, req.Password)
	if err != nil {
		if hasCallbackURL {
			h.redirectWithError(c, req.CallbackURL, "registration_failed", err.Error())
		} else {
			c.JSON(400, gin.H{"error": "Registration failed", "details": err.Error()})
		}
		return
	}

	// 生成 Token Pair
	tokens, err := h.jwtUtil.GenerateTokenPair(user.ID.Hex(), user.Email)
	if err != nil {
		if hasCallbackURL {
			h.redirectWithError(c, req.CallbackURL, "token_generation_failed", err.Error())
		} else {
			c.JSON(500, gin.H{"error": "Token generation failed", "details": err.Error()})
		}
		return
	}

	// 根据情况返回响应
	if hasCallbackURL {
		redirectURL, parseErr := url.Parse(req.CallbackURL)
		if parseErr != nil {
			c.JSON(500, gin.H{"error": "Internal server error: invalid callback URL"})
			return
		}

		query := redirectURL.Query()
		query.Set("access_token", tokens.AccessToken)
		query.Set("refresh_token", tokens.RefreshToken)
		redirectURL.RawQuery = query.Encode()

		c.Redirect(302, redirectURL.String())
	} else {
		c.JSON(201, tokens)
	}
}

// 本地用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required"`
		CallbackURL string `json:"callback_url,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// 验证回调URL（如果提供）
	hasCallbackURL := req.CallbackURL != ""
	if hasCallbackURL {
		isValidCallback := false
		for _, re := range h.callbackWhitelist {
			if re.MatchString(req.CallbackURL) {
				isValidCallback = true
				break
			}
		}

		if !isValidCallback {
			fmt.Printf("SECURITY ALERT: Callback URL validation failed. Provided URL: '%s'\n", req.CallbackURL)
			c.JSON(400, gin.H{"error": "Provided callback_url is not allowed"})
			return
		}
	}

	// 验证用户凭据
	user, err := h.userService.AuthenticateLocalUser(req.Email, req.Password)
	if err != nil {
		if hasCallbackURL {
			h.redirectWithError(c, req.CallbackURL, "authentication_failed", err.Error())
		} else {
			c.JSON(401, gin.H{"error": "Authentication failed", "details": err.Error()})
		}
		return
	}

	// 生成 Token Pair
	tokens, err := h.jwtUtil.GenerateTokenPair(user.ID.Hex(), user.Email)
	if err != nil {
		if hasCallbackURL {
			h.redirectWithError(c, req.CallbackURL, "token_generation_failed", err.Error())
		} else {
			c.JSON(500, gin.H{"error": "Token generation failed", "details": err.Error()})
		}
		return
	}

	// 根据情况返回响应
	if hasCallbackURL {
		redirectURL, parseErr := url.Parse(req.CallbackURL)
		if parseErr != nil {
			c.JSON(500, gin.H{"error": "Internal server error: invalid callback URL"})
			return
		}

		query := redirectURL.Query()
		query.Set("access_token", tokens.AccessToken)
		query.Set("refresh_token", tokens.RefreshToken)
		redirectURL.RawQuery = query.Encode()

		c.Redirect(302, redirectURL.String())
	} else {
		c.JSON(200, tokens)
	}
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

// ShowLogoutPage 显示登出页面
func (h *AuthHandler) ShowLogoutPage(c *gin.Context) {
	c.HTML(200, "logout.html", gin.H{
		"title": "登出成功 - GamePass",
	})
}

// 显示登录页面
func (h *AuthHandler) ShowLoginPage(c *gin.Context) {
	c.HTML(200, "login.html", gin.H{
		"title": "用户登录",
	})
}

// 显示注册页面
func (h *AuthHandler) ShowRegisterPage(c *gin.Context) {
	c.HTML(200, "register.html", gin.H{
		"title": "用户注册",
	})
}
