package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ModeHandler 模式检测处理器
type ModeHandler struct {
	startTime time.Time
	version   string
	mode      string
}

// NewModeHandler 创建新的模式检测处理器
func NewModeHandler(version string, mode string) *ModeHandler {
	return &ModeHandler{
		startTime: time.Now(),
		version:   version,
		mode:      mode,
	}
}

// ModeResponse 模式检测响应
type ModeResponse struct {
	Mode      string `json:"mode"`
	Version   string `json:"version"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Uptime    string `json:"uptime"`
}

// GetMode 获取当前运行模式
// @Summary 获取运行模式
// @Description 获取当前服务的运行模式（服务端/客户端）
// @Tags mode
// @Accept json
// @Produce json
// @Success 200 {object} ModeResponse
// @Router /api/v1/mode [get]
func (h *ModeHandler) GetMode(c *gin.Context) {
	uptime := time.Since(h.startTime)

	response := ModeResponse{
		Mode:      h.mode,
		Version:   h.version,
		Status:    "running",
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    uptime.String(),
	}

	c.JSON(http.StatusOK, response)
}

// DetectMode 检测运行模式
// @Summary 检测运行模式
// @Description 检测当前环境应该运行的模式（服务端/客户端）
// @Tags mode
// @Accept json
// @Produce json
// @Success 200 {object} ModeResponse
// @Router /api/v1/mode/detect [get]
func (h *ModeHandler) DetectMode(c *gin.Context) {
	// 检测运行模式逻辑
	detectedMode := h.detectRuntimeMode()

	uptime := time.Since(h.startTime)

	response := ModeResponse{
		Mode:      detectedMode,
		Version:   h.version,
		Status:    "detected",
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    uptime.String(),
	}

	c.JSON(http.StatusOK, response)
}

// SwitchMode 切换运行模式
// @Summary 切换运行模式
// @Description 请求切换运行模式（需要重启服务）
// @Tags mode
// @Accept json
// @Produce json
// @Param request body SwitchModeRequest true "切换模式请求"
// @Success 200 {object} SwitchModeResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/mode/switch [post]
func (h *ModeHandler) SwitchMode(c *gin.Context) {
	var req SwitchModeRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// 验证模式参数
	validModes := map[string]bool{
		"server": true,
		"client": true,
		"auto":   true,
	}
	if !validModes[req.Mode] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_MODE",
			Message: "不支持的运行模式: " + req.Mode,
			Code:    http.StatusBadRequest,
		})
		return
	}

	// 这里可以实现配置更新逻辑
	// 实际模式切换需要重启服务

	response := SwitchModeResponse{
		CurrentMode: h.mode,
		TargetMode:  req.Mode,
		Message:     "模式切换请求已接受，需要重启服务生效",
		RestartRequired: true,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// GetModeStatus 获取模式状态
// @Summary 获取模式状态
// @Description 获取详细的模式状态信息
// @Tags mode
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/mode/status [get]
func (h *ModeHandler) GetModeStatus(c *gin.Context) {
	uptime := time.Since(h.startTime)

	status := map[string]interface{}{
		"mode": map[string]interface{}{
			"current": h.mode,
			"detected": h.detectRuntimeMode(),
			"supported_modes": []string{"server", "client", "auto"},
		},
		"service": map[string]interface{}{
			"version":    h.version,
			"uptime":     uptime.String(),
			"start_time": h.startTime.Format(time.RFC3339),
			"status":     "running",
		},
		"detection": map[string]interface{}{
			"method": "health_check",
			"timeout": "3s",
			"endpoint": "http://localhost:8080/api/health",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, status)
}

// detectRuntimeMode 检测运行模式
func (h *ModeHandler) detectRuntimeMode() string {
	// 尝试连接本地服务端
	client := &http.Client{Timeout: 3 * time.Second}
	url := "http://localhost:8080/api/health"

	resp, err := client.Get(url)
	if err == nil && resp.StatusCode == http.StatusOK {
		return "client"
	}

	return "server"
}

// SwitchModeRequest 切换模式请求
type SwitchModeRequest struct {
	Mode string `json:"mode" binding:"required,oneof=server client auto"`
}

// SwitchModeResponse 切换模式响应
type SwitchModeResponse struct {
	CurrentMode     string `json:"current_mode"`
	TargetMode      string `json:"target_mode"`
	Message         string `json:"message"`
	RestartRequired bool   `json:"restart_required"`
	Timestamp       string `json:"timestamp"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// RegisterRoutes 注册路由
func (h *ModeHandler) RegisterRoutes(router *gin.RouterGroup) {
	mode := router.Group("/mode")
	{
		mode.GET("", h.GetMode)
		mode.GET("/detect", h.DetectMode)
		mode.GET("/status", h.GetModeStatus)
		mode.POST("/switch", h.SwitchMode)
	}
}