package repository

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"gorm.io/gorm"
)

// TaskRepository 任务数据访问接口
type TaskRepository interface {
	// Create 创建任务
	Create(ctx context.Context, task *model.Task) error
	// GetByID 根据ID获取任务
	GetByID(ctx context.Context, id uint) (*model.Task, error)
	// Update 更新任务
	Update(ctx context.Context, task *model.Task) error
	// Delete 删除任务（软删除）
	Delete(ctx context.Context, id uint) error
	// List 获取任务列表
	List(ctx context.Context, page, pageSize int) ([]*model.Task, int64, error)
	// ListByStatus 根据状态获取任务列表
	ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.Task, int64, error)
	// ListByType 根据类型获取任务列表
	ListByType(ctx context.Context, taskType string, page, pageSize int) ([]*model.Task, int64, error)
	// ListByCreator 根据创建者获取任务列表
	ListByCreator(ctx context.Context, creatorID uint, page, pageSize int) ([]*model.Task, int64, error)
	// ListByTimeRange 根据时间范围获取任务
	ListByTimeRange(ctx context.Context, start, end time.Time, page, pageSize int) ([]*model.Task, int64, error)
	// UpdateStatus 更新任务状态
	UpdateStatus(ctx context.Context, id uint, status string) error
	// UpdateResult 更新任务结果
	UpdateResult(ctx context.Context, id uint, result string, status string) error
	// GetPendingTasks 获取待执行的任务
	GetPendingTasks(ctx context.Context) ([]*model.Task, error)
	// GetRunningTasks 获取运行中的任务
	GetRunningTasks(ctx context.Context) ([]*model.Task, error)
	// CountByStatus 统计各状态任务数量
	CountByStatus(ctx context.Context) (map[string]int64, error)
	// GetScheduledTasks 获取定时任务
	GetScheduledTasks(ctx context.Context) ([]*model.Task, error)
}

// taskRepository 任务数据访问实现
type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 创建任务数据访问实例
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

// Create 创建任务
func (r *taskRepository) Create(ctx context.Context, task *model.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetByID 根据ID获取任务
func (r *taskRepository) GetByID(ctx context.Context, id uint) (*model.Task, error) {
	var task model.Task
	err := r.db.WithContext(ctx).First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// Update 更新任务
func (r *taskRepository) Update(ctx context.Context, task *model.Task) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// Delete 删除任务（软删除）
func (r *taskRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Task{}, id).Error
}

// List 获取任务列表
func (r *taskRepository) List(ctx context.Context, page, pageSize int) ([]*model.Task, int64, error) {
	var tasks []*model.Task
	var total int64

	// 计算总数
	if err := r.db.WithContext(ctx).Model(&model.Task{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(pageSize).
		Order("id DESC").
		Find(&tasks).Error

	return tasks, total, err
}

// ListByStatus 根据状态获取任务列表
func (r *taskRepository) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.Task, int64, error) {
	var tasks []*model.Task
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Task{}).Where("status = ?", status)

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
		Find(&tasks).Error

	return tasks, total, err
}

// ListByType 根据类型获取任务列表
func (r *taskRepository) ListByType(ctx context.Context, taskType string, page, pageSize int) ([]*model.Task, int64, error) {
	var tasks []*model.Task
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Task{}).Where("type = ?", taskType)

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
		Find(&tasks).Error

	return tasks, total, err
}

// ListByCreator 根据创建者获取任务列表
func (r *taskRepository) ListByCreator(ctx context.Context, creatorID uint, page, pageSize int) ([]*model.Task, int64, error) {
	var tasks []*model.Task
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Task{}).Where("created_by = ?", creatorID)

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
		Find(&tasks).Error

	return tasks, total, err
}

// ListByTimeRange 根据时间范围获取任务
func (r *taskRepository) ListByTimeRange(ctx context.Context, start, end time.Time, page, pageSize int) ([]*model.Task, int64, error) {
	var tasks []*model.Task
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Task{}).
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
		Find(&tasks).Error

	return tasks, total, err
}

// UpdateStatus 更新任务状态
func (r *taskRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// 如果状态是running，更新started_at
	if status == "running" {
		updates["started_at"] = gorm.Expr("NOW()")
	}

	// 如果状态是completed或failed，更新finished_at
	if status == "completed" || status == "failed" {
		updates["finished_at"] = gorm.Expr("NOW()")
	}

	return r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ?", id).
		Updates(updates).
		Error
}

// UpdateResult 更新任务结果
func (r *taskRepository) UpdateResult(ctx context.Context, id uint, result string, status string) error {
	updates := map[string]interface{}{
		"result":      result,
		"status":      status,
		"finished_at": gorm.Expr("NOW()"),
	}

	return r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ?", id).
		Updates(updates).
		Error
}

// GetPendingTasks 获取待执行的任务
func (r *taskRepository) GetPendingTasks(ctx context.Context) ([]*model.Task, error) {
	var tasks []*model.Task

	err := r.db.WithContext(ctx).
		Where("status = ?", "pending").
		Order("id ASC").
		Find(&tasks).Error

	return tasks, err
}

// GetRunningTasks 获取运行中的任务
func (r *taskRepository) GetRunningTasks(ctx context.Context) ([]*model.Task, error) {
	var tasks []*model.Task

	err := r.db.WithContext(ctx).
		Where("status = ?", "running").
		Order("id ASC").
		Find(&tasks).Error

	return tasks, err
}

// CountByStatus 统计各状态任务数量
func (r *taskRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	type StatusCount struct {
		Status string
		Count  int64
	}

	var results []StatusCount
	err := r.db.WithContext(ctx).
		Model(&model.Task{}).
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

// GetScheduledTasks 获取定时任务
func (r *taskRepository) GetScheduledTasks(ctx context.Context) ([]*model.Task, error) {
	var tasks []*model.Task

	err := r.db.WithContext(ctx).
		Where("schedule != ?", "").
		Find(&tasks).Error

	return tasks, err
}
