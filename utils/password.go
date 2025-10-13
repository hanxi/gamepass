package utils

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidPassword = errors.New("invalid password")
)

type PasswordUtil struct {
	cost int
}

func NewPasswordUtil() *PasswordUtil {
	return &PasswordUtil{
		cost: bcrypt.DefaultCost,
	}
}

// HashPassword 对密码进行哈希加密
func (p *PasswordUtil) HashPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", errors.New("password cannot be empty")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), p.cost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

// CheckPassword 验证密码是否正确
func (p *PasswordUtil) CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// IsValidPassword 检查密码强度（可根据需要调整规则）
func (p *PasswordUtil) IsValidPassword(password string) bool {
	// 基本密码强度检查：至少6位
	return len(password) >= 6
}