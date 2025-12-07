package grpc

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 业务错误类型
var (
	ErrAgentNotFound    = errors.New("agent not found")
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrConnectionFailed = errors.New("connection failed")
	ErrTimeout          = errors.New("operation timeout")
)

// convertGRPCError 将gRPC错误转换为业务错误
func convertGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// 检查是否是gRPC状态错误
	st, ok := status.FromError(err)
	if !ok {
		// 如果不是gRPC错误，直接返回
		return err
	}

	// 根据gRPC错误码转换为业务错误
	switch st.Code() {
	case codes.NotFound:
		return ErrAgentNotFound
	case codes.InvalidArgument:
		return ErrInvalidArgument
	case codes.DeadlineExceeded:
		return ErrTimeout
	case codes.Unavailable:
		return ErrConnectionFailed
	case codes.Internal:
		// 内部错误，返回原始错误信息
		return err
	default:
		// 其他错误，返回原始错误
		return err
	}
}
