package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// SysUserMenuVisit tracks menu usage for the authenticated user.
type SysUserMenuVisit struct {
	global.GVA_MODEL
	UserID        uint      `json:"userId" gorm:"uniqueIndex:idx_user_menu_visit;not null"`
	MenuID        uint      `json:"menuId" gorm:"uniqueIndex:idx_user_menu_visit;not null"`
	VisitCount    uint64    `json:"visitCount" gorm:"not null;default:0"`
	LastVisitedAt time.Time `json:"lastVisitedAt" gorm:"not null"`
}

func (SysUserMenuVisit) TableName() string { return "sys_user_menu_visits" }

type RecentMenuVisit struct {
	MenuID        uint      `json:"menuId"`
	Name          string    `json:"name"`
	Title         string    `json:"title"`
	Path          string    `json:"path"`
	Icon          string    `json:"icon"`
	VisitCount    uint64    `json:"visitCount"`
	LastVisitedAt time.Time `json:"lastVisitedAt"`
}
