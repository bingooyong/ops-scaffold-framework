package middleware

import (
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/repository"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Audit 审计日志中间件
func Audit(auditRepo repository.AuditLogRepository, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 处理请求
		c.Next()

		// 记录审计日志
		go func() {
			// 获取用户ID
			userID, exists := c.Get("user_id")
			if !exists {
				return // 未认证的请求不记录审计日志
			}

			// 计算耗时
			duration := time.Since(start).Milliseconds()

			// 创建审计日志
			log := &model.AuditLog{
				UserID:   userID.(uint),
				Action:   c.Request.Method,
				Resource: c.Request.URL.Path,
				Method:   c.Request.Method,
				Path:     c.Request.URL.Path,
				IP:       c.ClientIP(),
				Status:   c.Writer.Status(),
				Duration: duration,
			}

			// 保存审计日志
			if err := auditRepo.Create(c.Request.Context(), log); err != nil {
				logger.Error("failed to create audit log", zap.Error(err))
			}
		}()
	}
}
