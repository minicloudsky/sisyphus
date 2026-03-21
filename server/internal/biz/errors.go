package biz

import "errors"

var (
	ErrUserNotFound      = errors.New("用户不存在")
	ErrUserExists        = errors.New("用户已存在")
	ErrEmailExists       = errors.New("邮箱已存在")
	ErrInvalidAuth       = errors.New("用户名或密码错误")
	ErrOrderNotFound     = errors.New("工单不存在")
	ErrNoPermission      = errors.New("无权限")
	ErrDuplicateFavorite = errors.New("收藏已存在")
	ErrFavoriteNotFound  = errors.New("收藏不存在")
)
