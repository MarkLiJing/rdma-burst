package models

import (
	"fmt"
	"time"
)

// TransferTask 定义传输任务
type TransferTask struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	SourcePath  string    `json:"source_path"`
	TargetPath  string    `json:"target_path"`
	Mode        string    `json:"mode"` // hugepages, tmpfs, filesystem
	Direction   string    `json:"direction"` // put, get
	ServerIP    string    `json:"server_ip,omitempty"` // 服务端地址
	Status      string    `json:"status"`
	Progress    float64   `json:"progress"`
	BytesTransferred int64 `json:"bytes_transferred"`
	TotalBytes  int64     `json:"total_bytes"`
	StartTime   time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Error       string    `json:"error,omitempty"`
	Message     string    `json:"message,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TransferConfig 定义传输配置
type TransferConfig struct {
	Device            string        `json:"device"`
	BaseDir           string        `json:"base_dir"`
	TransferInterval  time.Duration `json:"transfer_interval"`
	MaxConcurrent     int           `json:"max_concurrent"`
	ChunkSize         int           `json:"chunk_size"`
	LogFile           string        `json:"log_file"`
	
	// 模式特定配置
	HugepagesConfig   *ModeConfig   `json:"hugepages_config,omitempty"`
	TmpfsConfig       *ModeConfig   `json:"tmpfs_config,omitempty"`
	FilesystemConfig  *ModeConfig   `json:"filesystem_config,omitempty"`
}

// TransferRequest 定义传输请求
type TransferRequest struct {
	Filename  string `json:"filename" binding:"required"`
	Mode      string `json:"mode" binding:"required,oneof=hugepages tmpfs filesystem"`
	Direction string `json:"direction" binding:"required,oneof=put get"`
	ServerIP  string `json:"server_ip,omitempty"` // 客户端使用
}

// TransferResponse 定义传输响应
type TransferResponse struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	ClientCommand string   `json:"client_command,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ProgressResponse 定义进度响应
type ProgressResponse struct {
	ID               string    `json:"id"`
	Status           string    `json:"status"`
	Progress         float64   `json:"progress"`
	BytesTransferred int64     `json:"bytes_transferred"`
	TotalBytes       int64     `json:"total_bytes"`
	TransferRate     float64   `json:"transfer_rate"` // MB/s
	ElapsedTime      string    `json:"elapsed_time"`
	EstimatedTime    string    `json:"estimated_time,omitempty"`
	Error            string    `json:"error,omitempty"`
	LastUpdated      time.Time `json:"last_updated"`
}

// TaskListResponse 定义任务列表响应
type TaskListResponse struct {
	Tasks []*TransferTask `json:"tasks"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
}

// HealthResponse 定义健康检查响应
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// ErrorResponse 定义错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// 状态常量
const (
	StatusPending    = "pending"
	StatusPrepared   = "prepared"  // 传输环境准备就绪
	StatusStarting   = "starting"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
)

// 传输模式常量
const (
	ModeHugepages  = "hugepages"
	ModeTmpfs      = "tmpfs"
	ModeFilesystem = "filesystem"
)

// 传输方向常量
const (
	DirectionPut = "put"
	DirectionGet = "get"
)

// NewTransferTask 创建新的传输任务
func NewTransferTask(filename, mode, direction string) *TransferTask {
	now := time.Now()
	return &TransferTask{
		ID:          generateID(),
		Filename:    filename,
		Mode:        mode,
		Direction:   direction,
		Status:      StatusPending,
		Progress:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewTransferTaskWithServer 创建包含服务端地址的传输任务
func NewTransferTaskWithServer(filename, mode, direction, serverIP string) *TransferTask {
	now := time.Now()
	return &TransferTask{
		ID:          generateID(),
		Filename:    filename,
		Mode:        mode,
		Direction:   direction,
		ServerIP:    serverIP,
		Status:      StatusPending,
		Progress:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// UpdateProgress 更新任务进度
func (t *TransferTask) UpdateProgress(bytesTransferred, totalBytes int64) {
	t.BytesTransferred = bytesTransferred
	t.TotalBytes = totalBytes
	
	if totalBytes > 0 {
		t.Progress = float64(bytesTransferred) / float64(totalBytes) * 100
	}
	
	t.UpdatedAt = time.Now()
}

// MarkStarted 标记任务开始
func (t *TransferTask) MarkStarted() {
	t.Status = StatusStarting
	t.StartTime = time.Now()
	t.UpdatedAt = time.Now()
}

// MarkInProgress 标记任务进行中
func (t *TransferTask) MarkInProgress() {
	t.Status = StatusInProgress
	t.UpdatedAt = time.Now()
}

// MarkCompleted 标记任务完成
func (t *TransferTask) MarkCompleted() {
	now := time.Now()
	t.Status = StatusCompleted
	t.Progress = 100
	t.EndTime = &now
	t.UpdatedAt = now
}

// MarkFailed 标记任务失败
func (t *TransferTask) MarkFailed(errorMsg string) {
	now := time.Now()
	t.Status = StatusFailed
	t.Error = errorMsg
	t.EndTime = &now
	t.UpdatedAt = now
}

// MarkCancelled 标记任务取消
func (t *TransferTask) MarkCancelled() {
	now := time.Now()
	t.Status = StatusCancelled
	t.EndTime = &now
	t.UpdatedAt = now
}

// IsActive 检查任务是否活跃
func (t *TransferTask) IsActive() bool {
	return t.Status == StatusStarting || t.Status == StatusInProgress
}

// IsFinished 检查任务是否完成
func (t *TransferTask) IsFinished() bool {
	return t.Status == StatusCompleted || t.Status == StatusFailed || t.Status == StatusCancelled
}

// 生成任务ID的简单实现
func generateID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}