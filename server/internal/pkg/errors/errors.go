package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode 错误码
type ErrorCode int

const (
	// 通用错误码
	ErrCodeUnknown        ErrorCode = 1000
	ErrCodeInvalidParams  ErrorCode = 1001
	ErrCodeUnauthorized   ErrorCode = 1002
	ErrCodeForbidden      ErrorCode = 1003
	ErrCodeNotFound       ErrorCode = 1004
	ErrCodeInternalServer ErrorCode = 1005

	// 业务错误码
	ErrCodeUserExists      ErrorCode = 2001
	ErrCodeUserNotFound    ErrorCode = 2002
	ErrCodeInvalidPassword ErrorCode = 2003
	ErrCodeOrderNotFound   ErrorCode = 3001
	ErrCodeOrderStatusErr  ErrorCode = 3002
	ErrCodeSQLExecuteErr   ErrorCode = 3003
	ErrCodeInstanceNotFound ErrorCode = 4001
	ErrCodePermissionDenied ErrorCode = 5001
)

// AppError 应用错误
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatus 返回HTTP状态码
func (e *AppError) HTTPStatus() int {
	switch e.Code {
	case ErrCodeInvalidParams:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeInternalServer:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// NewAppError 创建应用错误
func NewAppError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewInvalidParamsError 创建参数错误
func NewInvalidParamsError(message string) *AppError {
	return NewAppError(ErrCodeInvalidParams, message, nil)
}

// NewUnauthorizedError 创建未授权错误
func NewUnauthorizedError(message string) *AppError {
	return NewAppError(ErrCodeUnauthorized, message, nil)
}

// NewForbiddenError 创建禁止访问错误
func NewForbiddenError(message string) *AppError {
	return NewAppError(ErrCodeForbidden, message, nil)
}

// NewNotFoundError 创建未找到错误
func NewNotFoundError(message string) *AppError {
	return NewAppError(ErrCodeNotFound, message, nil)
}

// NewInternalServerError 创建内部服务器错误
func NewInternalServerError(message string, err error) *AppError {
	return NewAppError(ErrCodeInternalServer, message, err)
}
