package services

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TokenBlacklist struct {
	Token     string    `bson:"token"`
	ExpiresAt time.Time `bson:"expires_at"`
}

type TokenService struct {
	db *mongo.Database
}

func NewTokenService(db *mongo.Database) *TokenService {
	return &TokenService{db: db}
}

// 将 token 加入黑名单
func (s *TokenService) BlacklistToken(token string, expiresAt time.Time) error {
	_, err := s.db.Collection("token_blacklist").InsertOne(context.Background(), TokenBlacklist{
		Token:     token,
		ExpiresAt: expiresAt,
	})
	return err
}

// 检查 token 是否在黑名单中
func (s *TokenService) IsTokenBlacklisted(token string) bool {
	result := s.db.Collection("token_blacklist").FindOne(context.Background(), bson.M{"token": token})
	return result.Err() == nil
}

// 清理过期的黑名单 token
func (s *TokenService) CleanupExpiredTokens() error {
	_, err := s.db.Collection("token_blacklist").DeleteMany(
		context.Background(),
		bson.M{"expires_at": bson.M{"$lt": time.Now()}},
	)
	return err
}
