package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/repository"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// VersionService 版本服务接口
type VersionService interface {
	// Create 创建版本
	Create(ctx context.Context, version *model.Version) error
	// GetByID 根据ID获取版本
	GetByID(ctx context.Context, id uint) (*model.Version, error)
	// GetByComponentAndVersion 根据组件和版本号获取版本
	GetByComponentAndVersion(ctx context.Context, component, version string) (*model.Version, error)
	// Update 更新版本
	Update(ctx context.Context, version *model.Version) error
	// Delete 删除版本
	Delete(ctx context.Context, id uint) error
	// List 获取版本列表
	List(ctx context.Context, page, pageSize int) ([]*model.Version, int64, error)
	// ListByComponent 根据组件获取版本列表
	ListByComponent(ctx context.Context, component string, page, pageSize int) ([]*model.Version, int64, error)
	// GetLatestReleased 获取最新发布的版本
	GetLatestReleased(ctx context.Context, component string) (*model.Version, error)
	// GetAllReleased 获取所有已发布的版本
	GetAllReleased(ctx context.Context, component string) ([]*model.Version, error)
	// Release 发布版本
	Release(ctx context.Context, versionID uint) error
	// Deprecate 废弃版本
	Deprecate(ctx context.Context, versionID uint) error
	// VerifyFile 验证文件哈希
	VerifyFile(filePath, expectedHash string) (bool, error)
	// CalculateFileHash 计算文件哈希
	CalculateFileHash(filePath string) (string, error)
}

// versionService 版本服务实现
type versionService struct {
	versionRepo repository.VersionRepository
	auditRepo   repository.AuditLogRepository
	logger      *zap.Logger
}

// NewVersionService 创建版本服务实例
func NewVersionService(
	versionRepo repository.VersionRepository,
	auditRepo repository.AuditLogRepository,
	logger *zap.Logger,
) VersionService {
	return &versionService{
		versionRepo: versionRepo,
		auditRepo:   auditRepo,
		logger:      logger,
	}
}

// Create 创建版本
func (s *versionService) Create(ctx context.Context, version *model.Version) error {
	// 检查版本是否已存在
	existingVersion, err := s.versionRepo.GetByComponentAndVersion(ctx, version.Component, version.Version)
	if err != nil && err != gorm.ErrRecordNotFound {
		s.logger.Error("failed to check version", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}
	if existingVersion != nil {
		return errors.ErrVersionAlreadyExistsMsg
	}

	// 创建版本
	if err := s.versionRepo.Create(ctx, version); err != nil {
		s.logger.Error("failed to create version", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "创建版本失败", err)
	}

	s.logger.Info("version created",
		zap.Uint("version_id", version.ID),
		zap.String("component", version.Component),
		zap.String("version", version.Version))

	return nil
}

// GetByID 根据ID获取版本
func (s *versionService) GetByID(ctx context.Context, id uint) (*model.Version, error) {
	version, err := s.versionRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrVersionNotFoundMsg
		}
		s.logger.Error("failed to get version", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}
	return version, nil
}

// GetByComponentAndVersion 根据组件和版本号获取版本
func (s *versionService) GetByComponentAndVersion(ctx context.Context, component, version string) (*model.Version, error) {
	v, err := s.versionRepo.GetByComponentAndVersion(ctx, component, version)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrVersionNotFoundMsg
		}
		s.logger.Error("failed to get version", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}
	return v, nil
}

// Update 更新版本
func (s *versionService) Update(ctx context.Context, version *model.Version) error {
	if err := s.versionRepo.Update(ctx, version); err != nil {
		s.logger.Error("failed to update version", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "更新版本失败", err)
	}
	s.logger.Info("version updated", zap.Uint("version_id", version.ID))
	return nil
}

// Delete 删除版本
func (s *versionService) Delete(ctx context.Context, id uint) error {
	// 检查版本是否已发布
	version, err := s.versionRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrVersionNotFoundMsg
		}
		return errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}

	if version.Status == "released" {
		return errors.New(errors.ErrInvalidParams, "不能删除已发布的版本")
	}

	if err := s.versionRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete version", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "删除版本失败", err)
	}

	s.logger.Info("version deleted", zap.Uint("version_id", id))
	return nil
}

// List 获取版本列表
func (s *versionService) List(ctx context.Context, page, pageSize int) ([]*model.Version, int64, error) {
	versions, total, err := s.versionRepo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list versions", zap.Error(err))
		return nil, 0, errors.Wrap(errors.ErrDatabase, "查询版本列表失败", err)
	}
	return versions, total, nil
}

// ListByComponent 根据组件获取版本列表
func (s *versionService) ListByComponent(ctx context.Context, component string, page, pageSize int) ([]*model.Version, int64, error) {
	versions, total, err := s.versionRepo.ListByComponent(ctx, component, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list versions by component", zap.Error(err))
		return nil, 0, errors.Wrap(errors.ErrDatabase, "查询版本列表失败", err)
	}
	return versions, total, nil
}

// GetLatestReleased 获取最新发布的版本
func (s *versionService) GetLatestReleased(ctx context.Context, component string) (*model.Version, error) {
	version, err := s.versionRepo.GetLatestReleased(ctx, component)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrVersionNotFoundMsg
		}
		s.logger.Error("failed to get latest released version", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}
	return version, nil
}

// GetAllReleased 获取所有已发布的版本
func (s *versionService) GetAllReleased(ctx context.Context, component string) ([]*model.Version, error) {
	versions, err := s.versionRepo.GetAllReleased(ctx, component)
	if err != nil {
		s.logger.Error("failed to get all released versions", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}
	return versions, nil
}

// Release 发布版本
func (s *versionService) Release(ctx context.Context, versionID uint) error {
	// 获取版本
	version, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrVersionNotFoundMsg
		}
		return errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}

	// 检查版本状态
	if version.Status == "released" {
		return errors.New(errors.ErrInvalidParams, "版本已经发布")
	}

	// 更新版本状态为已发布
	if err := s.versionRepo.UpdateStatus(ctx, versionID, "released"); err != nil {
		s.logger.Error("failed to release version", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "发布版本失败", err)
	}

	s.logger.Info("version released",
		zap.Uint("version_id", versionID),
		zap.String("component", version.Component),
		zap.String("version", version.Version))

	return nil
}

// Deprecate 废弃版本
func (s *versionService) Deprecate(ctx context.Context, versionID uint) error {
	// 获取版本
	version, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrVersionNotFoundMsg
		}
		return errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}

	// 更新版本状态为已废弃
	if err := s.versionRepo.UpdateStatus(ctx, versionID, "deprecated"); err != nil {
		s.logger.Error("failed to deprecate version", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "废弃版本失败", err)
	}

	s.logger.Info("version deprecated",
		zap.Uint("version_id", versionID),
		zap.String("component", version.Component),
		zap.String("version", version.Version))

	return nil
}

// VerifyFile 验证文件哈希
func (s *versionService) VerifyFile(filePath, expectedHash string) (bool, error) {
	actualHash, err := s.CalculateFileHash(filePath)
	if err != nil {
		return false, err
	}

	return actualHash == expectedHash, nil
}

// CalculateFileHash 计算文件哈希
func (s *versionService) CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		s.logger.Error("failed to open file", zap.Error(err))
		return "", errors.Wrap(errors.ErrFileOperation, "打开文件失败", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		s.logger.Error("failed to calculate hash", zap.Error(err))
		return "", errors.Wrap(errors.ErrFileOperation, "计算哈希失败", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
