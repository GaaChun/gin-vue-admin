package service

import (
	"testing"
	"time"

	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/shopspring/decimal"
)

func TestReconciliationSampleGroupsBySelectedDimensions(t *testing.T) {
	lines := sampleLines()
	brand, err := BuildProjection(lines, req.ReconciliationQuery{Columns: []string{"brand", "amount"}}, false)
	if err != nil || len(brand.List) != 1 || value(brand.List[0], "brand") != "客户A" || value(brand.List[0], "amount") != "68.50" {
		t.Fatalf("brand projection=%#v err=%v", brand, err)
	}
	material, err := BuildProjection(lines, req.ReconciliationQuery{Columns: []string{"materialName", "quantity", "baseQuantity", "unitPrice", "amount"}}, false)
	if err != nil || value(material.List[0], "quantity") != "" || value(material.List[0], "baseQuantity") != "28.0000000000" || value(material.List[0], "unitPrice") != "" || value(material.List[0], "amount") != "68.50" {
		t.Fatalf("material projection=%#v err=%v", material, err)
	}
	units, err := BuildProjection(lines, req.ReconciliationQuery{Columns: []string{"materialCode", "materialName", "unit", "quantity", "unitPrice", "amount"}}, false)
	box, piece := rowByUnit(units.List, "箱"), rowByUnit(units.List, "件")
	if err != nil || len(units.List) != 2 || value(box, "unit") != "箱" || value(piece, "unit") != "件" || value(box, "materialCode") != "M" || value(box, "materialName") != "物料M" || value(piece, "quantity") != "4.0000" || value(box, "quantity") != "2.0000" || value(piece, "unitPrice") != "" || value(box, "unitPrice") != "30.0000" {
		t.Fatalf("unit projection=%#v err=%v", units, err)
	}
}

func TestReconciliationGroupsSameMaterialByDepartment(t *testing.T) {
	first := line("件", 1, "2", "3", "6.00", "2")
	first.Department = "装配部"
	second := line("件", 1, "4", "3", "12.00", "4")
	second.Department = "仓储部"

	result, err := BuildProjection([]ReconciliationLine{first, second}, req.ReconciliationQuery{
		Columns: []string{"materialName", "department", "quantity", "amount"},
	}, false)
	if err != nil || len(result.List) != 2 {
		t.Fatalf("department projection=%#v err=%v", result, err)
	}
	rows := map[string]ReportRow{}
	for _, row := range result.List {
		rows[value(row, "department")] = row
	}
	if value(rows["装配部"], "materialName") != "物料M" || value(rows["装配部"], "quantity") != "2.0000" ||
		value(rows["仓储部"], "materialName") != "物料M" || value(rows["仓储部"], "quantity") != "4.0000" {
		t.Fatalf("department groups=%#v", result.List)
	}
}

func TestReconciliationFiltersByExactDepartment(t *testing.T) {
	first := line("件", 1, "2", "3", "6.00", "2")
	first.Department = "装配部"
	second := line("件", 1, "4", "3", "12.00", "4")
	second.Department = "仓储部"
	result, err := BuildProjection([]ReconciliationLine{first, second}, req.ReconciliationQuery{
		Filters: req.ReconciliationFilters{Department: "仓储部"},
		Columns: []string{"department", "amount"},
	}, false)
	if err != nil || result.Total != 1 || len(result.List) != 1 ||
		value(result.List[0], "department") != "仓储部" || value(result.List[0], "amount") != "12.00" {
		t.Fatalf("department filter projection=%#v err=%v", result, err)
	}
}

func TestReconciliationRejectsInvalidColumnsAndDrafts(t *testing.T) {
	lines := append(sampleLines(), ReconciliationLine{Status: DeliveryDraft, Amount: decimal.RequireFromString("100")})
	if _, err := BuildProjection(lines, req.ReconciliationQuery{Columns: []string{}}, false); err == nil {
		t.Fatal("empty columns accepted")
	}
	if _, err := BuildProjection(lines, req.ReconciliationQuery{Columns: []string{"amount", "drop table"}}, false); err == nil {
		t.Fatal("unknown column accepted")
	}
	result, err := BuildProjection(lines, req.ReconciliationQuery{Columns: []string{"amount"}}, false)
	if err != nil || len(result.List) != 1 || value(result.List[0], "amount") != "68.50" {
		t.Fatalf("draft leaked into amount=%#v err=%v", result, err)
	}
}

func TestReconciliationSummaryTotalsIgnorePagination(t *testing.T) {
	lines := sampleLines()
	result, err := BuildProjection(lines, req.ReconciliationQuery{Columns: []string{"unit", "quantity", "baseQuantity", "amount"}, PageQuery: req.PageQuery{Page: 1, PageSize: 1}}, true)
	if err != nil || len(result.List) != 1 || result.Total != 2 ||
		result.Summary.Amount == nil || *result.Summary.Amount != "68.50" ||
		result.Summary.Quantity == nil || *result.Summary.Quantity != "6.0000" ||
		result.Summary.BaseQuantity == nil || *result.Summary.BaseQuantity != "28.0000000000" {
		t.Fatalf("paged summary=%#v err=%v", result, err)
	}
}

func TestReconciliationSortsAllGroupsBeforePagination(t *testing.T) {
	result, err := BuildProjection(sampleLines(), req.ReconciliationQuery{
		Columns:   []string{"unit", "amount"},
		PageQuery: req.PageQuery{Page: 2, PageSize: 1},
		SortQuery: req.SortQuery{SortBy: "amount", SortOrder: "descending"},
	}, true)
	if err != nil || result.Total != 2 || len(result.List) != 1 || value(result.List[0], "amount") != "8.50" {
		t.Fatalf("expected second page of descending all-group sort, got %#v, err %v", result, err)
	}
	if _, err := BuildProjection(sampleLines(), req.ReconciliationQuery{
		Columns:   []string{"unit", "amount"},
		SortQuery: req.SortQuery{SortBy: "quantity", SortOrder: "asc"},
	}, true); err == nil {
		t.Fatal("sort field outside visible columns was accepted")
	}
}

func sampleLines() []ReconciliationLine {
	return []ReconciliationLine{
		line("箱", 2, "2", "30", "60.00", "24"),
		line("件", 1, "3", "2", "6.00", "3"),
		line("件", 1, "1", "2.5", "2.50", "1"),
	}
}

func line(unit string, unitID uint, quantity, price, amount, base string) ReconciliationLine {
	return ReconciliationLine{DeliveredAt: time.Date(2026, 10, 15, 2, 0, 0, 0, time.UTC), Status: DeliveryReviewed, BrandID: 1, BrandCode: "A", BrandName: "客户A", MaterialID: 1, MaterialCode: "M", MaterialName: "物料M", UnitID: unitID, UnitName: unit, BaseUnitID: 1, BaseUnitName: "件", Quantity: decimal.RequireFromString(quantity), UnitPrice: decimal.RequireFromString(price), Amount: decimal.RequireFromString(amount), BaseQuantity: decimal.RequireFromString(base)}
}

func value(row ReportRow, id string) string {
	item := row.Values[id]
	if item == nil {
		return ""
	}
	return *item
}

func rowByUnit(rows []ReportRow, name string) ReportRow {
	for _, row := range rows {
		if row.Values["unit"] != nil && len(*row.Values["unit"]) >= len(name) && (*row.Values["unit"])[:len(name)] == name {
			return row
		}
	}
	return ReportRow{}
}
