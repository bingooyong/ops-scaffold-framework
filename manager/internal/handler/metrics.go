package handler

import (
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/service"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MetricsHandler 监控指标处理器
type MetricsHandler struct {
	metricsService service.MetricsService
	logger         *zap.Logger
}

// NewMetricsHandler 创建监控指标处理器实例
func NewMetricsHandler(metricsService service.MetricsService, logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		metricsService: metricsService,
		logger:         logger,
	}
}

// GetLatestMetrics 获取节点最新指标
// GET /api/v1/metrics/nodes/:node_id/latest
func (h *MetricsHandler) GetLatestMetrics(c *gin.Context) {
	nodeID := c.Param("node_id")
	if nodeID == "" {
		response.BadRequest(c, "节点ID不能为空")
		return
	}

	// 调用 Service 层方法（将在 Task 1.2 中实现）
	metrics, err := h.metricsService.GetLatestMetricsByNodeID(c.Request.Context(), nodeID)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	// 如果节点不存在或没有数据，返回空 map
	if metrics == nil {
		metrics = make(map[string]*model.Metrics)
	}

	response.Success(c, metrics)
}

// GetMetricsHistory 获取历史指标数据
// GET /api/v1/metrics/nodes/:node_id/:type/history
func (h *MetricsHandler) GetMetricsHistory(c *gin.Context) {
	nodeID := c.Param("node_id")
	if nodeID == "" {
		response.BadRequest(c, "节点ID不能为空")
		return
	}

	metricType := c.Param("type")
	if metricType == "" {
		response.BadRequest(c, "指标类型不能为空")
		return
	}

	// 验证指标类型
	validTypes := map[string]bool{
		"cpu":     true,
		"memory":  true,
		"disk":    true,
		"network": true,
	}
	if !validTypes[metricType] {
		response.BadRequest(c, "无效的指标类型，支持的类型: cpu, memory, disk, network")
		return
	}

	// 解析时间参数
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		response.BadRequest(c, "start_time 和 end_time 参数不能为空")
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		response.BadRequest(c, "start_time 格式错误，请使用 ISO8601 格式 (RFC3339)")
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		response.BadRequest(c, "end_time 格式错误，请使用 ISO8601 格式 (RFC3339)")
		return
	}

	// 验证时间范围不超过 30 天
	maxDuration := 30 * 24 * time.Hour
	if endTime.Sub(startTime) > maxDuration {
		response.BadRequest(c, "时间范围不能超过 30 天")
		return
	}

	// 验证时间顺序
	if endTime.Before(startTime) {
		response.BadRequest(c, "end_time 必须大于 start_time")
		return
	}

	// 调用 Service 层方法（将在 Task 1.2 中实现）
	metrics, err := h.metricsService.GetMetricsHistoryWithSampling(c.Request.Context(), nodeID, metricType, startTime, endTime)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, metrics)
}

// GetMetricsSummary 获取指标统计摘要
// GET /api/v1/metrics/nodes/:node_id/summary
func (h *MetricsHandler) GetMetricsSummary(c *gin.Context) {
	nodeID := c.Param("node_id")
	if nodeID == "" {
		response.BadRequest(c, "节点ID不能为空")
		return
	}

	// 解析可选的时间参数，默认最近 24 小时
	now := time.Now()
	startTime := now.Add(-24 * time.Hour)
	endTime := now

	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if startTimeStr != "" {
		parsed, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			response.BadRequest(c, "start_time 格式错误，请使用 ISO8601 格式 (RFC3339)")
			return
		}
		startTime = parsed
	}

	if endTimeStr != "" {
		parsed, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			response.BadRequest(c, "end_time 格式错误，请使用 ISO8601 格式 (RFC3339)")
			return
		}
		endTime = parsed
	}

	// 验证时间范围不超过 30 天
	maxDuration := 30 * 24 * time.Hour
	if endTime.Sub(startTime) > maxDuration {
		response.BadRequest(c, "时间范围不能超过 30 天")
		return
	}

	// 验证时间顺序
	if endTime.Before(startTime) {
		response.BadRequest(c, "end_time 必须大于 start_time")
		return
	}

	// 调用 Service 层方法（将在 Task 1.2 中实现）
	summary, err := h.metricsService.GetMetricsSummaryStats(c.Request.Context(), nodeID, startTime, endTime)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, summary)
}

// GetClusterOverview 获取集群资源概览
// GET /api/v1/metrics/cluster/overview
func (h *MetricsHandler) GetClusterOverview(c *gin.Context) {
	// 调用 Service 层方法
	overview, err := h.metricsService.GetClusterOverview(c.Request.Context())
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, overview)
}

