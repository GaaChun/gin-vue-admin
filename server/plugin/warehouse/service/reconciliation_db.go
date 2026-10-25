package service

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ReconciliationService reads only persisted delivery snapshots, never current masters.
type ReconciliationService struct{ db *gorm.DB }

func NewReconciliationService(db *gorm.DB) *ReconciliationService {
	return &ReconciliationService{db: db}
}
func DefaultReconciliationService() *ReconciliationService {
	return NewReconciliationService(global.GVA_DB)
}

func (s *ReconciliationService) Query(ctx context.Context, query req.ReconciliationQuery, paginate bool) (ReportProjection, error) {
	if _, err := validateColumns(query.Columns); err != nil {
		return ReportProjection{}, err
	}
	if query.Page < 0 || query.PageSize < 0 || query.PageSize > 100 {
		return ReportProjection{}, domainError(ValidationFailed, "分页参数无效", "page")
	}
	f := query.Filters
	statuses, err := reconciliationStatuses(f.Statuses)
	if err != nil {
		return ReportProjection{}, err
	}
	if f.DeliveredFrom != nil && f.DeliveredTo != nil && !f.DeliveredFrom.Before(*f.DeliveredTo) {
		return ReportProjection{}, domainError(ValidationFailed, "送货结束时间必须晚于开始时间", "deliveredTo")
	}
	if f.ReviewedFrom != nil && f.ReviewedTo != nil && !f.ReviewedFrom.Before(*f.ReviewedTo) {
		return ReportProjection{}, domainError(ValidationFailed, "审核结束时间必须晚于开始时间", "reviewedTo")
	}
	var result ReportProjection
	err = s.readSnapshot(ctx, statuses, func(lines []ReconciliationLine) error {
		var err error
		result, err = BuildProjection(lines, query, paginate)
		return err
	})
	return result, err
}

func (s *ReconciliationService) readSnapshot(ctx context.Context, statuses []string, use func([]ReconciliationLine) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lines []ReconciliationLine
		// A single joined statement and a repeatable-read, read-only transaction
		// keep rows, grouping and summary on one audited-state snapshot.
		err := tx.Table("wh_delivery_details AS d").
			Joins("JOIN wh_delivery_notes AS n ON n.id = d.note_id").
			Where("n.deleted_at IS NULL AND d.deleted_at IS NULL AND n.status IN ?", statuses).
			Select(`n.id AS note_id, n.delivered_at, n.created_at, n.reviewed_at, n.status,
			 n.number AS note_number, n.department, n.brand_id, n.brand_code_snapshot AS brand_code, n.brand_name_snapshot AS brand_name,
			 d.material_id, d.material_code_snapshot AS material_code, d.material_name_snapshot AS material_name,
			 d.specification_snapshot AS specification, d.category_id, d.category_code_snapshot AS category_code, d.category_name_snapshot AS category_name,
			 d.unit_id, d.base_unit_id, d.unit_name_snapshot AS unit_name, d.base_unit_name_snapshot AS base_unit_name,
			 d.quantity, d.unit_price, d.base_quantity, d.amount`).
			Order("n.id, d.line_no, d.id").Scan(&lines).Error
		if err != nil {
			return err
		}
		return use(lines)
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
}

