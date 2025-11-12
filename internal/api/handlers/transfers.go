package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"rdma-burst/internal/models"
	"rdma-burst/internal/services/transfer"
)

// TransferHandler 传输处理器
type TransferHandler struct {
	transferService *transfer.TransferService
	clientMode      bool // 是否为客户端模式
	serverHost      string
	serverPort      int
	serverConfig    *models.TransferSettings // 服务端配置
}

// NewTransferHandler 创建新的传输处理器
func NewTransferHandler(transferService *transfer.TransferService, serverConfig *models.TransferSettings) *TransferHandler {
	return &TransferHandler{
		transferService: transferService,
		clientMode:      false, // 默认为服务端模式
		serverConfig:    serverConfig, // 保存服务端配置
	}
}

// NewClientTransferHandler 创建客户端传输处理器
func NewClientTransferHandler(serverHost string, serverPort int, serverConfig *models.TransferSettings) *TransferHandler {
	return &TransferHandler{
		clientMode:   true,
		serverHost:   serverHost,
		serverPort:   serverPort,
		serverConfig: serverConfig, // 保存服务端配置
	}
}

// CreateTransfer 创建传输任务
// @Summary 创建传输任务
// @Description 创建新的 RDMA 文件传输任务
// @Tags transfers
// @Accept json
// @Produce json
// @Param request body models.TransferRequest true "传输请求"
// @Success 201 {object} models.TransferResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/transfers [post]
func (h *TransferHandler) CreateTransfer(c *gin.Context) {
	var req models.TransferRequest
	
	// 绑定请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// 验证请求参数
	if err := validateTransferRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "VALIDATION_ERROR",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// 如果是客户端模式，调用服务端API
	if h.clientMode {
		// 创建客户端传输服务（传递配置）
		clientService := transfer.NewClientTransferService(h.serverHost, h.serverPort, h.serverConfig)
		response, err := clientService.CreateTransfer(&req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "CLIENT_TRANSFER_ERROR",
				Message: "客户端调用服务端API失败: " + err.Error(),
				Code:    http.StatusInternalServerError,
			})
			return
		}
		c.JSON(http.StatusCreated, response)
		return
	}

	// 服务端模式：使用本地传输服务
	// 使用从配置中加载的服务端配置
	if h.serverConfig == nil {
		// 如果配置为空，使用默认配置
		h.serverConfig = &models.TransferSettings{
			Device:                "mlx5_0",
			BaseDir:               "/var/lib/rtrans",
			TransferInterval:      5 * 1e9, // 5秒
			MaxConcurrentTransfers: 1,
			ChunkSize:             4194304,
			Modes: models.TransferModes{
				Hugepages: models.ModeConfig{
					Enabled: true,
					BaseDir: "/dev/hugepages/dir",
				},
				Tmpfs: models.ModeConfig{
					Enabled: true,
					BaseDir: "/dev/shm/dir",
				},
				Filesystem: models.ModeConfig{
					Enabled: true,
					BaseDir: "/var/lib/rtrans/files",
				},
			},
		}
	}
	
	serverConfig := h.serverConfig

	// 在服务端配置中设置服务端地址（用于客户端传输）
	// 创建一个副本，避免修改原始配置
	transferConfig := *serverConfig
	transferConfig.ServerAddress = h.getServerAddress()

	// 第一步：准备传输环境（启动服务端监听进程）
	if err := h.transferService.PrepareTransfer(&req, &transferConfig); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "PREPARE_ERROR",
			Message: "准备传输环境失败: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// 服务端只负责启动监听进程，不执行客户端传输
	// 客户端应该在收到准备就绪响应后，在自己的机器上执行传输命令
	response := &models.TransferResponse{
		ID:        fmt.Sprintf("prepared_%d", time.Now().Unix()),
		Status:    models.StatusPrepared,
		Message:   "传输环境准备就绪，请在客户端执行传输命令",
		CreatedAt: time.Now(),
	}

	c.JSON(http.StatusCreated, response)
}

// GetTransferStatus 获取传输状态
// @Summary 获取传输状态
// @Description 获取指定传输任务的状态和进度
// @Tags transfers
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} models.ProgressResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/transfers/{id} [get]
func (h *TransferHandler) GetTransferStatus(c *gin.Context) {
	taskID := c.Param("id")
	
	if taskID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "MISSING_PARAM",
			Message: "任务ID不能为空",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// 如果是客户端模式，调用服务端API
	if h.clientMode {
		// 创建客户端传输服务（传递配置）
		clientService := transfer.NewClientTransferService(h.serverHost, h.serverPort, h.serverConfig)
		status, err := clientService.GetTransferStatus(taskID)
		if err != nil {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "TASK_NOT_FOUND",
				Message: err.Error(),
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusOK, status)
		return
	}

	// 服务端模式：使用本地传输服务
	if h.transferService == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "SERVICE_ERROR",
			Message: "传输服务未初始化",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// 获取传输状态
	status, err := h.transferService.GetTransferStatus(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "TASK_NOT_FOUND",
			Message: err.Error(),
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ListTransfers 列出传输任务
// @Summary 列出传输任务
// @Description 获取传输任务列表，支持分页
// @Tags transfers
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(20)
// @Success 200 {object} models.TaskListResponse
// @Failure 400 {object} models.ErrorResponse
// @Router /api/v1/transfers [get]
func (h *TransferHandler) ListTransfers(c *gin.Context) {
	// 获取分页参数
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	size, err := strconv.Atoi(c.DefaultQuery("size", "20"))
	if err != nil || size < 1 || size > 100 {
		size = 20
	}

	// 如果是客户端模式，调用服务端API
	if h.clientMode {
		// 创建客户端传输服务（传递配置）
		clientService := transfer.NewClientTransferService(h.serverHost, h.serverPort, h.serverConfig)
		response, err := clientService.ListTransfers(page, size)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "CLIENT_TRANSFER_ERROR",
				Message: "客户端调用服务端API失败: " + err.Error(),
				Code:    http.StatusInternalServerError,
			})
			return
		}
		c.JSON(http.StatusOK, response)
		return
	}

	// 服务端模式：使用本地传输服务
	if h.transferService == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "SERVICE_ERROR",
			Message: "传输服务未初始化",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// 获取任务列表
	response := h.transferService.ListTransfers(page, size)
	c.JSON(http.StatusOK, response)
}

