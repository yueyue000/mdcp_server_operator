package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/wumitech-com/mdcp_common/logger"
	"github.com/wumitech-com/mdcp_proto/api/server_operator"
	"github.com/wumitech-com/mdcp_server_operator/internal/config"
	"github.com/wumitech-com/mdcp_server_operator/internal/handlers"
	"github.com/wumitech-com/mdcp_server_operator/internal/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// RunGRPCServer 启动gRPC服务器
func RunGRPCServer(ctx context.Context, cfg *config.Config) error {
	// 创建监听器
	addr := fmt.Sprintf("%s:%d", cfg.Server.GRPC.Host, cfg.Server.GRPC.Port)
	logger.InfoF("正在创建gRPC监听器: %s", addr)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("创建监听器失败: %v", err)
	}
	logger.InfoF("gRPC监听器创建成功")

	// 创建gRPC服务器选项
	keepaliveParams := keepalive.ServerParameters{
		Time:    time.Duration(cfg.Server.GRPC.Keepalive.Time) * time.Second,
		Timeout: time.Duration(cfg.Server.GRPC.Keepalive.Timeout) * time.Second,
	}

	enforcementPolicy := keepalive.EnforcementPolicy{
		PermitWithoutStream: cfg.Server.GRPC.Keepalive.PermitWithoutStream,
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(middleware.ChainUnaryServer(
			middleware.RecoveryInterceptor,
			middleware.TraceInterceptor,
		)),
		grpc.KeepaliveEnforcementPolicy(enforcementPolicy),
		grpc.MaxRecvMsgSize(cfg.Server.GRPC.MaxReceiveMessageSize),
		grpc.MaxSendMsgSize(cfg.Server.GRPC.MaxSendMessageSize),
		grpc.KeepaliveParams(keepaliveParams),
	}

	// 创建gRPC服务器
	logger.InfoF("正在创建gRPC服务器...")
	grpcServer := grpc.NewServer(opts...)
	logger.InfoF("gRPC服务器创建成功")

	// 创建处理器
	logger.InfoF("正在创建服务器操作处理器...")
	handler := handlers.NewServerOperatorHandler(cfg)
	logger.InfoF("服务器操作处理器创建成功")

	// 注册服务
	logger.InfoF("正在注册gRPC服务...")
	server_operator.RegisterServerOperatorServiceServer(grpcServer, handler)
	logger.InfoF("gRPC服务注册成功")

	// 启动服务器
	logger.InfoF("正在启动gRPC服务器...")
	go func() {
		logger.InfoF("gRPC服务器开始监听: %s", addr)
		if err := grpcServer.Serve(lis); err != nil {
			logger.ErrorF("gRPC服务器错误: %v", err)
		}
	}()

	logger.InfoF("gRPC服务器启动完成，等待上下文取消...")

	// 等待上下文取消
	<-ctx.Done()

	// 优雅关闭
	logger.InfoF("正在关闭gRPC服务器...")
	grpcServer.GracefulStop()
	logger.InfoF("gRPC服务器已关闭")

	return nil
}
