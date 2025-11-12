package transfer

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"rdma-burst/internal/models"
	"rdma-burst/internal/wrapper"
)

// TransferService 传输服务
type TransferService struct {
	mu               sync.RWMutex
	rtranfile        *wrapper.RtranfileWrapper
	processMgr       *wrapper.ProcessManager
	activeTasks      map[string]*TransferTask
	taskHistory      []*models.TransferTask
	maxConcurrent    int
	transferInterval time.Duration
	lastTransferTime time.Time
	singleTransfer   bool
	requireReconnect bool
	activeConnections map[string]time.Time // 活跃连接映射
	serverProcesses  map[string]*wrapper.ProcessManager // 服务端进程映射
	serverConfig     *models.TransferSettings // 服务端配置
}

// TransferTask 传输任务包装器
type TransferTask struct {
	Task      *models.TransferTask
	Monitor   *wrapper.TransferMonitor
	Process   *wrapper.ProcessManager
	Config    *wrapper.TransferConfig
	Cancel    context.CancelFunc
}

// NewTransferService 创建新的传输服务
func NewTransferService(rtranfilePath string, maxConcurrent int, transferInterval time.Duration) *TransferService {
	return &TransferService{
		rtranfile:        wrapper.NewRtranfileWrapper(rtranfilePath),
		processMgr:       wrapper.NewProcessManager(),
		activeTasks:      make(map[string]*TransferTask),
		taskHistory:      make([]*models.TransferTask, 0),
		maxConcurrent:    maxConcurrent,
		transferInterval: transferInterval,
		lastTransferTime: time.Now(),
		singleTransfer:   true,
		requireReconnect: true,
		activeConnections: make(map[string]time.Time),
		serverProcesses:  make(map[string]*wrapper.ProcessManager),
	}
}

// NewTransferServiceWithConfig 使用配置创建传输服务
func NewTransferServiceWithConfig(rtranfilePath string, config *models.TransferSettings, singleTransferConfig *models.SingleTransferSettings) *TransferService {
	service := &TransferService{
		rtranfile:        wrapper.NewRtranfileWrapper(rtranfilePath),
		processMgr:       wrapper.NewProcessManager(),
		activeTasks:      make(map[string]*TransferTask),
		taskHistory:      make([]*models.TransferTask, 0),
		maxConcurrent:    config.MaxConcurrentTransfers,
		transferInterval: config.TransferInterval,
		lastTransferTime: time.Now(),
		activeConnections: make(map[string]time.Time),
		serverProcesses:  make(map[string]*wrapper.ProcessManager),
		serverConfig:     config,
	}

	if singleTransferConfig != nil {
		service.singleTransfer = singleTransferConfig.Enabled
		service.requireReconnect = singleTransferConfig.RequireReconnect
	}

	return service
}

// PrepareTransfer 准备传输环境（启动服务端监听进程）
func (ts *TransferService) PrepareTransfer(req *models.TransferRequest, serverConfig *models.TransferSettings) error {
	// 构建传输配置
	transferConfig, err := ts.buildTransferConfig(req, serverConfig)
	if err != nil {
		return err
	}

	// 启动服务端监听进程
	if err := ts.ensureServerProcessStarted(transferConfig); err != nil {
		return fmt.Errorf("启动服务端监听进程失败: %v", err)
	}

	// 等待服务端进程启动
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	serverStarted := false
	attempts := 0
	for !serverStarted {
		select {
		case <-timeout:
			return fmt.Errorf("服务端进程启动超时（等待了5秒）")
		case <-ticker.C:
			attempts++
			ts.mu.RLock()
			processMgr, exists := ts.serverProcesses[string(transferConfig.Mode)]
			ts.mu.RUnlock()
			
			if exists && processMgr.IsRunning() {
				serverStarted = true
				break
			}
			
			// 记录调试信息
			if attempts%2 == 0 { // 每1秒记录一次
				fmt.Printf("等待服务端进程启动... 尝试次数: %d, 模式: %s, 进程存在: %v\n",
					attempts, transferConfig.Mode, exists)
			}
		}
	}

	return nil
}

