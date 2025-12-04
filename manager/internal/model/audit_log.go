package model

import (
	"time"
)

// AuditLog 审计日志模型
type AuditLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID   uint   `gorm:"index;not null" json:"user_id"` // 操作用户ID
	Username string `gorm:"size:50;not null" json:"username"` // 冗余字段，方便查询

	Action   string `gorm:"size:100;not null" json:"action"`   // 操作类型，如：login, create_node, delete_task
	Resource string `gorm:"size:100" json:"resource"`         // 操作的资源，如：node:123, task:456
	Method   string `gorm:"size:10" json:"method"`            // HTTP方法：GET, POST, PUT, DELETE
	Path     string `gorm:"size:200" json:"path"`             // 请求路径
	IP       string `gorm:"size:50" json:"ip"`                // 操作者IP

	Status  int    `gorm:"not null" json:"status"`   // 操作结果状态码：200, 400, 500等
	Message string `gorm:"size:500" json:"message"`  // 操作结果消息
	Details JSONMap `gorm:"type:json" json:"details"` // 详细信息，JSON格式

	Duration int64 `json:"duration"` // 请求耗时（毫秒）
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "audit_logs"
}

// IsSuccess 操作是否成功
func (a *AuditLog) IsSuccess() bool {
	return a.Status >= 200 && a.Status < 300
}
