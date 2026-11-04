package logic

import (
	"context"

	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/svc"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterLogic) Register(in *pb.RegisterRequest) (*pb.EmptyResponse, error) {
	// todo: add your logic here and delete this line

	return &pb.EmptyResponse{}, nil
}
