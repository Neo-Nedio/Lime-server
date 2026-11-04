package logic

import (
	"context"
	"strings"
	"time"

	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/svc"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/pb"
	"github.com/Neo-Nedio/Lime-server/pkg/contans"
	jwtutil "github.com/Neo-Nedio/Lime-server/pkg/jwt"

	"github.com/go-playground/validator/v10"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, status.Error(codes.InvalidArgument, "访问令牌不能为空")
	}
	accessToken := strings.TrimSpace(in.AccessToken)
	if err := logoutValidator.Struct(logoutValidation{AccessToken: accessToken}); err != nil {
		return nil, status.Error(codes.InvalidArgument, "访问令牌不能为空")
	}
	if l.svcCtx == nil || l.svcCtx.Redis == nil {
		return nil, logoutInternalError(l.Logger, "认证依赖未配置", nil)
	}

	tokenManager, err := jwtutil.NewTokenManager(
		l.svcCtx.Config.JWT.AccessSecret,
		l.svcCtx.Config.JWT.RefreshSecret,
		l.svcCtx.Config.JWT.AccessExpire,
		l.svcCtx.Config.JWT.RefreshExpire,
	)
	if err != nil {
		return nil, logoutInternalError(l.Logger, "初始化令牌工具失败", err)
	}

	claims, err := tokenManager.ParseAccessToken(accessToken)
	if err != nil || claims.ExpiresAt == nil {
		return nil, status.Error(codes.Unauthenticated, "访问令牌无效或已过期")
	}
	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, contans.RefreshTokenKey(claims.UserID)); err != nil {
		return nil, logoutInternalError(l.Logger, "删除刷新令牌失败", err)
	}

	remaining := logoutRemainingSeconds(claims.ExpiresAt.Time, time.Now())
	if remaining == 0 {
		return &pb.EmptyResponse{Success: true}, nil
	}
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, contans.BlacklistTokenKey(accessToken), "1", remaining); err != nil {
		return nil, logoutInternalError(l.Logger, "写入访问令牌黑名单失败", err)
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

func logoutInternalError(logger logx.Logger, message string, cause error) error {
	if logger != nil && cause != nil {
		logger.Errorf("%s：%v", message, cause)
	}
	return status.Error(codes.Internal, "系统繁忙，请稍后再试")
}