// StartTransfer 启动传输任务
func (ts *TransferService) StartTransfer(req *models.TransferRequest, serverConfig *models.TransferSettings) (*models.TransferResponse, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// 检查并发限制
	if len(ts.activeTasks) >= ts.maxConcurrent {
		return nil, fmt.Errorf("已达到最大并发传输限制 (%d)", ts.maxConcurrent)
	}

	// 检查传输间隔
	if err := ts.checkTransferInterval(); err != nil {
		return nil, err
	}

	// 检查单次传输连接要求
	if ts.singleTransfer && ts.requireReconnect {
		// 使用配置中的默认服务端地址，而不是请求中的 server_ip
		connectionKey := ts.getConnectionKeyWithConfig(req, serverConfig)
		if ts.isConnectionActive(connectionKey) {
			return nil, fmt.Errorf("需要重新建立连接才能开始新的传输")
		}
	}

	// 创建传输任务（使用配置中的服务端地址）
	task := models.NewTransferTaskWithServer(req.Filename, req.Mode, req.Direction, "")
	
	// 构建传输配置
	transferConfig, err := ts.buildTransferConfig(req, serverConfig)
	if err != nil {
		return nil, err
	}

	// 验证配置
	if err := ts.rtranfile.ValidateConfig(transferConfig); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	// 创建传输任务包装器
	transferTask := &TransferTask{
		Task:    task,
		Config:  transferConfig,
		Monitor: wrapper.NewTransferMonitor(transferConfig.LogFile),
		Process: wrapper.NewProcessManager(),
	}

	// 启动传输任务（无论是客户端还是服务端传输）
	if err := ts.startTransferTask(transferTask); err != nil {
		return nil, err
	}

	// 添加到活跃任务
	ts.activeTasks[task.ID] = transferTask
	ts.taskHistory = append(ts.taskHistory, task)

	// 记录连接（如果是单次传输模式）
	if ts.singleTransfer {
		connectionKey := ts.getConnectionKeyWithConfig(req, serverConfig)
		ts.activeConnections[connectionKey] = time.Now()
	}

	// 更新最后传输时间
	ts.updateLastTransferTime()

	return &models.TransferResponse{
		ID:        task.ID,
		Status:    task.Status,
		Message:   "传输任务已启动",
		CreatedAt: task.CreatedAt,
	}, nil
}

// GetTransferStatus 获取传输状态
func (ts *TransferService) GetTransferStatus(taskID string) (*models.ProgressResponse, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	taskWrapper, exists := ts.activeTasks[taskID]
	if !exists {
		// 检查历史任务
		for _, task := range ts.taskHistory {
			if task.ID == taskID {
				return ts.buildProgressResponse(task, nil), nil
			}
		}
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}

	// 获取实时进度
	progress := taskWrapper.Monitor.GetProgress()
	return ts.buildProgressResponse(taskWrapper.Task, progress), nil
}

// CancelTransfer 取消传输任务
func (ts *TransferService) CancelTransfer(taskID string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	taskWrapper, exists := ts.activeTasks[taskID]
	if !exists {
		return fmt.Errorf("任务不存在或已完成: %s", taskID)
	}

	// 停止监控
	taskWrapper.Monitor.StopMonitoring()

	// 停止进程
	if err := taskWrapper.Process.Stop(); err != nil {
		return fmt.Errorf("停止传输进程失败: %v", err)
	}

	// 取消上下文
	if taskWrapper.Cancel != nil {
		taskWrapper.Cancel()
	}

	// 更新任务状态
	taskWrapper.Task.MarkCancelled()

	// 从活跃任务中移除
	delete(ts.activeTasks, taskID)

	return nil
}

