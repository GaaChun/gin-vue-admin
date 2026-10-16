package api

import (
	"errors"
	"io"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/service"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
)

type MasterAPI struct{ service *service.MasterService }

func NewMasterAPI(s *service.MasterService) MasterAPI { return MasterAPI{service: s} }
func DefaultMasterAPI() MasterAPI                     { return NewMasterAPI(service.DefaultMasterService()) }
func (a MasterAPI) fail(c *gin.Context, err error) {
	var domain *service.DomainError
	if errors.As(err, &domain) {
		response.FailWithDetailed(domain.BusinessError(), domain.Message, c)
		return
	}
	response.FailWithDetailed(service.NewDomainError(service.ValidationFailed, "请求处理失败").BusinessError(), "请求处理失败", c)
}
func bind(c *gin.Context, value interface{}) bool {
	if err := c.ShouldBindJSON(value); err != nil {
		response.FailWithDetailed(service.NewDomainError(service.ValidationFailed, "请求参数无效").WithField("body").BusinessError(), "请求参数无效", c)
		return false
	}
	return true
}
func detailID(c *gin.Context) (uint, bool) {
	var in req.VersionedID
	if err := c.ShouldBindQuery(&in); err != nil {
		response.FailWithDetailed(service.NewDomainError(service.ValidationFailed, "请求参数无效").WithField("id").BusinessError(), "请求参数无效", c)
		return 0, false
	}
	return in.ID, true
}

