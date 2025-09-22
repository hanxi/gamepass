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

// services/user_service.go
func (s *UserService) CreateOrUpdateUser(gothUser goth.User) (*models.User, error) {
	now := time.Now()

	// 复制 gothUser，避免直接使用指针
	gothUserCopy := gothUser

	user := &models.User{
		Name:       gothUser.Name,
		Email:      gothUser.Email,
		Provider:   gothUser.Provider,
		ProviderID: gothUser.UserID,
		AvatarURL:  gothUser.AvatarURL,
		GothUser:   &gothUserCopy, // 存储完整的 goth.User 对象
		UpdatedAt:  now,
	}

	filter := bson.M{
		"provider":    gothUser.Provider,
		"provider_id": gothUser.UserID,
	}

	// 查找现有用户
	var existingUser models.User
	err := s.db.Collection("users").FindOne(context.Background(), filter).Decode(&existingUser)

	if err == mongo.ErrNoDocuments {
		// 新用户
		user.CreatedAt = now
		result, err := s.db.Collection("users").InsertOne(context.Background(), user)
		if err != nil {
			return nil, err
		}
		user.ID = result.InsertedID.(primitive.ObjectID)
		return user, nil
	} else if err != nil {
		return nil, err
	}

	// 更新现有用户
	user.ID = existingUser.ID
	user.CreatedAt = existingUser.CreatedAt

	update := bson.M{
		"$set": user,
	}

	_, err = s.db.Collection("users").UpdateOne(
		context.Background(),
		filter,
		update,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUserByID(id string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = s.db.Collection("users").FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// 更新游戏数据
func (s *UserService) UpdateGameData(userID string, updates bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"game_data":  updates,
			"updated_at": time.Now(),
		},
	}

	_, err = s.db.Collection("users").UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		update,
	)
	return err
}

// 更新特定的游戏数据字段
func (s *UserService) UpdateGameDataField(userID string, key string, value interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"game_data." + key: value,
			"updated_at":       time.Now(),
		},
	}

	_, err = s.db.Collection("users").UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		update,
	)
	return err
}
