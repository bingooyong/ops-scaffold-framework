package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Username string `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password string `gorm:"size:100;not null" json:"-"` // 加密后的密码
	Email    string `gorm:"uniqueIndex;size:100" json:"email"`
	Phone    string `gorm:"size:20" json:"phone"`
	RealName string `gorm:"size:50" json:"real_name"`

	Role   string `gorm:"size:20;not null;default:'user'" json:"role"` // admin, user
	Status string `gorm:"size:20;not null;default:'active'" json:"status"` // active, disabled

	LastLoginAt *time.Time `json:"last_login_at"`
	LastLoginIP string     `gorm:"size:50" json:"last_login_ip"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// IsAdmin 是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// IsActive 是否为活跃用户
func (u *User) IsActive() bool {
	return u.Status == "active"
}
