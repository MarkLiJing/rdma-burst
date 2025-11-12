package models

import (
	"time"
)

// CombinedConfig 定义统一配置
type CombinedConfig struct {
	Mode            string                 `mapstructure:"mode" json:"mode"`
	Server          ServerSettings         `mapstructure:"server" json:"server"`
	Client          ClientServerSettings   `mapstructure:"client" json:"client"`
	Transfer        TransferSettings       `mapstructure:"transfer" json:"transfer"`
	Logging         CombinedLoggingSettings `mapstructure:"logging" json:"logging"`
	Monitoring      CombinedMonitoringSettings `mapstructure:"monitoring" json:"monitoring"`
	Security        SecuritySettings       `mapstructure:"security" json:"security"`
	ClientSpecific  ClientSpecificSettings `mapstructure:"client_specific" json:"client_specific"`
	Mutex           MutexSettings          `mapstructure:"mutex" json:"mutex"`
	SingleTransfer  SingleTransferSettings `mapstructure:"single_transfer" json:"single_transfer"`
}

// ServerConfig 定义服务端配置
type ServerConfig struct {
	Server    ServerSettings    `mapstructure:"server" json:"server"`
	Transfer  TransferSettings  `mapstructure:"transfer" json:"transfer"`
	Logging   LoggingSettings   `mapstructure:"logging" json:"logging"`
	Monitoring MonitoringSettings `mapstructure:"monitoring" json:"monitoring"`
	Security  SecuritySettings  `mapstructure:"security" json:"security"`
}

// ClientConfig 定义客户端配置
type ClientConfig struct {
	Server    ClientServerSettings `mapstructure:"client" json:"server"`
	Transfer  TransferSettings     `mapstructure:"transfer" json:"transfer"`
	Logging   LoggingSettings      `mapstructure:"logging" json:"logging"`
	Monitoring ClientMonitoringSettings `mapstructure:"monitoring" json:"monitoring"`
	Security  SecuritySettings     `mapstructure:"security" json:"security"`
	Client    ClientSpecificSettings `mapstructure:"client_specific" json:"client"`
}

// ServerSettings 定义服务端设置
type ServerSettings struct {
	Host           string        `mapstructure:"host" json:"host"`
	Port           int           `mapstructure:"port" json:"port"`
	LogLevel       string        `mapstructure:"log_level" json:"log_level"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout" json:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout" json:"write_timeout"`
	MaxHeaderBytes int           `mapstructure:"max_header_bytes" json:"max_header_bytes"`
}

// ClientServerSettings 定义客户端服务端连接设置
type ClientServerSettings struct {
	Host         string        `mapstructure:"host" json:"host"`
	Port         int           `mapstructure:"port" json:"port"`
	Timeout      time.Duration `mapstructure:"timeout" json:"timeout"`
	RetryAttempts int          `mapstructure:"retry_attempts" json:"retry_attempts"`
	RetryDelay   time.Duration `mapstructure:"retry_delay" json:"retry_delay"`
}

// TransferSettings 定义传输设置
type TransferSettings struct {
	Device                string            `mapstructure:"device" json:"device"`
	BaseDir               string            `mapstructure:"base_dir" json:"base_dir"`
	TransferInterval      time.Duration     `mapstructure:"transfer_interval" json:"transfer_interval"`
	MaxConcurrentTransfers int              `mapstructure:"max_concurrent_transfers" json:"max_concurrent_transfers"`
	ChunkSize            int               `mapstructure:"chunk_size" json:"chunk_size"`
	Modes                TransferModes     `mapstructure:"modes" json:"modes"`
	DefaultMode          string            `mapstructure:"default_mode" json:"default_mode,omitempty"`
	ServerAddress        string            `mapstructure:"server_address,omitempty" json:"server_address,omitempty"` // 临时字段，用于传递服务端地址
}

// TransferModes 定义传输模式配置
type TransferModes struct {
	Hugepages  ModeConfig `mapstructure:"hugepages" json:"hugepages"`
	Tmpfs      ModeConfig `mapstructure:"tmpfs" json:"tmpfs"`
	Filesystem ModeConfig `mapstructure:"filesystem" json:"filesystem"`
}

