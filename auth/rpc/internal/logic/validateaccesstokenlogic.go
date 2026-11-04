package logic

import (
	"context"
	"strconv"
	"strings"

	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/svc"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/pb"
	"github.com/Neo-Nedio/Lime-server/pkg/contans"
	errorHelper "github.com/Neo-Nedio/Lime-server/pkg/error/helper"
	jwtutil "github.com/Neo-Nedio/Lime-server/pkg/jwt"

	"github.com/go-playground/validator/v10"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ValidateAccessTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

type validateAccessTokenValidation struct {
	AccessToken string `validate:"required"`
}

var validateAccessTokenValidator = validator.New()

func NewValidateAccessTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidateAccessTokenLogic {
	return &ValidateAccessTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ValidateAccessTokenLogic) ValidateAccessToken(in *pb.ValidateAccessTokenRequest) (*pb.ValidateAccessTokenResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "访问令牌不能为空")
	}
	accessToken := strings.TrimSpace(in.AccessToken)
	if err := validateAccessTokenValidator.Struct(validateAccessTokenValidation{AccessToken: accessToken}); err != nil {
		return nil, status.Error(codes.InvalidArgument, "访问令牌不能为空")
	}
	if l.svcCtx == nil || l.svcCtx.Redis == nil {
		return nil, errorHelper.InternalError(l.Logger, "认证依赖未配置", nil)
	}

	tokenManager, err := jwtutil.NewTokenManager(
		l.svcCtx.Config.JWT.AccessSecret,
		l.svcCtx.Config.JWT.RefreshSecret,
		l.svcCtx.Config.JWT.AccessExpire,
		l.svcCtx.Config.JWT.RefreshExpire,
	)
	if err != nil {
		return nil, errorHelper.InternalError(l.Logger, "初始化令牌工具失败", err)
	}

	claims, err := tokenManager.ParseAccessToken(accessToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "访问令牌无效或已过期")
	}

	blacklisted, err := l.svcCtx.Redis.ExistsCtx(l.ctx, contans.BlacklistTokenKey(accessToken))
	if err != nil {
		return nil, errorHelper.InternalError(l.Logger, "检查访问令牌黑名单失败", err)
	}
	if blacklisted {
		return nil, status.Error(codes.Unauthenticated, "访问令牌已失效")
	}

	issueValue, err := l.svcCtx.Redis.GetCtx(l.ctx, contans.AuthIssueKey(claims.UserID))
	if err != nil {
		return nil, errorHelper.InternalError(l.Logger, "读取密码版本失败", err)
	}
	if issueValue == "" {
		return nil, status.Error(codes.Unauthenticated, "登录状态已失效")
	}
	issueVersion, err := strconv.ParseInt(strings.TrimSpace(issueValue), 10, 64)
	if err != nil || issueVersion < 0 {
		return nil, errorHelper.InternalError(l.Logger, "读取到无效的密码版本", err)
	}
	if claims.CredentialVersion < issueVersion {
		return nil, status.Error(codes.Unauthenticated, "访问令牌已失效")
	}

	return &pb.ValidateAccessTokenResponse{
		Valid:             true,
		UserId:            claims.UserID,
		CredentialVersion: claims.CredentialVersion,
		Subject:           claims.Subject,
	}, nil
}
