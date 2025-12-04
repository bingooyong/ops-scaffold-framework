package handler

import (
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/middleware"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/service"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService service.AuthService
	logger      *zap.Logger
}

// NewAuthHandler 创建认证处理器实例
func NewAuthHandler(authService service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=100"`
	Email    string `json:"email" binding:"omitempty,email"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=100"`
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Username, req.Password, req.Email)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	// 隐藏密码
	user.Password = ""

	response.Created(c, gin.H{
		"user": user,
	})
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	token, user, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	// 隐藏密码
	user.Password = ""

	response.Success(c, gin.H{
		"token": token,
		"user":  user,
	})
}

// GetProfile 获取当前用户信息
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	// 隐藏密码
	user.Password = ""

	response.Success(c, gin.H{
		"user": user,
	})
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.authService.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, gin.H{
		"message": "密码修改成功",
	})
}

// ListUsers 获取用户列表（管理员）
func (h *AuthHandler) ListUsers(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := h.authService.ListUsers(c.Request.Context(), page, pageSize)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	// 隐藏所有用户的密码
	for _, user := range users {
		user.Password = ""
	}

	response.Page(c, users, page, pageSize, total)
}

// DisableUser 禁用用户（管理员）
func (h *AuthHandler) DisableUser(c *gin.Context) {
	userID := parseUintParam(c, "id")
	if userID == 0 {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	if err := h.authService.DisableUser(c.Request.Context(), userID); err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, gin.H{
		"message": "用户已禁用",
	})
}

// EnableUser 启用用户（管理员）
func (h *AuthHandler) EnableUser(c *gin.Context) {
	userID := parseUintParam(c, "id")
	if userID == 0 {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	if err := h.authService.EnableUser(c.Request.Context(), userID); err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, gin.H{
		"message": "用户已启用",
	})
}