// ModeConfig 定义模式配置
type ModeConfig struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled"`
	BaseDir string `mapstructure:"base_dir" json:"base_dir"`
}

// LoggingSettings 定义日志设置
type LoggingSettings struct {
	FilePath   string `mapstructure:"file_path" json:"file_path"`
	MaxSize    int    `mapstructure:"max_size" json:"max_size"`
	MaxBackups int    `mapstructure:"max_backups" json:"max_backups"`
	MaxAge     int    `mapstructure:"max_age" json:"max_age"`
	Level      string `mapstructure:"level" json:"level"`
	Format     string `mapstructure:"format" json:"format"`
}

// MonitoringSettings 定义监控设置
type MonitoringSettings struct {
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval" json:"health_check_interval"`
	EnableMetrics       bool          `mapstructure:"enable_metrics" json:"enable_metrics"`
	MetricsPort         int           `mapstructure:"metrics_port" json:"metrics_port"`
}

// ClientMonitoringSettings 定义客户端监控设置
type ClientMonitoringSettings struct {
	ProgressUpdateInterval time.Duration `mapstructure:"progress_update_interval" json:"progress_update_interval"`
	TransferTimeout       time.Duration `mapstructure:"transfer_timeout" json:"transfer_timeout"`
}

// SecuritySettings 定义安全设置
type SecuritySettings struct {
	CORS      CORSSettings      `mapstructure:"cors" json:"cors"`
	RateLimit RateLimitSettings `mapstructure:"rate_limit" json:"rate_limit"`
	TLS       TLSSettings       `mapstructure:"tls" json:"tls,omitempty"`
	Auth      AuthSettings      `mapstructure:"auth" json:"auth,omitempty"`
}

// CORSSettings 定义 CORS 设置
type CORSSettings struct {
	Enabled         bool     `mapstructure:"enabled" json:"enabled"`
	AllowedOrigins  []string `mapstructure:"allowed_origins" json:"allowed_origins"`
	AllowedMethods  []string `mapstructure:"allowed_methods" json:"allowed_methods"`
	AllowedHeaders  []string `mapstructure:"allowed_headers" json:"allowed_headers"`
}

// RateLimitSettings 定义速率限制设置
type RateLimitSettings struct {
	Enabled           bool `mapstructure:"enabled" json:"enabled"`
	RequestsPerSecond int  `mapstructure:"requests_per_second" json:"requests_per_second"`
	Burst             int  `mapstructure:"burst" json:"burst"`
}

// TLSSettings 定义 TLS 设置
type TLSSettings struct {
	Enabled     bool   `mapstructure:"enabled" json:"enabled"`
	CACert      string `mapstructure:"ca_cert" json:"ca_cert"`
	ClientCert  string `mapstructure:"client_cert" json:"client_cert"`
	ClientKey   string `mapstructure:"client_key" json:"client_key"`
}

// AuthSettings 定义认证设置
type AuthSettings struct {
	Enabled  bool   `mapstructure:"enabled" json:"enabled"`
	Token    string `mapstructure:"token" json:"token"`
	Username string `mapstructure:"username" json:"username"`
	Password string `mapstructure:"password" json:"password"`
}

// CombinedLoggingSettings 定义统一日志设置
type CombinedLoggingSettings struct {
	Server LoggingSettings `mapstructure:"server" json:"server"`
	Client LoggingSettings `mapstructure:"client" json:"client"`
}

// CombinedMonitoringSettings 定义统一监控设置
type CombinedMonitoringSettings struct {
	Server MonitoringSettings       `mapstructure:"server" json:"server"`
	Client ClientMonitoringSettings `mapstructure:"client" json:"client"`
}

// MutexSettings 定义互斥启动设置
type MutexSettings struct {
	Enabled       bool          `mapstructure:"enabled" json:"enabled"`
	CheckTimeout  time.Duration `mapstructure:"check_timeout" json:"check_timeout"`
	RetryCount    int           `mapstructure:"retry_count" json:"retry_count"`
	RetryInterval time.Duration `mapstructure:"retry_interval" json:"retry_interval"`
}

