package main

import (
	"fmt"
	"log"
	"os"

	"rdma-burst/internal/models"
	"rdma-burst/internal/services/config"
)

func main() {
	// 测试配置文件路径
	configPath := "configs/combined.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("配置文件不存在: %s", configPath)
	}

	fmt.Printf("测试配置文件: %s\n", configPath)

	// 测试客户端配置解析
	fmt.Println("\n=== 测试客户端配置解析 ===")
	clientManager := config.NewConfigManager("client")
	
	// 先测试直接获取配置值
	fmt.Println("直接获取配置值:")
	if timeout := clientManager.GetConfigValue("client.timeout"); timeout != nil {
		fmt.Printf("  - client.timeout: %v (类型: %T)\n", timeout, timeout)
	}
	if logPath := clientManager.GetConfigValue("logging.client.file_path"); logPath != nil {
		fmt.Printf("  - logging.client.file_path: %v (类型: %T)\n", logPath, logPath)
	}
	
	clientConfig, err := clientManager.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("客户端配置解析失败: %v", err)
	}

	if cfg, ok := clientConfig.(*models.ClientConfig); ok {
		fmt.Printf("客户端配置解析成功:\n")
		fmt.Printf("  - 服务端主机: %s\n", cfg.Server.Host)
		fmt.Printf("  - 服务端端口: %d\n", cfg.Server.Port)
		fmt.Printf("  - 连接超时: %v (类型: %T)\n", cfg.Server.Timeout, cfg.Server.Timeout)
		fmt.Printf("  - 重试延迟: %v (类型: %T)\n", cfg.Server.RetryDelay, cfg.Server.RetryDelay)
		fmt.Printf("  - 传输间隔: %v (类型: %T)\n", cfg.Transfer.TransferInterval, cfg.Transfer.TransferInterval)
		fmt.Printf("  - 日志文件路径: %s\n", cfg.Logging.FilePath)
	} else {
		log.Fatalf("客户端配置类型断言失败")
	}

	// 测试服务端配置解析
	fmt.Println("\n=== 测试服务端配置解析 ===")
	serverManager := config.NewConfigManager("server")
	serverConfig, err := serverManager.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("服务端配置解析失败: %v", err)
	}

	if cfg, ok := serverConfig.(*models.ServerConfig); ok {
		fmt.Printf("服务端配置解析成功:\n")
		fmt.Printf("  - 服务端主机: %s\n", cfg.Server.Host)
		fmt.Printf("  - 服务端端口: %d\n", cfg.Server.Port)
		fmt.Printf("  - 读取超时: %v\n", cfg.Server.ReadTimeout)
		fmt.Printf("  - 写入超时: %v\n", cfg.Server.WriteTimeout)
		fmt.Printf("  - 传输间隔: %v\n", cfg.Transfer.TransferInterval)
	} else {
		log.Fatalf("服务端配置类型断言失败")
	}

	fmt.Println("\n=== 配置解析测试完成 ===")
}