package initialize

import (
	"testing"

	adapter "github.com/casbin/gorm-adapter/v3"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDepartmentMenuInheritsMasterRolesWithoutReplacingGrants(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.TempDir()+"/department-menu.db"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SysBaseMenu{}, &model.SysAuthorityMenu{}); err != nil {
		t.Fatal(err)
	}
	parent := menu("warehouseMaster", "warehouseMaster", "view/routerHolder.vue", "基础资料", 1)
	child := menu("warehouseBrand", "brand", "plugin/warehouse/view/master.vue", "客户", 2)
	department := menu("warehouseDepartment", "old", "old.vue", "old", 99)
	for _, item := range []*model.SysBaseMenu{&parent, &child, &department} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, grant := range []struct {
		role string
		menu uint
	}{{"101", parent.ID}, {"202", child.ID}, {"303", department.ID}} {
		if err := db.Model(&model.SysAuthorityMenu{}).Create(map[string]any{"sys_authority_authority_id": grant.role, "sys_base_menu_id": grant.menu}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := registerDepartmentMenu(db); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.First(&department, department.ID).Error; err != nil {
		t.Fatal(err)
	}
	if department.ParentId != parent.ID || department.Path != "department" || department.Component != "plugin/warehouse/view/department.vue" || department.Title != "部门" {
		t.Fatalf("department metadata=%#v", department)
	}
	for _, role := range []string{"101", "202", "303"} {
		var count int64
		if err := db.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ? AND sys_base_menu_id = ?", role, department.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("role %s grants=%d", role, count)
		}
	}
}

func TestDepartmentPermissionsAreAdditiveAndPreserveActionBoundaries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.TempDir()+"/department-policies.db"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&adapter.CasbinRule{}); err != nil {
		t.Fatal(err)
	}
	original := []adapter.CasbinRule{
		{Ptype: "p", V0: "101", V1: "/warehouse/master/brands/list", V2: "GET"},
		{Ptype: "p", V0: "101", V1: "/warehouse/master/brands/update", V2: "PUT"},
		{Ptype: "p", V0: "202", V1: "/warehouse/delivery/create", V2: "POST"},
		{Ptype: "p", V0: "303", V1: "/unrelated", V2: "GET"},
		{Ptype: "p", V0: "404", V1: "/warehouse/master/departments/delete", V2: "DELETE"},
	}
	if err := db.Create(&original).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := registerDepartmentPermissions(db); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range original {
		var count int64
		if err := db.Model(&adapter.CasbinRule{}).Where("id = ?", item.ID).Count(&count).Error; err != nil || count != 1 {
			t.Fatalf("original grant replaced=%#v err=%v", item, err)
		}
	}
	for _, check := range []struct {
		role, action, method string
		want                 int64
	}{
		{"101", "list", "GET", 1}, {"101", "update", "PUT", 1}, {"101", "delete", "DELETE", 0},
		{"202", "list", "GET", 1}, {"202", "create", "POST", 0}, {"303", "list", "GET", 0}, {"404", "delete", "DELETE", 1},
	} {
		var count int64
		if err := db.Model(&adapter.CasbinRule{}).Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", check.role, "/warehouse/master/departments/"+check.action, check.method).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != check.want {
			t.Fatalf("%#v got=%d", check, count)
		}
	}
}
