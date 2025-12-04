package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Task 任务模型
type Task struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string `gorm:"size:100;not null" json:"name"`
	Description string `gorm:"size:500" json:"description"`
	Type        string `gorm:"size:20;not null" json:"type"` // script, file_upload, service_control

	TargetNodes JSONArray `gorm:"type:json" json:"target_nodes"` // 目标节点ID列表

	Script string `gorm:"type:text" json:"script"` // 脚本内容（对于script类型）

	Params JSONMap `gorm:"type:json" json:"params"` // 任务参数，JSON格式

	Status     string     `gorm:"size:20;not null;default:'pending'" json:"status"` // pending, running, completed, failed
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`

	Result JSONMap `gorm:"type:json" json:"result"` // 执行结果

	CreatedBy uint `gorm:"index" json:"created_by"` // 创建者ID
}

// TableName 指定表名
func (Task) TableName() string {
	return "tasks"
}

// IsCompleted 任务是否已完成
func (t *Task) IsCompleted() bool {
	return t.Status == "completed" || t.Status == "failed"
}

// IsRunning 任务是否正在运行
func (t *Task) IsRunning() bool {
	return t.Status == "running"
}

// JSONArray 用于存储JSON格式的数组
type JSONArray []string

// Scan 实现sql.Scanner接口
func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONArray, 0)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value 实现driver.Valuer接口
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}
