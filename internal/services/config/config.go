package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"rdma-burst/internal/models"
	"rdma-burst/internal/utils"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	configType string // "server" 或 "client"
	viper      *viper.Viper
}

// NewConfigManager 创建新的配置管理器
func NewConfigManager(configType string) *ConfigManager {
	v := viper.New()
	
	// 设置配置类型
	v.SetConfigType("yaml")
	
	// 设置环境变量前缀
	v.SetEnvPrefix("RDMA")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	
	// 设置时间解析
	v.SetTypeByDefaultValue(true)
	
	return &ConfigManager{
		configType: configType,
		viper:      v,
	}
}

// LoadConfig 加载配置
func (cm *ConfigManager) LoadConfig(configPath string) (interface{}, error) {
	// 如果配置文件路径为空，使用默认配置
	if configPath == "" {
		return cm.getDefaultConfig(), nil
	}
	
	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}
	
	// 设置配置文件路径
	cm.viper.SetConfigFile(configPath)
	
	// 读取配置文件
	if err := cm.viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}
	
	// 根据配置类型加载不同的配置结构
	switch cm.configType {
	case "server":
		return cm.loadServerConfig()
	case "client":
		return cm.loadClientConfig()
	default:
		return nil, fmt.Errorf("不支持的配置类型: %s", cm.configType)
	}
}

