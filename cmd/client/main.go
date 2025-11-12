package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"

	"rdma-burst/internal/models"
	"rdma-burst/internal/services/config"
	"rdma-burst/pkg/logger"
)

const (
	version = "1.0.0"
)

func main() {
	// 初始化日志
	logger, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()

	// 加载配置
	configManager := config.NewConfigManager("client")
	configPath := getConfigPath()
	
	clientConfig, err := configManager.LoadConfig(configPath)
	if err != nil {
		logger.Fatal("加载配置失败", zap.Error(err))
	}

	cfg := clientConfig.(*models.ClientConfig)

	// 解析命令行参数
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "transfer":
		handleTransferCommand(cfg, logger)
	case "status":
		handleStatusCommand(cfg, logger)
	case "list":
		handleListCommand(cfg, logger)
	case "cancel":
		handleCancelCommand(cfg, logger)
	case "health":
		handleHealthCommand(cfg, logger)
	default:
		fmt.Printf("未知命令: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// handleTransferCommand 处理传输命令
func handleTransferCommand(cfg *models.ClientConfig, logger *zap.Logger) {
	if len(os.Args) < 5 {
		fmt.Println("用法: client transfer <filename> <mode> <direction> [server_ip]")
		fmt.Println("模式: hugepages, tmpfs, filesystem")
		fmt.Println("方向: put (上传), get (下载)")
		os.Exit(1)
	}

	filename := os.Args[2]
	mode := os.Args[3]
	direction := os.Args[4]
	
	var serverIP string
	if len(os.Args) > 5 {
		serverIP = os.Args[5]
	} else {
		serverIP = cfg.Server.Host
	}

	// 构建传输请求
	req := &models.TransferRequest{
		Filename:  filename,
		Mode:      mode,
		Direction: direction,
		ServerIP:  serverIP,
	}

	// 发送传输请求
	client := createHTTPClient(cfg)
	url := fmt.Sprintf("http://%s:%d/api/v1/transfers", cfg.Server.Host, cfg.Server.Port)

	response, err := sendTransferRequest(client, url, req)
	if err != nil {
		logger.Error("传输请求失败", zap.Error(err))
		os.Exit(1)
	}

	fmt.Printf("传输任务已创建:\n")
	fmt.Printf("任务ID: %s\n", response.ID)
	fmt.Printf("状态: %s\n", response.Status)
	fmt.Printf("消息: %s\n", response.Message)
	fmt.Printf("创建时间: %s\n", response.CreatedAt.Format(time.RFC3339))
}

// handleStatusCommand 处理状态查询命令
func handleStatusCommand(cfg *models.ClientConfig, logger *zap.Logger) {
	if len(os.Args) < 3 {
		fmt.Println("用法: client status <task_id>")
		os.Exit(1)
	}

	taskID := os.Args[2]

	// 查询传输状态
	client := createHTTPClient(cfg)
	url := fmt.Sprintf("http://%s:%d/api/v1/transfers/%s", cfg.Server.Host, cfg.Server.Port, taskID)

	status, err := getTransferStatus(client, url)
	if err != nil {
		logger.Error("查询状态失败", zap.Error(err))
		os.Exit(1)
	}

	fmt.Printf("传输任务状态:\n")
	fmt.Printf("任务ID: %s\n", status.ID)
	fmt.Printf("状态: %s\n", status.Status)
	fmt.Printf("进度: %.2f%%\n", status.Progress)
	fmt.Printf("已传输: %d / %d 字节\n", status.BytesTransferred, status.TotalBytes)
	fmt.Printf("传输速率: %.2f MB/s\n", status.TransferRate)
	fmt.Printf("已用时间: %s\n", status.ElapsedTime)
	
	if status.EstimatedTime != "" {
		fmt.Printf("预计剩余: %s\n", status.EstimatedTime)
	}
	
	if status.Error != "" {
		fmt.Printf("错误: %s\n", status.Error)
	}
}

// handleListCommand 处理列表命令
func handleListCommand(cfg *models.ClientConfig, logger *zap.Logger) {
	page := 1
	size := 20

	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &page)
	}
	if len(os.Args) > 3 {
		fmt.Sscanf(os.Args[3], "%d", &size)
	}

	// 获取任务列表
	client := createHTTPClient(cfg)
	url := fmt.Sprintf("http://%s:%d/api/v1/transfers?page=%d&size=%d", cfg.Server.Host, cfg.Server.Port, page, size)

	taskList, err := getTaskList(client, url)
	if err != nil {
		logger.Error("获取任务列表失败", zap.Error(err))
		os.Exit(1)
	}

	fmt.Printf("传输任务列表 (第 %d 页, 每页 %d 条, 共 %d 条):\n", taskList.Page, taskList.Size, taskList.Total)
	fmt.Println("==================================================================")
	
	for i, task := range taskList.Tasks {
		fmt.Printf("%d. 任务ID: %s\n", i+1, task.ID)
		fmt.Printf("   文件名: %s\n", task.Filename)
		fmt.Printf("   模式: %s, 方向: %s\n", task.Mode, task.Direction)
		fmt.Printf("   状态: %s, 进度: %.2f%%\n", task.Status, task.Progress)
		fmt.Printf("   创建时间: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println("   ---")
	}
}

// handleCancelCommand 处理取消命令
func handleCancelCommand(cfg *models.ClientConfig, logger *zap.Logger) {
	if len(os.Args) < 3 {
		fmt.Println("用法: client cancel <task_id>")
		os.Exit(1)
	}

	taskID := os.Args[2]

	// 取消传输任务
	client := createHTTPClient(cfg)
	url := fmt.Sprintf("http://%s:%d/api/v1/transfers/%s", cfg.Server.Host, cfg.Server.Port, taskID)

	response, err := cancelTransfer(client, url)
	if err != nil {
		logger.Error("取消任务失败", zap.Error(err))
		os.Exit(1)
	}

	fmt.Printf("任务取消成功:\n")
	fmt.Printf("任务ID: %s\n", response.ID)
	fmt.Printf("状态: %s\n", response.Status)
	fmt.Printf("消息: %s\n", response.Message)
}

// handleHealthCommand 处理健康检查命令
func handleHealthCommand(cfg *models.ClientConfig, logger *zap.Logger) {
	// 检查服务健康状态
	client := createHTTPClient(cfg)
	url := fmt.Sprintf("http://%s:%d/api/health", cfg.Server.Host, cfg.Server.Port)

	health, err := checkHealth(client, url)
	if err != nil {
		logger.Error("健康检查失败", zap.Error(err))
		os.Exit(1)
	}

	fmt.Printf("服务健康状态:\n")
	fmt.Printf("状态: %s\n", health.Status)
	fmt.Printf("版本: %s\n", health.Version)
	fmt.Printf("时间: %s\n", health.Timestamp)
}

// createHTTPClient 创建 HTTP 客户端
func createHTTPClient(cfg *models.ClientConfig) *http.Client {
	return &http.Client{
		Timeout: cfg.Server.Timeout,
	}
}

// sendTransferRequest 发送传输请求
func sendTransferRequest(client *http.Client, url string, req *models.TransferRequest) (*models.TransferResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errorResp models.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("请求失败: %s", resp.Status)
		}
		return nil, fmt.Errorf("%s: %s", errorResp.Error, errorResp.Message)
	}

	var response models.TransferResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// getTransferStatus 获取传输状态
