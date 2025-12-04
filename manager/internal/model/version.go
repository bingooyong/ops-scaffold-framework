package model

import (
	"time"

	"gorm.io/gorm"
)

// Version 版本模型
type Version struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Component string `gorm:"size:20;not null" json:"component"` // agent, daemon
	Version   string `gorm:"size:20;not null" json:"version"`   // 版本号，如：1.0.0

	FileName    string `gorm:"size:200;not null" json:"file_name"`    // 文件名
	FileSize    int64  `gorm:"not null" json:"file_size"`             // 文件大小（字节）
	DownloadURL string `gorm:"size:500;not null" json:"download_url"` // 下载URL

	Hash      string `gorm:"size:100;not null" json:"hash"`      // SHA-256哈希
	Signature string `gorm:"type:text;not null" json:"signature"` // 数字签名（Base64编码）

	Description string `gorm:"type:text" json:"description"` // 版本说明

	Status string `gorm:"size:20;not null;default:'draft'" json:"status"` // draft, testing, released, deprecated

	ReleaseType string `gorm:"size:20" json:"release_type"` // major, minor, patch, hotfix

	UploadedBy uint `gorm:"index" json:"uploaded_by"` // 上传者ID
}

// TableName 指定表名
func (Version) TableName() string {
	return "versions"
}

// IsReleased 是否已发布
func (v *Version) IsReleased() bool {
	return v.Status == "released"
}

// IsDraft 是否为草稿
func (v *Version) IsDraft() bool {
	return v.Status == "draft"
}
