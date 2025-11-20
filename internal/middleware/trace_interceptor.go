package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/wumitech-com/mdcp_common/enum"
	"github.com/wumitech-com/mdcp_common/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// TraceIDKey metadata中trace_id的键名
	TraceIDKey = "trace-id"
	// TraceIDLength trace_id的字节长度（128位 = 16字节）
	TraceIDLength = 16
)

// generateTraceID 生成符合OpenTelemetry标准的128位trace_id
func generateTraceID() string {
	// 生成16字节随机数据
	traceBytes := make([]byte, TraceIDLength)

	// 使用crypto/rand生成高质量随机数
	_, err := rand.Read(traceBytes)
	if err != nil {
		// 如果crypto/rand失败，使用时间戳作为fallback
		return generateTimeBasedTraceID()
	}

	// 确保非零值（OpenTelemetry规范要求）
	if isAllZeros(traceBytes) {
		// 如果随机生成全零（极低概率），设置最后一个字节为1
		traceBytes[TraceIDLength-1] = 1
	}

	// 转换为32字符小写十六进制字符串
	return hex.EncodeToString(traceBytes)
}

// generateTimeBasedTraceID 基于时间戳生成trace_id（fallback方案）
func generateTimeBasedTraceID() string {
	now := time.Now()

	// 使用纳秒时间戳确保唯一性
	timestamp := now.UnixNano()

	// 构造16字节数据
	traceBytes := make([]byte, TraceIDLength)

	// 前8字节：纳秒时间戳
	for i := 0; i < 8; i++ {
		traceBytes[i] = byte(timestamp >> (8 * (7 - i)))
	}

	// 后8字节：进程ID + 微秒 + 计数器（模拟随机性）
	pid := uint32(1000) // 简化的进程标识
	microsecond := uint32(now.Nanosecond() / 1000)

	for i := 0; i < 4; i++ {
		traceBytes[8+i] = byte(pid >> (8 * (3 - i)))
		traceBytes[12+i] = byte(microsecond >> (8 * (3 - i)))
	}

	return hex.EncodeToString(traceBytes)
}

// isAllZeros 检查字节数组是否全为0
func isAllZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// validateTraceID 验证trace_id格式是否符合OpenTelemetry标准
func validateTraceID(traceID string) bool {
	// 检查长度（32字符）
	if len(traceID) != TraceIDLength*2 {
		return false
	}

	// 检查是否为有效的十六进制字符串
	_, err := hex.DecodeString(traceID)
	if err != nil {
		return false
	}

	// 检查是否为全零（OpenTelemetry不允许）
	decoded, _ := hex.DecodeString(traceID)
	return !isAllZeros(decoded)
}

// extractTraceIDFromContext 从gRPC metadata中提取trace_id
func extractTraceIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	traceIDs := md.Get(TraceIDKey)
	if len(traceIDs) == 0 {
		return ""
	}

	traceID := traceIDs[0]

	// 验证trace_id格式
	if !validateTraceID(traceID) {
		return ""
	}

	return traceID
}

// getOrGenerateTraceID 从上下文获取trace_id，如果不存在则生成新的
func getOrGenerateTraceID(ctx context.Context) string {
	// 尝试从metadata中提取现有trace_id
	traceID := extractTraceIDFromContext(ctx)

	if traceID != "" {
		return traceID
	}

	// 生成新的trace_id
	return generateTraceID()
}

// TraceInterceptor 是一个gRPC拦截器，确保每个请求都有trace_id
func TraceInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 获取或生成trace_id
	traceID := getOrGenerateTraceID(ctx)

	// 如果incoming metadata中没有trace_id，我们需要将生成的trace_id添加到context中
	// 这样下游的handler和服务调用就能获取到它
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	// 确保metadata中有trace_id
	existingTraceIDs := md.Get(TraceIDKey)
	if len(existingTraceIDs) == 0 || !validateTraceID(existingTraceIDs[0]) {
		md.Set(TraceIDKey, traceID)
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	// 兼容公共logger：将 Trace 写入 enum.CtxKeyTrace，便于 LogWithContext 提取
	ctx = context.WithValue(ctx, enum.CtxKeyTrace, traceID)

	logger.InfoFWithContext(ctx, "请求开始 - 方法: %s, TraceID: %s", info.FullMethod, traceID)

	// 调用实际的handler
	resp, err := handler(ctx, req)

	// 在响应中设置trace_id到outgoing metadata，确保客户端能够获取到trace_id
	if err := grpc.SendHeader(ctx, metadata.Pairs(TraceIDKey, traceID)); err != nil {
		logger.WarnFWithContext(ctx, "设置响应头trace_id失败: %v", err)
	}

	if err != nil {
		logger.ErrorFWithContext(ctx, "请求失败 - 方法: %s, TraceID: %s, 错误: %v", info.FullMethod, traceID, err)
	} else {
		logger.InfoFWithContext(ctx, "请求成功 - 方法: %s, TraceID: %s", info.FullMethod, traceID)
	}

	return resp, err
}
