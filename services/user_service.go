package services

import (
	"context"
	"time"

	"github.com/markbates/goth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/hanxi/gamepass/models"
)

type UserService struct {
	db *mongo.Database
}

func NewUserService(db *mongo.Database) *UserService {
	return &UserService{db: db}
}

func (s *UserService) CreateOrUpdateUser(gothUser goth.User) (*models.User, error) {
	now := time.Now()

	// 首先尝试查找用户
	filter := bson.M{
		"provider":    gothUser.Provider,
		"provider_id": gothUser.UserID,
	}

	var existingUser models.User
	err := s.db.Collection("users").FindOne(context.Background(), filter).Decode(&existingUser)

	if err == mongo.ErrNoDocuments {
		// 用户不存在，创建新用户
		user := &models.User{
			Name:       gothUser.Name,
			Email:      gothUser.Email,
			Provider:   gothUser.Provider,
			ProviderID: gothUser.UserID,
			AvatarURL:  gothUser.AvatarURL,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		result, err := s.db.Collection("users").InsertOne(context.Background(), user)
		if err != nil {
			return nil, err
		}
		user.ID = result.InsertedID.(primitive.ObjectID)
		return user, nil
	} else if err != nil {
		return nil, err
	}

	// 用户存在，更新信息
	update := bson.M{
		"$set": bson.M{
			"name":       gothUser.Name,
			"email":      gothUser.Email,
			"avatar_url": gothUser.AvatarURL,
			"updated_at": now,
		},
	}

	_, err = s.db.Collection("users").UpdateOne(
		context.Background(),
		filter,
		update,
	)
	if err != nil {
		return nil, err
	}

	// 重新获取更新后的用户信息
	var updatedUser models.User
	err = s.db.Collection("users").FindOne(context.Background(), filter).Decode(&updatedUser)
	if err != nil {
		return nil, err
	}

	return &updatedUser, nil
}
