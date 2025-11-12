package wrapper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// TransferMode 定义传输模式
type TransferMode string

const (
	ModeHugepages  TransferMode = "hugepages"
	ModeTmpfs      TransferMode = "tmpfs"
	ModeFilesystem TransferMode = "filesystem"
)

// TransferDirection 定义传输方向
type TransferDirection string

const (
	DirectionPut TransferDirection = "put" // 上传文件
	DirectionGet TransferDirection = "get" // 下载文件
)

// TransferConfig 定义传输配置
type TransferConfig struct {
	// RDMA 设备
	Device string `json:"device"`
	
	// 传输目录
	Directory string `json:"directory"`
	
	// 传输模式
	Mode TransferMode `json:"mode"`
	
	// 传输方向
	Direction TransferDirection `json:"direction"`
	
	// 文件名
	Filename string `json:"filename"`
	
	// 服务端地址 (客户端使用)
	ServerAddress string `json:"server_address,omitempty"`
	
	// 块大小
	ChunkSize int `json:"chunk_size"`
	
	// 日志文件路径
	LogFile string `json:"log_file"`
	
	// 是否使用大页内存
	NoHuge bool `json:"no_huge"`
	
	// 是否使用内存映射
	MMan bool `json:"mman"`
}

// TransferResult 定义传输结果
type TransferResult struct {
	Success          bool          `json:"success"`
	Error            string        `json:"error,omitempty"`
	Duration         time.Duration `json:"duration"`
	BytesTransferred int64         `json:"bytes_transferred"`
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
}

// RtranfileWrapper rtranfile 包装器
type RtranfileWrapper struct {
	binPath string // rtranfile 二进制文件路径
}

// NewRtranfileWrapper 创建新的 rtranfile 包装器
func NewRtranfileWrapper(binPath string) *RtranfileWrapper {
	return &RtranfileWrapper{
		binPath: binPath,
	}
}

