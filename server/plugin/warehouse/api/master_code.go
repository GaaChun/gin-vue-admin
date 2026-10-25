package api

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
)

// ReserveCode 预占基础资料编号
// @Summary 预占可手动修改的基础资料流水编号
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.ReserveMasterCodeInput true "资料类型"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/reserveCode [post]
func (a MasterAPI) ReserveCode(c *gin.Context) {
	var in req.ReserveMasterCodeInput
	if !bind(c, &in) {
		return
	}
	value, err := a.service.ReserveCode(c, utils.GetUserID(c), in.Resource)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(value, "操作成功", c)
}
