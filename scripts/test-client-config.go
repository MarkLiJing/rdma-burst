package main

import (
	"fmt"
	"log"

	"rdma-burst/internal/models"
	"rdma-burst/internal/services/config"
)

func main() {
	// 测试客户端配置加载
	fmt.Println("=== 测试客户端配置加载 ===")
	
	configPath := "./configs/combined.yaml"
	
	// 创建客户端配置管理器
	configManager := config.NewConfigManager("client")
	
	// 加载配置
	clientConfig, err := configManager.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	
	// 打印配置信息
	if cfg, ok := clientConfig.(*models.ClientConfig); ok {
		fmt.Printf("客户端配置加载成功:\n")
		fmt.Printf("  服务端地址: %s\n", cfg.Server.Host)
		fmt.Printf("  服务端端口: %d\n", cfg.Server.Port)
		fmt.Printf("  RDMA设备: %s\n", cfg.Transfer.Device)
	} else {
		fmt.Printf("配置类型断言失败\n")
	}
}