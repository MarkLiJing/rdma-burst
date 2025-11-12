package main

import (
	"fmt"
	"log"
	"strings"

	"rdma-burst/internal/utils"
)

func main() {
	// 测试根据RDMA设备获取IP地址
	fmt.Println("=== 测试RDMA设备到IP地址映射 ===")
	
	// 测试不同的RDMA设备
	devices := []string{"mlx5_0", "mlx5_1", "mlx5_2", "ib0", "ib1"}
	
	for _, device := range devices {
		fmt.Printf("\n测试设备: %s\n", device)
		ip, err := utils.GetIPFromRDMAInterface(device)
		if err != nil {
			fmt.Printf("  错误: %v\n", err)
		} else {
			fmt.Printf("  获取到的IP: %s\n", ip)
		}
	}
	
	// 测试获取本地IP
	fmt.Println("\n=== 测试获取本地IP ===")
	localIP, err := utils.GetLocalIP()
	if err != nil {
		log.Printf("获取本地IP失败: %v", err)
	} else {
		fmt.Printf("本地IP: %s\n", localIP)
	}
	
	// 测试网络接口推断
	fmt.Println("\n=== 测试网络接口推断 ===")
	testInferInterface()
}

func testInferInterface() {
	// 测试接口推断逻辑
	devices := []string{"mlx5_0", "mlx5_1", "mlx5_2", "ib0", "ib1", "unknown"}
	
	for _, device := range devices {
		// 这里我们直接测试推断逻辑
		var inferredInterface string
		
		if len(device) >= 5 && device[:5] == "mlx5_" {
			parts := strings.Split(device, "_")
			if len(parts) >= 2 {
				inferredInterface = fmt.Sprintf("ib%s", parts[1])
			}
		} else if len(device) >= 2 && device[:2] == "ib" {
			inferredInterface = device
		}
		
		if inferredInterface != "" {
			fmt.Printf("设备 %s -> 推断接口: %s\n", device, inferredInterface)
		} else {
			fmt.Printf("设备 %s -> 无法推断接口\n", device)
		}
	}
}