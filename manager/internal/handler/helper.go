package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// parseIntQuery 解析整数查询参数
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// parseUintParam 解析URL路径参数为uint
func parseUintParam(c *gin.Context, key string) uint {
	valueStr := c.Param(key)
	if valueStr == "" {
		return 0
	}

	value, err := strconv.ParseUint(valueStr, 10, 32)
	if err != nil {
		return 0
	}

	return uint(value)
}

// parseStringQuery 解析字符串查询参数
func parseStringQuery(c *gin.Context, key string, defaultValue string) string {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}
	return value
}
