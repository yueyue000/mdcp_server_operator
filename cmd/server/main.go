package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wumitech-com/mdcp_common/logger"
	"github.com/wumitech-com/mdcp_server_operator/internal/config"
	"github.com/wumitech-com/mdcp_server_operator/server"
)

var configPath = flag.String("config", "configs/testing.yaml", "配置文件路径")

func main() {
	flag.Parse()

	// 设置全局时区为东八区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Printf("[%s] ❌ 加载时区失败: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), err)
		os.Exit(1)
	}
	time.Local = loc
	fmt.Printf("[%s] 🌏 已设置时区为: %s (当前时间: %s)\n", time.Now().Format("2006-01-02 15:04:05.000"), loc.String(), time.Now().Format("2006-01-02 15:04:05 MST"))

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 加载配置
	if err := config.LoadConfig(*configPath); err != nil {
		fmt.Printf("[%s] ❌ 加载配置失败: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), err)
		os.Exit(1)
	}
	cfg := config.GetConfig()

	// 初始化日志
	if err := logger.InitFromConfig(&cfg.Logging); err != nil {
		fmt.Printf("[%s] ❌ 初始化日志失败: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), err)
		os.Exit(1)
	}

	// 创建错误通道
	errChan := make(chan error, 1)

	// 启动 gRPC 服务器
	go func() {
		fmt.Printf("[%s] 🚀 正在启动gRPC服务器 (%s:%d)...\n", time.Now().Format("2006-01-02 15:04:05.000"), cfg.Server.GRPC.Host, cfg.Server.GRPC.Port)
		if err := server.RunGRPCServer(ctx, cfg); err != nil {
			errChan <- fmt.Errorf("gRPC服务器错误: %v", err)
		}
	}()

	// 等待gRPC服务器启动
	time.Sleep(time.Second)
	fmt.Printf("[%s] ✅ 服务器启动完成！\n", time.Now().Format("2006-01-02 15:04:05.000"))
	fmt.Printf("[%s] 📍 gRPC服务地址: %s:%d\n", time.Now().Format("2006-01-02 15:04:05.000"), cfg.Server.GRPC.Host, cfg.Server.GRPC.Port)

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待错误或信号
	select {
	case err := <-errChan:
		fmt.Printf("[%s] ❌ 服务器错误: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), err)
		logger.ErrorF("服务器错误: %v", err)
	case sig := <-sigChan:
		fmt.Printf("\n[%s] 📢 收到信号: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), sig)
		logger.InfoF("收到信号: %v", sig)
	}

	// 优雅关闭
	fmt.Printf("[%s] 🔄 正在关闭服务器...\n", time.Now().Format("2006-01-02 15:04:05.000"))
	cancel() // 触发上下文取消
	// 刷新日志缓冲
	_ = logger.Sync()

	// 给服务器一些时间来完成关闭
	time.Sleep(time.Second)
	fmt.Printf("[%s] 👋 服务器已关闭\n", time.Now().Format("2006-01-02 15:04:05.000"))
}

