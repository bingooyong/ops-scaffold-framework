package grpc

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryClientInterceptor 客户端一元RPC拦截器
// 用于记录客户端RPC调用的日志和性能指标
func UnaryClientInterceptor(logger *zap.Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()

		// 调用RPC
		err := invoker(ctx, method, req, reply, cc, opts...)

		// 计算耗时
		duration := time.Since(start)

		// 记录日志
		fields := []zap.Field{
			zap.String("method", method),
			zap.Duration("duration", duration),
		}

		if err != nil {
			// 提取gRPC错误码
			st, ok := status.FromError(err)
			if ok {
				fields = append(fields, zap.String("code", st.Code().String()))
			}
			fields = append(fields, zap.Error(err))

			// 根据错误类型选择日志级别
			if st.Code() == codes.DeadlineExceeded || st.Code() == codes.Unavailable {
				logger.Warn("grpc client call failed", fields...)
			} else {
				logger.Error("grpc client call failed", fields...)
			}
		} else {
			logger.Debug("grpc client call success", fields...)
		}

		return err
	}
}

// UnaryServerInterceptor 服务端一元RPC拦截器
// 用于记录服务端RPC处理的日志和性能指标
func UnaryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// 处理请求
		resp, err := handler(ctx, req)

		// 计算耗时
		duration := time.Since(start)

		// 记录日志
		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
		}

		if err != nil {
			// 提取gRPC错误码
			st, ok := status.FromError(err)
			if ok {
				fields = append(fields, zap.String("code", st.Code().String()))
			}
			fields = append(fields, zap.Error(err))

			// 根据错误类型选择日志级别
			if st.Code() == codes.InvalidArgument || st.Code() == codes.NotFound {
				logger.Info("grpc server call failed", fields...)
			} else {
				logger.Error("grpc server call failed", fields...)
			}
		} else {
			logger.Debug("grpc server call success", fields...)
		}

		return resp, err
	}
}