// ListTransfers 列出传输任务
func (ts *TransferService) ListTransfers(page, size int) *models.TaskListResponse {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// 计算分页
	total := len(ts.taskHistory)
	start := (page - 1) * size
	end := start + size

	if start >= total {
		return &models.TaskListResponse{
			Tasks: []*models.TransferTask{},
			Total: total,
			Page:  page,
			Size:  size,
		}
	}

	if end > total {
		end = total
	}

	tasks := make([]*models.TransferTask, end-start)
	copy(tasks, ts.taskHistory[start:end])

	return &models.TaskListResponse{
		Tasks: tasks,
		Total: total,
		Page:  page,
		Size:  size,
	}
}

// GetActiveTransfers 获取活跃传输任务数量
func (ts *TransferService) GetActiveTransfers() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return len(ts.activeTasks)
}

// buildTransferConfig 构建传输配置
func (ts *TransferService) buildTransferConfig(req *models.TransferRequest, serverConfig *models.TransferSettings) (*wrapper.TransferConfig, error) {
	config := &wrapper.TransferConfig{
		Device:    serverConfig.Device,
		ChunkSize: serverConfig.ChunkSize,
	}

	// 设置传输模式特定的配置
	switch req.Mode {
	case models.ModeHugepages:
		config.Mode = wrapper.ModeHugepages
		config.Directory = serverConfig.Modes.Hugepages.BaseDir
		// hugepages模式：服务端开启大页，禁用mman；客户端禁用大页，开启mman
		if req.Direction == models.DirectionPut || req.Direction == models.DirectionGet {
			// 客户端：禁用大页（包含--nohuge），开启mman（包含--mman）
			config.NoHuge = true
			config.MMan = true
		} else {
			// 服务端：开启大页（不包含--nohuge），禁用mman（不包含--mman）
			config.NoHuge = false
			config.MMan = false
		}
	case models.ModeTmpfs:
		config.Mode = wrapper.ModeTmpfs
		config.Directory = serverConfig.Modes.Tmpfs.BaseDir
		// tmpfs模式：服务端禁用大页，开启mman；客户端开启大页，开启mman
		if req.Direction == models.DirectionPut || req.Direction == models.DirectionGet {
			// 客户端：开启大页（不包含--nohuge），开启mman（包含--mman）
			config.NoHuge = false
			config.MMan = true
		} else {
			// 服务端：禁用大页（包含--nohuge），开启mman（包含--mman）
			config.NoHuge = true
			config.MMan = true
		}
	case models.ModeFilesystem:
		config.Mode = wrapper.ModeFilesystem
		// 对于文件系统模式，根据传输方向设置不同的目录
		if req.Direction == models.DirectionPut {
			// 客户端上传：使用文件所在目录作为工作目录
			config.Directory = getFileDirectory(req.Filename)
		} else {
			// 服务端下载：使用服务端配置的目录
			config.Directory = serverConfig.Modes.Filesystem.BaseDir
		}
		// filesystem模式：服务端禁用大页，禁用mman；客户端开启大页，禁用mman
		if req.Direction == models.DirectionPut || req.Direction == models.DirectionGet {
			// 客户端：开启大页（不包含--nohuge），禁用mman（不包含--mman）
			config.NoHuge = false
			config.MMan = false
		} else {
			// 服务端：禁用大页（包含--nohuge），禁用mman（不包含--mman）
			config.NoHuge = true
			config.MMan = false
		}
	default:
		return nil, fmt.Errorf("不支持的传输模式: %s", req.Mode)
	}

	// 设置传输方向
	switch req.Direction {
	case models.DirectionPut:
		config.Direction = wrapper.DirectionPut
		config.Filename = getFileName(req.Filename)  // 只使用文件名，不包含路径
	case models.DirectionGet:
		config.Direction = wrapper.DirectionGet
		config.Filename = getFileName(req.Filename)  // 只使用文件名，不包含路径
	default:
		return nil, fmt.Errorf("不支持的传输方向: %s", req.Direction)
	}

	// 设置服务端地址（客户端使用）- 从配置中获取服务端地址
	// 如果配置中有服务端地址，使用配置中的地址；否则使用默认地址
	if serverConfig.ServerAddress != "" {
		config.ServerAddress = serverConfig.ServerAddress
	} else {
		// 默认使用本地地址
		config.ServerAddress = "localhost"
	}

	// 设置日志文件路径
	config.LogFile = fmt.Sprintf("/var/log/rtrans/rtrans_%s_%s.log", req.Direction, time.Now().Format("20060102_150405"))

	return config, nil
}

