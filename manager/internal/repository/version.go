package repository

import (
	"context"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"gorm.io/gorm"
)

// VersionRepository 版本数据访问接口
type VersionRepository interface {
	// Create 创建版本
	Create(ctx context.Context, version *model.Version) error
	// GetByID 根据ID获取版本
	GetByID(ctx context.Context, id uint) (*model.Version, error)
	// GetByComponentAndVersion 根据组件和版本号获取版本
	GetByComponentAndVersion(ctx context.Context, component, version string) (*model.Version, error)
	// Update 更新版本
	Update(ctx context.Context, version *model.Version) error
	// Delete 删除版本（软删除）
	Delete(ctx context.Context, id uint) error
	// List 获取版本列表
	List(ctx context.Context, page, pageSize int) ([]*model.Version, int64, error)
	// ListByComponent 根据组件获取版本列表
	ListByComponent(ctx context.Context, component string, page, pageSize int) ([]*model.Version, int64, error)
	// ListByStatus 根据状态获取版本列表
	ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.Version, int64, error)
	// GetLatestReleased 获取最新发布的版本
	GetLatestReleased(ctx context.Context, component string) (*model.Version, error)
	// GetAllReleased 获取所有已发布的版本
	GetAllReleased(ctx context.Context, component string) ([]*model.Version, error)
	// UpdateStatus 更新版本状态
	UpdateStatus(ctx context.Context, id uint, status string) error
	// CountByComponent 统计各组件版本数量
	CountByComponent(ctx context.Context) (map[string]int64, error)
	// CountByStatus 统计各状态版本数量
	CountByStatus(ctx context.Context) (map[string]int64, error)
}

// versionRepository 版本数据访问实现
type versionRepository struct {
	db *gorm.DB
}

// NewVersionRepository 创建版本数据访问实例
func NewVersionRepository(db *gorm.DB) VersionRepository {
	return &versionRepository{db: db}
}

// Create 创建版本
func (r *versionRepository) Create(ctx context.Context, version *model.Version) error {
	return r.db.WithContext(ctx).Create(version).Error
}

// GetByID 根据ID获取版本
func (r *versionRepository) GetByID(ctx context.Context, id uint) (*model.Version, error) {
	var version model.Version
	err := r.db.WithContext(ctx).First(&version, id).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// GetByComponentAndVersion 根据组件和版本号获取版本
func (r *versionRepository) GetByComponentAndVersion(ctx context.Context, component, version string) (*model.Version, error) {
	var v model.Version
	err := r.db.WithContext(ctx).
		Where("component = ? AND version = ?", component, version).
		First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// Update 更新版本
func (r *versionRepository) Update(ctx context.Context, version *model.Version) error {
	return r.db.WithContext(ctx).Save(version).Error
}

// Delete 删除版本（软删除）
func (r *versionRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Version{}, id).Error
}

// List 获取版本列表
func (r *versionRepository) List(ctx context.Context, page, pageSize int) ([]*model.Version, int64, error) {
	var versions []*model.Version
	var total int64

	// 计算总数
	if err := r.db.WithContext(ctx).Model(&model.Version{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(pageSize).
		Order("id DESC").
		Find(&versions).Error

	return versions, total, err
}

// ListByComponent 根据组件获取版本列表
func (r *versionRepository) ListByComponent(ctx context.Context, component string, page, pageSize int) ([]*model.Version, int64, error) {
	var versions []*model.Version
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Version{}).Where("component = ?", component)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Offset(offset).
		Limit(pageSize).
		Order("id DESC").
		Find(&versions).Error

	return versions, total, err
}

// ListByStatus 根据状态获取版本列表
func (r *versionRepository) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.Version, int64, error) {
	var versions []*model.Version
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Version{}).Where("status = ?", status)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Offset(offset).
		Limit(pageSize).
		Order("id DESC").
		Find(&versions).Error

	return versions, total, err
}

// GetLatestReleased 获取最新发布的版本
func (r *versionRepository) GetLatestReleased(ctx context.Context, component string) (*model.Version, error) {
	var version model.Version

	err := r.db.WithContext(ctx).
		Where("component = ? AND status = ?", component, "released").
		Order("id DESC").
		First(&version).Error

	if err != nil {
		return nil, err
	}

	return &version, nil
}

// GetAllReleased 获取所有已发布的版本
func (r *versionRepository) GetAllReleased(ctx context.Context, component string) ([]*model.Version, error) {
	var versions []*model.Version

	query := r.db.WithContext(ctx).Where("status = ?", "released")

	if component != "" {
		query = query.Where("component = ?", component)
	}

	err := query.Order("id DESC").Find(&versions).Error
	return versions, err
}

// UpdateStatus 更新版本状态
func (r *versionRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.Version{}).
		Where("id = ?", id).
		Update("status", status).
		Error
}

// CountByComponent 统计各组件版本数量
func (r *versionRepository) CountByComponent(ctx context.Context) (map[string]int64, error) {
	type ComponentCount struct {
		Component string
		Count     int64
	}

	var results []ComponentCount
	err := r.db.WithContext(ctx).
		Model(&model.Version{}).
		Select("component, COUNT(*) as count").
		Group("component").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, result := range results {
		counts[result.Component] = result.Count
	}

	return counts, nil
}

// CountByStatus 统计各状态版本数量
func (r *versionRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	type StatusCount struct {
		Status string
		Count  int64
	}

	var results []StatusCount
	err := r.db.WithContext(ctx).
		Model(&model.Version{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, result := range results {
		counts[result.Status] = result.Count
	}

	return counts, nil
}
