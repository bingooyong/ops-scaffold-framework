package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Metrics 监控指标模型
type Metrics struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	NodeID    string    `gorm:"index;size:50;not null" json:"node_id"` // 关联的节点ID
	Type      string    `gorm:"index;size:20;not null" json:"type"` // cpu, memory, disk, network
	Timestamp time.Time `gorm:"index;not null" json:"timestamp"`       // 指标采集时间

	Values JSONMap `gorm:"type:json;not null" json:"values"` // 指标数据，JSON格式
}

// TableName 指定表名
func (Metrics) TableName() string {
	return "metrics"
}

// JSONMap 用于存储JSON格式的map[string]interface{}
type JSONMap map[string]interface{}

// Scan 实现sql.Scanner接口
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value 实现driver.Valuer接口
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// MetricsAggregate 指标聚合数据（用于仪表盘）
type MetricsAggregate struct {
	NodeID    string    `json:"node_id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`

	// 聚合统计
	Avg float64 `json:"avg"`
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Sum float64 `json:"sum"`
}