// getFileName 从文件路径中提取文件名
func getFileName(filepath string) string {
	// 查找最后一个斜杠
	lastSlash := -1
	for i := len(filepath) - 1; i >= 0; i-- {
		if filepath[i] == '/' {
			lastSlash = i
			break
		}
	}
	
	if lastSlash >= 0 && lastSlash < len(filepath)-1 {
		return filepath[lastSlash+1:]
	}
	
	return filepath
}

// getFileDirectory 获取文件所在目录
func getFileDirectory(filename string) string {
	// 如果文件路径是绝对路径，返回其目录
	// 否则返回当前工作目录
	if len(filename) > 0 && filename[0] == '/' {
		// 提取目录部分
		lastSlash := -1
		for i := len(filename) - 1; i >= 0; i-- {
			if filename[i] == '/' {
				lastSlash = i
				break
			}
		}
		if lastSlash > 0 {
			return filename[:lastSlash]
		}
	}
	// 默认返回当前目录
	return "."
}

// startTransferTask 启动传输任务
func (ts *TransferService) startTransferTask(taskWrapper *TransferTask) error {
	// 创建上下文
	_, cancel := context.WithCancel(context.Background())
	taskWrapper.Cancel = cancel

	// 标记任务开始
	taskWrapper.Task.MarkStarted()

	// 确保目标目录存在（对于下载操作）
	if taskWrapper.Config.Direction == wrapper.DirectionGet {
		if err := ts.ensureDirectoryExists(taskWrapper.Config.Directory); err != nil {
			taskWrapper.Task.MarkFailed(fmt.Sprintf("创建目标目录失败: %v", err))
			return err
		}
	}

	// 启动监控
	if err := taskWrapper.Monitor.StartMonitoring(); err != nil {
		taskWrapper.Task.MarkFailed(fmt.Sprintf("启动监控失败: %v", err))
		return err
	}

	// 检查传输方向
	if taskWrapper.Config.Direction == wrapper.DirectionPut || taskWrapper.Config.Direction == wrapper.DirectionGet {
		// 客户端传输 - 在服务端模式下不应该执行客户端传输命令
		// 客户端传输应该在客户端机器上执行，而不是在服务端机器上
		taskWrapper.Task.MarkFailed("服务端模式下不能执行客户端传输命令。客户端传输应该在客户端机器上执行")
		return fmt.Errorf("服务端模式下不能执行客户端传输命令。客户端传输应该在客户端机器上执行")
	} else {
		// 服务端监听（如果需要）
		taskWrapper.Task.MarkFailed("服务端监听模式暂不支持")
		return fmt.Errorf("服务端监听模式暂不支持")
	}
}

// monitorTransferProgress 监控传输进度
func (ts *TransferService) monitorTransferProgress(taskWrapper *TransferTask) {
	// 定期检查进度
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			progress := taskWrapper.Monitor.GetProgress()
			
			// 更新任务进度
			taskWrapper.Task.UpdateProgress(progress.BytesTransferred, progress.TotalBytes)
			
			// 检查传输状态
			switch progress.Status {
			case wrapper.StatusCompleted:
				taskWrapper.Task.MarkCompleted()
				ts.cleanupCompletedTask(taskWrapper)
				return
			case wrapper.StatusFailed:
				taskWrapper.Task.MarkFailed(progress.Error)
				ts.cleanupCompletedTask(taskWrapper)
				return
			case wrapper.StatusCancelled:
				taskWrapper.Task.MarkCancelled()
				ts.cleanupCompletedTask(taskWrapper)
				return
			}
			
		default:
			// 检查进程是否已退出
			processInfo := taskWrapper.Process.GetInfo()
			if processInfo.ExitTime != nil {
				// 进程已退出
				if processInfo.State == wrapper.StateError {
					taskWrapper.Task.MarkFailed(processInfo.Error)
				} else if taskWrapper.Task.Status != models.StatusCompleted {
					taskWrapper.Task.MarkFailed("进程异常退出")
				}
				ts.cleanupCompletedTask(taskWrapper)
				return
			}
		}
	}
}