// SingleTransferSettings 定义单次传输设置
type SingleTransferSettings struct {
	Enabled           bool          `mapstructure:"enabled" json:"enabled"`
	AutoClose         bool          `mapstructure:"auto_close" json:"auto_close"`
	RequireReconnect  bool          `mapstructure:"require_reconnect" json:"require_reconnect"`
	KeepAliveTimeout  time.Duration `mapstructure:"keep_alive_timeout" json:"keep_alive_timeout"`
}

// ClientSpecificSettings 定义客户端特定设置
type ClientSpecificSettings struct {
	MaxParallelTransfers int           `mapstructure:"max_parallel_transfers" json:"max_parallel_transfers"`
	EnableChecksum       bool          `mapstructure:"enable_checksum" json:"enable_checksum"`
	ChecksumAlgorithm    string        `mapstructure:"checksum_algorithm" json:"checksum_algorithm"`
	EnableResume         bool          `mapstructure:"enable_resume" json:"enable_resume"`
	ResumeCheckInterval  time.Duration `mapstructure:"resume_check_interval" json:"resume_check_interval"`
}

// GetDefaultServerConfig 获取默认服务端配置
func GetDefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Server: ServerSettings{
			Host:           "0.0.0.0",
			Port:           8080,
			LogLevel:       "info",
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1048576,
		},
		Transfer: TransferSettings{
			Device:                "mlx5_0",
			BaseDir:               "/var/lib/rtrans",
			TransferInterval:      5 * time.Second,
			MaxConcurrentTransfers: 1,
			ChunkSize:             4194304, // 4MB
			Modes: TransferModes{
				Hugepages: ModeConfig{
					Enabled: true,
					BaseDir: "/dev/hugepages/dir",
				},
				Tmpfs: ModeConfig{
					Enabled: true,
					BaseDir: "/dev/shm/dir",
				},
				Filesystem: ModeConfig{
					Enabled: true,
					BaseDir: "/var/lib/rtrans/files",
				},
			},
		},
		Logging: LoggingSettings{
			FilePath:   "/var/log/rtrans/rtrans_server.log",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
			Level:      "info",
			Format:     "json",
		},
		Monitoring: MonitoringSettings{
			HealthCheckInterval: 30 * time.Second,
			EnableMetrics:       true,
			MetricsPort:         9090,
		},
		Security: SecuritySettings{
			CORS: CORSSettings{
				Enabled:         true,
				AllowedOrigins:  []string{"*"},
				AllowedMethods:  []string{"GET", "POST", "DELETE"},
				AllowedHeaders:  []string{"Content-Type", "Authorization"},
			},
			RateLimit: RateLimitSettings{
				Enabled:           true,
				RequestsPerSecond: 10,
				Burst:             20,
			},
		},
	}
}

