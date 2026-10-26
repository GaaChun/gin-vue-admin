package initialize

import (
	"strings"

	adapter "github.com/casbin/gorm-adapter/v3"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"gorm.io/gorm"
)

func registerDepartmentMenu(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var parent, department model.SysBaseMenu
		if err := tx.Where("name = ?", "warehouseMaster").First(&parent).Error; err != nil {
			return err
		}
		if err := tx.Where("name = ?", "warehouseDepartment").First(&department).Error; err != nil {
			return err
		}
		if err := tx.Model(&department).Updates(map[string]any{
			"parent_id": parent.ID, "path": "department", "component": "plugin/warehouse/view/department.vue",
			"title": "部门", "sort": 5, "hidden": false, "active_name": "",
		}).Error; err != nil {
			return err
		}
		var sources []uint
		if err := tx.Model(&model.SysBaseMenu{}).Where("name IN ?", []string{"warehouseMaster", "warehouseUnit", "warehouseBrand", "warehouseCategory", "warehouseMaterial"}).Pluck("id", &sources).Error; err != nil {
			return err
		}
		var grants []model.SysAuthorityMenu
		if err := tx.Where("sys_base_menu_id IN ?", sources).Find(&grants).Error; err != nil {
			return err
		}
		for _, grant := range grants {
			for _, menuID := range []uint{parent.ID, department.ID} {
				if err := tx.Where("sys_authority_authority_id = ? AND sys_base_menu_id = ?", grant.AuthorityId, menuID).
					FirstOrCreate(&model.SysAuthorityMenu{}, map[string]any{"sys_authority_authority_id": grant.AuthorityId, "sys_base_menu_id": menuID}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// Existing action-level grants are copied, never replaced or escalated to full CRUD.
// Delivery readers additionally need list access for editable department candidates.
func registerDepartmentPermissions(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var policies []adapter.CasbinRule
		if err := tx.Where("ptype = ?", "p").Find(&policies).Error; err != nil {
			return err
		}
		methods := map[string]string{"list": "GET", "detail": "GET", "create": "POST", "update": "PUT", "delete": "DELETE"}
		for _, policy := range policies {
			target := ""
			for _, resource := range []string{"units", "brands", "categories", "materials"} {
				prefix := "/warehouse/master/" + resource + "/"
				if !strings.HasPrefix(policy.V1, prefix) {
					continue
				}
				action := strings.TrimPrefix(policy.V1, prefix)
				if methods[action] == policy.V2 {
					target = "/warehouse/master/departments/" + action
				}
			}
			if (policy.V1 == "/warehouse/delivery/departments" && policy.V2 == "GET") ||
				(policy.V1 == "/warehouse/delivery/list" && policy.V2 == "GET") ||
				(policy.V1 == "/warehouse/delivery/create" && policy.V2 == "POST") ||
				(policy.V1 == "/warehouse/delivery/update" && policy.V2 == "PUT") {
				target = "/warehouse/master/departments/list"
			}
			if target == "" {
				continue
			}
			method := policy.V2
			if strings.HasSuffix(target, "/list") {
				method = "GET"
			}
			grant := adapter.CasbinRule{Ptype: "p", V0: policy.V0, V1: target, V2: method}
			if err := tx.Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", grant.Ptype, grant.V0, grant.V1, grant.V2).FirstOrCreate(&grant).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
