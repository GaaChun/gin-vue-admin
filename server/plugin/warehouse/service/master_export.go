package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"gorm.io/gorm"
)

func (s *MasterService) ExportFilteredMasterExcel(ctx context.Context, kind string, filters req.MasterExportQuery) ([]byte, error) {
	var output []byte
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		state := NewMasterExcelState()
		if err := state.load(tx); err != nil {
			return err
		}
		query := tx
		switch kind {
		case "units":
			query = query.Model(&model.Unit{})
			if name := strings.TrimSpace(filters.Name); name != "" {
				query = query.Where("name LIKE ?", "%"+name+"%")
			}
		case "brands":
			query = query.Model(&model.Brand{})
		case "categories":
			query = query.Model(&model.Category{})
			if filters.ParentID != nil {
				query = query.Where("parent_id = ?", *filters.ParentID)
			}
		case "materials":
			query = query.Model(&model.Material{})
			if filters.BrandID != nil {
				query = query.Where("brand_id = ?", *filters.BrandID)
			}
			if filters.CategoryID != nil {
				ids, err := NewMasterService(tx).CategoryDescendants(ctx, *filters.CategoryID)
				if err != nil {
					return err
				}
				query = query.Where("category_id IN ?", ids)
			}
		default:
			return domainError(ValidationFailed, "未知资料导出类型", "kind")
		}
		if keyword := strings.TrimSpace(filters.Keyword); keyword != "" && kind != "units" {
			like := "%" + keyword + "%"
			query = query.Where("name LIKE ? OR code LIKE ? OR mnemonic LIKE ?", like, like, like)
		}
		var ids []uint
		if err := query.Order("id").Pluck("id", &ids).Error; err != nil {
			return err
		}
		selected := make(map[uint]bool, len(ids))
		for _, id := range ids {
			selected[id] = true
		}
		switch kind {
		case "units":
			for id := range state.units {
				if !selected[id] {
					delete(state.units, id)
				}
			}
		case "brands":
			for id := range state.brands {
				if !selected[id] {
					delete(state.brands, id)
				}
			}
		case "materials":
			for id := range state.materials {
				if !selected[id] {
					delete(state.materials, id)
				}
			}
		case "categories":
			rows := make([][]string, 0, len(ids))
			for _, id := range ids {
				item := state.categories[id]
				parent := "ROOT"
				if item.ParentID != 0 {
					parent = state.categories[item.ParentID].Code
				}
				rows = append(rows, []string{item.Code, item.Name, item.Mnemonic, parent, strconv.FormatUint(item.Version, 10)})
			}
			var err error
			output, err = writeMasterWorkbook(kind, rows, nil)
			return err
		}
		var err error
		output, err = state.Export(kind)
		return err
	})
	return output, dbError(err)
}
