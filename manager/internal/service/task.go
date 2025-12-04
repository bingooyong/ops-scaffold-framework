package service

import (
	"context"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/repository"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TaskService 任务服务接口
type TaskService interface {
	// Create 创建任务
	Create(ctx context.Context, task *model.Task) error
	// GetByID 根据ID获取任务
	GetByID(ctx context.Context, id uint) (*model.Task, error)
	// Update 更新任务
	Update(ctx context.Context, task *model.Task) error
	// Delete 删除任务
	Delete(ctx context.Context, id uint) error
	// List 获取任务列表
	List(ctx context.Context, page, pageSize int) ([]*model.Task, int64, error)
	// ListByStatus 根据状态获取任务列表
	ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.Task, int64, error)
	// ListByCreator 根据创建者获取任务列表
	ListByCreator(ctx context.Context, creatorID uint, page, pageSize int) ([]*model.Task, int64, error)
	// Execute 执行任务
	Execute(ctx context.Context, taskID uint) error
	// Cancel 取消任务
	Cancel(ctx context.Context, taskID uint) error
	// UpdateStatus 更新任务状态
	UpdateStatus(ctx context.Context, taskID uint, status string) error
	// UpdateResult 更新任务结果
	UpdateResult(ctx context.Context, taskID uint, result string, status string) error
	// GetPendingTasks 获取待执行的任务
	GetPendingTasks(ctx context.Context) ([]*model.Task, error)
	// GetRunningTasks 获取运行中的任务
	GetRunningTasks(ctx context.Context) ([]*model.Task, error)
	// GetStatistics 获取任务统计信息
	GetStatistics(ctx context.Context) (map[string]int64, error)
}

// taskService 任务服务实现
type taskService struct {
	taskRepo  repository.TaskRepository
	nodeRepo  repository.NodeRepository
	auditRepo repository.AuditLogRepository
	logger    *zap.Logger
}

// NewTaskService 创建任务服务实例
func NewTaskService(
	taskRepo repository.TaskRepository,
	nodeRepo repository.NodeRepository,
	auditRepo repository.AuditLogRepository,
	logger *zap.Logger,
) TaskService {
	return &taskService{
		taskRepo:  taskRepo,
		nodeRepo:  nodeRepo,
		auditRepo: auditRepo,
		logger:    logger,
	}
}

// Create 创建任务
func (s *taskService) Create(ctx context.Context, task *model.Task) error {
	// 验证目标节点是否存在
	for _, nodeID := range task.TargetNodes {
		_, err := s.nodeRepo.GetByNodeID(ctx, nodeID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.New(errors.ErrNodeNotFound, "目标节点不存在: "+nodeID)
			}
			s.logger.Error("failed to check node", zap.Error(err))
			return errors.Wrap(errors.ErrDatabase, "数据库错误", err)
		}
	}

	// 创建任务
	if err := s.taskRepo.Create(ctx, task); err != nil {
		s.logger.Error("failed to create task", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "创建任务失败", err)
	}

	s.logger.Info("task created", zap.Uint("task_id", task.ID), zap.String("name", task.Name))
	return nil
}

// GetByID 根据ID获取任务
func (s *taskService) GetByID(ctx context.Context, id uint) (*model.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrTaskNotFoundMsg
		}
		s.logger.Error("failed to get task", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}
	return task, nil
}

// Update 更新任务
func (s *taskService) Update(ctx context.Context, task *model.Task) error {
	if err := s.taskRepo.Update(ctx, task); err != nil {
		s.logger.Error("failed to update task", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "更新任务失败", err)
	}
	s.logger.Info("task updated", zap.Uint("task_id", task.ID))
	return nil
}

// Delete 删除任务
func (s *taskService) Delete(ctx context.Context, id uint) error {
	// 检查任务是否正在运行
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrTaskNotFoundMsg
		}
		return errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}

	if task.Status == "running" {
		return errors.ErrTaskRunningMsg
	}

	if err := s.taskRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete task", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "删除任务失败", err)
	}

	s.logger.Info("task deleted", zap.Uint("task_id", id))
	return nil
}

// List 获取任务列表
func (s *taskService) List(ctx context.Context, page, pageSize int) ([]*model.Task, int64, error) {
	tasks, total, err := s.taskRepo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list tasks", zap.Error(err))
		return nil, 0, errors.Wrap(errors.ErrDatabase, "查询任务列表失败", err)
	}
	return tasks, total, nil
}

