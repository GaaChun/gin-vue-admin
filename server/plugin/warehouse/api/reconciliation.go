package api

import (
	"net/http"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/service"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
)

type ReconciliationAPI struct {
	report  *service.ReconciliationService
	schemes *service.ReportSchemeService
}

func NewReconciliationAPI(report *service.ReconciliationService, schemes *service.ReportSchemeService) ReconciliationAPI {
	return ReconciliationAPI{report, schemes}
}
func DefaultReconciliationAPI() ReconciliationAPI {
	return NewReconciliationAPI(service.DefaultReconciliationService(), service.DefaultReportSchemeService())
}
func (a ReconciliationAPI) fail(c *gin.Context, err error) { MasterAPI{}.fail(c, err) }

// Columns returns the controlled report column registry.
// @Summary 对账可选列
// @Tags warehouse-reconciliation
// @Produce json
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/reconciliation/columns [get]
func (a ReconciliationAPI) Columns(c *gin.Context) {
	response.OkWithDetailed(service.ReconciliationColumns(), "操作成功", c)
}

// Options returns historical snapshot filter choices for every selectable state.
// @Summary 对账历史客户、类别、部门候选
// @Tags warehouse-reconciliation
// @Produce json
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/reconciliation/options [get]
func (a ReconciliationAPI) Options(c *gin.Context) {
	result, err := a.report.Options(c.Request.Context())
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(result, "操作成功", c)
}

// Query returns grouped report rows and global summaries.
// @Summary 按所选单据状态查询送货对账表
// @Tags warehouse-reconciliation
// @Accept json
// @Produce json
// @Param body body request.ReconciliationQuery true "筛选、有序列及分页"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/reconciliation/query [post]
func (a ReconciliationAPI) Query(c *gin.Context) {
	var in req.ReconciliationQuery
	if !bind(c, &in) {
		return
	}
	result, err := a.report.Query(c.Request.Context(), in, true)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(result, "操作成功", c)
}

// Export returns every group in the requested projection as text-safe XLSX.
// @Summary 导出完整对账表Excel
// @Tags warehouse-reconciliation
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param body body request.ReconciliationQuery true "筛选和有序列（忽略分页）"
// @Success 200 {file} binary
// @Security ApiKeyAuth
// @Router /warehouse/reconciliation/export [post]
func (a ReconciliationAPI) Export(c *gin.Context) {
	var in req.ReconciliationQuery
	if !bind(c, &in) {
		return
	}
	data, err := a.report.Export(c.Request.Context(), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=reconciliation.xlsx; filename*=UTF-8''%E9%80%81%E8%B4%A7%E5%AF%B9%E8%B4%A6%E8%A1%A8.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// PrintData returns the same projection without pagination.
// @Summary 获取完整对账打印数据
// @Tags warehouse-reconciliation
// @Accept json
// @Produce json
// @Param body body request.ReconciliationQuery true "筛选和有序列（忽略分页）"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/reconciliation/printData [post]
func (a ReconciliationAPI) PrintData(c *gin.Context) {
	var in req.ReconciliationQuery
	if !bind(c, &in) {
		return
	}
	result, err := a.report.Query(c.Request.Context(), in, false)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(gin.H{"projection": result, "filterDescription": service.ReportFilterDescription(in.Filters), "summaryLines": service.ReportSummaryLines(result.Summary), "title": "送货对账表"}, "操作成功", c)
}

// SchemeList returns only the actor's personal schemes plus shared schemes.
// @Summary 查询可用报表方案
// @Tags warehouse-scheme
// @Produce json
// @Param query query request.ReportSchemeListQuery false "分页筛选"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/schemes/list [get]
func (a ReconciliationAPI) SchemeList(c *gin.Context) {
	var in req.ReportSchemeListQuery
	if err := c.ShouldBindQuery(&in); err != nil {
		a.fail(c, service.NewDomainError(service.ValidationFailed, "请求参数无效"))
		return
	}
	result, err := a.schemes.List(c.Request.Context(), utils.GetUserID(c), utils.GetUserAuthorityId(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(result, "操作成功", c)
}

// SchemeDetail enforces personal ownership at the service boundary.
// @Summary 查询报表方案详情
// @Tags warehouse-scheme
// @Produce json
// @Param id query int true "方案ID"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/schemes/detail [get]
func (a ReconciliationAPI) SchemeDetail(c *gin.Context) {
	id, ok := detailID(c)
	if !ok {
		return
	}
	result, err := a.schemes.Detail(c.Request.Context(), utils.GetUserID(c), utils.GetUserAuthorityId(c), id)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(result, "操作成功", c)
}

// SchemeCreate persists a personal or separately authorized shared scheme.
// @Summary 创建报表方案
// @Tags warehouse-scheme
// @Accept json
// @Produce json
// @Param body body request.CreateReportSchemeInput true "名称范围有序列"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/schemes/create [post]
func (a ReconciliationAPI) SchemeCreate(c *gin.Context) {
	var in req.CreateReportSchemeInput
	if !bind(c, &in) {
		return
	}
	result, err := a.schemes.Create(c.Request.Context(), utils.GetUserID(c), utils.GetUserAuthorityId(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(result, "操作成功", c)
}

// SchemeUpdate checks ownership, scope transitions and optimistic version.
// @Summary 修改报表方案
// @Tags warehouse-scheme
// @Accept json
// @Produce json
// @Param body body request.UpdateReportSchemeInput true "方案ID版本及配置"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/schemes/update [put]
func (a ReconciliationAPI) SchemeUpdate(c *gin.Context) {
	var in req.UpdateReportSchemeInput
	if !bind(c, &in) {
		return
	}
	result, err := a.schemes.Update(c.Request.Context(), utils.GetUserID(c), utils.GetUserAuthorityId(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(result, "操作成功", c)
}

// SchemePublish is the independent shared-management Casbin permission endpoint.
// @Summary 发布个人报表方案为共享
// @Tags warehouse-scheme
// @Accept json
// @Produce json
// @Param body body request.UpdateReportSchemeInput true "方案ID版本及配置（scope必须shared）"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/schemes/publish [post]
func (a ReconciliationAPI) SchemePublish(c *gin.Context) {
	var in req.UpdateReportSchemeInput
	if !bind(c, &in) {
		return
	}
	if in.Scope != "shared" {
		a.fail(c, service.NewDomainError(service.ValidationFailed, "发布范围必须为shared").WithField("scope"))
		return
	}
	result, err := a.schemes.Update(c.Request.Context(), utils.GetUserID(c), utils.GetUserAuthorityId(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(result, "操作成功", c)
}

// SchemeDelete soft-deletes a scheme and increments its audited version once.
// @Summary 删除报表方案
// @Tags warehouse-scheme
// @Accept json
// @Produce json
// @Param body body request.DeleteInput true "方案ID和可选版本"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/schemes/delete [delete]
func (a ReconciliationAPI) SchemeDelete(c *gin.Context) {
	var in req.DeleteInput
	if !bind(c, &in) {
		return
	}
	if err := a.schemes.Delete(c.Request.Context(), utils.GetUserID(c), utils.GetUserAuthorityId(c), in); err != nil {
		a.fail(c, err)
		return
	}
	response.Ok(c)
}