// StartServer 启动 rtranfile 服务端
func (w *RtranfileWrapper) StartServer(ctx context.Context, config *TransferConfig) (*exec.Cmd, error) {
	// 确保工作目录存在
	if err := w.ensureDirectoryExists(config.Directory); err != nil {
		return nil, fmt.Errorf("创建工作目录失败: %v", err)
	}
	
	args := w.buildServerArgs(config)
	
	// 调试信息：显示完整的命令（去掉方括号）
	cmdStr := w.binPath
	for _, arg := range args {
		cmdStr += " " + arg
	}
	fmt.Printf("执行 rtranfile 命令: %s\n", cmdStr)
	
	cmd := exec.CommandContext(ctx, w.binPath, args...)
	
	// 设置日志文件输出
	if config.LogFile != "" {
		logFile, err := w.createLogFile(config.LogFile)
		if err != nil {
			return nil, fmt.Errorf("创建日志文件失败: %v", err)
		}
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		// 如果没有日志文件，输出到标准输出以便调试
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	
	return cmd, nil
}

// StartClient 启动 rtranfile 客户端传输
func (w *RtranfileWrapper) StartClient(ctx context.Context, config *TransferConfig) (*exec.Cmd, error) {
	// 确保工作目录存在
	if err := w.ensureDirectoryExists(config.Directory); err != nil {
		return nil, fmt.Errorf("创建工作目录失败: %v", err)
	}
	
	args := w.buildClientArgs(config)
	
	// 调试信息：显示完整的命令（去掉方括号）
	cmdStr := w.binPath
	for _, arg := range args {
		cmdStr += " " + arg
	}
	fmt.Printf("执行 rtranfile 命令: %s\n", cmdStr)
	
	cmd := exec.CommandContext(ctx, w.binPath, args...)
	
	// 设置日志文件输出
	if config.LogFile != "" {
		logFile, err := w.createLogFile(config.LogFile)
		if err != nil {
			return nil, fmt.Errorf("创建日志文件失败: %v", err)
		}
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}
	
	return cmd, nil
}

// buildServerArgs 构建服务端命令行参数
func (w *RtranfileWrapper) buildServerArgs(config *TransferConfig) []string {
	args := []string{
		"-d", config.Device,
		"--dir", config.Directory,
		"-l", "0", // 服务端监听模式，端口0表示自动选择
		"--logfile", config.LogFile,
	}
	
	// 根据传输模式添加参数
	args = w.addModeSpecificArgs(args, config)
	
	return args
}

// buildClientArgs 构建客户端命令行参数
func (w *RtranfileWrapper) buildClientArgs(config *TransferConfig) []string {
	args := []string{
		"-d", config.Device,
		"-c", config.ServerAddress,
		"--dir", config.Directory,
		"--logfile", config.LogFile,
		"-m", "4096", // 固定使用4096块大小
	}
	
	// 根据传输模式添加参数
	args = w.addModeSpecificArgs(args, config)
	
	// 添加传输方向参数
	// 只使用文件名，不包含路径
	filename := filepath.Base(config.Filename)
	if config.Direction == DirectionPut {
		args = append(args, "--put", filename)
	} else {
		args = append(args, "--get", filename)
	}
	
	return args
}

// addModeSpecificArgs 添加模式特定的参数
func (w *RtranfileWrapper) addModeSpecificArgs(args []string, config *TransferConfig) []string {
	switch config.Mode {
	case ModeHugepages:
		// 大页内存模式: --nohuge --mman
		args = append(args, "--nohuge", "--mman")
	case ModeTmpfs:
		// tmpfs 模式: --nohuge --mman
		args = append(args, "--nohuge", "--mman")
	case ModeFilesystem:
		// 文件系统模式: 服务端总是禁用大页和mman
		// 客户端根据配置决定
		if config.Direction == "" {
			// 服务端模式: 禁用大页和mman
			args = append(args, "--nohuge")
		} else {
			// 客户端模式: 根据配置决定
			if config.NoHuge {
				args = append(args, "--nohuge")
			}
			if config.MMan {
				args = append(args, "--mman")
			}
		}
	}
	
	return args
}

// createLogFile 创建日志文件
func (w *RtranfileWrapper) createLogFile(logPath string) (*os.File, error) {
	// 确保日志目录存在
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	
	// 创建或打开日志文件
	return os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

// ensureDirectoryExists 确保目录存在
func (w *RtranfileWrapper) ensureDirectoryExists(dirPath string) error {
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

// ValidateConfig 验证传输配置
func (w *RtranfileWrapper) ValidateConfig(config *TransferConfig) error {
	if config.Device == "" {
		return fmt.Errorf("RDMA 设备不能为空")
	}
	
	if config.Directory == "" {
		return fmt.Errorf("传输目录不能为空")
	}
	
	if config.LogFile == "" {
		return fmt.Errorf("日志文件路径不能为空")
	}
	
	// 验证传输模式
	switch config.Mode {
	case ModeHugepages, ModeTmpfs, ModeFilesystem:
		// 有效的传输模式
	default:
		return fmt.Errorf("不支持的传输模式: %s", config.Mode)
	}
	
	// 验证传输方向（服务端不需要传输方向）
	if config.Direction != "" {
		switch config.Direction {
		case DirectionPut, DirectionGet:
			// 有效的传输方向
		default:
			return fmt.Errorf("不支持的传输方向: %s", config.Direction)
		}
		
		// 客户端需要服务端地址和文件名
		if config.Direction == DirectionPut || config.Direction == DirectionGet {
			if config.ServerAddress == "" {
				return fmt.Errorf("客户端传输需要指定服务端地址")
			}
			if config.Filename == "" {
				return fmt.Errorf("客户端传输需要指定文件名")
			}
		}
	}
	
	return nil
}

// GetDefaultConfig 获取默认配置
func (w *RtranfileWrapper) GetDefaultConfig(mode TransferMode) *TransferConfig {
	config := &TransferConfig{
		Device:    "mlx5_0",
		ChunkSize: 4096,
		NoHuge:    true,
		MMan:      true,
	}
	
	switch mode {
	case ModeHugepages:
		config.Directory = "/dev/hugepages/dir"
		config.Mode = ModeHugepages
	case ModeTmpfs:
		config.Directory = "/dev/shm/dir"
		config.Mode = ModeTmpfs
	case ModeFilesystem:
		config.Directory = "/var/lib/rtrans/files"
		config.Mode = ModeFilesystem
		config.NoHuge = false
		config.MMan = false
	}
	
	return config
}