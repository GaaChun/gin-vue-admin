package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	resp "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/response"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MaterialDetail exposes the aggregate; clients never supply its audit fields or binding IDs.
type MaterialDetail struct {
	model.Material
	Units    []model.MaterialUnit `json:"units"`
	ImageURL string               `json:"imageUrl,omitempty"`
}

type materialUnitCommand struct {
	unitID  uint
	factor  decimal.Decimal
	isBase  bool
	enabled bool
}

func (s *MasterService) CreateMaterial(ctx context.Context, actor uint, in req.CreateMaterialInput) (*MaterialDetail, error) {
	item, units, err := prepareMaterialInput(in.MaterialInput)
	if err != nil {
		return nil, err
	}
	ApplyCreateAudit(&item.BusinessModel, actor, time.Now())
	result := &MaterialDetail{}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockAndValidateMaterialReferences(tx, item.BrandID, item.CategoryID, item.ImageFileID, units); err != nil {
			return err
		}
		if err := materialCodeConflict(tx, 0, item.Code); err != nil {
			return err
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		bindings := materialBindings(item.ID, actor, units, nil)
		if err := tx.Create(&bindings).Error; err != nil {
			return err
		}
		result.Material = item
		result.Units = bindings
		return nil
	})
	return result, dbError(err)
}

func (s *MasterService) UpdateMaterial(ctx context.Context, actor uint, in req.UpdateMaterialInput) (*MaterialDetail, error) {
	item, units, err := prepareMaterialInput(in.MaterialInput)
	if err != nil {
		return nil, err
	}
	result := &MaterialDetail{}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.Material
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, in.ID).Error; err != nil {
			return err
		}
		if err := CheckVersion(current.Version, in.Version); err != nil {
			return err
		}
		if err := s.lockAndValidateMaterialReferences(tx, item.BrandID, item.CategoryID, item.ImageFileID, units); err != nil {
			return err
		}
		if err := materialCodeConflict(tx, current.ID, item.Code); err != nil {
			return err
		}
		var existing []model.MaterialUnit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("material_id = ?", current.ID).Order("id").Find(&existing).Error; err != nil {
			return err
		}
		if err := s.replaceMaterialUnits(tx, current.ID, actor, existing, units); err != nil {
			return err
		}
		current.Name = item.Name
		current.Mnemonic = item.Mnemonic
		current.Code = item.Code
		current.BrandID = item.BrandID
		current.CategoryID = item.CategoryID
		current.Specification = item.Specification
		current.Alias = item.Alias
		current.ImageFileID = item.ImageFileID
		ApplyUpdateAudit(&current.BusinessModel, actor)
		if err := tx.Model(&current).Select("name", "mnemonic", "code", "brand_id", "category_id", "specification", "alias", "image_file_id", "updated_by", "version").Updates(&current).Error; err != nil {
			return err
		}
		var persisted []model.MaterialUnit
		if err := tx.Where("material_id = ?", current.ID).Order("id").Find(&persisted).Error; err != nil {
			return err
		}
		result.Material = current
		result.Units = persisted
		return nil
	})
	if err == gorm.ErrRecordNotFound {
		return nil, notFound()
	}
	return result, dbError(err)
}

func (s *MasterService) DeleteMaterial(ctx context.Context, actor uint, in req.DeleteInput) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item model.Material
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, in.ID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return notFound()
			}
			return err
		}
		if err := CheckVersion(item.Version, in.Version); err != nil {
			return err
		}
		var bindings []model.MaterialUnit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("material_id = ?", item.ID).Order("id").Find(&bindings).Error; err != nil {
			return err
		}
		var refs int64
		if err := tx.Model(&model.DeliveryDetail{}).Where("material_id = ?", item.ID).Count(&refs).Error; err != nil {
			return err
		}
		if refs > 0 {
			return NewDomainError(ReferenceInUse, "物料正在被送货明细引用")
		}
		// The material and its bindings are one aggregate; public Unit records remain untouched.
		for i := range bindings {
			ApplyDeleteAudit(&bindings[i].BusinessModel, actor)
			if err := tx.Model(&bindings[i]).Select("updated_by", "version", "deleted_by", "is_deleted").Updates(&bindings[i]).Error; err != nil {
				return err
			}
			if err := tx.Delete(&bindings[i]).Error; err != nil {
				return err
			}
		}
		ApplyDeleteAudit(&item.BusinessModel, actor)
		if err := tx.Model(&item).Select("updated_by", "version", "deleted_by", "is_deleted").Updates(&item).Error; err != nil {
			return err
		}
		return tx.Delete(&item).Error
	})
}