// cleanupCompletedTask 清理已完成的任务
func (ts *TransferService) cleanupCompletedTask(taskWrapper *TransferTask) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// 停止监控
	taskWrapper.Monitor.StopMonitoring()

	// 清理进程
	taskWrapper.Process.Cleanup()

	// 从活跃任务中移除
	delete(ts.activeTasks, taskWrapper.Task.ID)

	// 清理连接状态（如果是单次传输模式）
	if ts.singleTransfer {
		// 使用固定的连接标识符清理连接
		connectionKey := fmt.Sprintf("default_%s", taskWrapper.Task.Direction)
		delete(ts.activeConnections, connectionKey)
	}
}

// checkTransferInterval 检查传输间隔
func (ts *TransferService) checkTransferInterval() error {
	// 实现传输间隔检查逻辑
	// 这里需要记录最后传输时间并检查间隔
	// 简化实现：总是返回 nil
	return nil
}

// updateLastTransferTime 更新最后传输时间
func (ts *TransferService) updateLastTransferTime() {
	// 实现最后传输时间更新逻辑
}

// buildProgressResponse 构建进度响应
func (ts *TransferService) buildProgressResponse(task *models.TransferTask, progress *wrapper.ProgressInfo) *models.ProgressResponse {
	resp := &models.ProgressResponse{
		ID:               task.ID,
		Status:           task.Status,
		Progress:         task.Progress,
		BytesTransferred: task.BytesTransferred,
		TotalBytes:       task.TotalBytes,
		LastUpdated:      task.UpdatedAt,
	}

	if progress != nil {
		resp.TransferRate = progress.TransferRate
		resp.ElapsedTime = progress.ElapsedTime.String()
		if progress.EstimatedTime > 0 {
			resp.EstimatedTime = progress.EstimatedTime.String()
		}
		resp.Error = progress.Error
	}

	return resp
}

// Cleanup 清理资源
func (ts *TransferService) Cleanup() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// 停止所有活跃任务
	for _, taskWrapper := range ts.activeTasks {
		taskWrapper.Monitor.StopMonitoring()
		taskWrapper.Process.Cleanup()
		if taskWrapper.Cancel != nil {
			taskWrapper.Cancel()
		}
		taskWrapper.Task.MarkCancelled()
	}

	// 停止所有服务端进程
	for modeName, processMgr := range ts.serverProcesses {
		processMgr.Cleanup()
		delete(ts.serverProcesses, modeName)
	}

	ts.activeTasks = make(map[string]*TransferTask)
	ts.activeConnections = make(map[string]time.Time)
	ts.serverProcesses = make(map[string]*wrapper.ProcessManager)
}

// 连接管理相关方法

// getConnectionKey 获取连接标识符
func (ts *TransferService) getConnectionKey(req *models.TransferRequest) string {
	// 使用服务端地址和传输方向作为连接标识符
	return fmt.Sprintf("%s_%s", req.ServerIP, req.Direction)
}

// getConnectionKeyWithConfig 基于配置获取连接标识符
func (ts *TransferService) getConnectionKeyWithConfig(req *models.TransferRequest, serverConfig *models.TransferSettings) string {
	// 使用配置中的默认服务端地址和传输方向作为连接标识符
	// 这里简化实现，实际应该从配置中获取服务端地址
	// 使用固定的连接标识符，因为客户端已经预先配置了服务端地址
	return fmt.Sprintf("default_%s", req.Direction)
}

// isConnectionActive 检查连接是否活跃
func (ts *TransferService) isConnectionActive(connectionKey string) bool {
	lastActive, exists := ts.activeConnections[connectionKey]
	if !exists {
		return false
	}
	
	// 检查连接是否在超时时间内
	timeout := 10 * time.Second // 默认超时时间
	return time.Since(lastActive) < timeout
}

