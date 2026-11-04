package logic

import (
	"context"

	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/svc"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ValidateAccessTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewValidateAccessTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidateAccessTokenLogic {
	return &ValidateAccessTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ValidateAccessTokenLogic) ValidateAccessToken(in *pb.ValidateAccessTokenRequest) (*pb.ValidateAccessTokenResponse, error) {
	// todo: add your logic here and delete this line

	return &pb.ValidateAccessTokenResponse{}, nil
}
