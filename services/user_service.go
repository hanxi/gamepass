package services

import (
	"context"
	"errors"
	"time"

	"github.com/markbates/goth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/hanxi/gamepass/models"
	"github.com/hanxi/gamepass/utils"
)

type UserService struct {
	db           *mongo.Database
	passwordUtil *utils.PasswordUtil
}

func NewUserService(db *mongo.Database) *UserService {
	return &UserService{
		db:           db,
		passwordUtil: utils.NewPasswordUtil(),
	}
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

// CreateLocalUser 创建本地用户
func (s *UserService) CreateLocalUser(name, email, password string) (*models.User, error) {
	// 检查邮箱是否已存在
	existingUser, err := s.GetUserByEmail(email)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("email already exists")
	}

	// 验证密码强度
	if !s.passwordUtil.IsValidPassword(password) {
		return nil, errors.New("password must be at least 6 characters long")
	}

	// 哈希密码
	hashedPassword, err := s.passwordUtil.HashPassword(password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &models.User{
		Name:      name,
		Email:     email,
		Provider:  "local",
		Password:  hashedPassword,
		CreatedAt: now,
		UpdatedAt: now,
	}

	result, err := s.db.Collection("users").InsertOne(context.Background(), user)
	if err != nil {
		return nil, err
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	return user, nil
}

// AuthenticateLocalUser 验证本地用户登录
func (s *UserService) AuthenticateLocalUser(email, password string) (*models.User, error) {
	user, err := s.GetUserByEmail(email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("invalid email or password")
		}
		return nil, err
	}

	// 检查是否为本地用户
	if user.Provider != "local" {
		return nil, errors.New("this email is registered with a different provider")
	}

	// 验证密码
	if err := s.passwordUtil.CheckPassword(user.Password, password); err != nil {
		return nil, errors.New("invalid email or password")
	}

	return user, nil
}

// GetUserByEmail 通过邮箱获取用户
func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := s.db.Collection("users").FindOne(context.Background(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
