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
	cfg                *config.Config
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
		cfg:                cfg,
		portMappingExecutor: portMappingExecutor,
	}
}

// EnablePortMapping 启用端口映射
func (h *ServerOperatorHandler) EnablePortMapping(ctx context.Context, req *server_operator.EnablePortMappingRequest) (*server_operator.EnablePortMappingResponse, error) {
	logger.InfoF("启用端口映射: %s:%d", req.InternalIp, req.MappedPort)

	err := h.portMappingExecutor.EnablePortMapping(req.InternalIp, req.MappedPort)
	if err != nil {
		logger.ErrorF("启用端口映射失败: %v", err)
		return &server_operator.EnablePortMappingResponse{
			Success: false,
			Message: "启用端口映射失败: " + err.Error(),
		}, nil
	}

	return &server_operator.EnablePortMappingResponse{
		Success: true,
		Message: "端口映射已启用",
	}, nil
}

// DisablePortMapping 禁用端口映射
func (h *ServerOperatorHandler) DisablePortMapping(ctx context.Context, req *server_operator.DisablePortMappingRequest) (*server_operator.DisablePortMappingResponse, error) {
	logger.InfoF("禁用端口映射: %d", req.MappedPort)

	err := h.portMappingExecutor.DisablePortMapping(req.MappedPort)
	if err != nil {
		logger.ErrorF("禁用端口映射失败: %v", err)
		return &server_operator.DisablePortMappingResponse{
			Success: false,
			Message: "禁用端口映射失败: " + err.Error(),
		}, nil
	}

	return &server_operator.DisablePortMappingResponse{
		Success: true,
		Message: "端口映射已禁用",
	}, nil
}

// ListPortMappings 列出所有端口映射
func (h *ServerOperatorHandler) ListPortMappings(ctx context.Context, req *server_operator.ListPortMappingsRequest) (*server_operator.ListPortMappingsResponse, error) {
	logger.InfoF("列出端口映射")

	output, err := h.portMappingExecutor.ListPortMappings()
	if err != nil {
		logger.ErrorF("列出端口映射失败: %v", err)
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
	logger.InfoF("执行云手机Ping: %s", req.IpAddress)

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
		logger.ErrorF("执行Ping失败: %v", err)
		return &server_operator.ExecutePhonePingResponse{
			Success: false,
			Message: "执行Ping失败: " + err.Error(),
			Timeout: err.Error() == "ping超时",
		}, nil
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
	logger.InfoF("获取云手机SN码: %s", req.IpAddress)

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = int32(h.cfg.Phone.ADBTimeout)
	}

	sn, err := phone.GetSerialNumberViaADB(req.IpAddress, timeout)
	if err != nil {
		logger.ErrorF("获取SN码失败: %v", err)
		return &server_operator.GetPhoneSerialNumberResponse{
			Success:      false,
			Message:      "获取SN码失败: " + err.Error(),
			SerialNumber: "",
		}, nil
	}

	return &server_operator.GetPhoneSerialNumberResponse{
		Success:      true,
		Message:      "获取SN码成功",
		SerialNumber: sn,
	}, nil
}

// GetPhoneMACAddress 获取云手机MAC地址
func (h *ServerOperatorHandler) GetPhoneMACAddress(ctx context.Context, req *server_operator.GetPhoneMACAddressRequest) (*server_operator.GetPhoneMACAddressResponse, error) {
	logger.InfoF("获取云手机MAC地址: %s", req.IpAddress)

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = int32(h.cfg.Phone.ADBTimeout)
	}

	mac, err := phone.GetMACAddressViaADB(req.IpAddress, timeout)
	if err != nil {
		logger.ErrorF("获取MAC地址失败: %v", err)
		return &server_operator.GetPhoneMACAddressResponse{
			Success:    false,
			Message:    "获取MAC地址失败: " + err.Error(),
			MacAddress: "",
		}, nil
	}

	return &server_operator.GetPhoneMACAddressResponse{
		Success:    true,
		Message:    "获取MAC地址成功",
		MacAddress: mac,
	}, nil
}

// ExecutePhoneCommand 执行云手机命令
func (h *ServerOperatorHandler) ExecutePhoneCommand(ctx context.Context, req *server_operator.ExecutePhoneCommandRequest) (*server_operator.ExecutePhoneCommandResponse, error) {
	logger.InfoF("执行云手机命令: %s, 命令: %s", req.IpAddress, req.Command)

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	stdout, stderr, exitCode, err := phone.ExecutePhoneCommand(req.IpAddress, req.Command, timeout)
	if err != nil {
		logger.ErrorF("执行命令失败: %v", err)
		return &server_operator.ExecutePhoneCommandResponse{
			Success:  false,
			Message:  "执行命令失败: " + err.Error(),
			Stdout:   stdout,
			Stderr:   stderr,
			ExitCode: exitCode,
		}, nil
	}

	return &server_operator.ExecutePhoneCommandResponse{
		Success:  true,
		Message:  "命令执行完成",
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}, nil
}

