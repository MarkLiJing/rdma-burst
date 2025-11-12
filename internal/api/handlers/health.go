package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"rdma-burst/internal/models"
	"rdma-burst/internal/services/transfer"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	transferService *transfer.TransferService
	startTime       time.Time
	version         string
}

// NewHealthHandler 创建新的健康检查处理器
func NewHealthHandler(transferService *transfer.TransferService, version string) *HealthHandler {
	return &HealthHandler{
		transferService: transferService,
		startTime:       time.Now(),
		version:         version,
	}
}

// HealthCheck 健康检查
// @Summary 健康检查
// @Description 检查服务健康状态
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Router /api/health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	uptime := time.Since(h.startTime)
	activeTransfers := h.transferService.GetActiveTransfers()

	response := models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   h.version,
	}

	// 添加额外信息
	extraInfo := map[string]interface{}{
		"uptime":           uptime.String(),
		"active_transfers": activeTransfers,
		"start_time":       h.startTime.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     response.Status,
		"timestamp":  response.Timestamp,
		"version":    response.Version,
		"extra_info": extraInfo,
	})
}

// ReadyCheck 就绪检查
// @Summary 就绪检查
// @Description 检查服务是否就绪
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Router /api/ready [get]
func (h *HealthHandler) ReadyCheck(c *gin.Context) {
	// 这里可以添加更复杂的就绪检查逻辑
	// 例如检查数据库连接、外部服务依赖等
	
	response := models.HealthResponse{
		Status:    "ready",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   h.version,
	}

	c.JSON(http.StatusOK, response)
}

// LivenessCheck 存活检查
// @Summary 存活检查
// @Description 检查服务是否存活
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Router /api/live [get]
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	response := models.HealthResponse{
		Status:    "alive",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   h.version,
	}

	c.JSON(http.StatusOK, response)
}

// Metrics 指标端点
// @Summary 服务指标
// @Description 获取服务运行指标
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/metrics [get]
func (h *HealthHandler) Metrics(c *gin.Context) {
	uptime := time.Since(h.startTime)
	activeTransfers := h.transferService.GetActiveTransfers()

	metrics := map[string]interface{}{
		"service": map[string]interface{}{
			"name":          "rdma-burst",
			"version":       h.version,
			"uptime_seconds": uptime.Seconds(),
			"start_time":     h.startTime.Format(time.RFC3339),
		},
		"transfers": map[string]interface{}{
			"active": activeTransfers,
			"total":  h.getTotalTransfers(),
		},
		"system": map[string]interface{}{
			"goroutines": getGoroutineCount(),
			"timestamp":  time.Now().Format(time.RFC3339),
		},
	}

	c.JSON(http.StatusOK, metrics)
}

// getTotalTransfers 获取总传输任务数
func (h *HealthHandler) getTotalTransfers() int {
	// 这里需要实现获取总任务数的逻辑
	// 简化实现：返回 0
	return 0
}

// getGoroutineCount 获取 Goroutine 数量
func (h *HealthHandler) getGoroutineCount() int {
	// 这里需要实现获取 Goroutine 数量的逻辑
	// 简化实现：返回 0
	return 0
}

// getGoroutineCount 获取 Goroutine 数量（独立函数）
func getGoroutineCount() int {
	// 简化实现：返回 0
	return 0
}

// RegisterRoutes 注册路由
func (h *HealthHandler) RegisterRoutes(router *gin.RouterGroup) {
	health := router.Group("/health")
	{
		health.GET("", h.HealthCheck)
		health.GET("/ready", h.ReadyCheck)
		health.GET("/live", h.LivenessCheck)
		health.GET("/metrics", h.Metrics)
	}
}