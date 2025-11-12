package wrapper

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TransferStatus 定义传输状态
type TransferStatus string

const (
	StatusPending    TransferStatus = "pending"
	StatusStarting   TransferStatus = "starting"
	StatusInProgress TransferStatus = "in_progress"
	StatusCompleted  TransferStatus = "completed"
	StatusFailed     TransferStatus = "failed"
	StatusCancelled  TransferStatus = "cancelled"
)

// ProgressInfo 定义进度信息
type ProgressInfo struct {
	Status          TransferStatus `json:"status"`
	BytesTransferred int64         `json:"bytes_transferred"`
	TotalBytes      int64         `json:"total_bytes"`
	ProgressPercent float64       `json:"progress_percent"`
	TransferRate    float64       `json:"transfer_rate"` // MB/s
	ElapsedTime     time.Duration `json:"elapsed_time"`
	EstimatedTime   time.Duration `json:"estimated_time"`
	StartTime       time.Time     `json:"start_time"`
	LastUpdateTime  time.Time     `json:"last_update_time"`
	Error           string        `json:"error,omitempty"`
}

// LogParser 日志解析器
type LogParser struct {
	progressRegex *regexp.Regexp
	errorRegex    *regexp.Regexp
	completeRegex *regexp.Regexp
}

// NewLogParser 创建新的日志解析器
func NewLogParser() *LogParser {
	return &LogParser{
		// 匹配进度信息，例如: "Transferred 1024 MB of 2048 MB (50.0%)"
		progressRegex: regexp.MustCompile(`(?i)transferred\s+(\d+)\s*(MB|GB|KB|B)\s+of\s+(\d+)\s*(MB|GB|KB|B)\s*\(([\d.]+)%\)`),
		// 匹配错误信息
		errorRegex: regexp.MustCompile(`(?i)(error|failed|failure|exception)`),
		// 匹配完成信息
		completeRegex: regexp.MustCompile(`(?i)(completed|finished|success)`),
	}
}

// ParseLine 解析日志行
func (lp *LogParser) ParseLine(line string) (*ProgressInfo, error) {
	info := &ProgressInfo{
		LastUpdateTime: time.Now(),
	}

	// 检查错误信息
	if lp.errorRegex.MatchString(line) {
		info.Status = StatusFailed
		info.Error = strings.TrimSpace(line)
		return info, nil
	}

	// 检查完成信息
	if lp.completeRegex.MatchString(line) {
		info.Status = StatusCompleted
		info.ProgressPercent = 100.0
		return info, nil
	}

	// 解析进度信息
	matches := lp.progressRegex.FindStringSubmatch(line)
	if matches != nil {
		info.Status = StatusInProgress
		
		// 解析已传输字节数
		transferred, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("解析已传输字节数失败: %v", err)
		}
		
		// 解析总字节数
		total, err := strconv.ParseInt(matches[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("解析总字节数失败: %v", err)
		}
		
		// 转换单位
		transferredBytes := lp.convertToBytes(transferred, matches[2])
		totalBytes := lp.convertToBytes(total, matches[4])
		
		info.BytesTransferred = transferredBytes
		info.TotalBytes = totalBytes
		
		// 解析进度百分比
		percent, err := strconv.ParseFloat(matches[5], 64)
		if err != nil {
			return nil, fmt.Errorf("解析进度百分比失败: %v", err)
		}
		info.ProgressPercent = percent
		
		return info, nil
	}

	return nil, nil
}

// convertToBytes 将大小转换为字节
func (lp *LogParser) convertToBytes(value int64, unit string) int64 {
	switch strings.ToUpper(unit) {
	case "GB":
		return value * 1024 * 1024 * 1024
	case "MB":
		return value * 1024 * 1024
	case "KB":
		return value * 1024
	case "B":
		return value
	default:
		return value
	}
}

// TransferMonitor 传输监控器
type TransferMonitor struct {
	mu          sync.RWMutex
	progress    *ProgressInfo
	logFile     string
	parser      *LogParser
	stopChan    chan struct{}
	isMonitoring bool
}

// NewTransferMonitor 创建新的传输监控器
func NewTransferMonitor(logFile string) *TransferMonitor {
	return &TransferMonitor{
		progress: &ProgressInfo{
			Status:         StatusPending,
			StartTime:      time.Now(),
			LastUpdateTime: time.Now(),
		},
		logFile:  logFile,
		parser:   NewLogParser(),
		stopChan: make(chan struct{}),
	}
}

