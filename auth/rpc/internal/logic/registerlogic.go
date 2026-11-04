package logic

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	stderrors "errors"
	"math/big"
	"strings"
	"unicode"

	"github.com/Neo-Nedio/Lime-server/auth/models"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/svc"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/pb"
	"github.com/Neo-Nedio/Lime-server/pkg/contans"
	errorHelper "github.com/Neo-Nedio/Lime-server/pkg/error/helper"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

type registerValidation struct {
	Email    string `validate:"required,max=100,email"`
	Password string `validate:"required,min=6,max=32,password"`
	Code     string `validate:"required,len=6,numeric"`
	Phone    string `validate:"omitempty,len=11,numeric"`
}

var registerValidator = newRegisterValidator()

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterLogic) Register(in *pb.RegisterRequest) (*pb.EmptyResponse, error) {
	if in == nil {
		return nil, errors.Wrapf(errorHelper.NewBadRequest("请求不能为空"), "注册请求参数错误")
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	code := strings.TrimSpace(in.Code)
	phone := strings.TrimSpace(in.Phone)
	if err := registerValidator.Struct(registerValidation{
		Email:    email,
		Password: in.Password,
		Code:     code,
		Phone:    phone,
	}); err != nil {
		return nil, errors.Wrapf(errorHelper.NewBadRequest(registerValidationMessage(err)), "注册参数校验失败")
	}
	if l.svcCtx == nil || l.svcCtx.Redis == nil || l.svcCtx.UserModel == nil {
		return nil, errorHelper.InternalError("认证依赖未配置", nil)
	}

	storedCode, err := l.svcCtx.Redis.GetCtx(l.ctx, contans.EmailCodeKey(email))
	if err != nil {
		return nil, errorHelper.InternalError("读取验证码失败", err)
	}
	if storedCode == "" || subtle.ConstantTimeCompare([]byte(storedCode), []byte(code)) != 1 {
		return nil, errors.Wrapf(errorHelper.NewBadRequest("验证码错误或已过期"), "注册验证码校验失败")
	}
	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, contans.EmailCodeKey(email)); err != nil {
		return nil, errorHelper.InternalError("消费验证码失败", err)
	}

	existing, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, email)
	switch {
	case err == nil && existing != nil:
		return nil, errors.Wrapf(status.Error(codes.AlreadyExists, "邮箱已注册"), "注册邮箱重复")
	case err != nil && !registerNotFound(err):
		return nil, errorHelper.InternalError("查询邮箱唯一性失败", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errorHelper.InternalError("加密密码失败", err)
	}
	nickname, handle, uid, err := generateRegisterProfile()
	if err != nil {
		return nil, errors.Wrapf(err, "生成用户身份信息失败")
	}

	user := &models.User{
		Email:    email,
		Password: string(hashedPassword),
		Nickname: registerNullableString(nickname),
		Handle:   registerNullableString(handle),
		Uid:      registerNullableString(uid),
		Phone:    registerNullableString(phone),
		Role:     contans.UserRole,
		Status:   contans.UserStatusEnabled,
		Deleted:  contans.UserNotDeleted,
	}
	if _, err := l.svcCtx.UserModel.CreateAuthUser(l.ctx, user); err != nil {
		if registerDuplicate(err) {
			return nil, errors.Wrapf(status.Error(codes.AlreadyExists, "用户信息已存在"), "创建用户时发现重复信息")
		}
		return nil, errorHelper.InternalError("创建用户失败", err)
	}

	return &pb.EmptyResponse{Success: true}, nil
}

func newRegisterValidator() *validator.Validate {
	validate := validator.New()
	_ = validate.RegisterValidation("password", func(field validator.FieldLevel) bool {
		hasLetter, hasDigit := false, false
		for _, char := range field.Field().String() {
			hasLetter = hasLetter || unicode.IsLetter(char)
			hasDigit = hasDigit || unicode.IsDigit(char)
		}
		return hasLetter && hasDigit
	})
	return validate
}

func registerValidationMessage(err error) string {
	var validationErrors validator.ValidationErrors
	if !stderrors.As(err, &validationErrors) || len(validationErrors) == 0 {
		return "注册参数无效"
	}

	switch validationErrors[0].Field() {
	case "Email":
		return "邮箱格式无效"
	case "Password":
		return "密码必须为 6 到 32 位，并同时包含字母和数字"
	case "Code":
		return "验证码必须为 6 位数字"
	case "Phone":
		return "手机号必须为 11 位数字"
	default:
		return "注册参数无效"
	}
}

func generateRegisterProfile() (string, string, string, error) {
	nicknameSuffix, err := generateRegisterDigits(contans.VerificationCodeLength)
	if err != nil {
		return "", "", "", errorHelper.InternalError("生成用户昵称失败", err)
	}
	handleSuffix, err := generateRegisterHex(8)
	if err != nil {
		return "", "", "", errorHelper.InternalError("生成用户标识失败", err)
	}
	uidSuffix, err := generateRegisterHex(32)
	if err != nil {
		return "", "", "", errorHelper.InternalError("生成用户唯一标识失败", err)
	}
	return "用户" + nicknameSuffix, "user_" + handleSuffix, "uid_" + uidSuffix, nil
}

func generateRegisterDigits(length int) (string, error) {
	limit := big.NewInt(int64(len(contans.VerificationCodeDigits)))
	var builder strings.Builder
	builder.Grow(length)
	for index := 0; index < length; index++ {
		digit, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", errorHelper.InternalError("生成随机数字失败", err)
		}
		builder.WriteByte(contans.VerificationCodeDigits[digit.Int64()])
	}
	return builder.String(), nil
}

func generateRegisterHex(length int) (string, error) {
	if length <= 0 {
		return "", errorHelper.InternalError("随机字符串长度必须大于零", nil)
	}
	bytes := make([]byte, (length+1)/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", errorHelper.InternalError("生成随机字符串失败", err)
	}
	return hex.EncodeToString(bytes)[:length], nil
}

func registerNullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func registerNotFound(err error) bool {
	return stderrors.Is(err, models.ErrNotFound)
}

func registerDuplicate(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate") || strings.Contains(message, "unique constraint") || strings.Contains(message, "唯一")
}
