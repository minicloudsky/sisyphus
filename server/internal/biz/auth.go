package biz

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthUsecase 认证用例
type AuthUsecase struct {
	userRepo UserRepo
	secret   string
	expire   int64
}

// NewAuthUsecase 创建认证用例
func NewAuthUsecase(userRepo UserRepo, secret string, expire int64) *AuthUsecase {
	return &AuthUsecase{
		userRepo: userRepo,
		secret:   secret,
		expire:   expire,
	}
}

// Register 用户注册
func (uc *AuthUsecase) Register(ctx context.Context, username, password, email, realName string) (*User, string, error) {
	// 检查用户名是否已存在
	_, err := uc.userRepo.GetByUsername(ctx, username)
	if err == nil {
		return nil, "", ErrUserExists
	}

	// 检查邮箱是否已存在
	_, err = uc.userRepo.GetByEmail(ctx, email)
	if err == nil {
		return nil, "", ErrEmailExists
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	// 创建用户
	user := &User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
		RealName: realName,
		Status:   1,
	}

	user, err = uc.userRepo.Create(ctx, user)
	if err != nil {
		return nil, "", err
	}

	// 生成token
	token, err := uc.generateToken(user.ID, username)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Login 用户登录
func (uc *AuthUsecase) Login(ctx context.Context, username, password string) (*User, string, error) {
	// 获取用户
	user, err := uc.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, "", ErrInvalidAuth
	}

	// 检查用户状态
	if user.Status != 1 {
		return nil, "", errors.New("用户已被禁用")
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, "", ErrInvalidAuth
	}

	// 生成token
	token, err := uc.generateToken(user.ID, user.Username)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// RefreshToken 刷新token：验证旧token并生成新token（重置过期时间）
func (uc *AuthUsecase) RefreshToken(tokenString string) (string, error) {
	claims, err := uc.ValidateToken(tokenString)
	if err != nil {
		return "", errors.New("token无效或已过期")
	}
	newToken, err := uc.generateToken(claims.UserID, claims.Username)
	if err != nil {
		return "", err
	}
	return newToken, nil
}

// ValidateToken 验证token
func (uc *AuthUsecase) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(uc.secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的token")
}

// generateToken 生成JWT token
func (uc *AuthUsecase) generateToken(userID uint64, username string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(uc.expire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uc.secret))
}

// Claims JWT claims
type Claims struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}