// CancelTransfer 取消传输任务
// @Summary 取消传输任务
// @Description 取消指定的传输任务
// @Tags transfers
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} models.TransferResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/transfers/{id} [delete]
func (h *TransferHandler) CancelTransfer(c *gin.Context) {
	taskID := c.Param("id")
	
	if taskID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "MISSING_PARAM",
			Message: "任务ID不能为空",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// 如果是客户端模式，调用服务端API
	if h.clientMode {
		// 创建客户端传输服务（传递配置）
		clientService := transfer.NewClientTransferService(h.serverHost, h.serverPort, h.serverConfig)
		err := clientService.CancelTransfer(taskID)
		if err != nil {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "CANCEL_ERROR",
				Message: err.Error(),
				Code:    http.StatusNotFound,
			})
			return
		}

		c.JSON(http.StatusOK, models.TransferResponse{
			ID:      taskID,
			Status:  models.StatusCancelled,
			Message: "传输任务已取消",
		})
		return
	}

	// 服务端模式：使用本地传输服务
	if h.transferService == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "SERVICE_ERROR",
			Message: "传输服务未初始化",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// 取消传输任务
	err := h.transferService.CancelTransfer(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "CANCEL_ERROR",
			Message: err.Error(),
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, models.TransferResponse{
		ID:      taskID,
		Status:  models.StatusCancelled,
		Message: "传输任务已取消",
	})
}

// GetActiveTransfers 获取活跃传输数量
// @Summary 获取活跃传输数量
// @Description 获取当前活跃的传输任务数量
// @Tags transfers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/transfers/active [get]
func (h *TransferHandler) GetActiveTransfers(c *gin.Context) {
	// 如果是客户端模式，返回0（客户端不管理活跃传输）
	if h.clientMode {
		c.JSON(http.StatusOK, gin.H{
			"active_transfers": 0,
			"timestamp":        time.Now().Format(time.RFC3339),
		})
		return
	}

	// 服务端模式：使用本地传输服务
	if h.transferService == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "SERVICE_ERROR",
			Message: "传输服务未初始化",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	activeCount := h.transferService.GetActiveTransfers()
	
	c.JSON(http.StatusOK, gin.H{
		"active_transfers": activeCount,
		"timestamp":        time.Now().Format(time.RFC3339),
	})
}

// validateTransferRequest 验证传输请求
func validateTransferRequest(req *models.TransferRequest) error {
	// 验证文件名
	if req.Filename == "" {
		return fmt.Errorf("文件名不能为空")
	}

	// 验证传输模式
	validModes := map[string]bool{
		models.ModeHugepages:  true,
		models.ModeTmpfs:      true,
		models.ModeFilesystem: true,
	}
	if !validModes[req.Mode] {
		return fmt.Errorf("不支持的传输模式: %s", req.Mode)
	}

	// 验证传输方向
	validDirections := map[string]bool{
		models.DirectionPut: true,
		models.DirectionGet: true,
	}
	if !validDirections[req.Direction] {
		return fmt.Errorf("不支持的传输方向: %s", req.Direction)
	}

	// 客户端传输不再需要请求中包含服务端地址
	// 服务端地址从配置中获取

	return nil
}

// buildClientCommand 构建客户端执行命令
func (h *TransferHandler) buildClientCommand(req *models.TransferRequest, serverConfig *models.TransferSettings) string {
	// 获取客户端IP（从请求头中获取，简化实现使用默认值）
	clientIP := "客户端IP" // 实际应该从请求中获取
	
	// 构建rtranfile命令
	command := fmt.Sprintf("./bin/rtranfile --%s %s", req.Direction, req.Filename)
	
	// 添加模式参数
	switch req.Mode {
	case models.ModeHugepages:
		command += " --hugepages"
	case models.ModeTmpfs:
		command += " --tmpfs"
	case models.ModeFilesystem:
		command += " --filesystem"
	}
	
	// 添加服务端地址
	command += fmt.Sprintf(" --server %s", serverConfig.Device) // 简化实现
	
	return fmt.Sprintf("在客户端机器 (%s) 上执行: %s", clientIP, command)
}

// getServerAddress 获取服务端地址
func (h *TransferHandler) getServerAddress() string {
	// 如果是客户端模式，使用客户端配置的服务端地址
	if h.clientMode {
		return h.serverHost
	}
	// 服务端模式，使用默认地址
	return "localhost"
}

// RegisterRoutes 注册路由
func (h *TransferHandler) RegisterRoutes(router *gin.RouterGroup) {
	transfers := router.Group("/transfers")
	{
		transfers.POST("", h.CreateTransfer)
		transfers.GET("", h.ListTransfers)
		transfers.GET("/active", h.GetActiveTransfers)
		transfers.GET("/:id", h.GetTransferStatus)
		transfers.DELETE("/:id", h.CancelTransfer)
	}
}