// loadServerConfig 加载服务端配置
func (cm *ConfigManager) loadServerConfig() (*models.ServerConfig, error) {
	var config models.ServerConfig
	
	// 绑定环境变量
	cm.bindServerEnvVars()
	
	// 解析配置到结构体
	if err := cm.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析服务端配置失败: %v", err)
	}
	
	// 手动解析时间字段（如果自动解析失败）
	cm.fixTimeFields(&config)
	
	// 验证配置
	if err := cm.validateServerConfig(&config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// loadClientConfig 加载客户端配置
func (cm *ConfigManager) loadClientConfig() (*models.ClientConfig, error) {
	var config models.ClientConfig
	
	// 绑定环境变量
	cm.bindClientEnvVars()
	
	// 解析配置到结构体
	if err := cm.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析客户端配置失败: %v", err)
	}
	
	// 手动解析时间字段（如果自动解析失败）
	cm.fixTimeFields(&config)
	
	// 自动检测服务端地址（如果配置为localhost）
	cm.autoDetectServerAddress(&config)
	
	// 验证配置
	if err := cm.validateClientConfig(&config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// bindServerEnvVars 绑定服务端环境变量
func (cm *ConfigManager) bindServerEnvVars() {
	// 服务端设置
	cm.viper.BindEnv("server.host", "RDMA_SERVER_HOST")
	cm.viper.BindEnv("server.port", "RDMA_SERVER_PORT")
	cm.viper.BindEnv("server.log_level", "RDMA_SERVER_LOG_LEVEL")
	
	// 传输设置
	cm.viper.BindEnv("transfer.device", "RDMA_TRANSFER_DEVICE")
	cm.viper.BindEnv("transfer.base_dir", "RDMA_TRANSFER_BASE_DIR")
	cm.viper.BindEnv("transfer.transfer_interval", "RDMA_TRANSFER_INTERVAL")
	cm.viper.BindEnv("transfer.max_concurrent_transfers", "RDMA_MAX_CONCURRENT_TRANSFERS")
	cm.viper.BindEnv("transfer.chunk_size", "RDMA_CHUNK_SIZE")
	
	// 日志设置
	cm.viper.BindEnv("logging.file_path", "RDMA_LOG_FILE_PATH")
	cm.viper.BindEnv("logging.level", "RDMA_LOG_LEVEL")
	
	// 监控设置
	cm.viper.BindEnv("monitoring.health_check_interval", "RDMA_HEALTH_CHECK_INTERVAL")
	cm.viper.BindEnv("monitoring.enable_metrics", "RDMA_ENABLE_METRICS")
	cm.viper.BindEnv("monitoring.metrics_port", "RDMA_METRICS_PORT")
}

// bindClientEnvVars 绑定客户端环境变量
func (cm *ConfigManager) bindClientEnvVars() {
	// 服务端连接设置
	cm.viper.BindEnv("server.host", "RDMA_SERVER_HOST")
	cm.viper.BindEnv("server.port", "RDMA_SERVER_PORT")
	cm.viper.BindEnv("server.timeout", "RDMA_SERVER_TIMEOUT")
	cm.viper.BindEnv("server.retry_attempts", "RDMA_RETRY_ATTEMPTS")
	cm.viper.BindEnv("server.retry_delay", "RDMA_RETRY_DELAY")
	
	// 传输设置
	cm.viper.BindEnv("transfer.device", "RDMA_TRANSFER_DEVICE")
	cm.viper.BindEnv("transfer.base_dir", "RDMA_TRANSFER_BASE_DIR")
	cm.viper.BindEnv("transfer.transfer_interval", "RDMA_TRANSFER_INTERVAL")
	cm.viper.BindEnv("transfer.chunk_size", "RDMA_CHUNK_SIZE")
	cm.viper.BindEnv("transfer.default_mode", "RDMA_DEFAULT_MODE")
	
	// 日志设置
	cm.viper.BindEnv("logging.file_path", "RDMA_LOG_FILE_PATH")
	cm.viper.BindEnv("logging.level", "RDMA_LOG_LEVEL")
	
	// 客户端特定设置
	cm.viper.BindEnv("client.max_parallel_transfers", "RDMA_MAX_PARALLEL_TRANSFERS")
	cm.viper.BindEnv("client.enable_checksum", "RDMA_ENABLE_CHECKSUM")
	cm.viper.BindEnv("client.checksum_algorithm", "RDMA_CHECKSUM_ALGORITHM")
}

// validateServerConfig 验证服务端配置
func (cm *ConfigManager) validateServerConfig(config *models.ServerConfig) error {
	// 验证服务端设置
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("服务端口必须在 1-65535 范围内")
	}
	
	if config.Server.ReadTimeout <= 0 {
		return fmt.Errorf("读取超时必须大于 0")
	}
	
	if config.Server.WriteTimeout <= 0 {
		return fmt.Errorf("写入超时必须大于 0")
	}
	
	// 验证传输设置
	if config.Transfer.Device == "" {
		return fmt.Errorf("RDMA 设备不能为空")
	}
	
	if config.Transfer.BaseDir == "" {
		return fmt.Errorf("基础目录不能为空")
	}
	
	if config.Transfer.TransferInterval <= 0 {
		return fmt.Errorf("传输间隔必须大于 0")
	}
	
	if config.Transfer.MaxConcurrentTransfers <= 0 {
		return fmt.Errorf("最大并发传输数必须大于 0")
	}
	
	if config.Transfer.ChunkSize <= 0 {
		return fmt.Errorf("块大小必须大于 0")
	}
	
	// 验证传输模式配置
	if err := cm.validateTransferModes(&config.Transfer.Modes); err != nil {
		return err
	}
	
	// 验证日志设置
	if config.Logging.FilePath == "" {
		return fmt.Errorf("日志文件路径不能为空")
	}
	
	// 验证监控设置
	if config.Monitoring.HealthCheckInterval <= 0 {
		return fmt.Errorf("健康检查间隔必须大于 0")
	}
	
	return nil
}

// validateClientConfig 验证客户端配置
func (cm *ConfigManager) validateClientConfig(config *models.ClientConfig) error {
	// 验证服务端连接设置
	if config.Server.Host == "" {
		return fmt.Errorf("服务端主机不能为空")
	}
	
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("服务端口必须在 1-65535 范围内")
	}
	
	if config.Server.Timeout <= 0 {
		return fmt.Errorf("连接超时必须大于 0")
	}
	
	// 验证传输设置
	if config.Transfer.Device == "" {
		return fmt.Errorf("RDMA 设备不能为空")
	}
	
	if config.Transfer.BaseDir == "" {
		return fmt.Errorf("基础目录不能为空")
	}
	
	if config.Transfer.TransferInterval <= 0 {
		return fmt.Errorf("传输间隔必须大于 0")
	}
	
	if config.Transfer.ChunkSize <= 0 {
		return fmt.Errorf("块大小必须大于 0")
	}
	
	// 验证传输模式
	if config.Transfer.DefaultMode != "" {
		validModes := map[string]bool{
			"hugepages":  true,
			"tmpfs":      true,
			"filesystem": true,
		}
		if !validModes[config.Transfer.DefaultMode] {
			return fmt.Errorf("不支持的默认传输模式: %s", config.Transfer.DefaultMode)
		}
	}
	
	// 验证传输模式配置
	if err := cm.validateTransferModes(&config.Transfer.Modes); err != nil {
		return err
	}
	
	// 验证日志设置
	if config.Logging.FilePath == "" {
		return fmt.Errorf("日志文件路径不能为空")
	}
	
	// 验证客户端设置
	if config.Client.MaxParallelTransfers <= 0 {
		return fmt.Errorf("最大并行传输数必须大于 0")
	}
	
	return nil
}

