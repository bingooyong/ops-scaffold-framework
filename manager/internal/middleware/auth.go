package middleware

import (
	"strings"

	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/jwt"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/response"
	"github.com/gin-gonic/gin"
)

// JWTAuth JWT认证中间件
func JWTAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, errors.ErrUnauthorizedMsg)
			c.Abort()
			return
		}

		// 解析Token
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Error(c, errors.ErrInvalidTokenMsg)
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			if err == jwt.ErrExpiredToken {
				response.Error(c, errors.ErrTokenExpiredMsg)
			} else {
				response.Error(c, errors.ErrInvalidTokenMsg)
			}
			c.Abort()
			return
		}

		// 将用户信息保存到上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RequireAdmin 需要管理员权限的中间件
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Error(c, errors.ErrUnauthorizedMsg)
			c.Abort()
			return
		}

		if role != "admin" {
			response.Error(c, errors.ErrForbiddenMsg)
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	return userID.(uint), true
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}
	return username.(string), true
}

// GetRole 从上下文获取角色
func GetRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	return role.(string), true
}
