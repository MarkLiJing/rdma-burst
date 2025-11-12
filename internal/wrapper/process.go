package wrapper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// ProcessState 定义进程状态
type ProcessState string

const (
	StateStarting ProcessState = "starting"
	StateRunning  ProcessState = "running"
	StateStopping ProcessState = "stopping"
	StateStopped  ProcessState = "stopped"
	StateError    ProcessState = "error"
)

// ProcessInfo 定义进程信息
type ProcessInfo struct {
	PID         int          `json:"pid"`
	State       ProcessState `json:"state"`
	StartTime   time.Time    `json:"start_time"`
	ExitTime    *time.Time   `json:"exit_time,omitempty"`
	ExitCode    *int         `json:"exit_code,omitempty"`
	Error       string       `json:"error,omitempty"`
	CommandLine string       `json:"command_line"`
}

// ProcessManager 进程管理器
type ProcessManager struct {
	mu       sync.RWMutex
	process  *exec.Cmd
	info     *ProcessInfo
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewProcessManager 创建新的进程管理器
func NewProcessManager() *ProcessManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProcessManager{
		info: &ProcessInfo{
			State: StateStopped,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动进程
func (pm *ProcessManager) Start(cmd *exec.Cmd) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.process != nil && pm.info.State == StateRunning {
		return fmt.Errorf("进程已经在运行中")
	}

	// 更新进程信息
	pm.info = &ProcessInfo{
		State:       StateStarting,
		StartTime:   time.Now(),
		CommandLine: fmt.Sprintf("%s %v", cmd.Path, cmd.Args),
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		pm.info.State = StateError
		pm.info.Error = err.Error()
		return fmt.Errorf("启动进程失败: %v", err)
	}

	pm.process = cmd
	pm.info.PID = cmd.Process.Pid
	pm.info.State = StateRunning

	// 对于服务端进程（rtranfile服务端），不启动监控协程
	// 因为服务端进程应该在循环模式下持续运行
	// 只有客户端传输进程需要监控
	if !pm.isServerProcess(cmd) {
		// 启动监控协程
		go pm.monitorProcess()
	}

	return nil
}

// Stop 停止进程
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.process == nil || pm.info.State != StateRunning {
		return fmt.Errorf("进程未运行或已停止")
	}

	pm.info.State = StateStopping

	// 发送终止信号
	if err := pm.process.Process.Signal(os.Interrupt); err != nil {
		// 如果优雅终止失败，强制终止
		if err := pm.process.Process.Kill(); err != nil {
			pm.info.State = StateError
			pm.info.Error = err.Error()
			return fmt.Errorf("强制终止进程失败: %v", err)
		}
	}

	// 等待进程结束
	done := make(chan error, 1)
	go func() {
		done <- pm.process.Wait()
	}()

	select {
	case err := <-done:
		exitTime := time.Now()
		pm.info.ExitTime = &exitTime
		
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode := exitErr.ExitCode()
				pm.info.ExitCode = &exitCode
			}
			pm.info.State = StateError
			pm.info.Error = err.Error()
		} else {
			pm.info.State = StateStopped
		}
	case <-time.After(10 * time.Second):
		// 超时强制终止
		if err := pm.process.Process.Kill(); err != nil {
			pm.info.State = StateError
			pm.info.Error = err.Error()
			return fmt.Errorf("进程终止超时: %v", err)
		}
		<-done // 等待进程真正结束
		exitTime := time.Now()
		pm.info.ExitTime = &exitTime
		pm.info.State = StateStopped
	}

	pm.process = nil
	return nil
}

// GetInfo 获取进程信息
func (pm *ProcessManager) GetInfo() *ProcessInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 返回副本以避免并发修改
	info := *pm.info
	return &info
}

// GetPID 获取进程PID
func (pm *ProcessManager) GetPID() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	if pm.info == nil {
		return 0
	}
	return pm.info.PID
}

// IsRunning 检查进程是否在运行
func (pm *ProcessManager) IsRunning() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 如果状态不是运行中，直接返回false
	if pm.info.State != StateRunning {
		return false
	}

	// 如果进程为nil，返回false
	if pm.process == nil {
		return false
	}

	// 检查进程是否真的在运行
	process := pm.process.Process
	if process == nil {
		return false
	}

	// 检查进程是否已经退出（通过ExitTime判断）
	if pm.info.ExitTime != nil {
		return false
	}

	// 尝试向进程发送信号0（不发送信号，只检查进程是否存在）
	// 在Unix系统上，向进程发送信号0可以检查进程是否存在
	err := process.Signal(syscall.Signal(0))
	return err == nil
}

// monitorProcess 监控进程状态
func (pm *ProcessManager) monitorProcess() {
	if pm.process == nil {
		return
	}

	// 等待进程结束
	err := pm.process.Wait()
	
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.info.State == StateStopping {
		// 如果是主动停止，状态已经在 Stop 方法中更新
		return
	}

	exitTime := time.Now()
	pm.info.ExitTime = &exitTime

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			pm.info.ExitCode = &exitCode
			fmt.Printf("进程异常退出，退出码: %d, 错误: %v\n", exitCode, err)
		} else {
			fmt.Printf("进程退出错误: %v\n", err)
		}
		pm.info.State = StateError
		pm.info.Error = err.Error()
	} else {
		fmt.Printf("进程正常退出\n")
		pm.info.State = StateStopped
	}

	pm.process = nil
}

// Wait 等待进程结束
func (pm *ProcessManager) Wait() error {
	pm.mu.RLock()
	process := pm.process
	pm.mu.RUnlock()

	if process == nil {
		return fmt.Errorf("进程未运行")
	}

	return process.Wait()
}

// Cleanup 清理资源
func (pm *ProcessManager) Cleanup() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cancel != nil {
		pm.cancel()
	}

	if pm.process != nil && pm.info.State == StateRunning {
		// 尝试优雅终止
		_ = pm.process.Process.Signal(os.Interrupt)
		
		// 等待一段时间后强制终止
		select {
		case <-time.After(5 * time.Second):
			_ = pm.process.Process.Kill()
		case <-pm.ctx.Done():
		}
	}

	pm.process = nil
	pm.info.State = StateStopped
}

// isServerProcess 判断是否为服务端进程
func (pm *ProcessManager) isServerProcess(cmd *exec.Cmd) bool {
	// 检查命令参数，如果包含 -l 参数，说明是服务端进程
	for _, arg := range cmd.Args {
		if arg == "-l" {
			return true
		}
	}
	return false
}