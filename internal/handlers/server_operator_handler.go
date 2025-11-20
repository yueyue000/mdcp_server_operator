package handlers

import (
	"context"

	"github.com/wumitech-com/mdcp_common/logger"
	"github.com/wumitech-com/mdcp_proto/api/server_operator"
	"github.com/wumitech-com/mdcp_server_operator/internal/config"
	"github.com/wumitech-com/mdcp_server_operator/internal/service/phone"
	"github.com/wumitech-com/mdcp_server_operator/internal/service/ubuntu"
)

// ServerOperatorHandler 服务器操作处理器
type ServerOperatorHandler struct {
	server_operator.UnimplementedServerOperatorServiceServer
	cfg                 *config.Config
	portMappingExecutor *ubuntu.PortMappingExecutor
}

// NewServerOperatorHandler 创建服务器操作处理器
func NewServerOperatorHandler(cfg *config.Config) *ServerOperatorHandler {
	portMappingExecutor := ubuntu.NewPortMappingExecutor(
		cfg.Ubuntu.ExternalIP,
		cfg.Ubuntu.TargetPort,
		cfg.Ubuntu.TableName,
		cfg.Ubuntu.ChainName,
	)

	return &ServerOperatorHandler{
		cfg:                 cfg,
		portMappingExecutor: portMappingExecutor,
	}
}

// EnablePortMapping 启用端口映射
func (h *ServerOperatorHandler) EnablePortMapping(ctx context.Context, req *server_operator.EnablePortMappingRequest) (*server_operator.EnablePortMappingResponse, error) {
	logger.InfoFWithContext(ctx, "启用端口映射: %s:%d", req.InternalIp, req.MappedPort)

	err := h.portMappingExecutor.EnablePortMapping(ctx, req.InternalIp, req.MappedPort)
	if err != nil {
		logger.ErrorFWithContext(ctx, "启用端口映射失败: %v", err)
		return &server_operator.EnablePortMappingResponse{
			Success: false,
			Message: "启用端口映射失败: " + err.Error(),
		}, nil
	}

	logger.InfoFWithContext(ctx, "端口映射启用成功: %s:%d", req.InternalIp, req.MappedPort)
	return &server_operator.EnablePortMappingResponse{
		Success: true,
		Message: "端口映射已启用",
	}, nil
}

// DisablePortMapping 禁用端口映射
func (h *ServerOperatorHandler) DisablePortMapping(ctx context.Context, req *server_operator.DisablePortMappingRequest) (*server_operator.DisablePortMappingResponse, error) {
	logger.InfoFWithContext(ctx, "禁用端口映射: %d", req.MappedPort)

	err := h.portMappingExecutor.DisablePortMapping(ctx, req.MappedPort)
	if err != nil {
		logger.ErrorFWithContext(ctx, "禁用端口映射失败: %v", err)
		return &server_operator.DisablePortMappingResponse{
			Success: false,
			Message: "禁用端口映射失败: " + err.Error(),
		}, nil
	}

	logger.InfoFWithContext(ctx, "端口映射禁用成功: %d", req.MappedPort)
	return &server_operator.DisablePortMappingResponse{
		Success: true,
		Message: "端口映射已禁用",
	}, nil
}

// ListPortMappings 列出所有端口映射
func (h *ServerOperatorHandler) ListPortMappings(ctx context.Context, req *server_operator.ListPortMappingsRequest) (*server_operator.ListPortMappingsResponse, error) {
	logger.InfoFWithContext(ctx, "列出端口映射")

	output, err := h.portMappingExecutor.ListPortMappings(ctx)
	if err != nil {
		logger.ErrorFWithContext(ctx, "列出端口映射失败: %v", err)
		return &server_operator.ListPortMappingsResponse{
			Success: false,
			Message: "列出端口映射失败: " + err.Error(),
		}, nil
	}

	// 这里可以解析output并转换为PortMappingInfo列表
	// 为了简化，暂时返回原始输出
	return &server_operator.ListPortMappingsResponse{
		Success: true,
		Message: output,
	}, nil
}

