package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
)

func main() {
	// 测试配置文件路径
	configPath := "configs/combined.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	fmt.Printf("测试配置文件: %s\n", configPath)

	// 创建 Viper 实例
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	fmt.Println("\n=== 直接获取配置值 ===")
	
	// 测试客户端超时配置
	timeout := v.Get("client.timeout")
	fmt.Printf("client.timeout: %v (类型: %T)\n", timeout, timeout)
	
	// 如果是字符串，尝试解析
	if strVal, ok := timeout.(string); ok {
		fmt.Printf("  是字符串，尝试解析...\n")
		if duration, err := time.ParseDuration(strVal); err == nil {
			fmt.Printf("  解析成功: %v\n", duration)
			// 设置回 Viper
			v.Set("client.timeout", duration)
			fmt.Printf("  设置后: %v (类型: %T)\n", v.Get("client.timeout"), v.Get("client.timeout"))
		} else {
			fmt.Printf("  解析失败: %v\n", err)
		}
	}

	// 测试其他时间字段
	fields := []string{
		"client.retry_delay",
		"server.read_timeout", 
		"server.write_timeout",
		"transfer.transfer_interval",
	}

	for _, field := range fields {
		val := v.Get(field)
		fmt.Printf("%s: %v (类型: %T)\n", field, val, val)
		if strVal, ok := val.(string); ok {
			if duration, err := time.ParseDuration(strVal); err == nil {
				fmt.Printf("  解析为: %v\n", duration)
			}
		}
	}

	fmt.Println("\n=== 测试完成 ===")
}