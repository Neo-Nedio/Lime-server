// Package contans 保存认证模块共享的常量和 Redis Key 构造函数。
package contans

import (
	"strconv"
	"time"
)

const (
	// Redis 键前缀。
	RedisEmailCodePrefix         = "email:code:"
	RedisEmailCodeCooldownPrefix = "email:code:cd:"
	RedisRefreshTokenPrefix      = "refresh:token:"
	RedisBlacklistTokenPrefix    = "blacklist:token:"
	RedisAuthIssuePrefix         = "auth:issue:"

	// 验证码规则和生命周期。
	VerificationCodeLength  = 6
	VerificationCodeDigits  = "0123456789"
	VerificationCodeTTL     = 5 * time.Minute
	VerificationCooldownTTL = 60 * time.Second

	// 注册和登录参数约束。
	PasswordMinLength = 6
	PasswordMaxLength = 32
	PhoneLength       = 11

	// 默认用户角色和状态。user.status=0 表示启用，1 表示禁用。
	UserRole = "USER"

	UserStatusEnabled  int64 = 0
	UserStatusDisabled int64 = 1
	UserNotDeleted     int64 = 0
	UserDeleted        int64 = 1

	// JWT 类型。
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// EmailCodeKey 返回邮箱验证码对应的 Redis 键。
func EmailCodeKey(email string) string {
	return RedisEmailCodePrefix + email
}

// EmailCodeCooldownKey 返回邮箱验证码冷却对应的 Redis 键。
func EmailCodeCooldownKey(email string) string {
	return RedisEmailCodeCooldownPrefix + email
}

// RefreshTokenKey 返回用户当前刷新令牌对应的 Redis 键。
func RefreshTokenKey(userID int64) string {
	return RedisRefreshTokenPrefix + strconv.FormatInt(userID, 10)
}

// BlacklistTokenKey 返回访问令牌黑名单对应的 Redis 键。
func BlacklistTokenKey(token string) string {
	return RedisBlacklistTokenPrefix + token
}

// AuthIssueKey 返回用户凭证版本对应的 Redis 键。
func AuthIssueKey(userID int64) string {
	return RedisAuthIssuePrefix + strconv.FormatInt(userID, 10)
}