// closeConnection 关闭连接
func (ts *TransferService) closeConnection(connectionKey string) {
	delete(ts.activeConnections, connectionKey)
}

// cleanupExpiredConnections 清理过期的连接
func (ts *TransferService) cleanupExpiredConnections() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	timeout := 10 * time.Second // 默认超时时间
	currentTime := time.Now()
	
	for key, lastActive := range ts.activeConnections {
		if currentTime.Sub(lastActive) > timeout {
			delete(ts.activeConnections, key)
		}
	}
}

// SetSingleTransferMode 设置单次传输模式
func (ts *TransferService) SetSingleTransferMode(enabled bool, requireReconnect bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	ts.singleTransfer = enabled
	ts.requireReconnect = requireReconnect
	
	if !enabled {
		// 禁用单次传输模式时清理所有连接
		ts.activeConnections = make(map[string]time.Time)
	}
}

// GetConnectionStatus 获取连接状态
func (ts *TransferService) GetConnectionStatus() map[string]interface{} {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	status := map[string]interface{}{
		"single_transfer_enabled": ts.singleTransfer,
		"require_reconnect":       ts.requireReconnect,
		"active_connections":      len(ts.activeConnections),
		"connections":             make(map[string]string),
	}
	
	for key, lastActive := range ts.activeConnections {
		status["connections"].(map[string]string)[key] = lastActive.Format(time.RFC3339)
	}
	
	return status
}

