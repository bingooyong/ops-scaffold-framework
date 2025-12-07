package handler

import (
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/service"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/errors"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NodeHandler 节点处理器
type NodeHandler struct {
	nodeService service.NodeService
	logger      *zap.Logger
}

// NewNodeHandler 创建节点处理器实例
func NewNodeHandler(nodeService service.NodeService, logger *zap.Logger) *NodeHandler {
	return &NodeHandler{
		nodeService: nodeService,
		logger:      logger,
	}
}

// RegisterNodeRequest 节点注册请求
type RegisterNodeRequest struct {
	NodeID   string            `json:"node_id" binding:"required"`
	Hostname string            `json:"hostname" binding:"required"`
	IP       string            `json:"ip" binding:"required"`
	OS       string            `json:"os" binding:"required"`
	Arch     string            `json:"arch" binding:"required"`
	Labels   map[string]string `json:"labels"`
}

// List 获取节点列表
func (h *NodeHandler) List(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)
	status := parseStringQuery(c, "status", "")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var nodes []*model.Node
	var total int64
	var err error

	if status != "" {
		nodes, total, err = h.nodeService.ListByStatus(c.Request.Context(), status, page, pageSize)
	} else {
		nodes, total, err = h.nodeService.List(c.Request.Context(), page, pageSize)
	}

	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Page(c, nodes, page, pageSize, total)
}

// Get 获取节点详情
func (h *NodeHandler) Get(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		response.BadRequest(c, "节点ID不能为空")
		return
	}

	node, err := h.nodeService.GetByNodeID(c.Request.Context(), nodeID)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, gin.H{
		"node": node,
	})
}

// Delete 删除节点
func (h *NodeHandler) Delete(c *gin.Context) {
	id := parseUintParam(c, "id")
	if id == 0 {
		response.BadRequest(c, "无效的节点ID")
		return
	}

	if err := h.nodeService.Delete(c.Request.Context(), id); err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, gin.H{
		"message": "节点已删除",
	})
}

// GetStatistics 获取节点统计信息
func (h *NodeHandler) GetStatistics(c *gin.Context) {
	stats, err := h.nodeService.GetStatistics(c.Request.Context())
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			response.Error(c, apiErr)
		} else {
			response.InternalServerError(c, err.Error())
		}
		return
	}

	// 构建完整的统计信息
	var total, online, offline int64

	// 从统计结果中提取各状态的数量
	if onlineCount, ok := stats["online"]; ok {
		online = onlineCount
		total += onlineCount
	}
	if offlineCount, ok := stats["offline"]; ok {
		offline = offlineCount
		total += offlineCount
	}
	// 处理其他可能的状态（如 unknown）
	for status, count := range stats {
		if status != "online" && status != "offline" {
			total += count
		}
	}

	statistics := gin.H{
		"total":   total,
		"online":  online,
		"offline": offline,
	}

	response.Success(c, gin.H{
		"statistics": statistics,
	})
}
