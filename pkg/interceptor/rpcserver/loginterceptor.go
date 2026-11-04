package rpcserver

import (
	"context"
	"errors"

	errorCode "github.com/Neo-Nedio/Lime-server/pkg/error/code"
	errorHelper "github.com/Neo-Nedio/Lime-server/pkg/error/helper"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LogInterceptor 区分记录 RPC 业务错误和内部错误，并保留业务错误码。
func LogInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	resp, err := handler(ctx, req)
	if err == nil {
		return resp, nil
	}

	var projectError *errorHelper.Error
	if errors.As(err, &projectError) {
		if projectError.Code == errorCode.InternalServerError {
			logInternalError(ctx, info.FullMethod, err)
			return nil, internalStatusError()
		}

		logBusinessError(ctx, info.FullMethod, int(projectError.Code), err)
		return nil, status.Error(codes.Code(projectError.Code), projectError.Message)
	}

	if rpcStatus, ok := rpcStatusFromError(err); ok {
		if rpcStatus.Code() == codes.Internal {
			logInternalError(ctx, info.FullMethod, err)
			return nil, internalStatusError()
		}

		logBusinessError(ctx, info.FullMethod, int(rpcStatus.Code()), err)
		return nil, rpcStatus.Err()
	}

	logInternalError(ctx, info.FullMethod, err)
	return nil, internalStatusError()
}

func rpcStatusFromError(err error) (*status.Status, bool) {
	var statusError interface {
		GRPCStatus() *status.Status
	}
	if !errors.As(err, &statusError) {
		return nil, false
	}

	rpcStatus := statusError.GRPCStatus()
	if rpcStatus == nil || rpcStatus.Code() == codes.Unknown {
		return nil, false
	}
	return rpcStatus, true
}

func logBusinessError(ctx context.Context, method string, code int, err error) {
	logx.WithContext(ctx).Infof("【RPC 业务错误】method=%s code=%d err=%v", method, code, err)
}

func logInternalError(ctx context.Context, method string, err error) {
	logx.WithContext(ctx).Errorf("【RPC 内部错误】method=%s err=%v", method, err)
}

func internalStatusError() error {
	return status.Error(codes.Internal, "系统繁忙，请稍后再试")
}
