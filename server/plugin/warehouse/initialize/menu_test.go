package initialize

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDeliveryDisplayMenuRegistrationRepairsMetadataAndInheritsDeliveryRoles(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.TempDir()+"/menus.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SysBaseMenu{}, &model.SysAuthorityMenu{}); err != nil {
		t.Fatal(err)
	}
	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })

	Menu(context.Background())
	findMenu := func(name string) model.SysBaseMenu {
		t.Helper()
		var result model.SysBaseMenu
		if err := db.Where("name = ?", name).First(&result).Error; err != nil {
			t.Fatal(err)
		}
		return result
	}
	parent := findMenu("warehouseNotes")
	delivery := findMenu("warehouseDelivery")
	settings := findMenu("warehouseDeliveryDisplaySettings")
	if err := db.Model(&settings).Updates(map[string]any{
		"parent_id": 0, "path": "oldSettings", "component": "old.vue",
		"title": "old", "sort": 99, "hidden": true, "active_name": "warehouseDelivery",
	}).Error; err != nil {
		t.Fatal(err)
	}
	grant := func(role string, menuID uint) {
		t.Helper()
		if err := db.Model(&model.SysAuthorityMenu{}).Create(map[string]any{
			"sys_authority_authority_id": role, "sys_base_menu_id": menuID,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	grant("101", delivery.ID)
	grant("202", delivery.ID)
	grant("202", settings.ID)
	grant("303", parent.ID)
	grant("404", settings.ID)

	// Repeated startup must repair an existing row, not create a duplicate or
	// replace manually assigned permissions. Later delivery grants also inherit.
	Menu(context.Background())
	grant("505", delivery.ID)
	Menu(context.Background())
	Menu(context.Background())
	settings = findMenu("warehouseDeliveryDisplaySettings")
	if settings.ParentId != parent.ID || settings.Path != "deliveryDisplaySettings" ||
		settings.Component != "plugin/warehouse/view/delivery-display-settings.vue" ||
		settings.Title != "单据展示设置" || settings.Sort != 2 || settings.Hidden || settings.ActiveName != "" {
		t.Fatalf("settings menu metadata was not repaired: %#v", settings)
	}
	var menuCount int64
	if err := db.Model(&model.SysBaseMenu{}).Where("name = ?", settings.Name).Count(&menuCount).Error; err != nil {
		t.Fatal(err)
	}
	if menuCount != 1 {
		t.Fatalf("restarts created %d settings menus", menuCount)
	}
	for role, expected := range map[string]int64{"101": 1, "202": 1, "303": 0, "404": 1, "505": 1} {
		var count int64
		if err := db.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ? AND sys_base_menu_id = ?", role, settings.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != expected {
			t.Fatalf("role %s settings grants = %d, want %d", role, count, expected)
		}
	}
}
