package service

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/shopspring/decimal"
)

type ColumnKind string

const (
	ColumnDimension ColumnKind = "dimension"
	ColumnMetric    ColumnKind = "metric"
)

type ReportColumn struct {
	ID       string     `json:"id"`
	Label    string     `json:"label"`
	Kind     ColumnKind `json:"kind"`
	DataType string     `json:"dataType"`
	Scale    *int       `json:"scale,omitempty"`
}

type ReportRow struct {
	NoteID      uint              `json:"noteId"`
	Values      map[string]*string `json:"values"`
	Unavailable map[string]string  `json:"unavailable,omitempty"`
	BaseUnits   []QuantityBucket   `json:"baseUnits,omitempty"`
}

type QuantityBucket struct {
	MaterialID   uint   `json:"materialId"`
	MaterialCode string `json:"materialCode"`
	MaterialName string `json:"materialName"`
	UnitID       uint   `json:"unitId"`
	UnitName     string `json:"unitName"`
	Value        string `json:"value"`
}

type ReportSummary struct {
	Amount       *string `json:"amount,omitempty"`
	Quantity     *string `json:"quantity,omitempty"`
	BaseQuantity *string `json:"baseQuantity,omitempty"`
}

type ReportProjection struct {
	List     []ReportRow   `json:"list"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
	Total    int64         `json:"total"`
	Columns  []string      `json:"columns"`
	Summary  ReportSummary `json:"summary"`
	Warnings []string      `json:"warnings"`
}

type ReconciliationLine struct {
	NoteID                                    uint `gorm:"column:note_id"`
	DeliveredAt, CreatedAt                    time.Time
	ReviewedAt                                *time.Time
	Deleted                                   bool
	Status                                    DeliveryStatus
	NoteNumber                                string
	BrandID                                   uint
	Department                                string
	BrandCode, BrandName                      string
	MaterialID                                uint
	MaterialCode, MaterialName                string
	Specification                             *string
	CategoryID                                uint
	CategoryCode, CategoryName                string
	UnitID, BaseUnitID                        uint
	UnitName, BaseUnitName                    string
	CreatedBy                                 uint
	CreatedByName                             string
	ReviewedBy                                *uint
	ReviewedByName                            *string
	Quantity, UnitPrice, BaseQuantity, Amount decimal.Decimal
}

func ReconciliationColumns() []ReportColumn {
	scale := func(value int) *int { return &value }
	return []ReportColumn{
		{ID: "deliveryDay", Label: "送货日", Kind: ColumnDimension, DataType: "date"},
		{ID: "deliveryMonth", Label: "送货月", Kind: ColumnDimension, DataType: "month"},
		{ID: "noteNumber", Label: "单号", Kind: ColumnDimension, DataType: "text"},
		{ID: "brand", Label: "客户", Kind: ColumnDimension, DataType: "text"},
		{ID: "materialCode", Label: "物料编号", Kind: ColumnDimension, DataType: "text"},
		{ID: "materialName", Label: "物料名称", Kind: ColumnDimension, DataType: "text"},
		{ID: "specification", Label: "规格", Kind: ColumnDimension, DataType: "text"},
		{ID: "category", Label: "类别", Kind: ColumnDimension, DataType: "text"},
		{ID: "unit", Label: "单位", Kind: ColumnDimension, DataType: "text"},
		{ID: "department", Label: "部门", Kind: ColumnDimension, DataType: "text"},
		{ID: "creator", Label: "制单人", Kind: ColumnDimension, DataType: "text"},
		{ID: "createdAt", Label: "制单时间", Kind: ColumnDimension, DataType: "datetime"},
		{ID: "reviewer", Label: "审核人", Kind: ColumnDimension, DataType: "text"},
		{ID: "reviewedAt", Label: "审核时间", Kind: ColumnDimension, DataType: "datetime"},
		{ID: "quantity", Label: "数量", Kind: ColumnMetric, DataType: "decimal", Scale: scale(4)},
		{ID: "baseQuantity", Label: "基本数量", Kind: ColumnMetric, DataType: "decimal", Scale: scale(10)},
		{ID: "unitPrice", Label: "单价", Kind: ColumnMetric, DataType: "decimal", Scale: scale(4)},
		{ID: "amount", Label: "金额", Kind: ColumnMetric, DataType: "decimal", Scale: scale(2)},
	}
}

func BuildProjection(lines []ReconciliationLine, query req.ReconciliationQuery, paginate bool) (ReportProjection, error) {
	columns, err := validateColumns(query.Columns)
	if err != nil {
		return ReportProjection{}, err
	}
	matched := filterLines(lines, query.Filters)
	groups := map[string]*reportGroup{}
	order := []string{}
	for _, line := range matched {
		key := groupKey(line, columns)
		group, ok := groups[key]
		if !ok {
			group = &reportGroup{key: key, values: dimensionValues(line, columns)}
			groups[key] = group
			order = append(order, key)
		}
		group.lines = append(group.lines, line)
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i] < order[j] })
	rows := make([]ReportRow, 0, len(order))
	for _, key := range order {
		rows = append(rows, groups[key].row(columns))
	}
	sortErr := sortReportRows(rows, columns, query.SortBy, query.SortOrder)
	if sortErr != nil {
		return ReportProjection{}, sortErr
	}
	page := query.PageQuery.WithDefaults()
	result := ReportProjection{List: rows, Page: page.Page, PageSize: page.PageSize, Total: int64(len(rows)), Columns: columnIDs(columns), Summary: summary(matched, columns), Warnings: []string{"仅统计未删除且已审核送货单的历史快照", "导出、打印按执行时数据重新生成完整结果"}}
	if paginate {
		result.List = pageRows(rows, page.Page, page.PageSize)
	}
	return result, nil
}

func sortReportRows(rows []ReportRow, columns []ReportColumn, sortBy, sortOrder string) error {
	if sortBy == "" {
		if sortOrder != "" {
			return domainError(ValidationFailed, "排序字段不能为空", "sortBy")
		}
		return nil
	}
	var selected *ReportColumn
	for i := range columns {
		if columns[i].ID == sortBy {
			selected = &columns[i]
			break
		}
	}
	if selected == nil {
		return domainError(ValidationFailed, "排序字段必须是当前显示列", "sortBy")
	}
	order := strings.ToLower(sortOrder)
	if order != "asc" && order != "desc" && order != "ascending" && order != "descending" {
		return domainError(ValidationFailed, "排序方向无效", "sortOrder")
	}
	descending := order == "desc" || order == "descending"
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := rows[i].Values[sortBy], rows[j].Values[sortBy]
		if left == nil || right == nil {
			if left == nil && right == nil {
				return false
			}
			return left != nil
		}
		comparison := 0
		if selected.DataType == "decimal" {
			leftValue, leftErr := decimal.NewFromString(*left)
			rightValue, rightErr := decimal.NewFromString(*right)
			if leftErr == nil && rightErr == nil {
				comparison = leftValue.Cmp(rightValue)
			} else {
				comparison = strings.Compare(strings.ToLower(*left), strings.ToLower(*right))
			}
		} else {
			comparison = strings.Compare(strings.ToLower(*left), strings.ToLower(*right))
		}
		if descending {
			return comparison > 0
		}
		return comparison < 0
	})
	return nil
}

type reportGroup struct {
	key    string
	values map[string]*string
	lines  []ReconciliationLine
}

func (g *reportGroup) row(columns []ReportColumn) ReportRow {
	values := map[string]*string{}
	unavailable := map[string]string{}
	for id, value := range g.values {
		values[id] = value
	}
	quantityOK, baseOK, priceOK := comparable(g.lines)
	for _, column := range columns {
		switch column.ID {
		case "quantity":
			if quantityOK {
				values[column.ID] = stringPointer(sum(g.lines, func(line ReconciliationLine) decimal.Decimal { return line.Quantity }).StringFixed(4))
			} else {
				values[column.ID] = nil
				unavailable[column.ID] = "混合物料/单位，不可直接合计"
			}
		case "baseQuantity":
			if baseOK {
				values[column.ID] = stringPointer(sum(g.lines, func(line ReconciliationLine) decimal.Decimal { return line.BaseQuantity }).StringFixed(10))
			} else {
				values[column.ID] = nil
				unavailable[column.ID] = "混合物料/单位，不可直接合计"
			}
		case "unitPrice":
			if priceOK {
				values[column.ID] = stringPointer(g.lines[0].UnitPrice.StringFixed(4))
			} else {
				values[column.ID] = nil
				unavailable[column.ID] = "混合单价，不可直接合计"
			}
		case "amount":
			values[column.ID] = stringPointer(sum(g.lines, func(line ReconciliationLine) decimal.Decimal { return line.Amount }).StringFixed(2))
		}
	}
	if len(unavailable) == 0 {
		unavailable = nil
	}
	result := ReportRow{NoteID: g.lines[0].NoteID, Values: values, Unavailable: unavailable}
	for _, column := range columns {
		if column.ID == "baseQuantity" {
			result.BaseUnits = buckets(g.lines, true)
		}
	}
	return result
}

func validateColumns(ids []string) ([]ReportColumn, error) {
	if len(ids) == 0 {
		return nil, domainError(ValidationFailed, "至少选择一列", "columns")
	}
	index := map[string]ReportColumn{}
	for _, column := range ReconciliationColumns() {
		index[column.ID] = column
	}
	seen := map[string]struct{}{}
	result := make([]ReportColumn, 0, len(ids))
	for _, id := range ids {
		column, ok := index[id]
		if !ok {
			return nil, domainError(ValidationFailed, "包含未知列", "columns")
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, domainError(ValidationFailed, "列不能重复", "columns")
		}
		seen[id] = struct{}{}
		result = append(result, column)
	}
	return result, nil
}

func filterLines(lines []ReconciliationLine, filters req.ReconciliationFilters) []ReconciliationLine {
	result := []ReconciliationLine{}
	for _, line := range lines {
		if line.Deleted || line.Status != DeliveryReviewed {
			continue
		}
		if filters.DeliveredFrom != nil && line.DeliveredAt.Before(*filters.DeliveredFrom) {
			continue
		}
		if filters.DeliveredTo != nil && !line.DeliveredAt.Before(*filters.DeliveredTo) {
			continue
		}
		if filters.ReviewedFrom != nil && (line.ReviewedAt == nil || line.ReviewedAt.Before(*filters.ReviewedFrom)) {
			continue
		}
		if filters.ReviewedTo != nil && (line.ReviewedAt == nil || !line.ReviewedAt.Before(*filters.ReviewedTo)) {
			continue
		}
		if filters.Number != "" && !strings.Contains(strings.ToLower(line.NoteNumber), strings.ToLower(filters.Number)) {
			continue
		}
		if filters.BrandID != nil && line.BrandID != *filters.BrandID {
			continue
		}
		if filters.Department != "" && line.Department != filters.Department {
			continue
		}
		if filters.MaterialKeyword != "" && !strings.Contains(strings.ToLower(line.MaterialCode), strings.ToLower(filters.MaterialKeyword)) && !strings.Contains(strings.ToLower(line.MaterialName), strings.ToLower(filters.MaterialKeyword)) {
			continue
		}
		result = append(result, line)
	}
	return result
}

func groupKey(line ReconciliationLine, columns []ReportColumn) string {
	// Encode tuples so snapshot text cannot collide with dimension delimiters.
	values := make([][]string, 0, len(columns))
	for _, column := range columns {
		if column.Kind == ColumnDimension {
			values = append(values, dimensionKey(line, column.ID))
		}
	}
	encoded, _ := json.Marshal(values)
	return string(encoded)
}

func dimensionKey(line ReconciliationLine, id string) []string {
	// Display labels are deliberately readable, but each identity and snapshot
	// component remains separate in the key (spaces and '#' are valid data).
	switch id {
	case "department":
		return []string{id, line.Department}
	case "materialCode":
		return []string{id, line.MaterialCode, decimalID(line.MaterialID)}
	case "materialName":
		return []string{id, line.MaterialName, decimalID(line.MaterialID)}
	case "category":
		return []string{id, line.CategoryCode, line.CategoryName, decimalID(line.CategoryID)}
	case "unit":
		return []string{id, line.UnitName, decimalID(line.UnitID)}
	case "creator":
		return []string{id, line.CreatedByName, decimalID(line.CreatedBy)}
	case "reviewer":
		if line.ReviewedBy == nil || line.ReviewedByName == nil {
			return []string{id, ""}
		}
		return []string{id, *line.ReviewedByName, decimalID(*line.ReviewedBy)}
	default:
		return []string{id, dimensionValue(line, id)}
	}
}

func dimensionValues(line ReconciliationLine, columns []ReportColumn) map[string]*string {
	values := map[string]*string{}
	for _, column := range columns {
		if column.Kind == ColumnDimension {
			values[column.ID] = stringPointer(dimensionValue(line, column.ID))
		}
	}
	return values
}

func dimensionValue(line ReconciliationLine, id string) string {
	local := line.DeliveredAt.In(time.FixedZone("Asia/Shanghai", 8*3600))
	switch id {
	case "deliveryDay":
		return local.Format("2006-01-02")
	case "deliveryMonth":
		return local.Format("2006-01")
	case "noteNumber":
		return line.NoteNumber
	case "brand":
		return line.BrandName
	case "department":
		return line.Department
	case "materialCode":
		return line.MaterialCode
	case "materialName":
		return line.MaterialName
	case "specification":
		if line.Specification == nil {
			return ""
		}
		return *line.Specification
	case "category":
		return line.CategoryCode + " " + line.CategoryName + " #" + decimalID(line.CategoryID)
	case "unit":
		return line.UnitName
	case "creator":
		return line.CreatedByName + " #" + decimalID(line.CreatedBy)
	case "createdAt":
		return line.CreatedAt.UTC().Format(time.RFC3339Nano)
	case "reviewer":
		if line.ReviewedBy == nil || line.ReviewedByName == nil {
			return ""
		}
		return *line.ReviewedByName + " #" + decimalID(*line.ReviewedBy)
	case "reviewedAt":
		if line.ReviewedAt == nil {
			return ""
		}
		return line.ReviewedAt.UTC().Format(time.RFC3339Nano)
	default:
		return ""
	}
}

func comparable(lines []ReconciliationLine) (bool, bool, bool) {
	if len(lines) == 0 {
		return false, false, false
	}
	first := lines[0]
	quantity, base, price := true, true, true
	for _, line := range lines[1:] {
		if line.MaterialID != first.MaterialID || line.UnitID != first.UnitID {
			quantity = false
		}
		if line.MaterialID != first.MaterialID || line.BaseUnitID != first.BaseUnitID {
			base = false
		}
		if line.MaterialID != first.MaterialID || line.UnitID != first.UnitID || !line.UnitPrice.Equal(first.UnitPrice) {
			price = false
		}
	}
	return quantity, base, price
}

func summary(lines []ReconciliationLine, columns []ReportColumn) ReportSummary {
	result := ReportSummary{}
	selected := map[string]struct{}{}
	for _, column := range columns {
		selected[column.ID] = struct{}{}
	}
	if _, ok := selected["amount"]; ok {
		result.Amount = stringPointer(sum(lines, func(line ReconciliationLine) decimal.Decimal { return line.Amount }).StringFixed(2))
	}
	if _, ok := selected["quantity"]; ok {
		result.Quantity = stringPointer(sum(lines, func(line ReconciliationLine) decimal.Decimal { return line.Quantity }).StringFixed(4))
	}
	if _, ok := selected["baseQuantity"]; ok {
		result.BaseQuantity = stringPointer(sum(lines, func(line ReconciliationLine) decimal.Decimal { return line.BaseQuantity }).StringFixed(10))
	}
	return result
}

func buckets(lines []ReconciliationLine, base bool) []QuantityBucket {
	type key struct {
		material, unit       uint
		code, name, unitName string
	}
	grouped := map[key]QuantityBucket{}
	for _, line := range lines {
		unitID, unitName, value := line.UnitID, line.UnitName, line.Quantity
		if base {
			unitID, unitName, value = line.BaseUnitID, line.BaseUnitName, line.BaseQuantity
		}
		bucketKey := key{line.MaterialID, unitID, line.MaterialCode, line.MaterialName, unitName}
		item := grouped[bucketKey]
		current := decimal.Zero
		if item.Value != "" {
			current, _ = decimal.NewFromString(item.Value)
		}
		scale := int32(4)
		if base {
			scale = 10
		}
		item = QuantityBucket{MaterialID: line.MaterialID, MaterialCode: line.MaterialCode, MaterialName: line.MaterialName, UnitID: unitID, UnitName: unitName, Value: current.Add(value).StringFixed(scale)}
		grouped[bucketKey] = item
	}
	result := make([]QuantityBucket, 0, len(grouped))
	for _, item := range grouped {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.MaterialID != b.MaterialID {
			return a.MaterialID < b.MaterialID
		}
		if a.UnitID != b.UnitID {
			return a.UnitID < b.UnitID
		}
		if a.MaterialCode != b.MaterialCode {
			return a.MaterialCode < b.MaterialCode
		}
		if a.MaterialName != b.MaterialName {
			return a.MaterialName < b.MaterialName
		}
		return a.UnitName < b.UnitName
	})
	return result
}

func pageRows(rows []ReportRow, page, size int) []ReportRow {
	start := (page - 1) * size
	if start >= len(rows) {
		return []ReportRow{}
	}
	end := start + size
	if end > len(rows) {
		end = len(rows)
	}
	return rows[start:end]
}

func sum(lines []ReconciliationLine, value func(ReconciliationLine) decimal.Decimal) decimal.Decimal {
	total := decimal.Zero
	for _, line := range lines {
		total = total.Add(value(line))
	}
	return total
}

func columnIDs(columns []ReportColumn) []string {
	ids := make([]string, 0, len(columns))
	for _, column := range columns {
		ids = append(ids, column.ID)
	}
	return ids
}
func decimalID(value uint) string { return decimal.NewFromInt(int64(value)).String() }
