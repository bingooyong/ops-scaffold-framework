package response

import (
	"net/http"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"github.com/gin-gonic/gin"
)

// Response API响应结构
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// PageInfo 分页信息
type PageInfo struct {
	Page     int   `json:"page"`      // 当前页码
	PageSize int   `json:"page_size"` // 每页大小
	Total    int64 `json:"total"`     // 总记录数
	Pages    int   `json:"pages"`     // 总页数
}

// PageData 分页数据
type PageData struct {
	List     interface{} `json:"list"`      // 数据列表
	PageInfo PageInfo    `json:"page_info"` // 分页信息
}

// Success 返回成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// SuccessWithMessage 返回带自定义消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      0,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// Error 返回错误响应
func Error(c *gin.Context, err *errors.APIError) {
	c.JSON(err.GetHTTPStatus(), Response{
		Code:      int(err.Code),
		Message:   err.Message,
		Data:      err.Details,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// ErrorWithMessage 返回带自定义消息的错误响应
func ErrorWithMessage(c *gin.Context, code errors.ErrorCode, message string) {
	err := errors.New(code, message)
	c.JSON(err.GetHTTPStatus(), Response{
		Code:      int(err.Code),
		Message:   err.Message,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// BadRequest 返回400错误
func BadRequest(c *gin.Context, message string) {
	ErrorWithMessage(c, errors.ErrInvalidParams, message)
}

// Unauthorized 返回401错误
func Unauthorized(c *gin.Context, message string) {
	ErrorWithMessage(c, errors.ErrUnauthorized, message)
}

// Forbidden 返回403错误
func Forbidden(c *gin.Context, message string) {
	ErrorWithMessage(c, errors.ErrForbidden, message)
}

// NotFound 返回404错误
func NotFound(c *gin.Context, message string) {
	ErrorWithMessage(c, errors.ErrNotFound, message)
}

// Conflict 返回409错误
func Conflict(c *gin.Context, message string) {
	ErrorWithMessage(c, errors.ErrConflict, message)
}

// InternalServerError 返回500错误
func InternalServerError(c *gin.Context, message string) {
	ErrorWithMessage(c, errors.ErrInternalServer, message)
}

// Page 返回分页响应
func Page(c *gin.Context, list interface{}, page, pageSize int, total int64) {
	pages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		pages++
	}

	Success(c, PageData{
		List: list,
		PageInfo: PageInfo{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
			Pages:    pages,
		},
	})
}

// Created 返回201创建成功响应
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:      0,
		Message:   "created",
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// NoContent 返回204无内容响应
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Accepted 返回202已接受响应
func Accepted(c *gin.Context, data interface{}) {
	c.JSON(http.StatusAccepted, Response{
		Code:      0,
		Message:   "accepted",
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
