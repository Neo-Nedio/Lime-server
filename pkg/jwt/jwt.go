// Package jwt 提供项目内统一的 JWT 签发和解析工具。
package jwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Neo-Nedio/Lime-server/pkg/contans"
	jwtv4 "github.com/golang-jwt/jwt/v4"
)

var (
	// ErrInvalidTokenConfig 表示 JWT 密钥或有效期配置无效。
	ErrInvalidTokenConfig = errors.New("JWT 配置无效")
	// ErrInvalidTokenClaims 表示 JWT 中缺少必要的身份字段。
	ErrInvalidTokenClaims = errors.New("JWT 载荷无效")
	// ErrInvalidToken 表示 JWT 为空、签名错误或已经失效。
	ErrInvalidToken = errors.New("JWT 无效")
	// ErrTokenSigning 表示 JWT 签发失败。
	ErrTokenSigning = errors.New("JWT 签发失败")
	// ErrUnexpectedSigningMethod 表示 JWT 使用了不允许的签名算法。
	ErrUnexpectedSigningMethod = errors.New("JWT 签名算法不受支持")
	// ErrTokenTypeMismatch 表示 JWT 类型与当前校验入口不匹配。
	ErrTokenTypeMismatch = errors.New("JWT 类型不匹配")
)

// Claims 是认证令牌的统一载荷。
//
// 除标准的 subject、iat、exp 外，这里保留认证流程后续经常使用的用户身份参数；
// 不放入密码、密码哈希、密钥等敏感信息。
type Claims struct {
	UserID            int64  `json:"user_id"`
	Email             string `json:"email,omitempty"`
	UID               string `json:"uid,omitempty"`
	Handle            string `json:"handle,omitempty"`
	Role              string `json:"role,omitempty"`
	CredentialVersion int64  `json:"credential_version"`
	TokenType         string `json:"token_type"`
	jwtv4.RegisteredClaims
}

// UserClaims 是签发令牌时传入的用户身份参数。
type UserClaims struct {
	UserID            int64
	Subject           string
	Email             string
	UID               string
	Handle            string
	Role              string
	CredentialVersion int64
}

// TokenPair 是一次签发得到的一对访问令牌和刷新令牌。
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// TokenManager 负责使用不同密钥管理访问令牌和刷新令牌。
type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	now           func() time.Time
}

// Option 是 TokenManager 的可选配置。
type Option func(*TokenManager)

// WithClock 为 TokenManager 注入时钟，便于测试固定的签发时间。
func WithClock(now func() time.Time) Option {
	return func(manager *TokenManager) {
		if now != nil {
			manager.now = now
		}
	}
}

// NewTokenManager 创建 JWT 工具实例。
func NewTokenManager(accessSecret, refreshSecret string, accessExpire, refreshExpire int64, options ...Option) (*TokenManager, error) {
	if strings.TrimSpace(accessSecret) == "" || strings.TrimSpace(refreshSecret) == "" {
		return nil, fmt.Errorf("%w：访问令牌和刷新令牌的密钥不能为空", ErrInvalidTokenConfig)
	}
	if accessExpire <= 0 || refreshExpire <= 0 {
		return nil, fmt.Errorf("%w：访问令牌和刷新令牌的有效期必须大于零", ErrInvalidTokenConfig)
	}

	manager := &TokenManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     time.Duration(accessExpire) * time.Second,
		refreshTTL:    time.Duration(refreshExpire) * time.Second,
		now:           time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(manager)
		}
	}

	return manager, nil
}

// IssueAccessToken 签发访问令牌。
func (m *TokenManager) IssueAccessToken(user UserClaims) (string, error) {
	return m.issue(user, contans.TokenTypeAccess, m.accessTTL, m.accessSecret)
}

// IssueRefreshToken 签发刷新令牌。
func (m *TokenManager) IssueRefreshToken(user UserClaims) (string, error) {
	return m.issue(user, contans.TokenTypeRefresh, m.refreshTTL, m.refreshSecret)
}

// IssueTokenPair 同时签发访问令牌和刷新令牌。
func (m *TokenManager) IssueTokenPair(user UserClaims) (TokenPair, error) {
	accessToken, err := m.IssueAccessToken(user)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, err := m.IssueRefreshToken(user)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(m.accessTTL / time.Second),
	}, nil
}

// ParseAccessToken 校验并解析访问令牌。
func (m *TokenManager) ParseAccessToken(rawToken string) (*Claims, error) {
	return m.parse(rawToken, contans.TokenTypeAccess, m.accessSecret)
}

// ParseRefreshToken 校验并解析刷新令牌。
func (m *TokenManager) ParseRefreshToken(rawToken string) (*Claims, error) {
	return m.parse(rawToken, contans.TokenTypeRefresh, m.refreshSecret)
}

func (m *TokenManager) issue(user UserClaims, tokenType string, ttl time.Duration, secret []byte) (string, error) {
	user = normalizeUserClaims(user)
	if user.UserID <= 0 || user.Subject == "" || user.CredentialVersion < 0 {
		return "", fmt.Errorf("%w：用户 ID、主题和凭证版本无效", ErrInvalidTokenClaims)
	}

	now := m.now().UTC()
	claims := Claims{
		UserID:            user.UserID,
		Email:             user.Email,
		UID:               user.UID,
		Handle:            user.Handle,
		Role:              user.Role,
		CredentialVersion: user.CredentialVersion,
		TokenType:         tokenType,
		RegisteredClaims: jwtv4.RegisteredClaims{
			Subject:   user.Subject,
			IssuedAt:  jwtv4.NewNumericDate(now),
			ExpiresAt: jwtv4.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwtv4.NewWithClaims(jwtv4.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("%w（类型：%s）", ErrTokenSigning, tokenType)
	}

	return signed, nil
}

func (m *TokenManager) parse(rawToken, expectedTokenType string, secret []byte) (*Claims, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, ErrInvalidToken
	}

	token, err := jwtv4.ParseWithClaims(rawToken, &Claims{}, func(token *jwtv4.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwtv4.SigningMethodHS256.Alg() {
			return nil, ErrUnexpectedSigningMethod
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w：签名、格式或有效期校验失败", ErrInvalidToken)
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.UserID <= 0 || claims.CredentialVersion < 0 || strings.TrimSpace(claims.Subject) == "" {
		return nil, ErrInvalidTokenClaims
	}
	if claims.TokenType != expectedTokenType {
		return nil, ErrTokenTypeMismatch
	}

	return claims, nil
}

func normalizeUserClaims(user UserClaims) UserClaims {
	user.Subject = strings.TrimSpace(user.Subject)
	user.Email = strings.TrimSpace(user.Email)
	user.UID = strings.TrimSpace(user.UID)
	user.Handle = strings.TrimSpace(user.Handle)
	user.Role = strings.TrimSpace(user.Role)
	if user.Subject == "" {
		user.Subject = user.Email
	}

	return user
}
