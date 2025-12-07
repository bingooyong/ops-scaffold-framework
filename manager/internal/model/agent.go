package model

import (
	"time"

	"gorm.io/gorm"
)

// Agent Agent模型
type Agent struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// NodeID 节点ID(外键关联到nodes表)
	NodeID string `gorm:"index;size:50;not null" json:"node_id"`

	// AgentID Agent唯一标识符(在节点内唯一)
	AgentID string `gorm:"index;size:100;not null" json:"agent_id"`

	// Type Agent类型(filebeat/telegraf/node_exporter等)
	Type string `gorm:"size:50" json:"type"`

	// Status 运行状态(running/stopped/error/starting/stopping)
	Status string `gorm:"size:20;not null;default:'stopped'" json:"status"`

	// PID 进程ID(0表示未运行)
	PID int `gorm:"default:0" json:"pid"`

	// LastHeartbeat 最后心跳时间
	LastHeartbeat *time.Time `json:"last_heartbeat"`

	// LastSyncTime 最后同步时间
	LastSyncTime time.Time `gorm:"not null" json:"last_sync_time"`
}

// TableName 指定表名
func (Agent) TableName() string {
	return "agents"
}

// IsRunning Agent是否正在运行
func (a *Agent) IsRunning() bool {
	return a.Status == "running"
}
