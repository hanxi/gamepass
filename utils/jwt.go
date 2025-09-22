package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type JWTClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	TokenType string `json:"token_type"` // "access" or "refresh"
	jwt.StandardClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type JWTUtil struct {
	accessSecret  string
	refreshSecret string
	accessExp     time.Duration
	refreshExp    time.Duration
}

func NewJWTUtil(accessSecret, refreshSecret string, accessExp, refreshExp time.Duration) *JWTUtil {
	return &JWTUtil{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessExp:     accessExp,
		refreshExp:    refreshExp,
	}
}

func (j *JWTUtil) GetAccessSecret() string {
	return j.accessSecret
}

func (j *JWTUtil) GetRefreshSecret() string {
	return j.refreshSecret
}

// 生成 token pair
func (j *JWTUtil) GenerateTokenPair(userID, email string) (*TokenPair, error) {
	// 生成访问令牌
	accessToken, err := j.generateToken(userID, email, "access", j.accessSecret, j.accessExp)
	if err != nil {
		return nil, err
	}

	// 生成刷新令牌
	refreshToken, err := j.generateToken(userID, email, "refresh", j.refreshSecret, j.refreshExp)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (j *JWTUtil) generateToken(userID, email, tokenType, secret string, exp time.Duration) (string, error) {
	claims := JWTClaims{
		UserID:    userID,
		Email:     email,
		TokenType: tokenType,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(exp).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// 验证 token
func (j *JWTUtil) ValidateToken(tokenString, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
