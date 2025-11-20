package phone

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultADBTimeout = 3    // 默认ADB超时时间（秒）
	adbPort           = 5555 // ADB连接端口
)

// GetSerialNumberViaADB 通过ADB获取设备的SN码
func GetSerialNumberViaADB(ipAddress string, timeout int32) (string, error) {
	if timeout <= 0 {
		timeout = defaultADBTimeout
	}

	deviceAddr := fmt.Sprintf("%s:%d", ipAddress, adbPort)

	// 先尝试连接设备
	connectCtx, connectCancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer connectCancel()

	connectCmd := exec.CommandContext(connectCtx, "adb", "connect", deviceAddr)
	_, err := connectCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ADB连接失败: %v", err)
	}

	// 等待连接稳定
	time.Sleep(500 * time.Millisecond)

	// 获取序列号
	getpropCtx, getpropCancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer getpropCancel()

	cmd := exec.CommandContext(getpropCtx, "adb", "-s", deviceAddr, "shell", "getprop", "ro.serialno")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("执行getprop命令失败: %v", err)
	}

	sn := strings.TrimSpace(string(output))
	if sn == "" || strings.HasPrefix(sn, "error") {
		return "", fmt.Errorf("获取到的SN码无效")
	}

	return sn, nil
}

// GetMACAddressViaADB 通过ADB获取设备的MAC地址
func GetMACAddressViaADB(ipAddress string, timeout int32) (string, error) {
	if timeout <= 0 {
		timeout = defaultADBTimeout
	}

	deviceAddr := fmt.Sprintf("%s:%d", ipAddress, adbPort)

	// 尝试连接设备（如果已连接会直接返回）
	connectCtx, connectCancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer connectCancel()

	connectCmd := exec.CommandContext(connectCtx, "adb", "connect", deviceAddr)
	_, _ = connectCmd.CombinedOutput() // 忽略错误，继续尝试获取MAC

	// 等待连接稳定
	time.Sleep(300 * time.Millisecond)

	// 方法1：通过 ip addr 获取（优先）
	getipCtx1, getipCancel1 := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer getipCancel1()

	// 获取 eth0 或 wlan0 的 MAC 地址
	cmd1 := exec.CommandContext(getipCtx1, "adb", "-s", deviceAddr, "shell", "ip addr show eth0 | grep 'link/ether' | awk '{print $2}'")
	output1, err1 := cmd1.CombinedOutput()

	if err1 == nil && len(output1) > 0 {
		mac := strings.TrimSpace(string(output1))
		if mac != "" && len(mac) == 17 { // MAC 地址格式: xx:xx:xx:xx:xx:xx
			return strings.ToLower(mac), nil
		}
	}

	// 方法2：直接读取系统文件
	getipCtx2, getipCancel2 := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer getipCancel2()

	cmd2 := exec.CommandContext(getipCtx2, "adb", "-s", deviceAddr, "shell", "cat /sys/class/net/eth0/address")
	output2, err2 := cmd2.CombinedOutput()

	if err2 == nil && len(output2) > 0 {
		mac := strings.TrimSpace(string(output2))
		if mac != "" && len(mac) == 17 {
			return strings.ToLower(mac), nil
		}
	}

	// 方法3：尝试 wlan0（无线网络）
	getipCtx3, getipCancel3 := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer getipCancel3()

	cmd3 := exec.CommandContext(getipCtx3, "adb", "-s", deviceAddr, "shell", "cat /sys/class/net/wlan0/address")
	output3, err3 := cmd3.CombinedOutput()

	if err3 == nil && len(output3) > 0 {
		mac := strings.TrimSpace(string(output3))
		if mac != "" && len(mac) == 17 {
			return strings.ToLower(mac), nil
		}
	}

	return "", fmt.Errorf("无法通过ADB获取MAC地址")
}

// ExecutePhoneCommand 执行云手机ADB命令
func ExecutePhoneCommand(ipAddress, command string, timeout int32) (string, string, int32, error) {
	if timeout <= 0 {
		timeout = 30 // 默认30秒
	}

	deviceAddr := fmt.Sprintf("%s:%d", ipAddress, adbPort)

	// 先尝试连接设备
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer connectCancel()

	connectCmd := exec.CommandContext(connectCtx, "adb", "connect", deviceAddr)
	_, _ = connectCmd.CombinedOutput() // 忽略连接错误，继续执行命令

	// 等待连接稳定
	time.Sleep(300 * time.Millisecond)

	// 执行命令
	execCtx, execCancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer execCancel()

	cmd := exec.CommandContext(execCtx, "adb", "-s", deviceAddr, "shell", command)
	output, err := cmd.CombinedOutput()

	exitCode := int32(0)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = int32(exitError.ExitCode())
		} else {
			exitCode = -1
		}
		return "", string(output), exitCode, fmt.Errorf("执行命令失败: %v", err)
	}

	return string(output), "", exitCode, nil
}