func (s *MasterService) ListMaterials(ctx context.Context, q req.MaterialListQuery) (resp.Page[model.Material], error) {
	q.PageQuery = q.PageQuery.WithDefaults()
	var result resp.Page[model.Material]
	db := s.db.WithContext(ctx).Model(&model.Material{})
	if q.BrandID != nil {
		db = db.Where("brand_id = ?", *q.BrandID)
	}
	if q.CategoryID != nil {
		ids, err := s.CategoryDescendants(ctx, *q.CategoryID)
		if err != nil {
			return result, err
		}
		db = db.Where("category_id IN ?", ids)
	}
	if keyword := strings.TrimSpace(q.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("code LIKE ? OR name LIKE ? OR mnemonic LIKE ?", like, like, like)
	}
	orderColumns := auditSortColumns("wh_materials")
	for _, field := range []string{"code", "name", "mnemonic", "specification", "alias"} {
		orderColumns[field] = "wh_materials." + field
	}
	orderColumns["brandName"] = "(SELECT name FROM wh_brands WHERE wh_brands.id = wh_materials.brand_id AND wh_brands.deleted_at IS NULL)"
	orderColumns["categoryName"] = "(SELECT name FROM wh_categories WHERE wh_categories.id = wh_materials.category_id AND wh_categories.deleted_at IS NULL)"
	order, err := resolveMasterOrder(q.SortQuery, orderColumns)
	if err != nil {
		return result, err
	}
	err = db.Count(&result.Total).Error
	if err == nil {
		err = db.Order(order).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&result.List).Error
	}
	result.Page = q.Page
	result.PageSize = q.PageSize
	return result, dbError(err)
}

func (s *MasterService) MaterialDetail(ctx context.Context, id uint) (*MaterialDetail, error) {
	var result MaterialDetail
	if err := s.db.WithContext(ctx).First(&result.Material, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, notFound()
		}
		return nil, dbError(err)
	}
	if err := s.db.WithContext(ctx).Where("material_id = ?", id).Order("id").Find(&result.Units).Error; err != nil {
		return nil, dbError(err)
	}
	if result.ImageFileID != nil {
		var file example.ExaFileUploadAndDownload
		err := s.db.WithContext(ctx).First(&file, *result.ImageFileID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, dbError(err)
		}
		result.ImageURL = file.Url
	}
	return &result, nil
}

func prepareMaterialInput(in req.MaterialInput) (model.Material, []materialUnitCommand, error) {
	name, err := requiredLimited(in.Name, "name", 200)
	if err != nil {
		return model.Material{}, nil, err
	}
	mnemonic, err := requiredLimited(in.Mnemonic, "mnemonic", 100)
	if err != nil {
		return model.Material{}, nil, err
	}
	code, err := requiredLimited(in.Code, "code", 64)
	if err != nil {
		return model.Material{}, nil, err
	}
	if in.Specification != nil && utf8RuneCount(*in.Specification) > 255 {
		return model.Material{}, nil, domainError(ValidationFailed, "长度超出限制", "specification")
	}
	if in.Alias != nil && utf8RuneCount(*in.Alias) > 200 {
		return model.Material{}, nil, domainError(ValidationFailed, "长度超出限制", "alias")
	}
	units, err := prepareMaterialUnits(in.Units)
	if err != nil {
		return model.Material{}, nil, err
	}
	return model.Material{Name: name, Mnemonic: mnemonic, Code: code, BrandID: in.BrandID, CategoryID: in.CategoryID, Specification: in.Specification, Alias: in.Alias, ImageFileID: in.ImageFileID}, units, nil
}

func utf8RuneCount(value string) int { return len([]rune(value)) }

func prepareMaterialUnits(inputs []req.MaterialUnitInput) ([]materialUnitCommand, error) {
	if len(inputs) == 0 {
		return nil, domainError(ValidationFailed, "至少需要一个单位", "units")
	}
	result := make([]materialUnitCommand, 0, len(inputs))
	seen := make(map[uint]struct{}, len(inputs))
	baseCount := 0
	for _, input := range inputs {
		if input.UnitID == 0 {
			return nil, domainError(ValidationFailed, "单位无效", "units")
		}
		if _, exists := seen[input.UnitID]; exists {
			return nil, domainError(ValidationFailed, "同一单位不能重复绑定", "units")
		}
		seen[input.UnitID] = struct{}{}
		factor, err := ParseDecimal(input.Factor, FactorRule, "factor")
		if err != nil {
			return nil, err
		}
		if input.IsBase {
			if !input.Enabled || !factor.Equal(decimal.NewFromInt(1)) {
				return nil, domainError(ValidationFailed, "基本单位必须启用且系数为1", "units")
			}
			baseCount++
		}
		result = append(result, materialUnitCommand{unitID: input.UnitID, factor: factor, isBase: input.IsBase, enabled: input.Enabled})
	}
	if baseCount != 1 {
		return nil, domainError(ValidationFailed, "必须且只能有一个启用基本单位", "units")
	}
	return result, nil
}

