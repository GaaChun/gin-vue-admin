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
	if err != nil || len(brand.List) != 1 || value(brand.List[0], "amount") != "68.50" {
		t.Fatalf("brand projection=%#v err=%v", brand, err)
	}
	material, err := BuildProjection(lines, req.ReconciliationQuery{Columns: []string{"materialName", "quantity", "baseQuantity", "unitPrice", "amount"}}, false)
	if err != nil || value(material.List[0], "quantity") != "" || value(material.List[0], "baseQuantity") != "28.0000000000" || value(material.List[0], "unitPrice") != "" || value(material.List[0], "amount") != "68.50" {
		t.Fatalf("material projection=%#v err=%v", material, err)
	}
	units, err := BuildProjection(lines, req.ReconciliationQuery{Columns: []string{"materialName", "unit", "quantity", "unitPrice", "amount"}}, false)
	box, piece := rowByUnit(units.List, "箱"), rowByUnit(units.List, "件")
	if err != nil || len(units.List) != 2 || value(box, "quantity") != "2.0000" || value(piece, "quantity") != "4.0000" || value(piece, "unitPrice") != "" || value(box, "unitPrice") != "30.0000" {
		t.Fatalf("unit projection=%#v err=%v", units, err)
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

func TestReconciliationGlobalBucketsIgnorePage(t *testing.T) {
	lines := sampleLines()
	result, err := BuildProjection(lines, req.ReconciliationQuery{Columns: []string{"unit", "quantity", "amount"}, PageQuery: req.PageQuery{Page: 1, PageSize: 1}}, true)
	if err != nil || len(result.List) != 1 || result.Total != 2 || result.Summary.Amount == nil || *result.Summary.Amount != "68.50" || len(result.Summary.QuantityBuckets) != 2 {
		t.Fatalf("paged summary=%#v err=%v", result, err)
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
	return ReconciliationLine{DeliveredAt: time.Date(2026, 10, 15, 2, 0, 0, 0, time.UTC), Status: DeliveryReviewed, BrandID: 1, BrandCode: "A", BrandName: "品牌A", MaterialID: 1, MaterialCode: "M", MaterialName: "物料M", UnitID: unitID, UnitName: unit, BaseUnitID: 1, BaseUnitName: "件", Quantity: decimal.RequireFromString(quantity), UnitPrice: decimal.RequireFromString(price), Amount: decimal.RequireFromString(amount), BaseQuantity: decimal.RequireFromString(base)}
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