// StartMonitoring 开始监控
func (tm *TransferMonitor) StartMonitoring() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.isMonitoring {
		return fmt.Errorf("已经在监控中")
	}

	tm.isMonitoring = true
	tm.progress.Status = StatusStarting
	tm.progress.StartTime = time.Now()

	// 启动监控协程
	go tm.monitorLogFile()

	return nil
}

// StopMonitoring 停止监控
func (tm *TransferMonitor) StopMonitoring() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.isMonitoring {
		close(tm.stopChan)
		tm.isMonitoring = false
		
		if tm.progress.Status == StatusInProgress {
			tm.progress.Status = StatusCancelled
		}
	}
}

// GetProgress 获取当前进度
func (tm *TransferMonitor) GetProgress() *ProgressInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 返回副本
	progress := *tm.progress
	
	// 计算实时统计信息
	if progress.Status == StatusInProgress {
		elapsed := time.Since(progress.StartTime)
		progress.ElapsedTime = elapsed
		
		// 计算传输速率
		if elapsed > 0 {
			rate := float64(progress.BytesTransferred) / elapsed.Seconds() / (1024 * 1024) // MB/s
			progress.TransferRate = rate
		}
		
		// 计算预计剩余时间
		if progress.ProgressPercent > 0 && progress.TransferRate > 0 {
			remainingBytes := progress.TotalBytes - progress.BytesTransferred
			estimatedSeconds := float64(remainingBytes) / (progress.TransferRate * 1024 * 1024)
			progress.EstimatedTime = time.Duration(estimatedSeconds) * time.Second
		}
	}
	
	return &progress
}

// monitorLogFile 监控日志文件
func (tm *TransferMonitor) monitorLogFile() {
	// 等待日志文件创建
	for {
		select {
		case <-tm.stopChan:
			return
		default:
			if _, err := os.Stat(tm.logFile); err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		break
	}

	// 打开日志文件
	file, err := os.Open(tm.logFile)
	if err != nil {
		tm.mu.Lock()
		tm.progress.Status = StatusFailed
		tm.progress.Error = fmt.Sprintf("打开日志文件失败: %v", err)
		tm.mu.Unlock()
		return
	}
	defer file.Close()

	// 从文件末尾开始读取
	_, err = file.Seek(0, 2)
	if err != nil {
		tm.mu.Lock()
		tm.progress.Status = StatusFailed
		tm.progress.Error = fmt.Sprintf("定位日志文件失败: %v", err)
		tm.mu.Unlock()
		return
	}

	tm.mu.Lock()
	tm.progress.Status = StatusInProgress
	tm.mu.Unlock()

	scanner := bufio.NewScanner(file)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-tm.stopChan:
			return
		case <-ticker.C:
			// 读取新的日志行
			for scanner.Scan() {
				line := scanner.Text()
				progressInfo, err := tm.parser.ParseLine(line)
				if err != nil {
					// 解析错误，记录但不中断监控
					continue
				}

				if progressInfo != nil {
					tm.mu.Lock()
					// 更新进度信息
					if progressInfo.Status != "" {
						tm.progress.Status = progressInfo.Status
					}
					if progressInfo.BytesTransferred > 0 {
						tm.progress.BytesTransferred = progressInfo.BytesTransferred
					}
					if progressInfo.TotalBytes > 0 {
						tm.progress.TotalBytes = progressInfo.TotalBytes
					}
					if progressInfo.ProgressPercent > 0 {
						tm.progress.ProgressPercent = progressInfo.ProgressPercent
					}
					if progressInfo.Error != "" {
						tm.progress.Error = progressInfo.Error
					}
					tm.progress.LastUpdateTime = time.Now()
					tm.mu.Unlock()
				}
			}

			if err := scanner.Err(); err != nil {
				tm.mu.Lock()
				tm.progress.Status = StatusFailed
				tm.progress.Error = fmt.Sprintf("读取日志文件失败: %v", err)
				tm.mu.Unlock()
				return
			}
		}
	}
}

// IsMonitoring 检查是否在监控中
func (tm *TransferMonitor) IsMonitoring() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.isMonitoring
}

// SetStatus 手动设置状态
func (tm *TransferMonitor) SetStatus(status TransferStatus, errorMsg string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.progress.Status = status
	if errorMsg != "" {
		tm.progress.Error = errorMsg
	}
	tm.progress.LastUpdateTime = time.Now()
}