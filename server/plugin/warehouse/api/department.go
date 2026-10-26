package api

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/service"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
)

type DepartmentAPI struct{ service *service.DepartmentService }

func DefaultDepartmentAPI() DepartmentAPI {
	return DepartmentAPI{service: service.DefaultDepartmentService()}
}

// Create 创建部门
// @Summary 创建部门资料
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.CreateDepartmentInput true "名称及多个助记符"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/departments/create [post]
func (a DepartmentAPI) Create(c *gin.Context) {
	var in req.CreateDepartmentInput
	if !bind(c, &in) {
		return
	}
	item, err := a.service.Create(c, utils.GetUserID(c), in)
	if err != nil {
		MasterAPI{}.fail(c, err)
		return
	}
	response.OkWithDetailed(item, "操作成功", c)
}

// Update 更新部门
// @Summary 更新部门资料，不改写历史送货单快照
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.UpdateDepartmentInput true "部门及版本"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/departments/update [put]
func (a DepartmentAPI) Update(c *gin.Context) {
	var in req.UpdateDepartmentInput
	if !bind(c, &in) {
		return
	}
	item, err := a.service.Update(c, utils.GetUserID(c), in)
	if err != nil {
		MasterAPI{}.fail(c, err)
		return
	}
	response.OkWithDetailed(item, "操作成功", c)
}

// Delete 删除部门
// @Summary 软删除部门资料，保留历史快照
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.DeleteInput true "部门及版本"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/departments/delete [delete]
func (a DepartmentAPI) Delete(c *gin.Context) {
	var in req.DeleteInput
	if !bind(c, &in) {
		return
	}
	if err := a.service.Delete(c, utils.GetUserID(c), in); err != nil {
		MasterAPI{}.fail(c, err)
		return
	}
	response.Ok(c)
}

// List 查询部门
// @Summary 分页查询未删除部门，匹配名称或任一助记符
// @Tags warehouse-master
// @Produce json
// @Param query query request.DepartmentListQuery false "分页和关键字"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/departments/list [get]
func (a DepartmentAPI) List(c *gin.Context) {
	var q req.DepartmentListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		MasterAPI{}.fail(c, service.NewDomainError(service.ValidationFailed, "请求参数无效").WithField("query"))
		return
	}
	items, err := a.service.List(c, q)
	if err != nil {
		MasterAPI{}.fail(c, err)
		return
	}
	response.OkWithDetailed(items, "操作成功", c)
}

// Detail 查询部门详情
// @Summary 查询未删除部门详情及审计版本
// @Tags warehouse-master
// @Produce json
// @Param id query integer true "部门ID"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/departments/detail [get]
func (a DepartmentAPI) Detail(c *gin.Context) {
	id, ok := detailID(c)
	if !ok {
		return
	}
	item, err := a.service.Detail(c, id)
	if err != nil {
		MasterAPI{}.fail(c, err)
		return
	}
	response.OkWithDetailed(item, "操作成功", c)
}