// ListByStatus 根据状态获取任务列表
func (s *taskService) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]*model.Task, int64, error) {
	tasks, total, err := s.taskRepo.ListByStatus(ctx, status, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list tasks by status", zap.Error(err))
		return nil, 0, errors.Wrap(errors.ErrDatabase, "查询任务列表失败", err)
	}
	return tasks, total, nil
}

// ListByCreator 根据创建者获取任务列表
func (s *taskService) ListByCreator(ctx context.Context, creatorID uint, page, pageSize int) ([]*model.Task, int64, error) {
	tasks, total, err := s.taskRepo.ListByCreator(ctx, creatorID, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list tasks by creator", zap.Error(err))
		return nil, 0, errors.Wrap(errors.ErrDatabase, "查询任务列表失败", err)
	}
	return tasks, total, nil
}

// Execute 执行任务
func (s *taskService) Execute(ctx context.Context, taskID uint) error {
	// 获取任务
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrTaskNotFoundMsg
		}
		return errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}

	// 检查任务状态
	if task.Status == "running" {
		return errors.ErrTaskRunningMsg
	}

	// 更新任务状态为运行中
	if err := s.taskRepo.UpdateStatus(ctx, taskID, "running"); err != nil {
		s.logger.Error("failed to update task status", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "更新任务状态失败", err)
	}

	s.logger.Info("task execution started", zap.Uint("task_id", taskID))

	// 这里应该调用实际的任务执行逻辑（通过gRPC或HTTP调用Agent）
	// 目前只是更新状态，实际执行逻辑需要在gRPC服务中实现

	return nil
}

// Cancel 取消任务
func (s *taskService) Cancel(ctx context.Context, taskID uint) error {
	// 获取任务
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrTaskNotFoundMsg
		}
		return errors.Wrap(errors.ErrDatabase, "数据库错误", err)
	}

	// 只有pending和running状态的任务可以取消
	if task.Status != "pending" && task.Status != "running" {
		return errors.New(errors.ErrInvalidParams, "任务状态不允许取消")
	}

	// 更新任务状态为取消
	if err := s.taskRepo.UpdateResult(ctx, taskID, "任务已取消", "cancelled"); err != nil {
		s.logger.Error("failed to cancel task", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "取消任务失败", err)
	}

	s.logger.Info("task cancelled", zap.Uint("task_id", taskID))
	return nil
}

// UpdateStatus 更新任务状态
func (s *taskService) UpdateStatus(ctx context.Context, taskID uint, status string) error {
	if err := s.taskRepo.UpdateStatus(ctx, taskID, status); err != nil {
		s.logger.Error("failed to update task status", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "更新任务状态失败", err)
	}
	s.logger.Info("task status updated", zap.Uint("task_id", taskID), zap.String("status", status))
	return nil
}

// UpdateResult 更新任务结果
func (s *taskService) UpdateResult(ctx context.Context, taskID uint, result string, status string) error {
	if err := s.taskRepo.UpdateResult(ctx, taskID, result, status); err != nil {
		s.logger.Error("failed to update task result", zap.Error(err))
		return errors.Wrap(errors.ErrDatabase, "更新任务结果失败", err)
	}
	s.logger.Info("task result updated", zap.Uint("task_id", taskID))
	return nil
}

// GetPendingTasks 获取待执行的任务
func (s *taskService) GetPendingTasks(ctx context.Context) ([]*model.Task, error) {
	tasks, err := s.taskRepo.GetPendingTasks(ctx)
	if err != nil {
		s.logger.Error("failed to get pending tasks", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "查询待执行任务失败", err)
	}
	return tasks, nil
}

// GetRunningTasks 获取运行中的任务
func (s *taskService) GetRunningTasks(ctx context.Context) ([]*model.Task, error) {
	tasks, err := s.taskRepo.GetRunningTasks(ctx)
	if err != nil {
		s.logger.Error("failed to get running tasks", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "查询运行中任务失败", err)
	}
	return tasks, nil
}

// GetStatistics 获取任务统计信息
func (s *taskService) GetStatistics(ctx context.Context) (map[string]int64, error) {
	stats, err := s.taskRepo.CountByStatus(ctx)
	if err != nil {
		s.logger.Error("failed to get task statistics", zap.Error(err))
		return nil, errors.Wrap(errors.ErrDatabase, "获取统计信息失败", err)
	}
	return stats, nil
}
