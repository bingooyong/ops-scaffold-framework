package service

import (
	"context"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/repository"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/jwt"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService 认证服务接口
type AuthService interface {
	// Register 用户注册
	Register(ctx context.Context, username, password, email string) (*model.User, error)
	// Login 用户登录
	Login(ctx context.Context, username, password string) (string, *model.User, error)
	// Logout 用户登出
	Logout(ctx context.Context, userID uint) error
	// RefreshToken 刷新Token
	RefreshToken(ctx context.Context, token string) (string, error)
	// ChangePassword 修改密码
	ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error
	// ResetPassword 重置密码（管理员操作）
	ResetPassword(ctx context.Context, userID uint, newPassword string) error
	// GetUserByID 根据ID获取用户
	GetUserByID(ctx context.Context, id uint) (*model.User, error)
	// UpdateUser 更新用户信息
	UpdateUser(ctx context.Context, user *model.User) error
	// DisableUser 禁用用户
	DisableUser(ctx context.Context, userID uint) error
	// EnableUser 启用用户
	EnableUser(ctx context.Context, userID uint) error
	// ListUsers 获取用户列表
	ListUsers(ctx context.Context, page, pageSize int) ([]*model.User, int64, error)
	// ValidateToken 验证Token
	ValidateToken(ctx context.Context, token string) (*jwt.Claims, error)
}

// authService 认证服务实现
type authService struct {
	userRepo   repository.UserRepository
	auditRepo  repository.AuditLogRepository
	jwtManager *jwt.Manager
	logger     *zap.Logger
}

// NewAuthService 创建认证服务实例
func NewAuthService(
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	jwtManager *jwt.Manager,
	logger *zap.Logger,
) AuthService {
	return &authService{
		userRepo:   userRepo,
		auditRepo:  auditRepo,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// Register 用户注册
func (s *authService) Register(ctx context.Context, username, password, email string) (*model.User, error) {
	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil && err != gorm.ErrRecordNotFound {
		s.logger.Error("failed to check username", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}
	if existingUser != nil {
		return nil, errors.ErrUserAlreadyExistsMsg
	}

	// 检查邮箱是否已存在
	if email != "" {
		existingUser, err = s.userRepo.GetByEmail(ctx, email)
		if err != nil && err != gorm.ErrRecordNotFound {
			s.logger.Error("failed to check email", zap.Error(err))
			return nil, errors.Wrap(errors.ErrDatabase, "数据库错误", err)
		}
		if existingUser != nil {
			return nil, errors.New(errors.ErrConflict, "邮箱已被使用")
		}
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return nil, errors.Wrap(errors.ErrInternalServer, "密码加密失败", err)
	}

	// 创建用户
	user := &model.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
		Role:     "user",  // 默认角色
		Status:   "active", // 默认状态
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("failed to create user", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "创建用户失败", err)
	}

	s.logger.Info("user registered", zap.String("username", username))

	return user, nil
}

// Login 用户登录
func (s *authService) Login(ctx context.Context, username, password string) (string, *model.User, error) {
	// 获取用户
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil, errors.ErrInvalidCredentialsMsg
		}
		s.logger.Error("failed to get user", zap.Error(err))
		return "", nil, errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}

	// 检查用户状态
	if user.Status != "active" {
		return "", nil, errors.ErrUserDisabledMsg
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.ErrInvalidCredentialsMsg
	}

	// 生成Token
	token, err := s.jwtManager.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		s.logger.Error("failed to generate token", zap.Error(err))
		return "", nil, errors.Wrap(errors.ErrInternalServer, "生成Token失败", err)
	}

	// 更新最后登录时间
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("failed to update last login", zap.Error(err))
	}

	s.logger.Info("user logged in", zap.String("username", username))

	return token, user, nil
}

// Logout 用户登出
func (s *authService) Logout(ctx context.Context, userID uint) error {
	// JWT是无状态的，这里主要用于记录日志
	s.logger.Info("user logged out", zap.Uint("user_id", userID))
	return nil
}

// RefreshToken 刷新Token
func (s *authService) RefreshToken(ctx context.Context, token string) (string, error) {
	newToken, err := s.jwtManager.RefreshToken(token)
	if err != nil {
		return "", errors.Wrap(errors.ErrInvalidToken, "Token刷新失败", err)
	}
	return newToken, nil
}

// ChangePassword 修改密码
func (s *authService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	// 获取用户
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrUserNotFoundMsg
		}
		s.logger.Error("failed to get user", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New(errors.ErrInvalidParams, "旧密码错误")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return errors.Wrap(errors.ErrInternalServer, "密码加密失败", err)
	}

	// 更新密码
	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		s.logger.Error("failed to update password", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "更新密码失败", err)
	}

	s.logger.Info("password changed", zap.Uint("user_id", userID))

	return nil
}

// ResetPassword 重置密码（管理员操作）
func (s *authService) ResetPassword(ctx context.Context, userID uint, newPassword string) error {
	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return errors.Wrap(errors.ErrInternalServer, "密码加密失败", err)
	}

	// 更新密码
	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		s.logger.Error("failed to reset password", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "重置密码失败", err)
	}

	s.logger.Info("password reset", zap.Uint("user_id", userID))

	return nil
}

// GetUserByID 根据ID获取用户
func (s *authService) GetUserByID(ctx context.Context, id uint) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFoundMsg
		}
		s.logger.Error("failed to get user", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}
	return user, nil
}

// UpdateUser 更新用户信息
func (s *authService) UpdateUser(ctx context.Context, user *model.User) error {
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("failed to update user", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "更新用户失败", err)
	}
	s.logger.Info("user updated", zap.Uint("user_id", user.ID))
	return nil
}

// DisableUser 禁用用户
func (s *authService) DisableUser(ctx context.Context, userID uint) error {
	if err := s.userRepo.UpdateStatus(ctx, userID, "disabled"); err != nil {
		s.logger.Error("failed to disable user", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "禁用用户失败", err)
	}
	s.logger.Info("user disabled", zap.Uint("user_id", userID))
	return nil
}

// EnableUser 启用用户
func (s *authService) EnableUser(ctx context.Context, userID uint) error {
	if err := s.userRepo.UpdateStatus(ctx, userID, "active"); err != nil {
		s.logger.Error("failed to enable user", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "启用用户失败", err)
	}
	s.logger.Info("user enabled", zap.Uint("user_id", userID))
	return nil
}

// ListUsers 获取用户列表
func (s *authService) ListUsers(ctx context.Context, page, pageSize int) ([]*model.User, int64, error) {
	users, total, err := s.userRepo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list users", zap.Error(err))
		return nil, 0, errors.Wrap(errors.ErrDatabase, "查询用户列表失败", err)
	}
	return users, total, nil
}

// ValidateToken 验证Token
func (s *authService) ValidateToken(ctx context.Context, token string) (*jwt.Claims, error) {
	claims, err := s.jwtManager.ParseToken(token)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInvalidToken, "Token验证失败", err)
	}
	return claims, nil
}
