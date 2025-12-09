package handler

import (
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/response"
	"github.com/gin-gonic/gin"
)

// validateNodeID 验证节点ID格式
func validateNodeID(nodeID string) error {
	if nodeID == "" {
		return &ValidationError{Field: "node_id", Message: "节点ID不能为空"}
	}
	if len(nodeID) > 100 {
		return &ValidationError{Field: "node_id", Message: "节点ID长度不能超过100字符"}
	}
	return nil
}

// validateAgentID 验证Agent ID格式
func validateAgentID(agentID string) error {
	if agentID == "" {
		return &ValidationError{Field: "agent_id", Message: "Agent ID不能为空"}
	}
	if len(agentID) > 100 {
		return &ValidationError{Field: "agent_id", Message: "Agent ID长度不能超过100字符"}
	}
	// 只允许字母、数字、连字符、下划线
	for _, r := range agentID {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return &ValidationError{Field: "agent_id", Message: "Agent ID包含非法字符"}
		}
	}
	return nil
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// validateAndRespond 验证参数并响应错误（如果验证失败）
func validateAndRespond(c *gin.Context, nodeID, agentID string) bool {
	if err := validateNodeID(nodeID); err != nil {
		response.BadRequest(c, err.Error())
		return false
	}
	if agentID != "" {
		if err := validateAgentID(agentID); err != nil {
			response.BadRequest(c, err.Error())
			return false
		}
	}
	return true
}
