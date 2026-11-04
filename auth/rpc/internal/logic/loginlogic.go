package logic

import (
	"context"
	"crypto/subtle"
	stderrors "errors"
	"strconv"
	"strings"
	"time"

	"github.com/Neo-Nedio/Lime-server/auth/models"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/svc"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/pb"
	"github.com/Neo-Nedio/Lime-server/pkg/contans"
	errorHelper "github.com/Neo-Nedio/Lime-server/pkg/error/helper"
	jwtutil "github.com/Neo-Nedio/Lime-server/pkg/jwt"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

type loginValidation struct {
	Email    string `validate:"required,max=100,email"`
	Password string `validate:"required_without=Code,excluded_with=Code,omitempty"`
	Code     string `validate:"required_without=Password,excluded_with=Password,omitempty,len=6,numeric"`
}

var loginValidator = validator.New()

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *pb.LoginRequest) (*pb.LoginResponse, error) {
	if in == nil {
		return nil, errors.Wrapf(errorHelper.NewBadRequest("请求不能为空"), "登录请求参数错误")
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	code := strings.TrimSpace(in.Code)
	if err := loginValidator.Struct(loginValidation{
		Email:    email,
		Password: in.Password,
		Code:     code,
	}); err != nil {
		return nil, errors.Wrapf(errorHelper.NewBadRequest(loginValidationMessage(err)), "登录参数校验失败")
	}
	if l.svcCtx == nil || l.svcCtx.Redis == nil || l.svcCtx.UserModel == nil {
		return nil, errorHelper.InternalError("认证依赖未配置", nil)
	}

	user, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, email)
	switch {
	case stderrors.Is(err, models.ErrNotFound):
		return nil, errors.Wrapf(errorHelper.NewUnauthorized("邮箱、密码或验证码错误"), "登录用户不存在")
	case err != nil:
		return nil, errorHelper.InternalError("查询用户失败", err)
	case user == nil:
		return nil, errors.Wrapf(errorHelper.NewUnauthorized("邮箱、密码或验证码错误"), "登录用户为空")
	case user.Deleted != contans.UserNotDeleted || user.Status == contans.UserStatusDisabled:
		return nil, errors.Wrapf(errorHelper.NewForbidden("账号已被禁用"), "登录用户状态不允许登录")
	}

	if in.Password != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)); err != nil {
			return nil, errors.Wrapf(errorHelper.NewUnauthorized("密码错误"), "用户 %s 密码比对失败", email)
		}
	} else if err := l.verifyLoginCode(email, code); err != nil {
		return nil, errors.Wrapf(err, "验证码登录失败")
	}

	credentialVersion := time.Now().Unix()
	if err := l.svcCtx.Redis.SetexCtx(
		l.ctx,
		contans.AuthIssueKey(user.Id),
		strconv.FormatInt(credentialVersion, 10),
		int(l.svcCtx.Config.JWT.RefreshExpire),
	); err != nil {
		return nil, errorHelper.InternalError("保存密码版本失败", err)
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
	pair, err := tokenManager.IssueTokenPair(loginClaims(user, credentialVersion))
	if err != nil {
		return nil, errorHelper.InternalError("签发登录令牌失败", err)
	}
	if err := l.svcCtx.Redis.SetexCtx(
		l.ctx,
		contans.RefreshTokenKey(user.Id),
		pair.RefreshToken,
		int(l.svcCtx.Config.JWT.RefreshExpire),
	); err != nil {
		return nil, errorHelper.InternalError("保存刷新令牌失败", err)
	}

	return &pb.LoginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}, nil
}

func (l *LoginLogic) verifyLoginCode(email, code string) error {
	storedCode, err := l.svcCtx.Redis.GetCtx(l.ctx, contans.EmailCodeKey(email))
	if err != nil {
		return errorHelper.InternalError("读取验证码失败", err)
	}
	if storedCode == "" || subtle.ConstantTimeCompare([]byte(storedCode), []byte(code)) != 1 {
		return errors.Wrapf(errorHelper.NewUnauthorized("邮箱、密码或验证码错误"), "登录验证码校验失败")
	}
	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, contans.EmailCodeKey(email)); err != nil {
		return errorHelper.InternalError("消费验证码失败", err)
	}
	return nil
}

func loginClaims(user *models.User, credentialVersion int64) jwtutil.UserClaims {
	return jwtutil.UserClaims{
		UserID:            user.Id,
		Subject:           user.Email,
		Email:             user.Email,
		UID:               user.Uid.String,
		Handle:            user.Handle.String,
		Role:              user.Role,
		CredentialVersion: credentialVersion,
	}
}

func loginValidationMessage(err error) string {
	var validationErrors validator.ValidationErrors
	if !stderrors.As(err, &validationErrors) || len(validationErrors) == 0 {
		return "登录参数无效"
	}

	switch validationErrors[0].Field() {
	case "Email":
		return "邮箱格式无效"
	case "Password", "Code":
		return "密码和验证码必须二选一，验证码必须为 6 位数字"
	default:
		return "登录参数无效"
	}
}
