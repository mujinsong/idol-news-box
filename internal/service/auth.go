package service

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yuanhuaxi/weibo-spider/internal/dto"
	"github.com/yuanhuaxi/weibo-spider/internal/store"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("weibo-spider-secret-key")

// AuthService 认证服务
type AuthService struct {
	userStore *store.UserStore
}

// NewAuthService 创建认证服务
func NewAuthService(userStore *store.UserStore) *AuthService {
	return &AuthService{userStore: userStore}
}

// Login 登录
func (s *AuthService) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userStore.GetByUsername(req.Username)
	if err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	if user.Status != 1 {
		return nil, fmt.Errorf("用户已被禁用")
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("生成token失败: %w", err)
	}

	return &dto.LoginResponse{
		Token: token,
		User: &dto.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Nickname:  user.Nickname,
			WeiboUID:  user.WeiboUID,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}, nil
}

// generateToken 生成JWT token
func (s *AuthService) generateToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析JWT token
func ParseToken(tokenString string) (uint, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("无效的token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("无效的token")
	}

	userID := uint(claims["user_id"].(float64))
	return userID, nil
}
