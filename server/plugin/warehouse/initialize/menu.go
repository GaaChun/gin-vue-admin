package initialize

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/plugin-tool/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Menu(_ context.Context) {
	utils.RegisterMenus(
		menu("warehouseMaster", "warehouseMaster", "view/routerHolder.vue", "基础资料", 1),
		menu("warehouseUnit", "unit", "plugin/warehouse/view/master.vue", "物料单位", 1),
		menu("warehouseBrand", "brand", "plugin/warehouse/view/master.vue", "客户", 2),
		menu("warehouseCategory", "category", "plugin/warehouse/view/master.vue", "物料类别", 3),
		menu("warehouseMaterial", "material", "plugin/warehouse/view/master.vue", "物料", 4),
		menu("warehouseDepartment", "department", "plugin/warehouse/view/department.vue", "部门", 5),
	)
	if err := registerDepartmentMenu(global.GVA_DB); err != nil {
		zap.L().Error("注册部门菜单权限失败", zap.Error(err))
	}
	if err := global.GVA_DB.Model(&model.SysBaseMenu{}).Where("name = ?", "warehouseBrand").Update("title", "客户").Error; err != nil {
		zap.L().Error("更新客户菜单标题失败", zap.Error(err))
	}
	utils.RegisterMenus(
		menu("warehouseNotes", "warehouseNotes", "view/routerHolder.vue", "仓库管理", 2),
		menu("warehouseDelivery", "delivery", "plugin/warehouse/view/delivery.vue", "送货单", 1),
		menu("warehouseDeliveryDisplaySettings", "deliveryDisplaySettings", "plugin/warehouse/view/delivery-display-settings.vue", "单据展示设置", 2),
	)
	if err := registerDeliveryDisplayMenu(); err != nil {
		zap.L().Error("注册单据展示设置菜单权限失败", zap.Error(err))
	}
	utils.RegisterMenus(
		menu("warehouseReports", "warehouseReports", "view/routerHolder.vue", "报表管理", 3),
		menu("warehouseReconciliation", "reconciliation", "plugin/warehouse/view/reconciliation.vue", "对账表", 1),
	)
}

// RegisterMenus only creates missing records; reconcile this independent page on
// every startup and inherit access from the delivery menu without replacing grants.
func registerDeliveryDisplayMenu() error {
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		var parent, delivery, settings model.SysBaseMenu
		if err := tx.Where("name = ?", "warehouseNotes").First(&parent).Error; err != nil {
			return err
		}
		if err := tx.Where("name = ?", "warehouseDelivery").First(&delivery).Error; err != nil {
			return err
		}
		if err := tx.Where("name = ?", "warehouseDeliveryDisplaySettings").First(&settings).Error; err != nil {
			return err
		}
		if err := tx.Model(&settings).Updates(map[string]any{
			"parent_id":   parent.ID,
			"path":        "deliveryDisplaySettings",
			"component":   "plugin/warehouse/view/delivery-display-settings.vue",
			"title":       "单据展示设置",
			"sort":        2,
			"hidden":      false,
			"active_name": "",
		}).Error; err != nil {
			return err
		}
		var grants []model.SysAuthorityMenu
		if err := tx.Where("sys_base_menu_id = ?", delivery.ID).Find(&grants).Error; err != nil {
			return err
		}
		for _, grant := range grants {
			if err := tx.Where("sys_authority_authority_id = ? AND sys_base_menu_id = ?", grant.AuthorityId, settings.ID).
				FirstOrCreate(&model.SysAuthorityMenu{}, map[string]any{
					"sys_authority_authority_id": grant.AuthorityId,
					"sys_base_menu_id":           settings.ID,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func menu(name, path, component, title string, sort int) model.SysBaseMenu {
	return model.SysBaseMenu{Path: path, Name: name, Component: component, Sort: sort, Meta: model.Meta{Title: title, Icon: "box"}}
}
