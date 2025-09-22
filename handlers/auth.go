package handlers

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"

	"github.com/hanxi/gamepass/services"
	"github.com/hanxi/gamepass/utils"
)

type AuthHandler struct {
	userService  *services.UserService
	tokenService *services.TokenService
	jwtUtil      *utils.JWTUtil
}

func NewAuthHandler(us *services.UserService, ts *services.TokenService, ju *utils.JWTUtil) *AuthHandler {
	return &AuthHandler{
		userService:  us,
		tokenService: ts,
		jwtUtil:      ju,
	}
}

func (h *AuthHandler) BeginAuth(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = gothic.GetContextWithProvider(c.Request, provider)
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func (h *AuthHandler) Callback(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = gothic.GetContextWithProvider(c.Request, provider)
	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	dbUser, err := h.userService.CreateOrUpdateUser(user)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	tokens, err := h.jwtUtil.GenerateTokenPair(dbUser.ID.Hex(), dbUser.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, tokens)
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