type ReportOption struct {
	ID   uint   `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}
type ReportOptions struct {
	Brands     []ReportOption `json:"brands"`
	Categories []ReportOption `json:"categories"`
	Departments []string `json:"departments"`
}

func (s *ReconciliationService) Options(ctx context.Context) (ReportOptions, error) {
	result := ReportOptions{Brands: []ReportOption{}, Categories: []ReportOption{}, Departments: []string{}}
	err := s.readSnapshot(ctx, []string{string(DeliveryDraft), string(DeliveryReviewed), string(DeliveryReReviewed)}, func(lines []ReconciliationLine) error {
		brands, categories := map[ReportOption]bool{}, map[ReportOption]bool{}
		departments := map[string]bool{}
		for _, line := range lines {
			brands[ReportOption{line.BrandID, line.BrandCode, line.BrandName}] = true
			categories[ReportOption{line.CategoryID, line.CategoryCode, line.CategoryName}] = true
			if strings.TrimSpace(line.Department) != "" {
				departments[line.Department] = true
			}
		}
		for item := range brands {
			result.Brands = append(result.Brands, item)
		}
		for item := range categories {
			result.Categories = append(result.Categories, item)
		}
		for department := range departments {
			result.Departments = append(result.Departments, department)
		}
		sort.Strings(result.Departments)
		less := func(a, b ReportOption) bool {
			if a.Code != b.Code {
				return a.Code < b.Code
			}
			if a.ID != b.ID {
				return a.ID < b.ID
			}
			return a.Name < b.Name
		}
		sort.Slice(result.Brands, func(i, j int) bool { return less(result.Brands[i], result.Brands[j]) })
		sort.Slice(result.Categories, func(i, j int) bool { return less(result.Categories[i], result.Categories[j]) })
		return nil
	})
	return result, err
}

// Export reuses the complete projection and writes every value explicitly as text.
func (s *ReconciliationService) Export(ctx context.Context, query req.ReconciliationQuery) ([]byte, error) {
	projection, err := s.Query(ctx, query, false)
	if err != nil {
		return nil, err
	}
	book := excelize.NewFile()
	defer book.Close()
	const sheet = "对账表"
	if err := book.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	columns, _ := validateColumns(projection.Columns)
	write := func(row, col int, value string) error {
		cell, err := excelize.CoordinatesToCellName(col, row)
		if err != nil {
			return err
		}
		return book.SetCellStr(sheet, cell, value)
	}
	if err := write(1, 1, "送货对账表（历史快照）"); err != nil {
		return nil, err
	}
	if err := write(2, 1, ReportFilterDescription(query.Filters)); err != nil {
		return nil, err
	}
	for i, column := range columns {
		if err := write(4, i+1, column.Label); err != nil {
			return nil, err
		}
	}
	rowNo := 5
	for _, row := range projection.List {
		for i, column := range columns {
			text := "—"
			if value := row.Values[column.ID]; value != nil {
				text = *value
			} else if reason := row.Unavailable[column.ID]; reason != "" {
				text += "（" + reason + "）"
			}
			if column.ID == "baseQuantity" && row.Values[column.ID] != nil {
				units := []string{}
				for _, bucket := range row.BaseUnits {
					units = append(units, bucket.UnitName+" #"+decimalID(bucket.UnitID))
				}
				text += "（基本单位：" + strings.Join(units, "、") + "）"
			}
			if err := write(rowNo, i+1, text); err != nil {
				return nil, err
			}
		}
		rowNo++
	}
	if err := write(rowNo+1, 1, fmt.Sprintf("全量分组数：%d", projection.Total)); err != nil {
		return nil, err
	}
	rowNo += 2
	for _, text := range ReportSummaryLines(projection.Summary) {
		if err := write(rowNo, 1, text); err != nil {
			return nil, err
		}
		rowNo++
	}
	for _, warning := range projection.Warnings {
		if err := write(rowNo, 1, warning); err != nil {
			return nil, err
		}
		rowNo++
	}
	buffer, err := book.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func ReportFilterDescription(f req.ReconciliationFilters) string {
	statuses, _ := reconciliationStatuses(f.Statuses)
	labels := map[string]string{"draft": "未审核", "reviewed": "已审核", "reReviewed": "已复审"}
	names := make([]string, 0, len(statuses))
	for _, status := range statuses {
		names = append(names, labels[status])
	}
	parts := []string{"范围：" + strings.Join(names, "、") + "、未删除；时间区间 [开始,结束)"}
	if f.DeliveredFrom != nil {
		parts = append(parts, "送货开始="+f.DeliveredFrom.Format("2006-01-02T15:04:05Z07:00"))
	}
	if f.DeliveredTo != nil {
		parts = append(parts, "送货结束="+f.DeliveredTo.Format("2006-01-02T15:04:05Z07:00"))
	}
	if f.ReviewedFrom != nil {
		parts = append(parts, "审核开始="+f.ReviewedFrom.Format("2006-01-02T15:04:05Z07:00"))
	}
	if f.ReviewedTo != nil {
		parts = append(parts, "审核结束="+f.ReviewedTo.Format("2006-01-02T15:04:05Z07:00"))
	}
	if f.BrandID != nil {
		parts = append(parts, "客户ID="+decimalID(*f.BrandID))
	}
	if f.CategoryID != nil {
		parts = append(parts, "类别ID="+decimalID(*f.CategoryID))
	}
	if f.Department != "" {
		parts = append(parts, "部门="+f.Department)
	}
	if f.Number != "" {
		parts = append(parts, "单号包含="+f.Number)
	}
	if f.MaterialKeyword != "" {
		parts = append(parts, "物料包含="+f.MaterialKeyword)
	}
	return strings.Join(parts, "；")
}
func ReportSummaryLines(s ReportSummary) []string {
	result := []string{}
	if s.Amount != nil {
		result = append(result, "全局金额合计："+*s.Amount)
	}
	if s.Quantity != nil {
		result = append(result, "全局数量合计："+*s.Quantity)
	}
	if s.BaseQuantity != nil {
		result = append(result, "全局基本数量合计："+*s.BaseQuantity)
	}
	return result
}
