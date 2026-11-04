// Package code 定义公共 API 响应格式使用的状态码。
package code

// Code 表示 API 状态码。
type Code int

// API 文档中定义的状态码。
const (
	Success             Code = 200
	BadRequest          Code = 400
	Unauthorized        Code = 401
	Forbidden           Code = 403
	NotFound            Code = 404
	InternalServerError Code = 500
)
