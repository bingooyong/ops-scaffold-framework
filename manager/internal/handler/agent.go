package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/service"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AgentHandler Agent处理器
type AgentHandler struct {
	agentService *service.AgentService
	logger       *zap.Logger
}

// NewAgentHandler 创建Agent处理器实例
func NewAgentHandler(agentService *service.AgentService, logger *zap.Logger) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
		logger:       logger,
	}
}

// OperateAgentRequest 操作Agent请求
type OperateAgentRequest struct {
	Operation string `json:"operation" binding:"required,oneof=start stop restart"`
}

// List 获取节点下的所有Agent
// GET /api/v1/nodes/:node_id/agents
func (h *AgentHandler) List(c *gin.Context) {
	nodeID := c.Param("node_id")
	if !validateAndRespond(c, nodeID, "") {
		return
	}

	// 调用Service层
	agents, err := h.agentService.ListAgents(c.Request.Context(), nodeID)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			// 避免泄露内部错误信息
			h.logger.Error("list agents failed",
				zap.String("node_id", nodeID),
				zap.Error(err))
			response.InternalServerError(c, "获取Agent列表失败，请稍后重试")
		}
		return
	}

	h.logger.Info("list agents success",
		zap.String("node_id", nodeID),
		zap.Int("count", len(agents)))

	response.Success(c, gin.H{
		"agents": agents,
		"count":  len(agents),
	})
}

// Operate 操作Agent(启动/停止/重启)
// POST /api/v1/nodes/:node_id/agents/:agent_id/operate
func (h *AgentHandler) Operate(c *gin.Context) {
	nodeID := c.Param("node_id")
	agentID := c.Param("agent_id")
	if !validateAndRespond(c, nodeID, agentID) {
		return
	}

	// 解析请求体
	var req OperateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 调用Service层
	h.logger.Info("calling daemon OperateAgent",
		zap.String("node_id", nodeID),
		zap.String("agent_id", agentID),
		zap.String("operation", req.Operation))

	err := h.agentService.OperateAgent(c.Request.Context(), nodeID, agentID, req.Operation)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			// 避免泄露内部错误信息
			h.logger.Error("operate agent failed",
				zap.String("node_id", nodeID),
				zap.String("agent_id", agentID),
				zap.String("operation", req.Operation),
				zap.Error(err))
			response.InternalServerError(c, "操作失败，请稍后重试")
		}
		return
	}

	h.logger.Info("operate agent success",
		zap.String("node_id", nodeID),
		zap.String("agent_id", agentID),
		zap.String("operation", req.Operation))

	response.Success(c, gin.H{
		"message": "操作成功",
	})
}

// GetLogs 获取Agent日志
// GET /api/v1/nodes/:node_id/agents/:agent_id/logs?lines=100
func (h *AgentHandler) GetLogs(c *gin.Context) {
	nodeID := c.Param("node_id")
	agentID := c.Param("agent_id")
	if !validateAndRespond(c, nodeID, agentID) {
		return
	}

	// 获取查询参数
	linesStr := c.Query("lines")
	lines := 100 // 默认100行
	if linesStr != "" {
		parsedLines, err := strconv.Atoi(linesStr)
		if err != nil || parsedLines <= 0 {
			response.BadRequest(c, "无效的行数参数")
			return
		}
		if parsedLines > 1000 {
			parsedLines = 1000 // 限制最大1000行
		}
		lines = parsedLines
	}

	// 调用Service层
	logs, err := h.agentService.GetAgentLogs(c.Request.Context(), nodeID, agentID, lines)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			// 检查是否是功能未实现错误
			if apiErr.Code == errors.ErrInternalServer {
				response.ErrorWithMessage(c, errors.ErrInternalServer, "获取日志功能暂未实现")
			} else {
				response.Error(c, apiErr)
			}
		} else {
			// 避免泄露内部错误信息
			h.logger.Error("get agent logs failed",
				zap.String("node_id", nodeID),
				zap.String("agent_id", agentID),
				zap.Int("lines", lines),
				zap.Error(err))
			response.InternalServerError(c, "获取日志失败，请稍后重试")
		}
		return
	}

	h.logger.Debug("get agent logs success",
		zap.String("node_id", nodeID),
		zap.String("agent_id", agentID),
		zap.Int("lines", lines))

	response.Success(c, gin.H{
		"logs":  logs,
		"count": len(logs),
	})
}

// GetMetrics 获取Agent资源使用指标
// GET /api/v1/nodes/:node_id/agents/:agent_id/metrics?duration=3600
func (h *AgentHandler) GetMetrics(c *gin.Context) {
	nodeID := c.Param("node_id")
	agentID := c.Param("agent_id")
	if !validateAndRespond(c, nodeID, agentID) {
		return
	}

	// 解析查询参数 duration（秒），默认1小时
	durationStr := c.DefaultQuery("duration", "3600")
	durationSeconds, err := strconv.ParseInt(durationStr, 10, 64)
	if err != nil || durationSeconds <= 0 {
		response.BadRequest(c, "无效的duration参数，必须为正整数（秒）")
		return
	}

	// 限制最大查询时间范围为7天
	maxDuration := int64(7 * 24 * 3600) // 7天，单位：秒
	if durationSeconds > maxDuration {
		response.BadRequest(c, fmt.Sprintf("duration不能超过%d秒（7天）", maxDuration))
		return
	}

	duration := time.Duration(durationSeconds) * time.Second

	// 调用Service层
	dataPoints, err := h.agentService.GetAgentMetrics(c.Request.Context(), nodeID, agentID, duration)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			// 避免泄露内部错误信息
			h.logger.Error("get agent metrics failed",
				zap.String("node_id", nodeID),
				zap.String("agent_id", agentID),
				zap.Duration("duration", duration),
				zap.Error(err))
			response.InternalServerError(c, "获取Agent指标失败，请稍后重试")
		}
		return
	}

	h.logger.Debug("get agent metrics success",
		zap.String("node_id", nodeID),
		zap.String("agent_id", agentID),
		zap.Duration("duration", duration),
		zap.Int("data_points_count", len(dataPoints)))

	// 转换为API响应格式
	responseData := gin.H{
		"agent_id":    agentID,
		"data_points": dataPoints,
		"count":       len(dataPoints),
	}

	response.Success(c, responseData)
}

// Sync 手动同步节点下所有Agent的状态
// POST /api/v1/nodes/:node_id/agents/sync
// 此接口用于前端手动触发同步，从Daemon获取最新的Agent状态并更新数据库
func (h *AgentHandler) Sync(c *gin.Context) {
	nodeID := c.Param("node_id")
	if !validateAndRespond(c, nodeID, "") {
		return
	}

	h.logger.Info("manually syncing agent states",
		zap.String("node_id", nodeID))

	// 调用Service层
	syncedCount, err := h.agentService.SyncAgentStatesFromDaemon(c.Request.Context(), nodeID)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			// 避免泄露内部错误信息
			h.logger.Error("sync agent states failed",
				zap.String("node_id", nodeID),
				zap.Error(err))
			response.InternalServerError(c, "同步Agent状态失败，请稍后重试")
		}
		return
	}

	h.logger.Info("sync agent states success",
		zap.String("node_id", nodeID),
		zap.Int("synced_count", syncedCount))

	response.Success(c, gin.H{
		"message":      "同步成功",
		"synced_count": syncedCount,
	})
}
