package api

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/service"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
)

type DeliveryAPI struct{ service *service.DeliveryService }

func NewDeliveryAPI(s *service.DeliveryService) DeliveryAPI { return DeliveryAPI{service: s} }
func DefaultDeliveryAPI() DeliveryAPI                       { return NewDeliveryAPI(service.DefaultDeliveryService()) }

// ReserveDeliveryNumber 预占正式送货单号
// @Summary 预占正式送货单号
// @Tags warehouse-delivery
// @Produce json
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/delivery/reserveNumber [post]
func (a DeliveryAPI) ReserveNumber(c *gin.Context) {
	v, err := a.service.ReserveNumber(c, utils.GetUserID(c))
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// CreateDelivery 创建送货单
// @Summary 创建送货单，可原子保存并审核
// @Tags warehouse-delivery
// @Accept json
// @Produce json
// @Param body body request.CreateDeliveryInput true "送货单"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/delivery/create [post]
func (a DeliveryAPI) Create(c *gin.Context) {
	var in req.CreateDeliveryInput
	if !bind(c, &in) {
		return
	}
	if in.Review && !a.requireReviewPermission(c) {
		return
	}
	v, err := a.service.Create(c, utils.GetUserID(c), utils.GetUserName(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// UpdateDelivery 更新未审核送货单
// @Summary 更新未审核送货单
// @Tags warehouse-delivery
// @Accept json
// @Produce json
// @Param body body request.UpdateDeliveryInput true "送货单"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/delivery/update [put]
func (a DeliveryAPI) Update(c *gin.Context) {
	var in req.UpdateDeliveryInput
	if !bind(c, &in) {
		return
	}
	if in.Review && !a.requireReviewPermission(c) {
		return
	}
	v, err := a.service.Update(c, utils.GetUserID(c), utils.GetUserName(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// DeleteDelivery 删除未审核送货单
// @Summary 软删除未审核送货单
// @Tags warehouse-delivery
// @Accept json
// @Produce json
// @Param body body request.DeleteInput true "删除参数"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/delivery/delete [delete]
func (a DeliveryAPI) Delete(c *gin.Context) {
	var in req.DeleteInput
	if !bind(c, &in) {
		return
	}
	if err := a.service.Delete(c, utils.GetUserID(c), in); err != nil {
		a.fail(c, err)
		return
	}
	response.Ok(c)
}

// ReviewDelivery 审核送货单
// @Summary 审核送货单
// @Tags warehouse-delivery
// @Accept json
// @Produce json
// @Param body body request.ReviewDeliveryInput true "审核参数"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/delivery/review [post]
func (a DeliveryAPI) Review(c *gin.Context) {
	var in req.ReviewDeliveryInput
	if !bind(c, &in) {
		return
	}
	v, err := a.service.Review(c, utils.GetUserID(c), utils.GetUserName(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// UnreviewDelivery 反审核送货单
// @Summary 反审核送货单
// @Tags warehouse-delivery
// @Accept json
// @Produce json
// @Param body body request.UnreviewDeliveryInput true "反审核参数"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/delivery/unreview [post]
func (a DeliveryAPI) Unreview(c *gin.Context) {
	var in req.UnreviewDeliveryInput
	if !bind(c, &in) {
		return
	}
	v, err := a.service.Unreview(c, utils.GetUserID(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// ReReviewDelivery 复审已审核送货单
// @Summary 复审已审核送货单
// @Tags warehouse-delivery
// @Accept json
// @Produce json
// @Param body body request.ReReviewDeliveryInput true "复审参数"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/delivery/reReview [post]
func (a DeliveryAPI) ReReview(c *gin.Context) {
	var in req.ReReviewDeliveryInput
	if !bind(c, &in) {
		return
	}
	v, err := a.service.ReReview(c, utils.GetUserID(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// CancelReReviewDelivery 取消复审送货单
// @Summary 取消复审送货单
// @Tags warehouse-delivery
// @Accept json
// @Produce json
// @Param body body request.CancelReReviewDeliveryInput true "取消复审参数"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/delivery/cancelReReview [post]
func (a DeliveryAPI) CancelReReview(c *gin.Context) {
	var in req.CancelReReviewDeliveryInput
	if !bind(c, &in) {
		return
	}
	v, err := a.service.CancelReReview(c, utils.GetUserID(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

func (a DeliveryAPI) requireReviewPermission(c *gin.Context) bool {
	if service.CanReviewDelivery(utils.GetUserAuthorityId(c)) {
		return true
	}
	a.fail(c, service.NewDomainError(service.Forbidden, "缺少送货单审核权限"))
	return false
}

// DeliveryDetail 查询送货单详情
// @Summary 查询送货单详情
// @Tags warehouse-delivery
// @Produce json
// @Param id query int true "送货单ID"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/delivery/detail [get]
func (a DeliveryAPI) Detail(c *gin.Context) {
	id, ok := detailID(c)
	if !ok {
		return
	}
	v, err := a.service.Detail(c, id)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// ListDeliveries 查询送货单列表
// @Summary 查询送货单列表
// @Tags warehouse-delivery
// @Produce json
// @Param query query request.DeliveryListQuery false "筛选"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/delivery/list [get]
func (a DeliveryAPI) List(c *gin.Context) {
	var query req.DeliveryListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		a.fail(c, service.NewDomainError(service.ValidationFailed, "请求参数无效").WithField("query"))
		return
	}
	v, err := a.service.List(c, query)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// Departments lists historical department values for free-text suggestions.
// @Summary 查询历史部门候选
// @Tags warehouse-delivery
// @Produce json
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/delivery/departments [get]
func (a DeliveryAPI) Departments(c *gin.Context) {
	v, err := a.service.Departments(c)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

func (a DeliveryAPI) fail(c *gin.Context, err error) { MasterAPI{}.fail(c, err) }