// validateTransferModes 验证传输模式配置
func (cm *ConfigManager) validateTransferModes(modes *models.TransferModes) error {
	// 验证大页内存模式
	if modes.Hugepages.Enabled && modes.Hugepages.BaseDir == "" {
		return fmt.Errorf("大页内存模式启用时，基础目录不能为空")
	}
	
	// 验证 tmpfs 模式
	if modes.Tmpfs.Enabled && modes.Tmpfs.BaseDir == "" {
		return fmt.Errorf("tmpfs 模式启用时，基础目录不能为空")
	}
	
	// 验证文件系统模式
	if modes.Filesystem.Enabled && modes.Filesystem.BaseDir == "" {
		return fmt.Errorf("文件系统模式启用时，基础目录不能为空")
	}
	
	return nil
}

// getDefaultConfig 获取默认配置
func (cm *ConfigManager) getDefaultConfig() interface{} {
	switch cm.configType {
	case "server":
		return models.GetDefaultServerConfig()
	case "client":
		return models.GetDefaultClientConfig()
	default:
		return nil
	}
}

// CreateConfigFile 创建配置文件
func (cm *ConfigManager) CreateConfigFile(configPath string, config interface{}) error {
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}
	
	// 根据配置类型设置默认值
	switch cfg := config.(type) {
	case *models.ServerConfig:
		if cfg == nil {
			cfg = models.GetDefaultServerConfig()
		}
	case *models.ClientConfig:
		if cfg == nil {
			cfg = models.GetDefaultClientConfig()
		}
	default:
		return fmt.Errorf("不支持的配置类型")
	}
	
	// 设置配置文件路径
	cm.viper.SetConfigFile(configPath)
	
	// 将配置写入文件
	if err := cm.viper.WriteConfig(); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}
	
	return nil
}

// GetConfigValue 获取配置值
func (cm *ConfigManager) GetConfigValue(key string) interface{} {
	return cm.viper.Get(key)
}

// SetConfigValue 设置配置值
func (cm *ConfigManager) SetConfigValue(key string, value interface{}) {
	cm.viper.Set(key, value)
}

// fixTimeFields 修复时间字段（如果自动解析失败）
func (cm *ConfigManager) fixTimeFields(config interface{}) {
	switch cfg := config.(type) {
	case *models.ServerConfig:
		cm.fixServerTimeFields(cfg)
	case *models.ClientConfig:
		cm.fixClientTimeFields(cfg)
	}
}

// fixServerTimeFields 修复服务端时间字段
func (cm *ConfigManager) fixServerTimeFields(config *models.ServerConfig) {
	// 如果超时字段为0，尝试从 Viper 获取字符串值并解析
	if config.Server.ReadTimeout == 0 {
		if strVal, ok := cm.viper.Get("server.read_timeout").(string); ok {
			if duration, err := time.ParseDuration(strVal); err == nil {
				config.Server.ReadTimeout = duration
			}
		}
	}
	
	if config.Server.WriteTimeout == 0 {
		if strVal, ok := cm.viper.Get("server.write_timeout").(string); ok {
			if duration, err := time.ParseDuration(strVal); err == nil {
				config.Server.WriteTimeout = duration
			}
		}
	}
	
	if config.Transfer.TransferInterval == 0 {
		if strVal, ok := cm.viper.Get("transfer.transfer_interval").(string); ok {
			if duration, err := time.ParseDuration(strVal); err == nil {
				config.Transfer.TransferInterval = duration
			}
		}
	}
	
	if config.Monitoring.HealthCheckInterval == 0 {
		if strVal, ok := cm.viper.Get("monitoring.server.health_check_interval").(string); ok {
			if duration, err := time.ParseDuration(strVal); err == nil {
				config.Monitoring.HealthCheckInterval = duration
			}
		}
	}
	
	// 修复服务端日志路径
	if config.Logging.FilePath == "" {
		if logPath, ok := cm.viper.Get("logging.server.file_path").(string); ok && logPath != "" {
			config.Logging.FilePath = logPath
		}
	}
}

