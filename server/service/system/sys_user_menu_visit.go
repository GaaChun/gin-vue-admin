package system

import (
	"errors"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrMenuVisitUnavailable = errors.New("菜单不可访问")

type UserMenuVisitService struct{}

var UserMenuVisitServiceApp = new(UserMenuVisitService)

func (s *UserMenuVisitService) Record(userID, authorityID, menuID uint) error {
	if userID == 0 || authorityID == 0 || menuID == 0 {
		return ErrMenuVisitUnavailable
	}
	var allowed int64
	if err := global.GVA_DB.Table("sys_authority_menus AS allowed").
		Joins("JOIN sys_base_menus AS menus ON menus.id = allowed.sys_base_menu_id").
		Where("allowed.sys_authority_authority_id = ? AND menus.id = ? AND menus.hidden = ?", authorityID, menuID, false).
		Where("menus.deleted_at IS NULL AND menus.path NOT LIKE 'http://%' AND menus.path NOT LIKE 'https://%'").
		Where("NOT EXISTS (SELECT 1 FROM sys_base_menus AS children WHERE children.parent_id = menus.id AND children.deleted_at IS NULL)").
		Count(&allowed).Error; err != nil {
		return err
	}
	if allowed == 0 {
		return ErrMenuVisitUnavailable
	}

	now := time.Now()
	return global.GVA_DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "menu_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"visit_count":     gorm.Expr("visit_count + ?", 1),
			"last_visited_at": now,
		}),
	}).Create(&system.SysUserMenuVisit{UserID: userID, MenuID: menuID, VisitCount: 1, LastVisitedAt: now}).Error
}

func (s *UserMenuVisitService) List(userID, authorityID uint) ([]system.RecentMenuVisit, error) {
	items := make([]system.RecentMenuVisit, 0)
	err := global.GVA_DB.Table("sys_user_menu_visits AS visits").
		Select("menus.id AS menu_id, menus.name, menus.title, menus.path, menus.icon, visits.visit_count, visits.last_visited_at").
		Joins("JOIN sys_base_menus AS menus ON menus.id = visits.menu_id").
		Joins("JOIN sys_authority_menus AS allowed ON allowed.sys_base_menu_id = menus.id").
		Where("visits.user_id = ? AND allowed.sys_authority_authority_id = ? AND menus.hidden = ?", userID, authorityID, false).
		Where("visits.deleted_at IS NULL AND menus.deleted_at IS NULL AND menus.path NOT LIKE 'http://%' AND menus.path NOT LIKE 'https://%'").
		Where("NOT EXISTS (SELECT 1 FROM sys_base_menus AS children WHERE children.parent_id = menus.id AND children.deleted_at IS NULL)").
		Order("visits.visit_count DESC, visits.last_visited_at DESC, menus.id ASC").
		Limit(6).Scan(&items).Error
	return items, err
}
