package initialize

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/plugin-tool/utils"
	"go.uber.org/zap"
)

func Menu(_ context.Context) {
	utils.RegisterMenus(
		menu("warehouseMaster", "warehouseMaster", "view/routerHolder.vue", "基础资料", 1),
		menu("warehouseUnit", "unit", "plugin/warehouse/view/master.vue", "物料单位", 1),
		menu("warehouseBrand", "brand", "plugin/warehouse/view/master.vue", "客户", 2),
		menu("warehouseCategory", "category", "plugin/warehouse/view/master.vue", "物料类别", 3),
		menu("warehouseMaterial", "material", "plugin/warehouse/view/master.vue", "物料", 4),
	)
	if err := global.GVA_DB.Model(&model.SysBaseMenu{}).Where("name = ?", "warehouseBrand").Update("title", "客户").Error; err != nil {
		zap.L().Error("更新客户菜单标题失败", zap.Error(err))
	}
	utils.RegisterMenus(
		menu("warehouseNotes", "warehouseNotes", "view/routerHolder.vue", "仓库管理", 2),
		menu("warehouseDelivery", "delivery", "plugin/warehouse/view/delivery.vue", "送货单", 1),
	)
	utils.RegisterMenus(
		menu("warehouseReports", "warehouseReports", "view/routerHolder.vue", "报表管理", 3),
		menu("warehouseReconciliation", "reconciliation", "plugin/warehouse/view/reconciliation.vue", "对账表", 1),
	)
}

func menu(name, path, component, title string, sort int) model.SysBaseMenu {
	return model.SysBaseMenu{Path: path, Name: name, Component: component, Sort: sort, Meta: model.Meta{Title: title, Icon: "box"}}
}
