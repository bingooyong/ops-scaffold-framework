package repository

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"gorm.io/gorm"
)

// AuditLogRepository 审计日志数据访问接口
type AuditLogRepository interface {
	// Create 创建审计日志
	Create(ctx context.Context, log *model.AuditLog) error
	// BatchCreate 批量创建审计日志
	BatchCreate(ctx context.Context, logs []*model.AuditLog) error
	// GetByID 根据ID获取审计日志
	GetByID(ctx context.Context, id uint) (*model.AuditLog, error)
	// List 获取审计日志列表
	List(ctx context.Context, page, pageSize int) ([]*model.AuditLog, int64, error)
	// ListByUserID 根据用户ID获取审计日志
	ListByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*model.AuditLog, int64, error)
	// ListByAction 根据操作类型获取审计日志
	ListByAction(ctx context.Context, action string, page, pageSize int) ([]*model.AuditLog, int64, error)
	// ListByResource 根据资源类型获取审计日志
	ListByResource(ctx context.Context, resource string, page, pageSize int) ([]*model.AuditLog, int64, error)
	// ListByTimeRange 根据时间范围获取审计日志
	ListByTimeRange(ctx context.Context, start, end time.Time, page, pageSize int) ([]*model.AuditLog, int64, error)
	// ListByIP 根据IP地址获取审计日志
	ListByIP(ctx context.Context, ip string, page, pageSize int) ([]*model.AuditLog, int64, error)
	// ListByStatus 根据状态码获取审计日志
	ListByStatus(ctx context.Context, status int, page, pageSize int) ([]*model.AuditLog, int64, error)
	// Search 多条件搜索审计日志
	Search(ctx context.Context, userID *uint, action, resource, ip string, start, end *time.Time, page, pageSize int) ([]*model.AuditLog, int64, error)
	// DeleteOlderThan 删除指定时间之前的审计日志
	DeleteOlderThan(ctx context.Context, duration time.Duration) (int64, error)
	// CountByAction 统计各操作类型数量
	CountByAction(ctx context.Context) (map[string]int64, error)
	// CountByUserID 统计用户操作次数
	CountByUserID(ctx context.Context, userID uint) (int64, error)
}

// auditLogRepository 审计日志数据访问实现
type auditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository 创建审计日志数据访问实例
func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

// Create 创建审计日志
func (r *auditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// BatchCreate 批量创建审计日志
func (r *auditLogRepository) BatchCreate(ctx context.Context, logs []*model.AuditLog) error {
	return r.db.WithContext(ctx).Create(&logs).Error
}

// GetByID 根据ID获取审计日志
func (r *auditLogRepository) GetByID(ctx context.Context, id uint) (*model.AuditLog, error) {
	var log model.AuditLog
	err := r.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// List 获取审计日志列表
func (r *auditLogRepository) List(ctx context.Context, page, pageSize int) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64

	// 计算总数
	if err := r.db.WithContext(ctx).Model(&model.AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(pageSize).
		Order("id DESC").
		Find(&logs).Error

	return logs, total, err
}

// ListByUserID 根据用户ID获取审计日志
func (r *auditLogRepository) ListByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AuditLog{}).Where("user_id = ?", userID)

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
		Find(&logs).Error

	return logs, total, err
}

// ListByAction 根据操作类型获取审计日志
func (r *auditLogRepository) ListByAction(ctx context.Context, action string, page, pageSize int) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AuditLog{}).Where("action = ?", action)

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
		Find(&logs).Error

	return logs, total, err
}

// ListByResource 根据资源类型获取审计日志
func (r *auditLogRepository) ListByResource(ctx context.Context, resource string, page, pageSize int) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AuditLog{}).Where("resource = ?", resource)

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
		Find(&logs).Error

	return logs, total, err
}

// ListByTimeRange 根据时间范围获取审计日志
func (r *auditLogRepository) ListByTimeRange(ctx context.Context, start, end time.Time, page, pageSize int) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AuditLog{}).
		Where("created_at BETWEEN ? AND ?", start, end)

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
		Find(&logs).Error

	return logs, total, err
}

// ListByIP 根据IP地址获取审计日志
func (r *auditLogRepository) ListByIP(ctx context.Context, ip string, page, pageSize int) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AuditLog{}).Where("ip = ?", ip)

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
		Find(&logs).Error

	return logs, total, err
}

// ListByStatus 根据状态码获取审计日志
func (r *auditLogRepository) ListByStatus(ctx context.Context, status int, page, pageSize int) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AuditLog{}).Where("status = ?", status)

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
		Find(&logs).Error

	return logs, total, err
}

// Search 多条件搜索审计日志
func (r *auditLogRepository) Search(ctx context.Context, userID *uint, action, resource, ip string, start, end *time.Time, page, pageSize int) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AuditLog{})

	// 构建查询条件
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if ip != "" {
		query = query.Where("ip = ?", ip)
	}
	if start != nil && end != nil {
		query = query.Where("created_at BETWEEN ? AND ?", *start, *end)
	}

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
		Find(&logs).Error

	return logs, total, err
}

// DeleteOlderThan 删除指定时间之前的审计日志
func (r *auditLogRepository) DeleteOlderThan(ctx context.Context, duration time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-duration)

	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoffTime).
		Delete(&model.AuditLog{})

	return result.RowsAffected, result.Error
}

// CountByAction 统计各操作类型数量
func (r *auditLogRepository) CountByAction(ctx context.Context) (map[string]int64, error) {
	type ActionCount struct {
		Action string
		Count  int64
	}

	var results []ActionCount
	err := r.db.WithContext(ctx).
		Model(&model.AuditLog{}).
		Select("action, COUNT(*) as count").
		Group("action").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, result := range results {
		counts[result.Action] = result.Count
	}

	return counts, nil
}

// CountByUserID 统计用户操作次数
func (r *auditLogRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&model.AuditLog{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	return count, err
}