func (s *MasterService) lockAndValidateMaterialReferences(tx *gorm.DB, brandID, categoryID uint, imageFileID *uint, units []materialUnitCommand) error {
	// Lock referenced stable IDs in ascending order, then recheck activity before writing references.
	unitIDs := make([]uint, 0, len(units))
	for _, unit := range units {
		unitIDs = append(unitIDs, unit.unitID)
	}
	sort.Slice(unitIDs, func(i, j int) bool { return unitIDs[i] < unitIDs[j] })
	if brandID != 0 {
		var brands []model.Brand
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", brandID).Order("id").Find(&brands).Error; err != nil {
			return err
		}
		if len(brands) != 1 {
			return domainError(ValidationFailed, "客户不存在或已删除", "brandId")
		}
	}
	if categoryID != 0 {
		var categories []model.Category
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", categoryID).Order("id").Find(&categories).Error; err != nil {
			return err
		}
		if len(categories) != 1 {
			return domainError(ValidationFailed, "类别不存在或已删除", "categoryId")
		}
	}
	var activeUnits []model.Unit
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", unitIDs).Order("id").Find(&activeUnits).Error; err != nil {
		return err
	}
	if len(activeUnits) != len(unitIDs) {
		return domainError(ValidationFailed, "单位不存在或已删除", "units")
	}
	if imageFileID != nil {
		var image model.MaterialImage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("file_id = ?", *imageFileID).First(&image).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return domainError(ValidationFailed, "图片未通过物料图片校验", "imageFileId")
			}
			return err
		}
	}
	return nil
}

func materialCodeConflict(tx *gorm.DB, excludeID uint, code string) error {
	var count int64
	if err := tx.Model(&model.Material{}).Where("id <> ? AND LOWER(TRIM(code)) = ?", excludeID, NormalizeUniqueKey(code)).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return duplicate("code")
	}
	return nil
}

func materialBindings(materialID, actor uint, commands []materialUnitCommand, existing map[uint]model.MaterialUnit) []model.MaterialUnit {
	bindings := make([]model.MaterialUnit, 0, len(commands))
	for _, command := range commands {
		binding, found := existing[command.unitID]
		if !found {
			binding = model.MaterialUnit{MaterialID: materialID, UnitID: command.unitID}
			ApplyCreateAudit(&binding.BusinessModel, actor, time.Now())
		}
		binding.Factor, binding.IsBase, binding.Enabled = command.factor, command.isBase, command.enabled
		bindings = append(bindings, binding)
	}
	return bindings
}

func (s *MasterService) replaceMaterialUnits(tx *gorm.DB, materialID, actor uint, existing []model.MaterialUnit, commands []materialUnitCommand) error {
	byUnit := make(map[uint]model.MaterialUnit, len(existing))
	for _, binding := range existing {
		byUnit[binding.UnitID] = binding
	}
	oldBase, newBase := uint(0), uint(0)
	for _, binding := range existing {
		if binding.IsBase && binding.Enabled {
			oldBase = binding.UnitID
		}
	}
	for _, command := range commands {
		if command.isBase {
			newBase = command.unitID
		}
	}
	if oldBase != 0 && oldBase != newBase {
		// Clear first so the final complete-set update never transiently violates the one-base key.
		if err := tx.Model(&model.MaterialUnit{}).Where("material_id = ? AND unit_id = ?", materialID, oldBase).Update("is_base", false).Error; err != nil {
			return err
		}
	}
	wanted := make(map[uint]materialUnitCommand, len(commands))
	for _, command := range commands {
		wanted[command.unitID] = command
	}
	for _, binding := range existing {
		if _, keep := wanted[binding.UnitID]; keep {
			continue
		}
		var count int64
		if err := tx.Model(&model.DeliveryDetail{}).Where("material_unit_id = ?", binding.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return NewDomainError(ReferenceInUse, "被送货明细引用的单位绑定不能删除")
		}
		ApplyDeleteAudit(&binding.BusinessModel, actor)
		if err := tx.Model(&binding).Select("updated_by", "version", "deleted_by", "is_deleted").Updates(&binding).Error; err != nil {
			return err
		}
		if err := tx.Delete(&binding).Error; err != nil {
			return err
		}
	}
	for _, binding := range materialBindings(materialID, actor, commands, byUnit) {
		if binding.ID == 0 {
			if err := tx.Create(&binding).Error; err != nil {
				return err
			}
			continue
		}
		ApplyUpdateAudit(&binding.BusinessModel, actor)
		if err := tx.Model(&binding).Select("factor", "is_base", "enabled", "updated_by", "version").Updates(&binding).Error; err != nil {
			return err
		}
	}
	return nil
}
