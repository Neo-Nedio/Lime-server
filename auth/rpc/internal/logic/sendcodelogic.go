package logic

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/config"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/svc"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/pb"
	"github.com/Neo-Nedio/Lime-server/pkg/contans"
	errorHelper "github.com/Neo-Nedio/Lime-server/pkg/error/helper"

	"github.com/go-playground/validator/v10"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SendCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendCodeLogic {
	return &SendCodeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendCodeLogic) SendCode(in *pb.SendCodeRequest) (*pb.EmptyResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "请求不能为空")
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	if err := sendCodeValidator.Struct(sendCodeValidation{Email: email}); err != nil {
		return nil, status.Error(codes.InvalidArgument, "邮箱格式无效")
	}
	if l.svcCtx == nil || l.svcCtx.Redis == nil {
		return nil, errorHelper.InternalError(l.Logger, "认证依赖未配置", nil)
	}

	cooldownKey := contans.EmailCodeCooldownKey(email)
	exists, err := l.svcCtx.Redis.ExistsCtx(l.ctx, cooldownKey)
	if err != nil {
		return nil, errorHelper.InternalError(l.Logger, "检查验证码发送频率失败", err)
	}
	if exists {
		return nil, status.Error(codes.ResourceExhausted, "验证码发送过于频繁，请稍后再试")
	}

	code, err := generateSendCode()
	if err != nil {
		return nil, errorHelper.InternalError(l.Logger, "生成验证码失败", err)
	}
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, contans.EmailCodeKey(email), code, int(contans.VerificationCodeTTL/time.Second)); err != nil {
		return nil, errorHelper.InternalError(l.Logger, "保存验证码失败", err)
	}
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, cooldownKey, "1", int(contans.VerificationCooldownTTL/time.Second)); err != nil {
		return nil, errorHelper.InternalError(l.Logger, "保存验证码冷却状态失败", err)
	}
	if err := sendSendCodeEmail(l.ctx, l.svcCtx.Config, email, code); err != nil {
		return nil, errorHelper.InternalError(l.Logger, "发送验证码邮件失败", err)
	}

	return &pb.EmptyResponse{Success: true}, nil
}

type sendCodeValidation struct {
	Email string `validate:"required,max=100,email"`
}

var sendCodeValidator = validator.New()

func generateSendCode() (string, error) {
	limit := big.NewInt(int64(len(contans.VerificationCodeDigits)))
	var builder strings.Builder
	builder.Grow(contans.VerificationCodeLength)

	for index := 0; index < contans.VerificationCodeLength; index++ {
		digit, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		builder.WriteByte(contans.VerificationCodeDigits[digit.Int64()])
	}

	return builder.String(), nil
}

func sendSendCodeEmail(ctx context.Context, c config.Config, email, code string) error {
	if err := ctx.Err(); err != nil {
		return errors.New("邮件发送已取消")
	}
	if strings.TrimSpace(c.Mailer.Host) == "" || c.Mailer.Port <= 0 || strings.TrimSpace(c.Mailer.From) == "" {
		return errors.New("SMTP 配置不完整")
	}

	recipient, err := mail.ParseAddress(email)
	if err != nil || recipient.Address != email {
		return errors.New("收件邮箱地址无效")
	}

	address := net.JoinHostPort(strings.TrimSpace(c.Mailer.Host), strconv.Itoa(c.Mailer.Port))
	message := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: Lime 验证码\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n您的验证码是 %s，有效期 5 分钟。\r\n",
		c.Mailer.From, recipient.Address, code,
	))

	var auth smtp.Auth
	if c.Mailer.Username != "" || c.Mailer.Password != "" {
		auth = smtp.PlainAuth("", c.Mailer.Username, c.Mailer.Password, c.Mailer.Host)
	}
	if err := smtp.SendMail(address, auth, c.Mailer.From, []string{recipient.Address}, message); err != nil {
		return errors.New("SMTP 邮件发送失败")
	}

	return nil
}
