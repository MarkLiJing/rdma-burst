package utils

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// GetIPFromRDMAInterface 根据RDMA设备名称获取对应的IP地址
func GetIPFromRDMAInterface(rdmaDevice string) (string, error) {
	// RDMA设备通常与网络接口有对应关系
	// 例如 mlx5_0 -> ib0, mlx5_1 -> ib1 等
	
	// 尝试从设备名称推断网络接口
	interfaceName := inferInterfaceFromRDMA(rdmaDevice)
	if interfaceName == "" {
		return "", fmt.Errorf("无法从RDMA设备 %s 推断网络接口", rdmaDevice)
	}
	
	// 获取网络接口的IP地址
	return getInterfaceIP(interfaceName)
}

// inferInterfaceFromRDMA 从RDMA设备名称推断网络接口名称
func inferInterfaceFromRDMA(rdmaDevice string) string {
	// 常见的RDMA设备到网络接口的映射
	// mlx5_0 -> ib0, mlx5_1 -> ib1, 等等
	if strings.HasPrefix(rdmaDevice, "mlx5_") {
		// 提取数字部分
		parts := strings.Split(rdmaDevice, "_")
		if len(parts) >= 2 {
			return fmt.Sprintf("ib%s", parts[1])
		}
	}
	
	// 尝试其他常见的RDMA设备前缀
	if strings.HasPrefix(rdmaDevice, "ib") {
		return rdmaDevice
	}
	
	// 如果无法推断，尝试查找可用的InfiniBand接口
	return findAvailableIBInterface()
}

// findAvailableIBInterface 查找可用的InfiniBand接口
func findAvailableIBInterface() string {
	// 读取网络接口信息
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return ""
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 查找以"ib"开头的接口
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				interfaceName := strings.TrimSpace(parts[0])
				if strings.HasPrefix(interfaceName, "ib") {
					return interfaceName
				}
			}
		}
	}
	
	return ""
}

// getInterfaceIP 获取网络接口的IP地址
func getInterfaceIP(interfaceName string) (string, error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return "", fmt.Errorf("无法找到网络接口 %s: %v", interfaceName, err)
	}
	
	addrs, err := iface.Addrs()
	if err != nil {
		return "", fmt.Errorf("无法获取接口 %s 的地址: %v", interfaceName, err)
	}
	
	for _, addr := range addrs {
		// 检查是否是IP地址（不是CIDR）
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		
		// 只返回IPv4地址
		if ip != nil && ip.To4() != nil {
			return ip.String(), nil
		}
	}
	
	return "", fmt.Errorf("接口 %s 没有可用的IPv4地址", interfaceName)
}

// GetLocalIP 获取本地主机的IP地址（回退方案）
func GetLocalIP() (string, error) {
	// 尝试通过连接到外部地址来获取本地IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// 如果失败，尝试获取第一个非回环地址
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return "", err
		}
		
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String(), nil
				}
			}
		}
		return "", fmt.Errorf("无法获取本地IP地址")
	}
	defer conn.Close()
	
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}