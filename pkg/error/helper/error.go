// Package helper 提供公共 API 错误对象和快捷构造函数。
package helper

import (
	"fmt"

	"github.com/Neo-Nedio/Lime-server/pkg/error/code"
	"github.com/pkg/errors"
)

// Error 是 API 返回的通用错误对象。
type Error struct {
	Code    code.Code `json:"code"`
	Message string    `json:"message"`
}

// Error 实现标准 error 接口，返回错误消息。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

// New 根据状态码和错误消息创建 API 错误。
func New(status code.Code, message string) *Error {
	return &Error{
		Code:    status,
		Message: message,
	}
}

// Newf 根据格式化消息创建 API 错误。
func Newf(status code.Code, format string, args ...any) *Error {
	return New(status, fmt.Sprintf(format, args...))
}

// NewBadRequest 创建 400 参数错误。
func NewBadRequest(message string) *Error {
	return New(code.BadRequest, message)
}

// NewUnauthorized 创建 401 未认证错误。
func NewUnauthorized(message string) *Error {
	return New(code.Unauthorized, message)
}

// NewForbidden 创建 403 无权限或业务规则错误。
func NewForbidden(message string) *Error {
	return New(code.Forbidden, message)
}

// NewNotFound 创建 404 资源不存在错误。
func NewNotFound(message string) *Error {
	return New(code.NotFound, message)
}

// NewInternalServerError 创建 500 操作失败错误。
func NewInternalServerError(message string) *Error {
	return New(code.InternalServerError, message)
}

// InternalError 包装内部错误，详细信息交给 RPC 拦截器记录。
func InternalError(message string, cause error) error {
	internalError := NewInternalServerError("系统繁忙，请稍后再试")
	if cause == nil {
		return errors.Wrapf(internalError, "%s", message)
	}
	return errors.Wrapf(internalError, "%s：%v", message, cause)
}
