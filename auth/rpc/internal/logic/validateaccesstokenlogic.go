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
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
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
		return nil, errors.Wrapf(errorHelper.NewBadRequest("访问令牌不能为空"), "校验访问令牌请求参数错误")
	}
	accessToken := strings.TrimSpace(in.AccessToken)
	if err := validateAccessTokenValidator.Struct(validateAccessTokenValidation{AccessToken: accessToken}); err != nil {
		return nil, errors.Wrapf(errorHelper.NewBadRequest("访问令牌不能为空"), "校验访问令牌参数失败")
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
	if err != nil {
		return nil, errors.Wrapf(errorHelper.NewUnauthorized("访问令牌无效或已过期"), "解析访问令牌失败")
	}

	blacklisted, err := l.svcCtx.Redis.ExistsCtx(l.ctx, contans.BlacklistTokenKey(accessToken))
	if err != nil {
		return nil, errorHelper.InternalError("检查访问令牌黑名单失败", err)
	}
	if blacklisted {
		return nil, errors.Wrapf(errorHelper.NewUnauthorized("访问令牌已失效"), "访问令牌命中黑名单")
	}

	issueValue, err := l.svcCtx.Redis.GetCtx(l.ctx, contans.AuthIssueKey(claims.UserID))
	if err != nil {
		return nil, errorHelper.InternalError("读取密码版本失败", err)
	}
	if issueValue == "" {
		return nil, errors.Wrapf(errorHelper.NewUnauthorized("登录状态已失效"), "登录凭证版本不存在")
	}
	issueVersion, err := strconv.ParseInt(strings.TrimSpace(issueValue), 10, 64)
	if err != nil || issueVersion < 0 {
		return nil, errorHelper.InternalError("读取到无效的密码版本", err)
	}
	if claims.CredentialVersion < issueVersion {
		return nil, errors.Wrapf(errorHelper.NewUnauthorized("访问令牌已失效"), "访问令牌凭证版本过低")
	}

	return &pb.ValidateAccessTokenResponse{
		Valid:             true,
		UserId:            claims.UserID,
		CredentialVersion: claims.CredentialVersion,
		Subject:           claims.Subject,
	}, nil
}