// GetDefaultCombinedConfig 获取默认统一配置
func GetDefaultCombinedConfig() *CombinedConfig {
	return &CombinedConfig{
		Mode: "auto",
		Server: ServerSettings{
			Host:           "0.0.0.0",
			Port:           8080,
			LogLevel:       "info",
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1048576,
		},
		Client: ClientServerSettings{
			Host:         "localhost",
			Port:         8080,
			Timeout:      30 * time.Second,
			RetryAttempts: 3,
			RetryDelay:   5 * time.Second,
		},
		Transfer: TransferSettings{
			Device:                "mlx5_0",
			BaseDir:               "/var/lib/rtrans",
			TransferInterval:      5 * time.Second,
			MaxConcurrentTransfers: 1,
			ChunkSize:             4194304, // 4MB
			DefaultMode:           "filesystem",
			Modes: TransferModes{
				Hugepages: ModeConfig{
					Enabled: true,
					BaseDir: "/dev/hugepages/dir",
				},
				Tmpfs: ModeConfig{
					Enabled: true,
					BaseDir: "/dev/shm/dir",
				},
				Filesystem: ModeConfig{
					Enabled: true,
					BaseDir: "/var/lib/rtrans/files",
				},
			},
		},
		Logging: CombinedLoggingSettings{
			Server: LoggingSettings{
				FilePath:   "/var/log/rtrans/rtrans_server.log",
				MaxSize:    100,
				MaxBackups: 5,
				MaxAge:     30,
				Level:      "info",
				Format:     "json",
			},
			Client: LoggingSettings{
				FilePath:   "/var/log/rtrans/rtrans_client.log",
				MaxSize:    50,
				MaxBackups: 3,
				MaxAge:     7,
				Level:      "info",
				Format:     "text",
			},
		},
		Monitoring: CombinedMonitoringSettings{
			Server: MonitoringSettings{
				HealthCheckInterval: 30 * time.Second,
				EnableMetrics:       true,
				MetricsPort:         9090,
			},
			Client: ClientMonitoringSettings{
				ProgressUpdateInterval: 5 * time.Second,
				TransferTimeout:       1 * time.Hour,
			},
		},
		Security: SecuritySettings{
			CORS: CORSSettings{
				Enabled:         true,
				AllowedOrigins:  []string{"*"},
				AllowedMethods:  []string{"GET", "POST", "DELETE"},
				AllowedHeaders:  []string{"Content-Type", "Authorization"},
			},
			RateLimit: RateLimitSettings{
				Enabled:           true,
				RequestsPerSecond: 10,
				Burst:             20,
			},
			TLS: TLSSettings{
				Enabled: false,
			},
			Auth: AuthSettings{
				Enabled: false,
			},
		},
		ClientSpecific: ClientSpecificSettings{
			MaxParallelTransfers: 1,
			EnableChecksum:       true,
			ChecksumAlgorithm:    "sha256",
			EnableResume:         true,
			ResumeCheckInterval:  10 * time.Second,
		},
		Mutex: MutexSettings{
			Enabled:       true,
			CheckTimeout:  3 * time.Second,
			RetryCount:    3,
			RetryInterval: 1 * time.Second,
		},
		SingleTransfer: SingleTransferSettings{
			Enabled:          true,
			AutoClose:        true,
			RequireReconnect: true,
			KeepAliveTimeout: 10 * time.Second,
		},
	}
}

// GetDefaultClientConfig 获取默认客户端配置
func GetDefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Server: ClientServerSettings{
			Host:         "localhost",
			Port:         8080,
			Timeout:      30 * time.Second,
			RetryAttempts: 3,
			RetryDelay:   5 * time.Second,
		},
		Transfer: TransferSettings{
			Device:           "mlx5_0",
			BaseDir:          "/var/lib/rtrans",
			TransferInterval: 5 * time.Second,
			ChunkSize:        4194304, // 4MB
			DefaultMode:      "filesystem",
			Modes: TransferModes{
				Hugepages: ModeConfig{
					Enabled: true,
					BaseDir: "/dev/hugepages/dir",
				},
				Tmpfs: ModeConfig{
					Enabled: true,
					BaseDir: "/dev/shm/dir",
				},
				Filesystem: ModeConfig{
					Enabled: true,
					BaseDir: "/var/lib/rtrans/files",
				},
			},
		},
		Logging: LoggingSettings{
			FilePath:   "/var/log/rtrans/rtrans_client.log",
			MaxSize:    50,
			MaxBackups: 3,
			MaxAge:     7,
			Level:      "info",
			Format:     "text",
		},
		Monitoring: ClientMonitoringSettings{
			ProgressUpdateInterval: 5 * time.Second,
			TransferTimeout:       1 * time.Hour,
		},
		Security: SecuritySettings{
			TLS: TLSSettings{
				Enabled: false,
			},
			Auth: AuthSettings{
				Enabled: false,
			},
		},
		Client: ClientSpecificSettings{
			MaxParallelTransfers: 1,
			EnableChecksum:       true,
			ChecksumAlgorithm:    "sha256",
			EnableResume:         true,
			ResumeCheckInterval:  10 * time.Second,
		},
	}
}