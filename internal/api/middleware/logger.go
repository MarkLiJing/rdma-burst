package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggerMiddleware 日志中间件
type LoggerMiddleware struct {
	logger *zap.Logger
}

// NewLoggerMiddleware 创建新的日志中间件
func NewLoggerMiddleware(logger *zap.Logger) *LoggerMiddleware {
	return &LoggerMiddleware{
		logger: logger,
	}
}

// Logger 日志记录中间件
func (lm *LoggerMiddleware) Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()
		
		// 处理请求
		c.Next()
		
		// 结束时间
		end := time.Now()
		latency := end.Sub(start)
		
		// 获取客户端IP
		clientIP := c.ClientIP()
		
		// 获取方法路径
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()
		
		// 记录日志
		lm.logger.Info("HTTP请求",
			zap.String("client_ip", clientIP),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}

// Recovery 恢复中间件
func (lm *LoggerMiddleware) Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录 panic 错误
				lm.logger.Error("HTTP请求发生panic",
					zap.Any("error", err),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("client_ip", c.ClientIP()),
				)
				
				// 返回错误响应
				c.AbortWithStatusJSON(500, gin.H{
					"error":   "INTERNAL_SERVER_ERROR",
					"message": "服务器内部错误",
				})
			}
		}()
		
		c.Next()
	}
}