// ensureServerProcessStarted 确保服务端监听进程已启动
func (ts *TransferService) ensureServerProcessStarted(config *wrapper.TransferConfig) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	// 首先检查该模式的进程是否已启动且正在运行
	if processMgr, exists := ts.serverProcesses[string(config.Mode)]; exists {
		// 检查进程是否在运行
		if processMgr.IsRunning() {
			fmt.Printf("模式 %s 的服务端进程已在运行，PID: %d\n", config.Mode, processMgr.GetPID())
			return nil // 进程已在运行，不需要重新启动
		}
		// 进程已停止，从映射中移除
		fmt.Printf("模式 %s 的服务端进程已停止，需要重新启动\n", config.Mode)
		delete(ts.serverProcesses, string(config.Mode))
	}
	
	// 检查是否有其他模式的进程在运行（只停止不同模式的进程）
	for modeName, processMgr := range ts.serverProcesses {
		if modeName != string(config.Mode) && processMgr.IsRunning() {
			// 停止其他模式的进程
			fmt.Printf("停止当前运行的模式: %s，切换到模式: %s\n", modeName, config.Mode)
			if err := processMgr.Stop(); err != nil {
				fmt.Printf("停止模式 %s 的进程失败: %v\n", modeName, err)
			}
			delete(ts.serverProcesses, modeName)
		}
	}
	
	// 根据传输模式确定服务端参数
	var baseDir string
	var noHuge, mMan bool
	
	// 如果 serverConfig 为 nil，使用默认值
	if ts.serverConfig == nil {
		// 使用默认配置
		switch config.Mode {
		case wrapper.ModeHugepages:
			baseDir = "/dev/hugepages/dir"
			noHuge = false // 大页模式服务端：开启大页
			mMan = false   // 大页模式服务端：禁用mman
		case wrapper.ModeTmpfs:
			baseDir = "/dev/shm/dir"
			noHuge = true  // tmpfs模式服务端：禁用大页
			mMan = true    // tmpfs模式服务端：开启mman
		case wrapper.ModeFilesystem:
			baseDir = "/var/lib/rtrans/files"
			noHuge = false  // 文件系统模式服务端：尝试开启大页（可能不支持）
			mMan = false   // 文件系统模式服务端：禁用mman
		default:
			return fmt.Errorf("不支持的传输模式: %s", config.Mode)
		}
	} else {
		// 使用配置中的值
		switch config.Mode {
		case wrapper.ModeHugepages:
			baseDir = ts.serverConfig.Modes.Hugepages.BaseDir
			noHuge = false // 大页模式服务端：开启大页
			mMan = false   // 大页模式服务端：禁用mman
		case wrapper.ModeTmpfs:
			baseDir = ts.serverConfig.Modes.Tmpfs.BaseDir
			noHuge = true  // tmpfs模式服务端：禁用大页
			mMan = true    // tmpfs模式服务端：开启mman
		case wrapper.ModeFilesystem:
			baseDir = ts.serverConfig.Modes.Filesystem.BaseDir
			noHuge = false  // 文件系统模式服务端：尝试开启大页（可能不支持）
			mMan = false   // 文件系统模式服务端：禁用mman
		default:
			return fmt.Errorf("不支持的传输模式: %s", config.Mode)
		}
	}
	
	// 创建服务端配置
	serverConfig := &wrapper.TransferConfig{
		Device:    config.Device,
		Directory: baseDir,
		Mode:      config.Mode,
		LogFile:   fmt.Sprintf("/var/log/rtrans/rtranfile_server_%s.log", config.Mode),
		NoHuge:    noHuge,
		MMan:      mMan,
		// 服务端配置不需要传输方向和文件名
		Direction: "",
		Filename:  "",
	}
	
	// 验证配置
	if err := ts.rtranfile.ValidateConfig(serverConfig); err != nil {
		return fmt.Errorf("服务端配置验证失败: %v", err)
	}
	
	// 启动服务端监听进程
	fmt.Printf("正在启动服务端监听进程... 模式: %s, 设备: %s, 目录: %s\n",
		config.Mode, serverConfig.Device, serverConfig.Directory)
	
	// 使用后台上下文启动服务端进程，避免进程立即退出
	serverCtx := context.Background()
	serverCmd, err := ts.rtranfile.StartServer(serverCtx, serverConfig)
	if err != nil {
		return fmt.Errorf("启动服务端监听进程失败: %v", err)
	}
	
	// 创建进程管理器来管理服务端进程
	serverProcessMgr := wrapper.NewProcessManager()
	if err := serverProcessMgr.Start(serverCmd); err != nil {
		return fmt.Errorf("管理服务端进程失败: %v", err)
	}
	
	// 保存进程管理器
	ts.serverProcesses[string(config.Mode)] = serverProcessMgr
	
	fmt.Printf("服务端监听进程已启动，PID: %d\n", serverProcessMgr.GetPID())
	
	// 等待服务端进程稳定运行（避免立即退出）
	time.Sleep(2 * time.Second)
	
	// 检查进程是否仍在运行
	if !serverProcessMgr.IsRunning() {
		// 获取进程信息以提供更详细的错误信息
		processInfo := serverProcessMgr.GetInfo()
		errorMsg := fmt.Sprintf("服务端监听进程启动后立即退出，PID: %d, 状态: %s",
			processInfo.PID, processInfo.State)
		
		if processInfo.ExitCode != nil {
			errorMsg += fmt.Sprintf(", 退出码: %d", *processInfo.ExitCode)
		}
		if processInfo.Error != "" {
			errorMsg += fmt.Sprintf(", 错误: %s", processInfo.Error)
		}
		if processInfo.ExitTime != nil {
			errorMsg += fmt.Sprintf(", 退出时间: %s", processInfo.ExitTime.Format(time.RFC3339))
		}
		
		errorMsg += "\n请检查以下可能的问题："
		errorMsg += "\n1. RDMA设备是否可用: " + serverConfig.Device
		errorMsg += "\n2. 目录权限: " + serverConfig.Directory
		errorMsg += "\n3. rtranfile日志文件: " + serverConfig.LogFile
		errorMsg += "\n4. 系统资源是否充足"
		
		return fmt.Errorf(errorMsg)
	}
	
	return nil
}

// ensureDirectoryExists 确保目录存在
func (ts *TransferService) ensureDirectoryExists(dirPath string) error {
	if dirPath == "" || dirPath == "." {
		return nil // 当前目录总是存在
	}
	
	// 检查目录是否存在
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// 创建目录
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("创建目录失败 %s: %v", dirPath, err)
		}
	}
	return nil
}