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
	Account   string `json:"account"`
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
	issuer        string
	audience      string
}

func NewJWTUtil(accessSecret, refreshSecret string, accessExp, refreshExp time.Duration, issuer, audience string) *JWTUtil {
	return &JWTUtil{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessExp:     accessExp,
		refreshExp:    refreshExp,
		issuer:        issuer,
		audience:      audience,
	}
}

func (j *JWTUtil) GetAccessSecret() string {
	return j.accessSecret
}

func (j *JWTUtil) GetRefreshSecret() string {
	return j.refreshSecret
}

// 生成 token pair
func (j *JWTUtil) GenerateTokenPair(account, email string) (*TokenPair, error) {
	// 生成访问令牌
	accessToken, err := j.generateToken(account, email, "access", j.accessSecret, j.accessExp)
	if err != nil {
		return nil, err
	}

	// 生成刷新令牌
	refreshToken, err := j.generateToken(account, email, "refresh", j.refreshSecret, j.refreshExp)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (j *JWTUtil) generateToken(account, email, tokenType, secret string, exp time.Duration) (string, error) {
	claims := JWTClaims{
		Account:   account,
		Email:     email,
		TokenType: tokenType,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(exp).Unix(),
			IssuedAt:  time.Now().Unix(),
			// 下游服务（如 skyext 登录）会校验签发方与接收方；未配置时保持省略
			Issuer:   j.issuer,
			Audience: j.audience,
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