// ExecutePhonePing 执行云手机Ping
func (h *ServerOperatorHandler) ExecutePhonePing(ctx context.Context, req *server_operator.ExecutePhonePingRequest) (*server_operator.ExecutePhonePingResponse, error) {
	logger.InfoFWithContext(ctx, "执行云手机Ping: %s", req.IpAddress)

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = int32(h.cfg.Phone.PingTimeout)
	}
	count := req.Count
	if count <= 0 {
		count = 3
	}

	success, latency, err := phone.ExecutePing(req.IpAddress, timeout, count)
	if err != nil {
		logger.ErrorFWithContext(ctx, "执行Ping失败: IP=%s, 错误=%v", req.IpAddress, err)
		return &server_operator.ExecutePhonePingResponse{
			Success: false,
			Message: "执行Ping失败: " + err.Error(),
			Timeout: err.Error() == "ping超时",
		}, nil
	}

	if success {
		logger.InfoFWithContext(ctx, "Ping成功: IP=%s, 延迟=%.2fms", req.IpAddress, latency)
	} else {
		logger.WarnFWithContext(ctx, "Ping失败: IP=%s", req.IpAddress)
	}

	return &server_operator.ExecutePhonePingResponse{
		Success: success,
		Message: "Ping执行完成",
		Latency: latency,
		Timeout: false,
	}, nil
}

// GetPhoneSerialNumber 获取云手机SN码
func (h *ServerOperatorHandler) GetPhoneSerialNumber(ctx context.Context, req *server_operator.GetPhoneSerialNumberRequest) (*server_operator.GetPhoneSerialNumberResponse, error) {
	logger.InfoFWithContext(ctx, "获取云手机SN码: IP=%s", req.IpAddress)

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = int32(h.cfg.Phone.ADBTimeout)
	}

	sn, err := phone.GetSerialNumberViaADB(req.IpAddress, timeout)
	if err != nil {
		logger.ErrorFWithContext(ctx, "获取SN码失败: IP=%s, 错误=%v", req.IpAddress, err)
		return &server_operator.GetPhoneSerialNumberResponse{
			Success:      false,
			Message:      "获取SN码失败: " + err.Error(),
			SerialNumber: "",
		}, nil
	}

	logger.InfoFWithContext(ctx, "获取SN码成功: IP=%s, SN=%s", req.IpAddress, sn)
	return &server_operator.GetPhoneSerialNumberResponse{
		Success:      true,
		Message:      "获取SN码成功",
		SerialNumber: sn,
	}, nil
}

// GetPhoneMACAddress 获取云手机MAC地址
func (h *ServerOperatorHandler) GetPhoneMACAddress(ctx context.Context, req *server_operator.GetPhoneMACAddressRequest) (*server_operator.GetPhoneMACAddressResponse, error) {
	logger.InfoFWithContext(ctx, "获取云手机MAC地址: IP=%s", req.IpAddress)

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = int32(h.cfg.Phone.ADBTimeout)
	}

	mac, err := phone.GetMACAddressViaADB(req.IpAddress, timeout)
	if err != nil {
		logger.ErrorFWithContext(ctx, "获取MAC地址失败: IP=%s, 错误=%v", req.IpAddress, err)
		return &server_operator.GetPhoneMACAddressResponse{
			Success:    false,
			Message:    "获取MAC地址失败: " + err.Error(),
			MacAddress: "",
		}, nil
	}

	logger.InfoFWithContext(ctx, "获取MAC地址成功: IP=%s, MAC=%s", req.IpAddress, mac)
	return &server_operator.GetPhoneMACAddressResponse{
		Success:    true,
		Message:    "获取MAC地址成功",
		MacAddress: mac,
	}, nil
}

// ExecutePhoneCommand 执行云手机命令
func (h *ServerOperatorHandler) ExecutePhoneCommand(ctx context.Context, req *server_operator.ExecutePhoneCommandRequest) (*server_operator.ExecutePhoneCommandResponse, error) {
	logger.InfoFWithContext(ctx, "执行云手机命令: IP=%s, 命令=%s", req.IpAddress, req.Command)

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	stdout, stderr, exitCode, err := phone.ExecutePhoneCommand(req.IpAddress, req.Command, timeout)
	if err != nil {
		logger.ErrorFWithContext(ctx, "执行命令失败: IP=%s, 命令=%s, 错误=%v, ExitCode=%d", req.IpAddress, req.Command, err, exitCode)
		return &server_operator.ExecutePhoneCommandResponse{
			Success:  false,
			Message:  "执行命令失败: " + err.Error(),
			Stdout:   stdout,
			Stderr:   stderr,
			ExitCode: exitCode,
		}, nil
	}

	if exitCode != 0 {
		logger.WarnFWithContext(ctx, "命令执行完成但退出码非0: IP=%s, 命令=%s, ExitCode=%d, Stderr=%s", req.IpAddress, req.Command, exitCode, stderr)
	} else {
		logger.InfoFWithContext(ctx, "命令执行成功: IP=%s, 命令=%s, ExitCode=%d", req.IpAddress, req.Command, exitCode)
	}

	return &server_operator.ExecutePhoneCommandResponse{
		Success:  true,
		Message:  "命令执行完成",
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}, nil
}
