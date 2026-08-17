package auth

import (
	"time"

	"tinh-tien-api/internal/pkg/model"
)

type Role string

const (
	RoleOwner Role = "owner"
	RoleStaff Role = "staff"
)

type User struct {
	model.Base
	Username     string `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string `gorm:"size:255;not null" json:"-"`
	FullName     string `gorm:"size:128;not null" json:"full_name"`
	Role         Role   `gorm:"size:16;not null" json:"role"`
	Active       bool   `gorm:"not null;default:true" json:"active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}
