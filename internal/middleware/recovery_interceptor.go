package middleware

import (
	"context"
	"runtime"

	"github.com/wumitech-com/mdcp_common/enum"
	"github.com/wumitech-com/mdcp_common/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryInterceptor 捕获 handler 中的 panic，记录包含 Trace 的错误日志并返回 Internal 错误
func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			trace, _ := ctx.Value(enum.CtxKeyTrace).(string)
			logger.ErrorFWithContext(ctx, "panic recovered - 方法: %s, Trace: %s, 错误: %v", info.FullMethod, trace, r)
			var pcs [8]uintptr
			n := runtime.Callers(3, pcs[:])
			if n > 0 {
				f := runtime.FuncForPC(pcs[0])
				file, line := f.FileLine(pcs[0])
				logger.ErrorFWithContext(ctx, "panic at %s:%d %s", file, line, f.Name())
			}
			err = status.Error(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}
