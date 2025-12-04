package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery 错误恢复中间件
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录错误日志
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				// 返回500错误
				c.JSON(http.StatusInternalServerError, response.Response{
					Code:    int(errors.ErrInternalServer),
					Message: "服务器内部错误",
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}
