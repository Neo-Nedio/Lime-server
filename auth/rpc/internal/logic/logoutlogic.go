package logic

import (
	"context"
	"strings"
	"time"

	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/svc"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/pb"
	"github.com/Neo-Nedio/Lime-server/pkg/contans"
	errorHelper "github.com/Neo-Nedio/Lime-server/pkg/error/helper"
	jwtutil "github.com/Neo-Nedio/Lime-server/pkg/jwt"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

type logoutValidation struct {
	AccessToken string `validate:"required"`
}

var logoutValidator = validator.New()

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LogoutLogic) Logout(in *pb.LogoutRequest) (*pb.EmptyResponse, error) {
	if in == nil {
		return nil, errors.Wrapf(errorHelper.NewBadRequest("访问令牌不能为空"), "退出登录请求参数错误")
	}
	accessToken := strings.TrimSpace(in.AccessToken)
	if err := logoutValidator.Struct(logoutValidation{AccessToken: accessToken}); err != nil {
		return nil, errors.Wrapf(errorHelper.NewBadRequest("访问令牌不能为空"), "退出登录参数校验失败")
	}
	if l.svcCtx == nil || l.svcCtx.Redis == nil {
		return nil, errorHelper.InternalError("认证依赖未配置", nil)
	}

	tokenManager, err := jwtutil.NewTokenManager(
		l.svcCtx.Config.JWT.AccessSecret,
		l.svcCtx.Config.JWT.RefreshSecret,
		l.svcCtx.Config.JWT.AccessExpire,
		l.svcCtx.Config.JWT.RefreshExpire,
	)
	if err != nil {
		return nil, errorHelper.InternalError("初始化令牌工具失败", err)
	}

	claims, err := tokenManager.ParseAccessToken(accessToken)
	if err != nil || claims.ExpiresAt == nil {
		return nil, errors.Wrapf(errorHelper.NewUnauthorized("访问令牌无效或已过期"), "解析访问令牌失败")
	}
	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, contans.RefreshTokenKey(claims.UserID)); err != nil {
		return nil, errorHelper.InternalError("删除刷新令牌失败", err)
	}

	remaining := logoutRemainingSeconds(claims.ExpiresAt.Time, time.Now())
	if remaining == 0 {
		return &pb.EmptyResponse{Success: true}, nil
	}
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, contans.BlacklistTokenKey(accessToken), "1", remaining); err != nil {
		return nil, errorHelper.InternalError("写入访问令牌黑名单失败", err)
	}

	return &pb.EmptyResponse{Success: true}, nil
}

func logoutRemainingSeconds(expiresAt, now time.Time) int {
	remaining := expiresAt.Sub(now)
	if remaining <= 0 {
		return 0
	}

	seconds := int(remaining / time.Second)
	if remaining%time.Second != 0 {
		seconds++
	}
	return seconds
}
