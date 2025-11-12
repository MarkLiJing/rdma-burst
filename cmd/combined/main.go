package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"rdma-burst/internal/api/handlers"
	"rdma-burst/internal/api/middleware"
	"rdma-burst/internal/models"
	"rdma-burst/internal/services/config"
	"rdma-burst/internal/services/transfer"
	"rdma-burst/internal/wrapper"
	"rdma-burst/pkg/logger"
)

// 构建信息
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// 运行模式
const (
	ModeServer = "server"
	ModeClient = "client"
	ModeAuto   = "auto"
)

// 应用配置
type AppConfig struct {
	Mode         string        `mapstructure:"mode"` // server, client, auto
	ServerConfig *models.ServerConfig
	ClientConfig *models.ClientConfig
}

func main() {
	// 解析命令行参数
	var configPath string
	var mode string
	var showVersion bool

	flag.StringVar(&configPath, "config", "", "配置文件路径")
	flag.StringVar(&mode, "mode", ModeAuto, "运行模式: server, client, auto")
	flag.BoolVar(&showVersion, "version", false, "显示版本信息")
	flag.Parse()

	if showVersion {
		printVersion()
		return
	}

	// 初始化日志
	logger, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()

	// 加载配置
	appConfig, err := loadConfig(configPath, mode)
	if err != nil {
		logger.Fatal("加载配置失败", zap.Error(err))
	}

	// 确定运行模式
	runtimeMode := determineRuntimeMode(appConfig.Mode, logger)
	logger.Info("确定运行模式", zap.String("mode", runtimeMode))

	// 根据模式启动应用
	switch runtimeMode {
	case ModeServer:
		startServer(appConfig.ServerConfig, logger)
	case ModeClient:
		startClient(appConfig.ClientConfig, logger)
	default:
		logger.Fatal("未知的运行模式", zap.String("mode", runtimeMode))
	}
}

// loadConfig 加载配置
func loadConfig(configPath string, mode string) (*AppConfig, error) {
	if configPath == "" {
		// 使用默认配置路径
		configPath = "./configs/combined.yaml"
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}

	// 根据模式加载配置
	appConfig := &AppConfig{
		Mode: mode,
	}

	switch mode {
	case ModeServer:
		// 服务端模式使用服务端配置
		serverConfigManager := config.NewConfigManager("server")
		serverConfig, err := serverConfigManager.LoadConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("加载服务端配置失败: %v", err)
		}
		appConfig.ServerConfig = serverConfig.(*models.ServerConfig)
		
	case ModeClient:
		// 客户端模式使用客户端配置
		clientConfigManager := config.NewConfigManager("client")
		clientConfig, err := clientConfigManager.LoadConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("加载客户端配置失败: %v", err)
		}
		appConfig.ClientConfig = clientConfig.(*models.ClientConfig)
		
	default:
		// 自动模式：先尝试加载服务端配置，如果失败则使用默认配置
		serverConfigManager := config.NewConfigManager("server")
		serverConfig, err := serverConfigManager.LoadConfig(configPath)
		if err != nil {
			// 如果加载失败，使用默认服务端配置
			appConfig.ServerConfig = models.GetDefaultServerConfig()
		} else {
			appConfig.ServerConfig = serverConfig.(*models.ServerConfig)
		}
		
		// 客户端配置使用默认值
		appConfig.ClientConfig = models.GetDefaultClientConfig()
	}

	return appConfig, nil
}

// determineRuntimeMode 确定运行模式
func determineRuntimeMode(configMode string, logger *zap.Logger) string {
	switch configMode {
	case ModeServer:
		return ModeServer
	case ModeClient:
		return ModeClient
	case ModeAuto:
		// 自动检测模式：尝试连接服务端，如果成功则为客户端，否则为服务端
		return autoDetectMode(logger)
	default:
		logger.Warn("未知的配置模式，使用自动检测", zap.String("mode", configMode))
		return autoDetectMode(logger)
	}
}

// autoDetectMode 自动检测运行模式
func autoDetectMode(logger *zap.Logger) string {
	// 尝试连接本地服务端
	client := &http.Client{Timeout: 3 * time.Second}
	url := "http://localhost:8080/api/health"

	resp, err := client.Get(url)
	if err == nil && resp.StatusCode == http.StatusOK {
		logger.Info("检测到运行中的服务端，启动客户端模式")
		return ModeClient
	}

	logger.Info("未检测到运行中的服务端，启动服务端模式")
	return ModeServer
}

