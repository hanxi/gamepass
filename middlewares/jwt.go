package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hanxi/gamepass/services"
	"github.com/hanxi/gamepass/utils"
)

func JWTAuth(jwtUtil *utils.JWTUtil, tokenService *services.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		// 提取 token
		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 {
			c.JSON(401, gin.H{"error": "invalid token format"})
			c.Abort()
			return
		}

		token := bearerToken[1]

		// 检查黑名单
		if tokenService.IsTokenBlacklisted(token) {
			c.JSON(401, gin.H{"error": "token has been revoked"})
			c.Abort()
			return
		}

		// 验证 token
		claims, err := jwtUtil.ValidateToken(token, jwtUtil.GetAccessSecret())
		if err != nil {
			c.JSON(401, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// 将用户信息存储在上下文中
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		c.Next()
	}
}
