package initialize

import (
	"context"

	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/plugin-tool/utils"
)

func Api(_ context.Context) {
	entities := []model.SysApi{
		api("POST", "/warehouse/master/mnemonic", "生成助记符建议"),
		api("GET", "/warehouse/master/categories/tree", "查询类别树"),
		api("POST", "/warehouse/master/materials/uploadImage", "上传物料图片"),
		api("POST", "/warehouse/delivery/reserveNumber", "预占送货单号"),
		api("POST", "/warehouse/delivery/create", "创建送货单"),
		api("PUT", "/warehouse/delivery/update", "更新送货单"),
		api("DELETE", "/warehouse/delivery/delete", "删除送货单"),
		api("POST", "/warehouse/delivery/review", "审核送货单"),
		api("POST", "/warehouse/delivery/unreview", "反审核送货单"),
		api("GET", "/warehouse/delivery/list", "查询送货单"),
		api("GET", "/warehouse/delivery/detail", "查询送货单详情"),
		api("GET", "/warehouse/reconciliation/columns", "查询对账列字典"),
		api("GET", "/warehouse/reconciliation/options", "查询历史快照筛选选项"),
		api("POST", "/warehouse/reconciliation/query", "查询对账表"),
		api("POST", "/warehouse/reconciliation/export", "导出全量对账表"),
		api("POST", "/warehouse/reconciliation/printData", "打印全量对账表"),
		api("GET", "/warehouse/schemes/list", "查询报表方案"),
		api("GET", "/warehouse/schemes/detail", "读取报表方案"),
		api("POST", "/warehouse/schemes/create", "创建报表方案"),
		api("PUT", "/warehouse/schemes/update", "修改报表方案"),
		api("DELETE", "/warehouse/schemes/delete", "删除报表方案"),
		api("POST", "/warehouse/schemes/publish", "管理共享报表方案"),
	}
	for _, resource := range []struct{ path, label string }{
		{"units", "单位"}, {"brands", "品牌"}, {"categories", "类别"}, {"materials", "物料"},
	} {
		prefix := "/warehouse/master/" + resource.path
		entities = append(entities,
			api("GET", prefix+"/list", "查询"+resource.label),
			api("GET", prefix+"/detail", "查询"+resource.label+"详情"),
			api("POST", prefix+"/create", "创建"+resource.label),
			api("PUT", prefix+"/update", "更新"+resource.label),
			api("DELETE", prefix+"/delete", "删除"+resource.label),
			api("GET", prefix+"/template", "下载"+resource.label+"模板"),
			api("POST", prefix+"/import", "导入"+resource.label),
			api("GET", prefix+"/export", "导出"+resource.label),
		)
	}
	utils.RegisterApis(entities...)
}

func api(method, path, description string) model.SysApi {
	return model.SysApi{Path: path, Method: method, ApiGroup: "仓储管理", Description: description}
}