// fixClientTimeFields 修复客户端时间字段
func (cm *ConfigManager) fixClientTimeFields(config *models.ClientConfig) {
	// 如果超时字段为0，尝试从 Viper 获取字符串值并解析
	if config.Server.Timeout == 0 {
		if strVal, ok := cm.viper.Get("client.timeout").(string); ok {
			if duration, err := time.ParseDuration(strVal); err == nil {
				config.Server.Timeout = duration
			}
		}
	}
	
	if config.Server.RetryDelay == 0 {
		if strVal, ok := cm.viper.Get("client.retry_delay").(string); ok {
			if duration, err := time.ParseDuration(strVal); err == nil {
				config.Server.RetryDelay = duration
			}
		}
	}
	
	if config.Transfer.TransferInterval == 0 {
		if strVal, ok := cm.viper.Get("transfer.transfer_interval").(string); ok {
			if duration, err := time.ParseDuration(strVal); err == nil {
				config.Transfer.TransferInterval = duration
			}
		}
	}
	
	if config.Monitoring.ProgressUpdateInterval == 0 {
		if strVal, ok := cm.viper.Get("monitoring.client.progress_update_interval").(string); ok {
			if duration, err := time.ParseDuration(strVal); err == nil {
				config.Monitoring.ProgressUpdateInterval = duration
			}
		}
	}
	
	if config.Monitoring.TransferTimeout == 0 {
		if strVal, ok := cm.viper.Get("monitoring.client.transfer_timeout").(string); ok {
			if duration, err := time.ParseDuration(strVal); err == nil {
				config.Monitoring.TransferTimeout = duration
			}
		}
	}
	
	if config.Client.ResumeCheckInterval == 0 {
		if strVal, ok := cm.viper.Get("client_specific.resume_check_interval").(string); ok {
			if duration, err := time.ParseDuration(strVal); err == nil {
				config.Client.ResumeCheckInterval = duration
			}
		}
	}
	
	// 修复客户端日志路径
	if config.Logging.FilePath == "" {
		if logPath, ok := cm.viper.Get("logging.client.file_path").(string); ok && logPath != "" {
			config.Logging.FilePath = logPath
		}
	}
	
	// 修复客户端并行传输数
	if config.Client.MaxParallelTransfers <= 0 {
		if maxTransfers := cm.viper.GetInt("client_specific.max_parallel_transfers"); maxTransfers > 0 {
			config.Client.MaxParallelTransfers = maxTransfers
		}
	}
}

// autoDetectServerAddress 自动检测服务端地址
func (cm *ConfigManager) autoDetectServerAddress(config *models.ClientConfig) {
	// 如果服务端地址是localhost，尝试根据RDMA设备自动检测
	if config.Server.Host == "localhost" || config.Server.Host == "127.0.0.1" {
		// 尝试根据RDMA设备获取IP地址
		if config.Transfer.Device != "" {
			ip, err := utils.GetIPFromRDMAInterface(config.Transfer.Device)
			if err == nil && ip != "" {
				config.Server.Host = ip
				return
			}
		}
		
		// 如果RDMA设备检测失败，尝试获取本地IP
		ip, err := utils.GetLocalIP()
		if err == nil && ip != "" {
			config.Server.Host = ip
		}
	}
}

// SaveConfig 保存配置到文件
func (cm *ConfigManager) SaveConfig() error {
	return cm.viper.WriteConfig()
}