package models

import (
	"time"

	"github.com/markbates/goth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name       string             `bson:"name" json:"name"`
	Email      string             `bson:"email" json:"email"`
	Provider   string             `bson:"provider" json:"provider"`
	ProviderID string             `bson:"provider_id" json:"provider_id"`
	AvatarURL  string             `bson:"avatar_url" json:"avatar_url"`
	GothUser   *goth.User         `bson:"goth_user" json:"goth_user"`
	Password   string             `bson:"password,omitempty" json:"-"` // 本地认证密码，不在JSON中暴露

	GameData bson.M `bson:"game_data" json:"game_data"` // 游戏数据

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

type UserView struct {
	ID         primitive.ObjectID `json:"id"`
	Name       string             `json:"name"`
	Email      string             `json:"email"`
	Provider   string             `json:"provider"`
	ProviderID string             `json:"provider_id"`
	AvatarURL  string             `json:"avatar_url"`
	GameData   bson.M             `json:"game_data,omitempty"` // 可选的游戏数据
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// User 结构体添加转换方法
func (u *User) ToView() *UserView {
	return &UserView{
		ID:         u.ID,
		Name:       u.Name,
		Email:      u.Email,
		Provider:   u.Provider,
		ProviderID: u.ProviderID,
		AvatarURL:  u.AvatarURL,
		GameData:   u.GameData,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}

// 添加更新游戏数据的方法
func (u *User) UpdateGameData(key string, value interface{}) {
	if u.GameData == nil {
		u.GameData = bson.M{}
	}
	u.GameData[key] = value
}
