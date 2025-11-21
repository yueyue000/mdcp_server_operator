package phone

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPingTimeout = 3 // 默认Ping超时时间（秒）
	defaultPingCount   = 3 // 默认Ping次数
)

// ExecutePing 执行Ping命令检测网络连通性
func ExecutePing(ipAddress string, timeout, count int32) (bool, float64, error) {
	if timeout <= 0 {
		timeout = defaultPingTimeout
	}
	if count <= 0 {
		count = defaultPingCount
	}

	// 使用ping命令检测网络
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout+1)*time.Second)
	defer cancel()

	// 直接使用ping命令（与mdcp_core_executor保持一致）
	// 容器使用Docker网络（online-hk_mdcp-network），可以直接访问宿主机网络中的IP
	pingArgs := []string{
		"-c", strconv.Itoa(int(count)),
		"-W", strconv.Itoa(int(timeout)),
		ipAddress,
	}
	cmd := exec.CommandContext(ctx, "ping", pingArgs...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// 如果返回非0退出码，说明ping失败
		if ctx.Err() == context.DeadlineExceeded {
			return false, 0, fmt.Errorf("ping超时")
		}
		return false, 0, nil
	}

	// 解析ping输出，提取平均延迟
	// 输出格式示例: "round-trip min/avg/max/stddev = 1.234/2.345/3.456/0.123 ms"
	outputStr := string(output)
	avgLatency := parsePingLatency(outputStr)

	return true, avgLatency, nil
}

// parsePingLatency 解析ping输出中的平均延迟
func parsePingLatency(output string) float64 {
	// 匹配 "min/avg/max/stddev = xx.xx/yy.yy/zz.zz/ww.ww ms" 或类似格式
	re := regexp.MustCompile(`min/avg/max/(?:stddev|mdev)\s*=\s*[\d.]+/([\d.]+)/[\d.]+/[\d.]+\s*ms`)
	matches := re.FindStringSubmatch(output)

	if len(matches) >= 2 {
		latency, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return latency
		}
	}

	return 0
}

// CompareMACAddress 比较两个MAC地址是否相同（忽略大小写和分隔符差异）
func CompareMACAddress(mac1, mac2 string) bool {
	// 移除分隔符并转为小写
	normalize := func(mac string) string {
		mac = strings.ToLower(mac)
		mac = strings.ReplaceAll(mac, ":", "")
		mac = strings.ReplaceAll(mac, "-", "")
		return mac
	}

	return normalize(mac1) == normalize(mac2)
}