func getTransferStatus(client *http.Client, url string) (*models.ProgressResponse, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp models.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("请求失败: %s", resp.Status)
		}
		return nil, fmt.Errorf("%s: %s", errorResp.Error, errorResp.Message)
	}

	var status models.ProgressResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

// getTaskList 获取任务列表
func getTaskList(client *http.Client, url string) (*models.TaskListResponse, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp models.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("请求失败: %s", resp.Status)
		}
		return nil, fmt.Errorf("%s: %s", errorResp.Error, errorResp.Message)
	}

	var taskList models.TaskListResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskList); err != nil {
		return nil, err
	}

	return &taskList, nil
}

// cancelTransfer 取消传输任务
func cancelTransfer(client *http.Client, url string) (*models.TransferResponse, error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp models.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("请求失败: %s", resp.Status)
		}
		return nil, fmt.Errorf("%s: %s", errorResp.Error, errorResp.Message)
	}

	var response models.TransferResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// checkHealth 检查健康状态
func checkHealth(client *http.Client, url string) (*models.HealthResponse, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务不可用: %s", resp.Status)
	}

	var health models.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, err
	}

	return &health, nil
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	if path := os.Getenv("RDMA_CONFIG_PATH"); path != "" {
		return path
	}
	return "./configs/client.yaml"
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Println("RDMA 文件传输客户端")
	fmt.Println("版本:", version)
	fmt.Println()
	fmt.Println("用法: client <command> [arguments]")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  transfer <filename> <mode> <direction> [server_ip]")
	fmt.Println("      创建新的传输任务")
	fmt.Println("  status <task_id>")
	fmt.Println("      查询传输任务状态")
	fmt.Println("  list [page] [size]")
	fmt.Println("      列出传输任务")
	fmt.Println("  cancel <task_id>")
	fmt.Println("      取消传输任务")
	fmt.Println("  health")
	fmt.Println("      检查服务健康状态")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  client transfer data.txt filesystem put 192.168.1.100")
	fmt.Println("  client status task_1234567890")
	fmt.Println("  client list 1 10")
	fmt.Println("  client cancel task_1234567890")
	fmt.Println("  client health")
}