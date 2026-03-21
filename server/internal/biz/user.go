package biz

import (
	"context"
	"time"
)

// User 用户业务实体
type User struct {
	ID         uint64
	Username   string
	Password   string
	Email      string
	RealName   string
	Phone      string
	Department string
	Status     int // 1-正常, 0-禁用
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// UserRepo 用户仓储接口
type UserRepo interface {
	Create(ctx context.Context, u *User) (*User, error)
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context, page, pageSize int) ([]*User, int64, error)
}

// UserUsecase 用户用例
type UserUsecase struct {
	repo UserRepo
}

// NewUserUsecase 创建用户用例
func NewUserUsecase(repo UserRepo) *UserUsecase {
	return &UserUsecase{repo: repo}
}

// CreateUser 创建用户
func (uc *UserUsecase) CreateUser(ctx context.Context, u *User) (*User, error) {
	// 检查用户名是否已存在
	_, err := uc.repo.GetByUsername(ctx, u.Username)
	if err == nil {
		return nil, ErrUserExists
	}

	// 检查邮箱是否已存在
	_, err = uc.repo.GetByEmail(ctx, u.Email)
	if err == nil {
		return nil, ErrEmailExists
	}

	return uc.repo.Create(ctx, u)
}

// UpdateUser 更新用户
func (uc *UserUsecase) UpdateUser(ctx context.Context, u *User) error {
	// 检查用户是否存在
	_, err := uc.repo.GetByID(ctx, u.ID)
	if err != nil {
		return ErrUserNotFound
	}

	// 如果更新邮箱，检查是否与其他用户冲突
	if u.Email != "" {
		existing, err := uc.repo.GetByEmail(ctx, u.Email)
		if err == nil && existing.ID != u.ID {
			return ErrEmailExists
		}
	}

	return uc.repo.Update(ctx, u)
}

// DeleteUser 删除用户
func (uc *UserUsecase) DeleteUser(ctx context.Context, id uint64) error {
	_, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
	}
	return uc.repo.Delete(ctx, id)
}

// GetUser 获取用户
func (uc *UserUsecase) GetUser(ctx context.Context, id uint64) (*User, error) {
	return uc.repo.GetByID(ctx, id)
}

// GetUserByUsername 根据用户名获取用户
func (uc *UserUsecase) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return uc.repo.GetByUsername(ctx, username)
}

// ListUsers 列出用户
func (uc *UserUsecase) ListUsers(ctx context.Context, page, pageSize int) ([]*User, int64, error) {
	return uc.repo.List(ctx, page, pageSize)
}
