package service

import (
	"bytes"
	"strings"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

func TestMasterWorkbookRejectsMalformedCategoryFile(t *testing.T) {
	contents := workbookWithSheets(t, map[string][][]string{
		"类别": {
			{"编号*", "类别名称*", "助记符*", "上级类别编号*", "版本号"},
			{"ROOT", "根", "g", "ROOT", "0"},
			{"C1", "子", "z", "MISSING", "1"},
			{"=C2", "循环", "x", "C1", ""},
		},
	})
	_, err := parseMasterWorkbook("categories", contents)
	domain, ok := AsDomainError(err)
	if !ok || domain.Code != ImportFailed {
		t.Fatalf("malformed workbook error = %#v", domain.BusinessError())
	}
	found := map[string]bool{}
	for _, item := range domain.Errors {
		found[item.Column+":"+item.Message] = true
		if item.Sheet != "类别" || item.LineNo == 0 {
			t.Fatalf("error location missing: %#v", item)
		}
	}
	for _, want := range []string{"编号:类别编号不能为ROOT", "版本号:版本号必须大于0", "编号:不导入公式单元格"} {
		if !found[want] {
			t.Fatalf("missing %s in %#v", want, domain.BusinessError())
		}
	}
}

func TestCategoryForwardReferenceImportsInParentOrder(t *testing.T) {
	state := NewMasterExcelState()
	contents := workbookWithSheets(t, map[string][][]string{
		"类别": {
			{"编号*", "类别名称*", "助记符*", "上级类别编号*", "版本号"},
			{"CHILD", "子类", "zl", "PARENT", ""},
			{"PARENT", "父类", "fl", "ROOT", ""},
		},
	})
	result, err := state.Import("categories", contents, 7)
	if err != nil || result.CreatedCount != 2 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	child, _ := state.categoryByCode("CHILD")
	parent, _ := state.categoryByCode("PARENT")
	if child.ParentID != parent.ID || parent.ParentID != 0 {
		t.Fatalf("forward parent was not resolved: child=%d parent=%d", child.ParentID, parent.ID)
	}
}

func TestCategoryCycleRejectsWholeFile(t *testing.T) {
	state := NewMasterExcelState()
	original := state.AddCategory("KEEP", "保留", "bl", 0, 1)
	contents := workbookWithSheets(t, map[string][][]string{
		"类别": {
			{"编号*", "类别名称*", "助记符*", "上级类别编号*", "版本号"},
			{"A", "甲", "j", "B", ""},
			{"B", "乙", "y", "A", ""},
		},
	})
	if _, err := state.Import("categories", contents, 2); err == nil {
		t.Fatal("category cycle accepted")
	}
	if got := state.categories[original.ID]; got.Name != "保留" || len(state.categories) != 1 {
		t.Fatalf("failed import changed state: %#v", state.categories)
	}
}

func TestMasterImportRollsBackAfterVersionConflict(t *testing.T) {
	state := NewMasterExcelState()
	existing := state.AddBrand("B001", "旧客户", "jkh", 3)
	contents := workbookWithSheets(t, map[string][][]string{
		"客户": {
			{"编号*", "客户名称*", "助记符*", "版本号"},
			{"B002", "新客户", "xkh", ""},
			{"B001", "修改客户", "xkh", "2"},
		},
	})
	_, err := state.Import("brands", contents, 9)
	domain, ok := AsDomainError(err)
	if !ok || domain.Code != VersionConflict || domain.LineNo != 3 || domain.CurrentVersion != 1 {
		t.Fatalf("version conflict = %#v", err)
	}
	if _, found := state.brandByCode("B002"); found || state.brands[existing.ID].Name != "旧客户" || state.brands[existing.ID].CreatedBy != 3 {
		t.Fatalf("failed file committed changes: %#v", state.brands)
	}
}

func TestMaterialWorkbookValidatesCompleteUnitSet(t *testing.T) {
	contents := workbookWithSheets(t, map[string][][]string{
		"物料": {{"编号*", "物料名称*", "助记符*", "规格", "别名", "客户编号*", "类别编号*", "图片文件ID", "版本号"}, {"M1", "物料", "wl", "", "", "B1", "C1", "", ""}, {"M2", "物料二", "we", "", "", "B1", "C1", "", ""}},
		"物料单位": {{"物料编号*", "单位名称*", "换算系数*", "是否基本单位*", "是否启用*"}, {"UNKNOWN", "个", "1", "是", "是"}, {"M1", "个", "1", "是", "是"}, {"M1", "个", "2", "否", "是"}},
	})
	_, err := parseMasterWorkbook("materials", contents)
	domain, ok := AsDomainError(err)
	if !ok {
		t.Fatal(err)
	}
	messages := map[string]bool{}
	for _, item := range domain.Errors {
		messages[item.Sheet+":"+item.Column+":"+item.Message] = true
	}
	if !messages["物料单位:物料编号:物料主表中不存在"] || !messages["物料:编号:缺少完整单位集合"] || !messages["物料单位:单位名称:与第3行重复"] {
		t.Fatalf("unit-set problems = %#v", domain.BusinessError())
	}
}

func TestMaterialImportPreservesReferencedBindingAndCreationAudit(t *testing.T) {
	state := NewMasterExcelState()
	customer := state.AddBrand("B001", "客户", "kh", 1)
	category := state.AddCategory("C001", "类别", "lb", 0, 1)
	unit := state.AddUnit("个", 1)
	box := state.AddUnit("箱", 1)
	material := state.AddMaterial(model.Material{Code: "M001", Name: "旧物料", Mnemonic: "wl", BrandID: customer.ID, CategoryID: category.ID}, []model.MaterialUnit{{UnitID: unit.ID, Factor: decimal.NewFromInt(1), IsBase: true, Enabled: true}}, 4)
	bindingID := onlyBindingID(state)
	state.ReferenceBinding(bindingID)
	rejected := workbookWithSheets(t, map[string][][]string{
		"物料": {{"编号*", "物料名称*", "助记符*", "规格", "别名", "客户编号*", "类别编号*", "图片文件ID", "版本号"}, {"M001", "新物料", "xwl", "", "", "B001", "C001", "", "1"}, {"M002", "新物料二", "xe", "", "", "B001", "C001", "", ""}},
		"物料单位": {{"物料编号*", "单位名称*", "换算系数*", "是否基本单位*", "是否启用*"}, {"M001", "箱", "1", "是", "是"}, {"M002", "个", "1", "是", "是"}},
	})
	if _, err := state.Import("materials", rejected, 8); err == nil {
		t.Fatal("referenced binding omission accepted")
	}
	if got := state.materials[material.ID]; got.Name != "旧物料" || got.CreatedBy != 4 || len(state.materials) != 1 {
		t.Fatalf("failed material import changed aggregate: %#v", state.materials)
	}
	accepted := workbookWithSheets(t, map[string][][]string{
		"物料": {{"编号*", "物料名称*", "助记符*", "规格", "别名", "客户编号*", "类别编号*", "图片文件ID", "版本号"}, {"M001", "普通文本", "wb", "规格", "", "B001", "C001", "", "1"}, {"M002", "新物料", "xwl", "", "", "B001", "C001", "", ""}},
		"物料单位": {{"物料编号*", "单位名称*", "换算系数*", "是否基本单位*", "是否启用*"}, {"M001", "个", "1", "是", "是"}, {"M001", "箱", "12.5", "否", "是"}, {"M002", "个", "1", "是", "是"}},
	})
	result, err := state.Import("materials", accepted, 8)
	if err != nil || result.CreatedCount != 1 || result.UpdatedCount != 1 {
		domain, _ := AsDomainError(err)
		t.Fatalf("result=%#v err=%#v", result, domain.BusinessError())
	}
	updated := state.materials[material.ID]
	if updated.Name != "普通文本" || updated.CreatedBy != 4 || updated.Version != 2 {
		t.Fatalf("update changed creation audit or text: %#v", updated)
	}
	preserved := state.bindings[bindingID]
	if preserved.ID != bindingID || preserved.UnitID != unit.ID || !preserved.IsBase {
		t.Fatalf("referenced binding rebuilt: %#v", preserved)
	}
	if !bindingFor(state, material.ID, box.ID).Factor.Equal(decimal.RequireFromString("12.5")) {
		t.Fatal("auxiliary factor precision lost")
	}
}

func TestMaterialImportAllowsBlankCustomerAndCategory(t *testing.T) {
	state := NewMasterExcelState()
	unit := state.AddUnit("个", 1)
	contents := workbookWithSheets(t, map[string][][]string{
		"物料": {{"编号*", "物料名称*", "助记符*", "规格", "别名", "客户编号", "类别编号", "图片文件ID", "版本号"}, {"M-OPTIONAL", "无分类物料", "wfl", "", "", "", "", "", ""}},
		"物料单位": {{"物料编号*", "单位名称*", "换算系数*", "是否基本单位*", "是否启用*"}, {"M-OPTIONAL", "个", "1", "是", "是"}},
	})
	result, err := state.Import("materials", contents, 1)
	if err != nil || result.CreatedCount != 1 {
		t.Fatalf("import failed: result=%#v err=%v", result, err)
	}
	material, found := state.materialByCode("M-OPTIONAL")
	if !found || material.BrandID != 0 || material.CategoryID != 0 || bindingFor(state, material.ID, unit.ID).ID == 0 {
		t.Fatalf("optional-reference material not stored correctly: %#v", material)
	}
}

func TestMasterExportPreservesTextAndMaterialUnits(t *testing.T) {
	state := NewMasterExcelState()
	template, err := masterTemplate("materials")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseMasterWorkbook("materials", template); err != nil {
		t.Fatalf("material template headings were not importable: %v", err)
	}
	templateBook, err := excelize.OpenReader(bytes.NewReader(template))
	if err != nil {
		t.Fatal(err)
	}
	defer templateBook.Close()
	if active := templateBook.GetSheetName(templateBook.GetActiveSheetIndex()); active != materialSheetName {
		t.Fatalf("material template opens on %q, want %q", active, materialSheetName)
	}
	customer := state.AddBrand("001", "=客户", "kh", 1)
	category := state.AddCategory("010", "类别", "lb", 0, 1)
	unit := state.AddUnit("个", 1)
	state.AddMaterial(model.Material{Code: "M001", Name: "=物料", Mnemonic: "wl", BrandID: customer.ID, CategoryID: category.ID}, []model.MaterialUnit{{UnitID: unit.ID, Factor: decimal.NewFromInt(1), IsBase: true, Enabled: true}}, 1)
	contents, err := state.Export("materials")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseMasterWorkbook("materials", contents)
	if err != nil {
		domain, _ := AsDomainError(err)
		t.Fatalf("exported workbook rejected: %#v", domain.BusinessError())
	}
	if parsed.materials[0].code != "M001" || parsed.materials[0].name != "=物料" || parsed.materials[0].brandCode != "001" {
		t.Fatalf("exported material changed text: %#v", parsed.materials[0])
	}
	if len(parsed.materialUnits) != 1 || parsed.materialUnits[0].factor.StringFixed(6) != "1.000000" {
		t.Fatalf("exported unit relation changed: %#v", parsed.materialUnits)
	}
	brandFile, err := state.Export("brands")
	if err != nil {
		t.Fatal(err)
	}
	customers, err := parseMasterWorkbook("brands", brandFile)
	if err != nil || customers.brands[0].code != "001" || customers.brands[0].name != "=客户" {
		t.Fatalf("customer export changed text or headings: %#v %v", customers.brands, err)
	}
}

func workbookWithSheets(t *testing.T, sheets map[string][][]string) []byte {
	t.Helper()
	book := excelize.NewFile()
	defer book.Close()
	first := true
	for name, rows := range sheets {
		if first {
			_ = book.SetSheetName("Sheet1", name)
			first = false
		} else {
			_, _ = book.NewSheet(name)
		}
		for rowIndex, row := range rows {
			for column, value := range row {
				cell, _ := excelize.CoordinatesToCellName(column+1, rowIndex+1)
				if strings.HasPrefix(value, "=") {
					if err := book.SetCellFormula(name, cell, strings.TrimPrefix(value, "=")); err != nil {
						t.Fatal(err)
					}
					continue
				}
				if err := book.SetCellStr(name, cell, value); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	var output bytes.Buffer
	if err := book.Write(&output); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func onlyBindingID(state *MasterExcelState) uint {
	for id := range state.bindings {
		return id
	}
	return 0
}

func bindingFor(state *MasterExcelState, materialID, unitID uint) model.MaterialUnit {
	for _, binding := range state.bindings {
		if binding.MaterialID == materialID && binding.UnitID == unitID {
			return binding
		}
	}
	return model.MaterialUnit{}
}
func TestUnitTemplateContainsHeadersAndImportsExample(t *testing.T) {
	contents, err := MasterTemplate("units")
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(contents))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	if got := book.GetSheetList(); len(got) != 1 || got[0] != unitSheetName {
		t.Fatalf("template sheets = %#v", got)
	}
	rows, err := book.GetRows(unitSheetName)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 || strings.Join(rows[0], "|") != "单位名称*|版本号" || rows[1][0] != "个" {
		t.Fatalf("template rows = %#v", rows)
	}
	state := NewMasterExcelState()
	result, err := state.Import("units", contents, 1)
	if err != nil || result.CreatedCount != 1 {
		t.Fatalf("template import result=%#v err=%v", result, err)
	}
	if _, found := state.unitByName("个"); !found {
		t.Fatal("template example unit was not imported")
	}
}