// startServer 启动服务端
func startServer(cfg *models.ServerConfig, logger *zap.Logger) {
	// 检查是否已有服务端在运行
	if isServerRunning(cfg.Server.Host, cfg.Server.Port) {
		logger.Fatal("服务端已在运行，无法启动新的服务端实例")
	}

	// 创建传输服务（使用配置中的传输设置）
	rtranfilePath := getRtranfilePath()
	transferService := transfer.NewTransferServiceWithConfig(
		rtranfilePath,
		&cfg.Transfer,
		nil, // 单次传输配置为空，使用默认值
	)

	// 创建进程映射（按需启动监听进程）
	serverProcesses := make(map[string]*wrapper.ProcessManager)
	
	logger.Info("服务端启动完成，等待客户端传输请求")
	logger.Info("rtranfile 监听进程将按需启动")

	// 设置 Gin 模式
	if cfg.Server.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 引擎
	router := gin.New()

	// 添加中间件
	middleware := middleware.NewLoggerMiddleware(logger)
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(CORSMiddleware(cfg.Security.CORS))

	// 创建 API 处理器
	transferHandler := handlers.NewTransferHandler(transferService, &cfg.Transfer)
	healthHandler := handlers.NewHealthHandler(transferService, version)
	modeHandler := handlers.NewModeHandler(version, ModeServer)

	// 注册路由
	api := router.Group("/api/v1")
	transferHandler.RegisterRoutes(api)
	healthHandler.RegisterRoutes(router.Group("/api"))
	modeHandler.RegisterRoutes(api)

	// 添加模式检测端点（兼容旧版本）
	router.GET("/api/mode", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"mode":    ModeServer,
			"version": version,
			"status":  "running",
		})
	})

	// 根路径健康检查
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "rdma-burst",
			"mode":    ModeServer,
			"version": version,
			"status":  "running",
		})
	})

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// 启动服务器
	go func() {
		logger.Info("启动 RDMA 文件传输服务端",
			zap.String("host", cfg.Server.Host),
			zap.Int("port", cfg.Server.Port),
			zap.String("version", version),
			zap.String("mode", ModeServer),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("启动服务器失败", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务端...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 清理传输服务
	transferService.Cleanup()

	// 停止所有 rtranfile 服务端进程
	for modeName, processMgr := range serverProcesses {
		if err := processMgr.Stop(); err != nil {
			logger.Error("停止 rtranfile 服务端进程失败",
				zap.String("mode", modeName),
				zap.Error(err))
		} else {
			logger.Info("rtranfile 服务端进程已停止",
				zap.String("mode", modeName))
		}
	}

	// 关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("关闭服务器失败", zap.Error(err))
	}

	logger.Info("服务端已关闭")
}

// startClient 启动客户端
func startClient(cfg *models.ClientConfig, logger *zap.Logger) {
	// 检查服务端是否可用
	if !isServerRunning(cfg.Server.Host, cfg.Server.Port) {
		logger.Fatal("服务端不可用，请先启动服务端",
			zap.String("host", cfg.Server.Host),
			zap.Int("port", cfg.Server.Port),
		)
	}

	logger.Info("RDMA 文件传输客户端已连接到服务端",
		zap.String("server_host", cfg.Server.Host),
		zap.Int("server_port", cfg.Server.Port),
		zap.String("version", version),
		zap.String("mode", ModeClient),
	)

	fmt.Printf("RDMA 文件传输客户端已连接到服务端\n")
	fmt.Printf("服务端地址: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	
	// 创建传输服务（客户端使用自己的传输服务）
	rtranfilePath := getRtranfilePath()
	transferService := transfer.NewTransferService(
		rtranfilePath,
		cfg.Transfer.MaxConcurrentTransfers,
		cfg.Transfer.TransferInterval,
	)

	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	// 创建 Gin 引擎
	router := gin.New()

	// 添加中间件
	middleware := middleware.NewLoggerMiddleware(logger)
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(CORSMiddleware(cfg.Security.CORS))

	// 创建 API 处理器（客户端模式使用客户端处理器）
	// 将客户端的传输配置转换为服务端传输配置格式
	serverTransferConfig := &models.TransferSettings{
		Device:                cfg.Transfer.Device,
		BaseDir:               cfg.Transfer.BaseDir,
		TransferInterval:      cfg.Transfer.TransferInterval,
		MaxConcurrentTransfers: cfg.Transfer.MaxConcurrentTransfers,
		ChunkSize:             cfg.Transfer.ChunkSize,
		ServerAddress:         cfg.Server.Host,
		Modes: models.TransferModes{
			Hugepages: models.ModeConfig{
				Enabled: true,
				BaseDir: cfg.Transfer.Modes.Hugepages.BaseDir,
			},
			Tmpfs: models.ModeConfig{
				Enabled: true,
				BaseDir: cfg.Transfer.Modes.Tmpfs.BaseDir,
			},
			Filesystem: models.ModeConfig{
				Enabled: true,
				BaseDir: cfg.Transfer.Modes.Filesystem.BaseDir,
			},
		},
	}
	transferHandler := handlers.NewClientTransferHandler(cfg.Server.Host, cfg.Server.Port, serverTransferConfig)
	healthHandler := handlers.NewHealthHandler(transferService, version)
	modeHandler := handlers.NewModeHandler(version, ModeClient)

	// 注册路由
	api := router.Group("/api/v1")
	transferHandler.RegisterRoutes(api)
	healthHandler.RegisterRoutes(router.Group("/api"))
	modeHandler.RegisterRoutes(api)

	// 添加模式检测端点（兼容旧版本）
	router.GET("/api/mode", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"mode":    ModeClient,
			"version": version,
			"status":  "running",
		})
	})

	// 根路径健康检查
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "rdma-burst",
			"mode":    ModeClient,
			"version": version,
			"status":  "running",
		})
	})

	// 创建 HTTP 服务器（客户端使用不同的端口，避免冲突）
	clientPort := cfg.Server.Port + 1 // 使用服务端端口+1
	server := &http.Server{
		Addr:           fmt.Sprintf("localhost:%d", clientPort),
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// 启动服务器
	go func() {
		logger.Info("启动 RDMA 文件传输客户端API服务",
			zap.String("host", "localhost"),
			zap.Int("port", clientPort),
			zap.String("version", version),
			zap.String("mode", ModeClient),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("启动客户端API服务失败", zap.Error(err))
		}
	}()

	logger.Info("RDMA 文件传输客户端已启动",
		zap.String("server_host", cfg.Server.Host),
		zap.Int("server_port", cfg.Server.Port),
		zap.Int("client_api_port", clientPort),
		zap.String("version", version),
		zap.String("mode", ModeClient),
	)

	fmt.Printf("RDMA 文件传输客户端已启动\n")
	fmt.Printf("服务端地址: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("客户端API地址: http://localhost:%d\n", clientPort)
	fmt.Printf("使用 'curl http://localhost:%d/api/v1/transfers' 发起自动化传输\n", clientPort)

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭客户端...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 清理传输服务
	transferService.Cleanup()

	// 关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("关闭客户端API服务失败", zap.Error(err))
	}

	logger.Info("客户端已关闭")
}

