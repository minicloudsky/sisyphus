package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	
	"sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
)

// AuthService 认证服务（实现 Protobuf 生成的接口）
type AuthService struct {
	authUsecase *biz.AuthUsecase
	userUsecase *biz.UserUsecase
	log         *log.Helper
}

// NewAuthService 创建认证服务
func NewAuthService(authUsecase *biz.AuthUsecase, userUsecase *biz.UserUsecase, logger log.Logger) *AuthService {
	return &AuthService{
		authUsecase: authUsecase,
		userUsecase: userUsecase,
		log:         log.NewHelper(logger),
	}
}

// Register 用户注册（实现 Protobuf 接口）
func (s *AuthService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	user, _, err := s.authUsecase.Register(ctx, req.Username, req.Password, req.Email, req.RealName)
	if err != nil {
		return &v1.RegisterResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.RegisterResponse{
		Code:    200,
		Message: "success",
		Data: &v1.User{
			Id:         user.ID,
			Username:   user.Username,
			Email:      user.Email,
			RealName:   user.RealName,
			Phone:      user.Phone,
			Department: user.Department,
			Status:     int32(user.Status),
		},
	}, nil
}

// Login 用户登录（实现 Protobuf 接口）
func (s *AuthService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	user, token, err := s.authUsecase.Login(ctx, req.Username, req.Password)
	if err != nil {
		return &v1.LoginResponse{
			Code:    401,
			Message: "用户名或密码错误",
		}, nil
	}

	return &v1.LoginResponse{
		Code:    200,
		Message: "success",
		Data: &v1.LoginData{
			Token: token,
			User: &v1.User{
				Id:       user.ID,
				Username: user.Username,
				Email:    user.Email,
				RealName: user.RealName,
				Phone:    user.Phone,
				Department: user.Department,
				Status:   int32(user.Status),
			},
		},
	}, nil
}

// RefreshToken 刷新token
func (s *AuthService) RefreshToken(ctx context.Context, oldToken string) (string, error) {
	newToken, err := s.authUsecase.RefreshToken(oldToken)
	if err != nil {
		return "", err
	}
	return newToken, nil
}

// UpdateUser 更新用户信息（实现 Protobuf 接口）
func (s *AuthService) UpdateUser(ctx context.Context, req *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
	userID := getUserIDFromContext(ctx)
	if userID == 0 {
		return &v1.UpdateUserResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}

	// 获取当前用户
	user, err := s.userUsecase.GetUser(ctx, userID)
	if err != nil {
		return &v1.UpdateUserResponse{
			Code:    404,
			Message: "用户不存在",
		}, nil
	}

	// 更新用户信息
	user.Email = req.Email
	user.RealName = req.RealName
	user.Phone = req.Phone
	user.Department = req.Department

	err = s.userUsecase.UpdateUser(ctx, user)
	if err != nil {
		return &v1.UpdateUserResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	// 重新获取更新后的用户信息
	updatedUser, err := s.userUsecase.GetUser(ctx, userID)
	if err != nil {
		return &v1.UpdateUserResponse{
			Code:    500,
			Message: "获取更新后的用户信息失败",
		}, nil
	}

	return &v1.UpdateUserResponse{
		Code:    200,
		Message: "success",
		Data: &v1.User{
			Id:         updatedUser.ID,
			Username:   updatedUser.Username,
			Email:      updatedUser.Email,
			RealName:   updatedUser.RealName,
			Phone:      updatedUser.Phone,
			Department: updatedUser.Department,
			Status:     int32(updatedUser.Status),
		},
	}, nil
}
