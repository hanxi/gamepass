package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/hanxi/gamepass/services"
	"go.mongodb.org/mongo-driver/bson"
)

type ProfileHandler struct {
	userService *services.UserService
}

func NewProfileHandler(us *services.UserService) *ProfileHandler {
	return &ProfileHandler{
		userService: us,
	}
}

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	// 从上下文获取用户ID（由JWT中间件设置）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	// 获取用户信息
	user, err := h.userService.GetUserByID(userID.(string))
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get user profile"})
		return
	}

	if user == nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}

	// 转换为视图模型并返回
	c.JSON(200, user.ToView())
}

// 更新游戏数据
func (h *ProfileHandler) UpdateGameData(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var updates bson.M
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	if err := h.userService.UpdateGameData(userID.(string), updates); err != nil {
		c.JSON(500, gin.H{"error": "failed to update game data"})
		return
	}

	// 获取更新后的用户信息
	user, err := h.userService.GetUserByID(userID.(string))
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get updated user data"})
		return
	}

	c.JSON(200, user.ToView())
}

// 更新单个游戏数据字段
func (h *ProfileHandler) UpdateGameDataField(c *gin.Context) {
	userID, _ := c.Get("user_id")
	field := c.Param("field")

	var value interface{}
	if err := c.ShouldBindJSON(&value); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	if err := h.userService.UpdateGameDataField(userID.(string), field, value); err != nil {
		c.JSON(500, gin.H{"error": "failed to update game data field"})
		return
	}

	c.JSON(200, gin.H{"message": "updated successfully"})
}
