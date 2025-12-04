package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Node 节点模型
type Node struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	NodeID   string `gorm:"uniqueIndex;size:50;not null" json:"node_id"` // Daemon生成的UUID
	Hostname string `gorm:"size:100;not null" json:"hostname"`
	IP       string `gorm:"size:50;not null" json:"ip"`
	OS       string `gorm:"size:50" json:"os"`
	Arch     string `gorm:"size:20" json:"arch"`

	Labels MapString `gorm:"type:json" json:"labels"` // 标签，用于分组和筛选

	DaemonVersion string `gorm:"size:20" json:"daemon_version"`
	AgentVersion  string `gorm:"size:20" json:"agent_version"`

	Status     string     `gorm:"size:20;not null;default:'offline'" json:"status"` // online, offline
	LastSeenAt *time.Time `json:"last_seen_at"`

	RegisterAt time.Time `json:"register_at"`
}

// TableName 指定表名
func (Node) TableName() string {
	return "nodes"
}

// IsOnline 节点是否在线
func (n *Node) IsOnline() bool {
	return n.Status == "online"
}

// MapString 用于存储JSON格式的map
type MapString map[string]string

// Scan 实现sql.Scanner接口
func (m *MapString) Scan(value interface{}) error {
	if value == nil {
		*m = make(MapString)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, m)
}

// Value 实现driver.Valuer接口
func (m MapString) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}
