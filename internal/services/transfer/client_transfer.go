package transfer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"rdma-burst/internal/models"
	"rdma-burst/internal/wrapper"
)

// ClientTransferService 客户端传输服务
type ClientTransferService struct {
	serverURL     string // 服务端API地址
	client        *http.Client
	rtranfilePath string // rtranfile工具路径
	config        *models.TransferSettings // 客户端配置
}

// NewClientTransferService 创建新的客户端传输服务
func NewClientTransferService(serverHost string, serverPort int, config *models.TransferSettings) *ClientTransferService {
	return &ClientTransferService{
		serverURL:     fmt.Sprintf("http://%s:%d/api/v1", serverHost, serverPort),
		rtranfilePath: "/usr/local/bin/rtranfile", // 默认rtranfile路径
		config:        config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientTransferServiceWithPath 使用指定rtranfile路径创建客户端传输服务
func NewClientTransferServiceWithPath(serverHost string, serverPort int, rtranfilePath string, config *models.TransferSettings) *ClientTransferService {
	return &ClientTransferService{
		serverURL:     fmt.Sprintf("http://%s:%d/api/v1", serverHost, serverPort),
		rtranfilePath: rtranfilePath,
		config:        config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateTransfer 通过服务端API创建传输任务
func (cts *ClientTransferService) CreateTransfer(req *models.TransferRequest) (*models.TransferResponse, error) {
	// 准备请求体
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	// 发送请求到服务端
	resp, err := cts.client.Post(cts.serverURL+"/transfers", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("调用服务端API失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("服务端返回错误状态: %d", resp.StatusCode)
	}

	// 解析响应
	var transferResp models.TransferResponse
	if err := json.NewDecoder(resp.Body).Decode(&transferResp); err != nil {
		return nil, fmt.Errorf("解析服务端响应失败: %v", err)
	}

	// 如果服务端返回准备就绪状态，客户端在后台执行实际传输
	if transferResp.Status == models.StatusPrepared {
		// 在后台异步执行客户端传输
		go cts.executeClientTransferAsync(req, transferResp.ID)
		
		// 立即返回，不等待传输完成
		transferResp.Status = models.StatusInProgress
		transferResp.Message = "客户端传输已开始执行，请通过查询接口获取进度"
	}

	return &transferResp, nil
}

// GetTransferStatus 获取传输状态
func (cts *ClientTransferService) GetTransferStatus(taskID string) (*models.ProgressResponse, error) {
	resp, err := cts.client.Get(cts.serverURL + "/transfers/" + taskID)
	if err != nil {
		return nil, fmt.Errorf("获取传输状态失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务端返回错误状态: %d", resp.StatusCode)
	}

	var progressResp models.ProgressResponse
	if err := json.NewDecoder(resp.Body).Decode(&progressResp); err != nil {
		return nil, fmt.Errorf("解析传输状态失败: %v", err)
	}

	return &progressResp, nil
}

// ListTransfers 列出传输任务
func (cts *ClientTransferService) ListTransfers(page, size int) (*models.TaskListResponse, error) {
	url := fmt.Sprintf("%s/transfers?page=%d&size=%d", cts.serverURL, page, size)
	resp, err := cts.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取任务列表失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务端返回错误状态: %d", resp.StatusCode)
	}

	var taskListResp models.TaskListResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskListResp); err != nil {
		return nil, fmt.Errorf("解析任务列表失败: %v", err)
	}

	return &taskListResp, nil
}

// CancelTransfer 取消传输任务
func (cts *ClientTransferService) CancelTransfer(taskID string) error {
	req, err := http.NewRequest("DELETE", cts.serverURL+"/transfers/"+taskID, nil)
	if err != nil {
		return fmt.Errorf("创建取消请求失败: %v", err)
	}

	resp, err := cts.client.Do(req)
	if err != nil {
		return fmt.Errorf("取消传输任务失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务端返回错误状态: %d", resp.StatusCode)
	}

	return nil
}

// executeClientTransfer 执行客户端传输命令
func (cts *ClientTransferService) executeClientTransfer(req *models.TransferRequest) error {
	// 构建传输配置
	config, err := cts.buildTransferConfig(req)
	if err != nil {
		return fmt.Errorf("构建传输配置失败: %v", err)
	}

	// 验证配置
	rtranfileWrapper := wrapper.NewRtranfileWrapper(cts.rtranfilePath)
	if err := rtranfileWrapper.ValidateConfig(config); err != nil {
		return fmt.Errorf("传输配置验证失败: %v", err)
	}

	// 执行客户端传输命令
	fmt.Printf("正在执行客户端传输命令...\n")
	fmt.Printf("文件: %s, 模式: %s, 方向: %s\n", req.Filename, req.Mode, req.Direction)
	
	cmd, err := rtranfileWrapper.StartClient(context.Background(), config)
	if err != nil {
		return fmt.Errorf("启动客户端传输失败: %v", err)
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动客户端传输进程失败: %v", err)
	}

	// 等待传输完成
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("客户端传输执行失败: %v", err)
	}

	fmt.Printf("客户端传输完成\n")
	return nil
}

// executeClientTransferAsync 异步执行客户端传输命令
func (cts *ClientTransferService) executeClientTransferAsync(req *models.TransferRequest, taskID string) {
	fmt.Printf("开始异步执行客户端传输，任务ID: %s\n", taskID)
	
	if err := cts.executeClientTransfer(req); err != nil {
		fmt.Printf("客户端传输执行失败，任务ID: %s, 错误: %v\n", taskID, err)
	} else {
		fmt.Printf("客户端传输完成，任务ID: %s\n", taskID)
	}
}

// buildTransferConfig 构建客户端传输配置
func (cts *ClientTransferService) buildTransferConfig(req *models.TransferRequest) (*wrapper.TransferConfig, error) {
	// 使用配置中的设备设置
	device := "mlx5_0" // 默认设备
	if cts.config != nil && cts.config.Device != "" {
		device = cts.config.Device
	}

	// 使用配置中的块大小
	chunkSize := 4194304 // 默认块大小
	if cts.config != nil && cts.config.ChunkSize > 0 {
		chunkSize = cts.config.ChunkSize
	}

	config := &wrapper.TransferConfig{
		Device:    device,
		ChunkSize: chunkSize,
	}

	// 设置传输模式
	switch req.Mode {
	case models.ModeHugepages:
		config.Mode = wrapper.ModeHugepages
		// 客户端大页模式：使用当前目录或文件所在目录
		config.Directory = getFileDirectory(req.Filename)
		// 客户端：禁用大页，开启mman
		config.NoHuge = true
		config.MMan = true
	case models.ModeTmpfs:
		config.Mode = wrapper.ModeTmpfs
		// 客户端tmpfs模式：使用当前目录或文件所在目录
		config.Directory = getFileDirectory(req.Filename)
		// 客户端：开启大页，开启mman
		config.NoHuge = false
		config.MMan = true
	case models.ModeFilesystem:
		config.Mode = wrapper.ModeFilesystem
		// 文件系统模式：使用文件所在目录作为工作目录
		config.Directory = getFileDirectory(req.Filename)
		// 客户端：开启大页，禁用mman
		config.NoHuge = false
		config.MMan = false
	default:
		return nil, fmt.Errorf("不支持的传输模式: %s", req.Mode)
	}

	// 设置传输方向
	switch req.Direction {
	case models.DirectionPut:
		config.Direction = wrapper.DirectionPut
		config.Filename = req.Filename
	case models.DirectionGet:
		config.Direction = wrapper.DirectionGet
		config.Filename = req.Filename
	default:
		return nil, fmt.Errorf("不支持的传输方向: %s", req.Direction)
	}

	// 设置服务端地址（从服务端URL中提取）
	// 假设服务端URL格式为 http://host:port/api/v1
	serverHost := cts.serverURL
	if len(serverHost) > 7 { // 跳过 "http://"
		serverHost = serverHost[7:]
	}
	// 移除端口和路径部分
	if idx := strings.Index(serverHost, ":"); idx > 0 {
		serverHost = serverHost[:idx]
	}
	config.ServerAddress = serverHost

	// 设置日志文件
	config.LogFile = fmt.Sprintf("/var/log/rtrans/client_%s_%s.log", req.Direction, time.Now().Format("20060102_150405"))

	return config, nil
}