// isServerRunning 检查服务端是否在运行
func isServerRunning(host string, port int) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	url := fmt.Sprintf("http://%s:%d/api/health", host, port)

	resp, err := client.Get(url)
	if err != nil {
		// 不显示连接失败的调试信息，这是正常情况
		return false
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return false
	}
	
	return true
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("RDMA 大文件传输服务\n")
	fmt.Printf("版本: %s\n", version)
	fmt.Printf("构建时间: %s\n", buildTime)
	fmt.Printf("Git提交: %s\n", gitCommit)
	fmt.Printf("运行模式: 统一模式（支持服务端/客户端自动检测）\n")
}

// CORSMiddleware CORS 中间件
func CORSMiddleware(corsConfig models.CORSSettings) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !corsConfig.Enabled {
			c.Next()
			return
		}

		// 设置 CORS 头
		origin := c.Request.Header.Get("Origin")
		if len(corsConfig.AllowedOrigins) > 0 {
			for _, allowedOrigin := range corsConfig.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					c.Header("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		c.Header("Access-Control-Allow-Methods", joinStrings(corsConfig.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", joinStrings(corsConfig.AllowedHeaders, ", "))
		c.Header("Access-Control-Allow-Credentials", "true")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// getRtranfilePath 获取 rtranfile 二进制文件路径
func getRtranfilePath() string {
	// 1. 检查环境变量
	if path := os.Getenv("RTRANFILE_PATH"); path != "" {
		return path
	}
	
	// 2. 检查系统路径
	if _, err := os.Stat("/usr/local/bin/rtranfile"); err == nil {
		return "/usr/local/bin/rtranfile"
	}
	
	// 3. 检查当前目录下的 bin 目录
	if _, err := os.Stat("./bin/rtranfile"); err == nil {
		return "./bin/rtranfile"
	}
	
	// 4. 检查是否在 PATH 中
	if path, err := exec.LookPath("rtranfile"); err == nil {
		return path
	}
	
	// 5. 默认返回硬编码路径（兼容旧版本）
	return "./bin/rtranfile"
}

// joinStrings 连接字符串切片
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}