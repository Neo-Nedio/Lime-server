package logic

import (
	"context"
	"crypto/subtle"
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

type RefreshLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

type refreshValidation struct {
	RefreshToken string `validate:"required"`
}

var refreshValidator = validator.New()

func NewRefreshLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshLogic {
	return &RefreshLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RefreshLogic) Refresh(in *pb.RefreshRequest) (*pb.LoginResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "刷新令牌不能为空")
	}
	refreshToken := strings.TrimSpace(in.RefreshToken)
	if err := refreshValidator.Struct(refreshValidation{RefreshToken: refreshToken}); err != nil {
		return nil, status.Error(codes.InvalidArgument, "刷新令牌不能为空")
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

	claims, err := tokenManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "刷新令牌无效或已过期")
	}

	storedToken, err := l.svcCtx.Redis.GetCtx(l.ctx, contans.RefreshTokenKey(claims.UserID))
	if err != nil {
		return nil, errorHelper.InternalError(l.Logger, "读取当前刷新令牌失败", err)
	}
	if storedToken == "" || subtle.ConstantTimeCompare([]byte(storedToken), []byte(refreshToken)) != 1 {
		return nil, status.Error(codes.Unauthenticated, "刷新令牌已失效")
	}

	credentialVersion, err := l.refreshCredentialVersion(claims.UserID)
	if err != nil {
		return nil, err
	}

	pair, err := tokenManager.IssueTokenPair(refreshClaims(claims, credentialVersion))
	if err != nil {
		return nil, errorHelper.InternalError(l.Logger, "轮换刷新令牌失败", err)
	}
	if err := l.svcCtx.Redis.SetexCtx(
		l.ctx,
		contans.RefreshTokenKey(claims.UserID),
		pair.RefreshToken,
		int(l.svcCtx.Config.JWT.RefreshExpire),
	); err != nil {
		return nil, errorHelper.InternalError(l.Logger, "保存轮换后的刷新令牌失败", err)
	}

	return &pb.LoginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}, nil
}

func (l *RefreshLogic) refreshCredentialVersion(userID int64) (int64, error) {
	issueValue, err := l.svcCtx.Redis.GetCtx(l.ctx, contans.AuthIssueKey(userID))
	if err != nil {
		return 0, errorHelper.InternalError(l.Logger, "读取密码版本失败", err)
	}
	if issueValue == "" {
		return 0, status.Error(codes.Unauthenticated, "登录状态已失效")
	}

	issueVersion, err := strconv.ParseInt(strings.TrimSpace(issueValue), 10, 64)
	if err != nil || issueVersion < 0 {
		return 0, errorHelper.InternalError(l.Logger, "读取到无效的密码版本", err)
	}
	return issueVersion, nil
}

func refreshClaims(claims *jwtutil.Claims, credentialVersion int64) jwtutil.UserClaims {
	return jwtutil.UserClaims{
		UserID:            claims.UserID,
		Subject:           claims.Subject,
		Email:             claims.Email,
		UID:               claims.UID,
		Handle:            claims.Handle,
		Role:              claims.Role,
		CredentialVersion: credentialVersion,
	}
}
