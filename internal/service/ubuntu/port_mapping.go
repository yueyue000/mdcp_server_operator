package ubuntu

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/wumitech-com/mdcp_common/logger"
)

// PortMappingExecutor nftables端口映射执行器
type PortMappingExecutor struct {
	externalIP string
	targetPort string
	tableName  string
	chainName  string
}

// NewPortMappingExecutor 创建nftables端口映射执行器
func NewPortMappingExecutor(externalIP, targetPort, tableName, chainName string) *PortMappingExecutor {
	return &PortMappingExecutor{
		externalIP: externalIP,
		targetPort: targetPort,
		tableName:  tableName,
		chainName:  chainName,
	}
}

// executeNFTCommand 执行nft命令（在宿主机网络命名空间中）
func (e *PortMappingExecutor) executeNFTCommand(args ...string) (string, error) {
	// 使用nsenter进入宿主机的网络命名空间执行nft
	cmdArgs := append([]string{"-t", "1", "-n", "nft"}, args...)
	logger.InfoF("执行宿主机nft命令: nsenter %v", strings.Join(cmdArgs, " "))

	cmd := exec.Command("nsenter", cmdArgs...)

	timeout := 10 * time.Second
	timer := time.AfterFunc(timeout, func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	})
	defer timer.Stop()

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.ErrorF("nft命令执行失败: %v, 输出: %s", err, string(output))
		return string(output), fmt.Errorf("nft命令执行失败: %v", err)
	}

	logger.InfoF("nft命令执行成功, 输出: %s", string(output))
	return string(output), nil
}

// initChain 初始化nftables链并确保OUTPUT链规则存在
func (e *PortMappingExecutor) initChain() error {
	// 检查PHONE_PORT_MAPPING链是否存在
	_, err := e.executeNFTCommand("list", "chain", e.tableName, e.chainName)
	if err != nil {
		// 链不存在，创建
		logger.InfoF("创建nftables链: %s", e.chainName)
		_, err = e.executeNFTCommand("add", "chain", e.tableName, e.chainName)
		if err != nil {
			return fmt.Errorf("创建nftables链失败: %v", err)
		}
	}

	// 检查OUTPUT链是否有跳转到PHONE_PORT_MAPPING的规则
	outputChain, _ := e.executeNFTCommand("list", "chain", e.tableName, "OUTPUT")
	if !strings.Contains(outputChain, "PHONE_PORT_MAPPING") {
		// 使用insert在OUTPUT链最前面添加规则（优先级最高，不影响其他规则）
		// 只匹配目标是206.119.108.2的流量，不影响NAT、xray等其他配置
		insertCmd := fmt.Sprintf("insert rule %s OUTPUT ip daddr %s jump %s",
			e.tableName, e.externalIP, e.chainName)
		_, err = e.executeNFTCommand(strings.Split(insertCmd, " ")...)
		if err != nil {
			return fmt.Errorf("添加OUTPUT链规则失败: %v", err)
		}
		logger.InfoF("已添加OUTPUT链规则（只匹配%s，不影响其他流量）", e.externalIP)
	}

	logger.InfoF("nftables链初始化完成: %s", e.chainName)
	return nil
}

// EnablePortMapping 启用端口映射
func (e *PortMappingExecutor) EnablePortMapping(internalIP string, mappedPort int32) error {
	// 初始化链
	if err := e.initChain(); err != nil {
		return fmt.Errorf("初始化nftables链失败: %v", err)
	}

	// 添加DNAT规则
	// nft add rule ip nat PHONE_PORT_MAPPING tcp dport 10196 dnat to 192.168.87.126:5555
	addRuleCmd := fmt.Sprintf("add rule %s %s tcp dport %d dnat to %s:%s",
		e.tableName, e.chainName, mappedPort, internalIP, e.targetPort)
	_, err := e.executeNFTCommand(strings.Split(addRuleCmd, " ")...)
	if err != nil {
		return fmt.Errorf("添加DNAT规则失败: %v", err)
	}

	// 添加MASQUERADE规则
	masqueradeCmd := fmt.Sprintf("add rule %s POSTROUTING ip daddr %s tcp dport %s masquerade",
		e.tableName, internalIP, e.targetPort)
	_, err = e.executeNFTCommand(strings.Split(masqueradeCmd, " ")...)
	if err != nil {
		logger.WarnF("添加MASQUERADE规则失败: %v（可能已存在或不影响功能）", err)
	}

	logger.InfoF("端口映射已启用: %s:%d -> %s:%s", e.externalIP, mappedPort, internalIP, e.targetPort)
	return nil
}

// DisablePortMapping 禁用端口映射
func (e *PortMappingExecutor) DisablePortMapping(mappedPort int32) error {
	// 列出PHONE_PORT_MAPPING链的所有规则（带handle）
	listCmd := fmt.Sprintf("--handle list chain %s %s", e.tableName, e.chainName)
	output, err := e.executeNFTCommand(strings.Split(listCmd, " ")...)
	if err != nil {
		logger.WarnF("查询nftables规则失败: %v", err)
		return nil
	}

	// 解析输出，查找包含目标端口的规则handle
	lines := strings.Split(output, "\n")
	portStr := fmt.Sprintf("dport %d", mappedPort)

	for _, line := range lines {
		if strings.Contains(line, portStr) && strings.Contains(line, "# handle") {
			// 提取handle编号
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "handle" && i+1 < len(fields) {
					handle := fields[i+1]
					// 删除规则
					deleteCmd := fmt.Sprintf("delete rule %s %s handle %s", e.tableName, e.chainName, handle)
					_, err = e.executeNFTCommand(strings.Split(deleteCmd, " ")...)
					if err != nil {
						logger.ErrorF("删除nftables规则失败: %v", err)
						return fmt.Errorf("删除端口映射规则失败: %v", err)
					}
					logger.InfoF("端口映射已禁用: 端口 %d（handle: %s）", mappedPort, handle)
					return nil
				}
			}
		}
	}

	logger.WarnF("未找到端口 %d 的映射规则", mappedPort)
	return nil
}

// ListPortMappings 列出所有端口映射
func (e *PortMappingExecutor) ListPortMappings() (string, error) {
	listCmd := fmt.Sprintf("list chain %s %s", e.tableName, e.chainName)
	output, err := e.executeNFTCommand(strings.Split(listCmd, " ")...)
	if err != nil {
		return "", fmt.Errorf("查询端口映射失败: %v", err)
	}
	return output, nil
}

// TestConnection 测试nft是否可用
func (e *PortMappingExecutor) TestConnection() error {
	cmd := exec.Command("nsenter", "-t", "1", "-n", "nft", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nft命令不可用: %v", err)
	}

	logger.InfoF("nft版本: %s", strings.TrimSpace(string(output)))
	return nil
}

