package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
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
	configManager := config.NewConfigManager("server")
	configPath := getConfigPath()
	
	serverConfig, err := configManager.LoadConfig(configPath)
	if err != nil {
		logger.Fatal("加载配置失败", zap.Error(err))
	}

	cfg := serverConfig.(*models.ServerConfig)

	// 创建传输服务（使用配置中的传输设置）
	rtranfilePath := "./bin/rtranfile" // rtranfile 二进制文件路径
	transferService := transfer.NewTransferServiceWithConfig(
		rtranfilePath,
		&cfg.Transfer,
		nil, // 单次传输配置为空，使用默认值
	)

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

	// 注册路由
	api := router.Group("/api/v1")
	transferHandler.RegisterRoutes(api)
	healthHandler.RegisterRoutes(router.Group("/api"))

	// 根路径健康检查
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "rdma-burst",
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
		logger.Info("启动 RDMA 文件传输服务",
			zap.String("host", cfg.Server.Host),
			zap.Int("port", cfg.Server.Port),
			zap.String("version", version),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("启动服务器失败", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 清理传输服务
	transferService.Cleanup()

	// 关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("关闭服务器失败", zap.Error(err))
	}

	logger.Info("服务已关闭")
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	// 优先使用环境变量指定的配置路径
	if path := os.Getenv("RDMA_CONFIG_PATH"); path != "" {
		return path
	}
	
	// 使用默认配置路径
	return "./configs/server.yaml"
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