// CreateUnit 创建单位
// @Summary 创建单位
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.CreateUnitInput true "单位"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/units/create [post]
func (a MasterAPI) CreateUnit(c *gin.Context) {
	var in req.CreateUnitInput
	if !bind(c, &in) {
		return
	}
	v, e := a.service.CreateUnit(c, utils.GetUserID(c), in)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// UpdateUnit 更新单位
// @Summary 更新单位
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.UpdateUnitInput true "单位"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/units/update [put]
func (a MasterAPI) UpdateUnit(c *gin.Context) {
	var in req.UpdateUnitInput
	if !bind(c, &in) {
		return
	}
	v, e := a.service.UpdateUnit(c, utils.GetUserID(c), in)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// DeleteUnit 删除单位
// @Summary 软删除单位
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.DeleteInput true "删除参数"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/units/delete [delete]
func (a MasterAPI) DeleteUnit(c *gin.Context) {
	var in req.DeleteInput
	if !bind(c, &in) {
		return
	}
	if e := a.service.DeleteUnit(c, utils.GetUserID(c), in); e != nil {
		a.fail(c, e)
		return
	}
	response.Ok(c)
}

// ListUnits 查询单位列表
// @Summary 查询单位列表
// @Tags warehouse-master
// @Produce json
// @Param query query request.UnitListQuery false "筛选"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/units/list [get]
func (a MasterAPI) ListUnits(c *gin.Context) {
	var q req.UnitListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		a.fail(c, service.NewDomainError(service.ValidationFailed, "请求参数无效").WithField("query"))
		return
	}
	v, e := a.service.ListUnits(c, q)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// UnitDetail 查询单位详情
// @Summary 查询单位详情
// @Tags warehouse-master
// @Produce json
// @Param id query integer true "ID"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/units/detail [get]
func (a MasterAPI) UnitDetail(c *gin.Context) {
	id, ok := detailID(c)
	if !ok {
		return
	}
	v, e := a.service.UnitDetail(c, id)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// CreateBrand 创建品牌
// @Summary 创建品牌
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.CreateBrandInput true "品牌"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/brands/create [post]
func (a MasterAPI) CreateBrand(c *gin.Context) {
	var in req.CreateBrandInput
	if !bind(c, &in) {
		return
	}
	v, e := a.service.CreateBrand(c, utils.GetUserID(c), in)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// UpdateBrand 更新品牌
// @Summary 更新品牌
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.UpdateBrandInput true "品牌"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/brands/update [put]
func (a MasterAPI) UpdateBrand(c *gin.Context) {
	var in req.UpdateBrandInput
	if !bind(c, &in) {
		return
	}
	v, e := a.service.UpdateBrand(c, utils.GetUserID(c), in)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// DeleteBrand 删除品牌
// @Summary 软删除品牌
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.DeleteInput true "删除参数"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/brands/delete [delete]
func (a MasterAPI) DeleteBrand(c *gin.Context) {
	var in req.DeleteInput
	if !bind(c, &in) {
		return
	}
	if e := a.service.DeleteBrand(c, utils.GetUserID(c), in); e != nil {
		a.fail(c, e)
		return
	}
	response.Ok(c)
}

// ListBrands 查询品牌列表
// @Summary 查询品牌列表
// @Tags warehouse-master
// @Produce json
// @Param query query request.BrandListQuery false "筛选"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/brands/list [get]
func (a MasterAPI) ListBrands(c *gin.Context) {
	var q req.BrandListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		a.fail(c, service.NewDomainError(service.ValidationFailed, "请求参数无效").WithField("query"))
		return
	}
	v, e := a.service.ListBrands(c, q)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// BrandDetail 查询品牌详情
// @Summary 查询品牌详情
// @Tags warehouse-master
// @Produce json
// @Param id query integer true "ID"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/brands/detail [get]
func (a MasterAPI) BrandDetail(c *gin.Context) {
	id, ok := detailID(c)
	if !ok {
		return
	}
	v, e := a.service.BrandDetail(c, id)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// CreateCategory 创建类别
// @Summary 创建类别
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.CreateCategoryInput true "类别"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/categories/create [post]
func (a MasterAPI) CreateCategory(c *gin.Context) {
	var in req.CreateCategoryInput
	if !bind(c, &in) {
		return
	}
	v, e := a.service.CreateCategory(c, utils.GetUserID(c), in)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// UpdateCategory 更新或移动类别
// @Summary 更新或移动类别
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.UpdateCategoryInput true "类别"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/categories/update [put]
func (a MasterAPI) UpdateCategory(c *gin.Context) {
	var in req.UpdateCategoryInput
	if !bind(c, &in) {
		return
	}
	v, e := a.service.UpdateCategory(c, utils.GetUserID(c), in)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// DeleteCategory 删除类别
// @Summary 软删除类别
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.DeleteInput true "删除参数"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/categories/delete [delete]
func (a MasterAPI) DeleteCategory(c *gin.Context) {
	var in req.DeleteInput
	if !bind(c, &in) {
		return
	}
	if e := a.service.DeleteCategory(c, utils.GetUserID(c), in); e != nil {
		a.fail(c, e)
		return
	}
	response.Ok(c)
}

// ListCategories 查询类别列表
// @Summary 查询类别列表
// @Tags warehouse-master
// @Produce json
// @Param query query request.CategoryListQuery false "筛选"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/categories/list [get]
func (a MasterAPI) ListCategories(c *gin.Context) {
	var q req.CategoryListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		a.fail(c, service.NewDomainError(service.ValidationFailed, "请求参数无效").WithField("query"))
		return
	}
	v, e := a.service.ListCategories(c, q)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// CategoryDetail 查询类别详情
// @Summary 查询类别详情
// @Tags warehouse-master
// @Produce json
// @Param id query integer true "ID"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/categories/detail [get]
func (a MasterAPI) CategoryDetail(c *gin.Context) {
	id, ok := detailID(c)
	if !ok {
		return
	}
	v, e := a.service.CategoryDetail(c, id)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// CategoryTree 查询类别树
// @Summary 查询类别树
// @Tags warehouse-master
// @Produce json
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/categories/tree [get]
func (a MasterAPI) CategoryTree(c *gin.Context) {
	v, e := a.service.CategoryTree(c)
	if e != nil {
		a.fail(c, e)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// SuggestMnemonic 建议助记符
// @Summary 根据名称生成助记符建议
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.MnemonicRequest true "名称"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/mnemonic [post]
func (a MasterAPI) SuggestMnemonic(c *gin.Context) {
	var in req.MnemonicRequest
	if !bind(c, &in) {
		return
	}
	response.OkWithDetailed(gin.H{"mnemonic": service.SuggestMnemonic(in.Name)}, "操作成功", c)
}

// CreateMaterial 创建物料及完整单位集合
// @Summary 创建物料
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.CreateMaterialInput true "物料"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/materials/create [post]
func (a MasterAPI) CreateMaterial(c *gin.Context) {
	var in req.CreateMaterialInput
	if !bind(c, &in) {
		return
	}
	v, err := a.service.CreateMaterial(c, utils.GetUserID(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// UpdateMaterial 更新物料及完整单位集合
// @Summary 更新物料
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.UpdateMaterialInput true "物料"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/materials/update [put]
func (a MasterAPI) UpdateMaterial(c *gin.Context) {
	var in req.UpdateMaterialInput
	if !bind(c, &in) {
		return
	}
	v, err := a.service.UpdateMaterial(c, utils.GetUserID(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// DeleteMaterial 软删除物料及未被外部引用的单位绑定
// @Summary 删除物料
// @Tags warehouse-master
// @Accept json
// @Produce json
// @Param body body request.DeleteInput true "删除参数"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/materials/delete [delete]
func (a MasterAPI) DeleteMaterial(c *gin.Context) {
	var in req.DeleteInput
	if !bind(c, &in) {
		return
	}
	if err := a.service.DeleteMaterial(c, utils.GetUserID(c), in); err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithMessage("操作成功", c)
}

// ListMaterials 查询物料列表
// @Summary 查询物料列表
// @Tags warehouse-master
// @Produce json
// @Param query query request.MaterialListQuery false "筛选"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/materials/list [get]
func (a MasterAPI) ListMaterials(c *gin.Context) {
	var q req.MaterialListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		a.fail(c, service.NewDomainError(service.ValidationFailed, "请求参数无效").WithField("query"))
		return
	}
	v, err := a.service.ListMaterials(c, q)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// MaterialDetail 查询物料及其单位绑定
// @Summary 查询物料详情
// @Tags warehouse-master
// @Produce json
// @Param id query integer true "ID"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/materials/detail [get]
func (a MasterAPI) MaterialDetail(c *gin.Context) {
	id, ok := detailID(c)
	if !ok {
		return
	}
	v, err := a.service.MaterialDetail(c, id)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

// UploadMaterialImage 严格校验并登记物料图片
// @Summary 上传物料图片
// @Tags warehouse-master
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "jpg/jpeg/png/bmp，最大1MB"
// @Success 200 {object} response.Response
// @Security ApiKeyAuth
// @Router /warehouse/master/materials/uploadImage [post]
func (a MasterAPI) UploadMaterialImage(c *gin.Context) {
	header, err := c.FormFile("file")
	if err != nil {
		a.fail(c, service.NewDomainError(service.ValidationFailed, "缺少图片文件").WithField("file"))
		return
	}
	if header.Size > service.MaterialImageMaxBytes {
		a.fail(c, service.NewDomainError(service.ValidationFailed, "图片大小必须小于或等于1MB").WithField("file"))
		return
	}
	file, err := header.Open()
	if err != nil {
		a.fail(c, service.NewDomainError(service.ValidationFailed, "读取图片失败").WithField("file"))
		return
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, service.MaterialImageMaxBytes+1))
	if err != nil {
		a.fail(c, service.NewDomainError(service.ValidationFailed, "读取图片失败").WithField("file"))
		return
	}
	v, err := a.service.UploadMaterialImage(c, utils.GetUserID(c), header.Filename, contents)
	if err != nil {
		a.fail(c, err)
		return
	}
	response.OkWithDetailed(v, "操作成功", c)
}

func (a MasterAPI) MasterTemplate(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		contents, err := service.MasterTemplate(kind)
		if err != nil {
			a.fail(c, err)
			return
		}
		c.Header("Content-Disposition", "attachment; filename="+kind+"-template.xlsx")
		c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", contents)
	}
}

func (a MasterAPI) MasterImport(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header, err := c.FormFile("file")
		if err != nil {
			a.fail(c, service.NewDomainError(service.ValidationFailed, "缺少Excel文件").WithField("file"))
			return
		}
		file, err := header.Open()
		if err != nil {
			a.fail(c, service.NewDomainError(service.ValidationFailed, "读取Excel失败").WithField("file"))
			return
		}
		defer file.Close()
		contents, err := io.ReadAll(file)
		if err != nil {
			a.fail(c, service.NewDomainError(service.ValidationFailed, "读取Excel失败").WithField("file"))
			return
		}
		result, err := a.service.ImportMasterExcel(c, utils.GetUserID(c), kind, contents)
		if err != nil {
			a.fail(c, err)
			return
		}
		response.OkWithDetailed(result, "操作成功", c)
	}
}

func (a MasterAPI) MasterExport(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var filters req.MasterExportQuery
		if err := c.ShouldBindQuery(&filters); err != nil {
			a.fail(c, service.NewDomainError(service.ValidationFailed, "导出筛选无效"))
			return
		}
		contents, err := a.service.ExportFilteredMasterExcel(c, kind, filters)
		if err != nil {
			a.fail(c, err)
			return
		}
		c.Header("Content-Disposition", "attachment; filename="+kind+".xlsx")
		c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", contents)
	}
}
